package semconnect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	readmodel "github.com/c360studio/semops/internal/egress/csapi"
)

const (
	MediaJSON = "application/json"
	MediaOMS  = "application/om+json"

	ResourceSystem      = "System"
	ResourceDeployment  = "Deployment"
	ResourceDatastream  = "Datastream"
	ResourceObservation = "Observation"
	ResourceSystemEvent = "SystemEvent"
)

const (
	defaultSystemIDPrefix     = "c360.semconnect.systems.csapi.system"
	defaultDatastreamIDPrefix = "c360.semconnect.systems.csapi.datastream"
	defaultDeploymentIDPrefix = "c360.semconnect.systems.csapi.deployment"
	defaultSystemEventPrefix  = "c360.semconnect.systems.csapi.systemevent"
)

var (
	tokenChars    = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	entityIDToken = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
)

type Prefixes struct {
	SystemIDPrefix      string `json:"system_id_prefix"`
	DatastreamIDPrefix  string `json:"datastream_id_prefix"`
	DeploymentIDPrefix  string `json:"deployment_id_prefix"`
	SystemEventIDPrefix string `json:"system_event_id_prefix"`
}

type Option func(*config)

type config struct {
	prefixes Prefixes
}

type ReadSidePlan struct {
	GeneratedAt      time.Time                   `json:"generated_at"`
	ClaimScope       string                      `json:"claim_scope"`
	Prefixes         Prefixes                    `json:"prefixes"`
	IDs              IDMap                       `json:"ids"`
	Requests         []Request                   `json:"requests"`
	DeferredSurfaces []readmodel.DeferredSurface `json:"deferred_surfaces,omitempty"`
	Warnings         []string                    `json:"warnings,omitempty"`
}

type IDMap struct {
	Systems      map[string]string `json:"systems,omitempty"`
	Datastreams  map[string]string `json:"datastreams,omitempty"`
	Deployments  map[string]string `json:"deployments,omitempty"`
	Observations map[string]string `json:"observations,omitempty"`
	SystemEvents map[string]string `json:"system_events,omitempty"`
}

type Request struct {
	Resource         string            `json:"resource"`
	SourceID         string            `json:"source_id"`
	SemConnectID     string            `json:"semconnect_id"`
	Method           string            `json:"method"`
	Path             string            `json:"path"`
	ContentType      string            `json:"content_type"`
	Body             any               `json:"body"`
	ProvenanceOwner  string            `json:"provenance_owner,omitempty"`
	ProvenanceSource string            `json:"provenance_source,omitempty"`
	ProvenanceRef    string            `json:"provenance_ref,omitempty"`
	DependsOn        []string          `json:"depends_on,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type featureBody struct {
	Type       string              `json:"type"`
	Geometry   *readmodel.Geometry `json:"geometry,omitempty"`
	Properties featureProperties   `json:"properties"`
}

type featureProperties struct {
	UID                  string `json:"uid"`
	Name                 string `json:"name,omitempty"`
	Description          string `json:"description,omitempty"`
	Definition           string `json:"definition,omitempty"`
	DeployedSystemsLinks []link `json:"deployedSystems@link,omitempty"`
}

type link struct {
	Href  string `json:"href"`
	Rel   string `json:"rel,omitempty"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

type datastreamBody struct {
	ID               string `json:"id"`
	Type             string `json:"type,omitempty"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	System           string `json:"system"`
	SystemID         string `json:"system@id,omitempty"`
	ObservedProperty string `json:"observedProperty"`
	OutputName       string `json:"outputName,omitempty"`
	ResultType       string `json:"resultType,omitempty"`
}

type observationBody struct {
	ID               string `json:"id,omitempty"`
	Procedure        string `json:"procedure"`
	ObservedProperty string `json:"observedProperty"`
	ResultTime       string `json:"resultTime"`
	PhenomenonTime   string `json:"phenomenonTime,omitempty"`
	Result           any    `json:"result"`
}

type systemEventBody struct {
	ID          string         `json:"id,omitempty"`
	Time        string         `json:"time,omitempty"`
	EventTime   string         `json:"eventTime,omitempty"`
	EventType   string         `json:"eventType,omitempty"`
	Message     string         `json:"message,omitempty"`
	Description string         `json:"description,omitempty"`
	SystemID    string         `json:"system@id,omitempty"`
	Severity    string         `json:"severity,omitempty"`
	Source      string         `json:"source,omitempty"`
	Keywords    []string       `json:"keywords,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

func DefaultPrefixes() Prefixes {
	return Prefixes{
		SystemIDPrefix:      defaultSystemIDPrefix,
		DatastreamIDPrefix:  defaultDatastreamIDPrefix,
		DeploymentIDPrefix:  defaultDeploymentIDPrefix,
		SystemEventIDPrefix: defaultSystemEventPrefix,
	}
}

func WithPrefixes(prefixes Prefixes) Option {
	return func(cfg *config) {
		cfg.prefixes = prefixes
	}
}

func BuildReadSidePlan(catalog readmodel.Catalog, options ...Option) (ReadSidePlan, error) {
	cfg := config{prefixes: DefaultPrefixes()}
	for _, opt := range options {
		opt(&cfg)
	}
	if err := validatePrefixes(cfg.prefixes); err != nil {
		return ReadSidePlan{}, err
	}

	ids := IDMap{
		Systems:      make(map[string]string, len(catalog.Systems)),
		Datastreams:  make(map[string]string, len(catalog.Datastreams)),
		Deployments:  make(map[string]string, len(catalog.Deployments)),
		Observations: make(map[string]string, len(catalog.Observations)),
		SystemEvents: make(map[string]string, len(catalog.SystemEvents)),
	}
	for _, system := range catalog.Systems {
		ids.Systems[system.ID] = cfg.prefixes.SystemIDPrefix + "." + stableToken(system.ID)
	}
	for _, stream := range catalog.Datastreams {
		ids.Datastreams[stream.ID] = cfg.prefixes.DatastreamIDPrefix + "." + stableToken(stream.ID)
	}
	for _, deployment := range catalog.Deployments {
		ids.Deployments[deployment.ID] = cfg.prefixes.DeploymentIDPrefix + "." + stableToken(deployment.ID)
	}
	for _, observation := range catalog.Observations {
		ids.Observations[observation.ID] = stableToken(observation.ID)
	}
	for _, event := range catalog.SystemEvents {
		ids.SystemEvents[event.ID] = cfg.prefixes.SystemEventIDPrefix + "." + stableToken(event.ID)
	}

	streamProperties := make(map[string]string, len(catalog.Datastreams))
	for _, stream := range catalog.Datastreams {
		streamProperties[stream.ID] = stream.ObservedProperty
	}

	var requests []Request
	var warnings []string
	for _, system := range catalog.Systems {
		id := ids.Systems[system.ID]
		requests = append(requests, Request{
			Resource:         ResourceSystem,
			SourceID:         system.ID,
			SemConnectID:     id,
			Method:           "POST",
			Path:             "/systems",
			ContentType:      MediaJSON,
			Body:             systemFeature(system, id),
			ProvenanceOwner:  system.Provenance.Owner,
			ProvenanceSource: system.Source,
			ProvenanceRef:    system.Provenance.SourceRef,
		})
	}
	for _, deployment := range catalog.Deployments {
		systemID, ok := ids.Systems[deployment.SystemID]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("deployment %q skipped: system %q not in catalog", deployment.ID, deployment.SystemID))
			continue
		}
		id := ids.Deployments[deployment.ID]
		requests = append(requests, Request{
			Resource:         ResourceDeployment,
			SourceID:         deployment.ID,
			SemConnectID:     id,
			Method:           "POST",
			Path:             "/deployments",
			ContentType:      MediaJSON,
			Body:             deploymentFeature(deployment, id, systemID),
			ProvenanceOwner:  deployment.Provenance.Owner,
			ProvenanceSource: deployment.Source,
			ProvenanceRef:    deployment.Provenance.SourceRef,
			DependsOn:        []string{systemID},
		})
	}
	for _, stream := range catalog.Datastreams {
		systemID, ok := ids.Systems[stream.SystemID]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("datastream %q skipped: system %q not in catalog", stream.ID, stream.SystemID))
			continue
		}
		id := ids.Datastreams[stream.ID]
		requests = append(requests, Request{
			Resource:         ResourceDatastream,
			SourceID:         stream.ID,
			SemConnectID:     id,
			Method:           "POST",
			Path:             "/datastreams",
			ContentType:      MediaJSON,
			Body:             datastreamRequestBody(stream, id, systemID),
			ProvenanceOwner:  stream.Provenance.Owner,
			ProvenanceSource: stream.Source,
			ProvenanceRef:    stream.Provenance.SourceRef,
			DependsOn:        []string{systemID},
		})
	}
	for _, observation := range catalog.Observations {
		streamID, ok := ids.Datastreams[observation.DatastreamID]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("observation %q skipped: datastream %q not in catalog", observation.ID, observation.DatastreamID))
			continue
		}
		systemID := ids.Systems[observation.SystemID]
		requests = append(requests, Request{
			Resource:         ResourceObservation,
			SourceID:         observation.ID,
			SemConnectID:     ids.Observations[observation.ID],
			Method:           "POST",
			Path:             "/datastreams/" + streamID + "/observations",
			ContentType:      MediaOMS,
			Body:             observationRequestBody(observation, ids.Observations[observation.ID], streamProperties[observation.DatastreamID]),
			ProvenanceOwner:  observation.Provenance.Owner,
			ProvenanceSource: observation.Source,
			ProvenanceRef:    observation.Provenance.SourceRef,
			DependsOn:        compactStrings(streamID, systemID),
			Metadata: map[string]string{
				"semops_entity_id":   observation.SemOpsEntityID,
				"semops_entity_type": observation.SemOpsEntityType,
				"claim_posture":      observation.ClaimPosture,
			},
		})
	}
	for _, event := range catalog.SystemEvents {
		systemID, ok := ids.Systems[event.SystemID]
		if event.SystemID != "" && !ok {
			warnings = append(warnings, fmt.Sprintf("system event %q skipped: system %q not in catalog", event.ID, event.SystemID))
			continue
		}
		id := ids.SystemEvents[event.ID]
		requests = append(requests, Request{
			Resource:        ResourceSystemEvent,
			SourceID:        event.ID,
			SemConnectID:    id,
			Method:          "POST",
			Path:            "/systemEvents",
			ContentType:     MediaJSON,
			Body:            systemEventRequestBody(event, id, systemID),
			ProvenanceOwner: event.Provenance.Owner,
			ProvenanceRef:   event.Provenance.SourceRef,
			DependsOn:       compactStrings(systemID),
		})
	}

	return ReadSidePlan{
		GeneratedAt:      catalog.GeneratedAt,
		ClaimScope:       catalog.ClaimScope,
		Prefixes:         cfg.prefixes,
		IDs:              ids,
		Requests:         requests,
		DeferredSurfaces: append([]readmodel.DeferredSurface(nil), catalog.DeferredSurfaces...),
		Warnings:         warnings,
	}, nil
}

func systemFeature(system readmodel.System, semConnectID string) featureBody {
	return featureBody{
		Type:     "Feature",
		Geometry: system.Location,
		Properties: featureProperties{
			UID:         lastToken(semConnectID),
			Name:        system.Name,
			Description: system.Description,
			Definition:  "https://c360.studio/semops/cop/system",
		},
	}
}

func deploymentFeature(deployment readmodel.Deployment, semConnectID, systemID string) featureBody {
	return featureBody{
		Type: "Feature",
		Properties: featureProperties{
			UID:         lastToken(semConnectID),
			Name:        deployment.Name,
			Description: "SemOps deployment projection for " + deployment.SystemID,
			Definition:  "https://c360.studio/semops/cop/deployment",
			DeployedSystemsLinks: []link{{
				Href:  "/systems/" + systemID,
				Rel:   "deployedSystem",
				Type:  MediaJSON,
				Title: systemID,
			}},
		},
	}
}

func datastreamRequestBody(stream readmodel.Datastream, semConnectID, systemID string) datastreamBody {
	return datastreamBody{
		ID:               semConnectID,
		Type:             "Datastream",
		Name:             stream.Name,
		Description:      fmt.Sprintf("SemOps %s datastream projected from %s", stream.ObservedProperty, stream.Source),
		System:           systemID,
		SystemID:         systemID,
		ObservedProperty: observedPropertyIRI(stream.ObservedProperty),
		OutputName:       "result",
		ResultType:       stream.ResultType,
	}
}

func observationRequestBody(observation readmodel.Observation, observationID, observedProperty string) observationBody {
	result := copyResult(observation.Result)
	if observation.Geometry != nil {
		result["geometry"] = observation.Geometry
	}
	if observation.ClaimPosture != "" {
		result["claim_posture"] = observation.ClaimPosture
	}
	if observation.SemOpsEntityID != "" {
		result["semops_entity_id"] = observation.SemOpsEntityID
	}
	if observation.SemOpsEntityType != "" {
		result["semops_entity_type"] = observation.SemOpsEntityType
	}
	return observationBody{
		ID:               observationID,
		Procedure:        procedureIRI(observation.Source),
		ObservedProperty: observedPropertyIRI(observedProperty),
		ResultTime:       formatTime(observation.PhenomenonTime),
		PhenomenonTime:   formatTime(observation.PhenomenonTime),
		Result:           result,
	}
}

func systemEventRequestBody(event readmodel.SystemEvent, semConnectID, systemID string) systemEventBody {
	payload := map[string]any{}
	if event.Status != "" {
		payload["status"] = event.Status
	}
	return systemEventBody{
		ID:        semConnectID,
		Time:      formatTime(event.ObservedAt),
		EventTime: formatTime(event.ObservedAt),
		EventType: firstNonEmpty(event.EventType, "SystemChanged"),
		Message:   event.Message,
		SystemID:  systemID,
		Severity:  event.Severity,
		Source:    "semops",
		Keywords:  []string{"semops", "cop", stableToken(event.EventType)},
		Payload:   payload,
	}
}

func observedPropertyIRI(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "state"
	}
	if strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "urn:") {
		return value
	}
	return "https://c360.studio/semops/cop/observed-property/" + stableToken(value)
}

func procedureIRI(source string) string {
	source = firstNonEmpty(source, "semops")
	return "https://c360.studio/semops/procedure/" + stableToken(source)
}

func validatePrefixes(prefixes Prefixes) error {
	checks := map[string]string{
		"system_id_prefix":       prefixes.SystemIDPrefix,
		"datastream_id_prefix":   prefixes.DatastreamIDPrefix,
		"deployment_id_prefix":   prefixes.DeploymentIDPrefix,
		"system_event_id_prefix": prefixes.SystemEventIDPrefix,
	}
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := validatePrefix(checks[name]); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func validatePrefix(prefix string) error {
	tokens := strings.Split(prefix, ".")
	if len(tokens) != 5 {
		return fmt.Errorf("must be 5 dotted tokens (got %d)", len(tokens))
	}
	for i, tok := range tokens {
		if !entityIDToken.MatchString(tok) {
			return fmt.Errorf("token[%d]=%q not alphanumeric-start", i, tok)
		}
	}
	return nil
}

func stableToken(value string) string {
	value = strings.TrimSpace(value)
	for {
		i := strings.IndexByte(value, ':')
		if i < 0 {
			break
		}
		value = value[i+1:]
	}
	token := tokenChars.ReplaceAllString(value, "_")
	token = strings.Trim(token, "_-")
	if token == "" {
		token = "unknown"
	}
	if len(token) <= 80 && entityIDToken.MatchString(token) {
		return token
	}
	sum := sha256.Sum256([]byte(value))
	shortHash := hex.EncodeToString(sum[:])[:12]
	token = strings.Trim(token, "_-")
	if len(token) > 48 {
		token = token[:48]
	}
	token = strings.Trim(token, "_-")
	if token == "" || !entityIDToken.MatchString(token) {
		token = "id"
	}
	return token + "_" + shortHash
}

func lastToken(id string) string {
	parts := strings.Split(id, ".")
	if len(parts) == 0 {
		return stableToken(id)
	}
	return parts[len(parts)-1]
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return time.Unix(0, 0).UTC().Format(time.RFC3339)
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func copyResult(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+4)
	for key, value := range in {
		out[key] = value
	}
	return out
}

func compactStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
