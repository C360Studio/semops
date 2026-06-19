package cap

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type MutationKind string

const (
	MutationCreate MutationKind = "create"
	MutationUpdate MutationKind = "update"
)

type Config struct {
	Org         string
	Platform    string
	OwnerTokens map[string]ownership.OwnerToken
	TraceID     string
	Confidence  float64
}

type SourceAlert struct {
	Alert     capcodec.Alert
	SourceRef string
}

type Projector struct {
	cfg         Config
	bornHazards map[string]struct{}
}

type Plan struct {
	Mutations []Mutation
}

type Mutation struct {
	Kind   MutationKind
	Create graph.CreateEntityWithTriplesRequest
	Update graph.UpdateEntityWithTriplesRequest
}

func NewProjector(cfg Config) *Projector {
	if cfg.Org == "" {
		cfg.Org = "c360"
	}
	if cfg.Platform == "" {
		cfg.Platform = "edge"
	}
	if cfg.Confidence == 0 {
		cfg.Confidence = 1.0
	}
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:         cfg,
		bornHazards: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectAlert(alert capcodec.Alert, sourceRef string) (Plan, error) {
	return p.projectAlert(SourceAlert{Alert: alert, SourceRef: sourceRef}, cloneStringSet(p.bornHazards))
}

func (p *Projector) ProjectAlerts(alerts []SourceAlert) (Plan, error) {
	bornHazards := cloneStringSet(p.bornHazards)
	var plan Plan
	for _, alert := range alerts {
		next, err := p.projectAlert(alert, bornHazards)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) projectAlert(source SourceAlert, bornHazards map[string]struct{}) (Plan, error) {
	alert := source.Alert
	if err := alert.Validate(); err != nil {
		return Plan{}, err
	}

	hazardID := p.hazardID(alert.Identifier)
	triples, err := p.hazardEvidenceTriples(hazardID, alert, source.SourceRef)
	if err != nil {
		return Plan{}, err
	}
	if len(triples) == 0 {
		return Plan{}, nil
	}

	observed := observedAt(alert)
	if _, ok := bornHazards[hazardID]; !ok {
		bornHazards[hazardID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          hazardID,
					MessageType: capHazardMessageType(),
					UpdatedAt:   observed,
				},
				Triples:         triples,
				IndexingProfile: cop.CAPHazardEvidenceContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerCAP),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-hazard-evidence", alert.Identifier),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: hazardID,
			},
			AddTriples:      triples,
			IndexingProfile: cop.CAPHazardEvidenceContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerCAP),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("append-hazard-evidence", alert.Identifier),
		},
	}}}, nil
}

func (p *Projector) MarkBornForPlan(plan Plan) int {
	if p == nil {
		return 0
	}
	var marked int
	for _, mutation := range plan.Mutations {
		if mutation.Kind != MutationCreate || mutation.Create.Entity == nil || mutation.Create.Entity.ID == "" {
			continue
		}
		if _, ok := p.bornHazards[mutation.Create.Entity.ID]; ok {
			continue
		}
		p.bornHazards[mutation.Create.Entity.ID] = struct{}{}
		marked++
	}
	return marked
}

func (p *Projector) MarkBornForAlert(alert capcodec.Alert, entityID string) bool {
	if p == nil || entityID == "" || entityID != p.hazardID(alert.Identifier) {
		return false
	}
	p.bornHazards[entityID] = struct{}{}
	return true
}

func (p *Projector) hazardEvidenceTriples(
	hazardID string,
	alert capcodec.Alert,
	sourceRef string,
) ([]message.Triple, error) {
	doc := evidenceDocument(alert)
	evidence, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal cap hazard evidence: %w", err)
	}
	text := advisoryText(alert)
	triples := []message.Triple{
		p.triple(hazardID, cop.HazardAdvisoryText, text, alert),
		p.triple(hazardID, cop.HazardEvidence, string(evidence), alert),
		p.triple(hazardID, cop.HazardSource, "cap", alert),
		p.triple(hazardID, cop.ProvenanceSource, "cap", alert),
		p.triple(hazardID, cop.ProvenanceConfidence, confidenceForAlert(alert, p.cfg.Confidence), alert),
		p.triple(hazardID, cop.ProvenanceObservedAt, observedAt(alert), alert),
	}
	if sourceRef != "" {
		triples = append(triples, p.triple(hazardID, cop.ProvenanceSourceRef, sourceRef, alert))
	}
	return triples, nil
}

func evidenceDocument(alert capcodec.Alert) cop.HazardEvidenceDocument {
	info, _ := alert.PrimaryInfo()
	doc := cop.HazardEvidenceDocument{
		Identifier:  alert.Identifier,
		MessageType: alert.MsgType,
		Status:      alert.Status,
		Event:       info.Event,
		Urgency:     info.Urgency,
		Severity:    info.Severity,
		Certainty:   info.Certainty,
		Sender:      alert.Sender,
		SenderName:  info.SenderName,
		Sent:        formatOptionalTime(alert.Sent),
		Effective:   formatOptionalTime(firstNonZeroTime(info.Effective, alert.Sent)),
		Expires:     formatOptionalTime(info.Expires),
		Parameters:  evidenceNameValues(info.Parameters),
		Resources:   evidenceResources(info.Resources),
	}
	if len(info.Areas) > 0 {
		area := info.Areas[0]
		doc.AreaDesc = area.AreaDesc
		doc.Polygons = evidencePolygons(area.Polygons)
		doc.Circles = evidenceCircles(area.Circles)
		doc.Geocodes = evidenceNameValues(area.Geocodes)
	}
	return doc
}

func advisoryText(alert capcodec.Alert) string {
	info, ok := alert.PrimaryInfo()
	if !ok {
		return alert.Identifier
	}
	text := info.AdvisoryText()
	if text != "" {
		return text
	}
	return firstNonEmpty(info.Headline, info.Event, alert.Identifier)
}

func observedAt(alert capcodec.Alert) time.Time {
	info, ok := alert.PrimaryInfo()
	if ok {
		if !info.Effective.IsZero() {
			return info.Effective.UTC()
		}
	}
	if !alert.Sent.IsZero() {
		return alert.Sent.UTC()
	}
	return time.Now().UTC()
}

func confidenceForAlert(alert capcodec.Alert, fallback float64) float64 {
	info, ok := alert.PrimaryInfo()
	if !ok {
		return fallback
	}
	switch strings.ToLower(info.Certainty) {
	case "observed":
		return 0.95
	case "likely":
		return 0.82
	case "possible":
		return 0.58
	case "unlikely":
		return 0.35
	case "unknown":
		return 0.4
	default:
		return fallback
	}
}

func (p *Projector) hazardID(identifier string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, identifier)
}

func EntityID(org, platform, identifier string) string {
	if org == "" {
		org = "c360"
	}
	if platform == "" {
		platform = "edge"
	}
	return message.EntityID{
		Org:      org,
		Platform: platform,
		Domain:   "cop",
		System:   "cap",
		Type:     cop.EntityHazardArea,
		Instance: entityToken(identifier),
	}.Key()
}

func (p *Projector) ownerToken(owner string) string {
	return p.cfg.OwnerTokens[owner].Wire()
}

func (p *Projector) triple(subject, predicate string, object any, alert capcodec.Alert) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "cap",
		Timestamp:  observedAt(alert),
		Confidence: confidenceForAlert(alert, p.cfg.Confidence),
	}
}

func evidencePolygons(polygons [][]capcodec.Point) [][]cop.HazardEvidencePoint {
	out := make([][]cop.HazardEvidencePoint, 0, len(polygons))
	for _, polygon := range polygons {
		next := make([]cop.HazardEvidencePoint, 0, len(polygon))
		for _, point := range polygon {
			next = append(next, cop.HazardEvidencePoint{Lat: point.Lat, Lon: point.Lon})
		}
		out = append(out, next)
	}
	return out
}

func evidenceCircles(circles []capcodec.Circle) []cop.HazardEvidenceCircle {
	out := make([]cop.HazardEvidenceCircle, 0, len(circles))
	for _, circle := range circles {
		out = append(out, cop.HazardEvidenceCircle{
			Center:   cop.HazardEvidencePoint{Lat: circle.Center.Lat, Lon: circle.Center.Lon},
			RadiusKM: circle.RadiusKM,
		})
	}
	return out
}

func evidenceNameValues(values []capcodec.NameValue) []cop.HazardEvidenceNameValue {
	out := make([]cop.HazardEvidenceNameValue, 0, len(values))
	for _, value := range values {
		out = append(out, cop.HazardEvidenceNameValue{Name: value.Name, Value: value.Value})
	}
	return out
}

func evidenceResources(resources []capcodec.Resource) []cop.HazardEvidenceResource {
	out := make([]cop.HazardEvidenceResource, 0, len(resources))
	for _, resource := range resources {
		out = append(out, cop.HazardEvidenceResource{
			Description: resource.Description,
			MimeType:    resource.MimeType,
			URI:         resource.URI,
			Digest:      resource.Digest,
		})
	}
	return out
}

func cloneStringSet(set map[string]struct{}) map[string]struct{} {
	clone := make(map[string]struct{}, len(set))
	for key := range set {
		clone[key] = struct{}{}
	}
	return clone
}

func cloneOwnerTokens(tokens map[string]ownership.OwnerToken) map[string]ownership.OwnerToken {
	clone := make(map[string]ownership.OwnerToken, len(tokens))
	for owner, token := range tokens {
		clone[owner] = token
	}
	return clone
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Time{}
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func entityToken(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(value) {
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "unknown"
	}
	return token
}

func requestID(prefix, identifier string) string {
	return prefix + "-" + entityToken(identifier)
}

func capHazardMessageType() message.Type {
	return messageType(cop.CAPHazardEvidenceContract().MessageType)
}

func messageType(key string) message.Type {
	parts := strings.Split(key, ".")
	if len(parts) < 3 {
		return message.Type{Domain: key, Category: "unknown", Version: "v1"}
	}
	return message.Type{
		Domain:   parts[0],
		Category: strings.Join(parts[1:len(parts)-1], "."),
		Version:  parts[len(parts)-1],
	}
}
