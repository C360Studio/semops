package sapient

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const (
	CoordinateSystemLatLngDegM = "LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M"
	DatumWGS84Ellipsoid        = "LOCATION_DATUM_WGS84_E"
	DatumWGS84Geoid            = "LOCATION_DATUM_WGS84_G"
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

type SourceMessage struct {
	Message   sapientcodec.Message
	SourceRef string
}

type Projector struct {
	cfg        Config
	bornTracks map[string]struct{}
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
		cfg.Confidence = 0.7
	}
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:        cfg,
		bornTracks: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectMessage(msg sapientcodec.Message, sourceRef string) (Plan, error) {
	return p.projectMessage(SourceMessage{Message: msg, SourceRef: sourceRef}, cloneStringSet(p.bornTracks))
}

func (p *Projector) ProjectMessages(messages []SourceMessage) (Plan, error) {
	bornTracks := cloneStringSet(p.bornTracks)
	var plan Plan
	for _, source := range messages {
		next, err := p.projectMessage(source, bornTracks)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) projectMessage(source SourceMessage, bornTracks map[string]struct{}) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("sapient projector is nil")
	}
	msg := source.Message
	if msg.Content != sapientcodec.ContentDetectionReport {
		return Plan{}, nil
	}
	report := msg.DetectionReport
	if report == nil {
		return Plan{}, fmt.Errorf("sapient detectionReport payload is required")
	}
	if strings.TrimSpace(msg.NodeID) == "" {
		return Plan{}, fmt.Errorf("sapient node_id is required")
	}
	if msg.Timestamp.IsZero() {
		return Plan{}, fmt.Errorf("sapient message timestamp is required")
	}
	if strings.TrimSpace(report.ReportID) == "" {
		return Plan{}, fmt.Errorf("sapient detection report_id is required")
	}
	if strings.TrimSpace(report.ObjectID) == "" {
		return Plan{}, fmt.Errorf("sapient detection object_id is required")
	}
	lat, lon, err := absoluteLatLon(report)
	if err != nil {
		return Plan{}, err
	}

	trackID := p.trackID(report.ObjectID)
	triples := p.trackTriples(trackID, msg, source.SourceRef, lat, lon)
	if len(triples) == 0 {
		return Plan{}, nil
	}

	if _, ok := bornTracks[trackID]; !ok {
		bornTracks[trackID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          trackID,
					MessageType: messageType(cop.SAPIENTTrackContract().MessageType),
					UpdatedAt:   msg.Timestamp.UTC(),
				},
				Triples:         triples,
				IndexingProfile: cop.SAPIENTTrackContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerSAPIENT),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-sapient-track", report.ObjectID),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: trackID,
			},
			AddTriples:      triples,
			IndexingProfile: cop.SAPIENTTrackContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerSAPIENT),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-sapient-track", report.ObjectID),
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
		if _, ok := p.bornTracks[mutation.Create.Entity.ID]; ok {
			continue
		}
		p.bornTracks[mutation.Create.Entity.ID] = struct{}{}
		marked++
	}
	return marked
}

func (p *Projector) MarkBornForMessage(msg sapientcodec.Message, entityID string) bool {
	if p == nil || entityID == "" || msg.DetectionReport == nil {
		return false
	}
	if entityID != p.trackID(msg.DetectionReport.ObjectID) {
		return false
	}
	p.bornTracks[entityID] = struct{}{}
	return true
}

func (p *Projector) trackTriples(
	trackID string,
	msg sapientcodec.Message,
	sourceRef string,
	lat float64,
	lon float64,
) []message.Triple {
	report := msg.DetectionReport
	when := msg.Timestamp.UTC()
	confidence := p.confidenceForDetection(report)
	triples := []message.Triple{
		p.triple(trackID, cop.TrackNativeID, nativeID(msg), when, confidence),
		p.triple(trackID, cop.TrackStatus, statusForDetection(report), when, confidence),
		p.triple(trackID, cop.TrackObservedAt, when, when, confidence),
		p.triple(trackID, cop.TrackPosition, wktPoint(lat, lon), when, confidence),
		p.triple(trackID, cop.ProvenanceSource, "sapient", when, confidence),
		p.triple(trackID, cop.ProvenanceConfidence, confidence, when, confidence),
		p.triple(trackID, cop.ProvenanceObservedAt, when, when, confidence),
	}
	if sourceRef != "" {
		triples = append(triples, p.triple(trackID, cop.ProvenanceSourceRef, sourceRef, when, confidence))
	}
	return triples
}

func (p *Projector) triple(subject string, predicate string, object any, when time.Time, confidence float64) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "sapient",
		Timestamp:  when.UTC(),
		Confidence: confidence,
	}
}

func (p *Projector) trackID(objectID string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, objectID)
}

func EntityID(org, platform, objectID string) string {
	return strings.Join([]string{
		entityToken(org),
		entityToken(platform),
		"cop",
		"sapient",
		cop.EntityTrack,
		entityToken(objectID),
	}, ".")
}

func (p *Projector) ownerToken(owner string) string {
	if p == nil || p.cfg.OwnerTokens == nil {
		return ""
	}
	return p.cfg.OwnerTokens[owner].Wire()
}

func (p *Projector) confidenceForDetection(report *sapientcodec.DetectionReport) float64 {
	if report != nil && report.DetectionConfidence != nil && isFinite(*report.DetectionConfidence) {
		return *report.DetectionConfidence
	}
	return p.cfg.Confidence
}

func absoluteLatLon(report *sapientcodec.DetectionReport) (float64, float64, error) {
	if report.RangeBearing != nil {
		return 0, 0, fmt.Errorf("sapient rangeBearing detections require sensor pose and uncertainty before projection")
	}
	if report.Location == nil {
		return 0, 0, fmt.Errorf("sapient detection location is required")
	}
	location := report.Location
	if location.CoordinateSystem != CoordinateSystemLatLngDegM {
		return 0, 0, fmt.Errorf("sapient location coordinate_system %q is not supported for first projection", location.CoordinateSystem)
	}
	if location.Datum != DatumWGS84Ellipsoid && location.Datum != DatumWGS84Geoid {
		return 0, 0, fmt.Errorf("sapient location datum %q is not supported for first projection", location.Datum)
	}
	lon := location.X
	lat := location.Y
	if !isFinite(lat) || !isFinite(lon) {
		return 0, 0, fmt.Errorf("sapient location coordinates must be finite")
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("sapient latitude/longitude out of range")
	}
	return lat, lon, nil
}

func nativeID(msg sapientcodec.Message) string {
	report := msg.DetectionReport
	if report == nil {
		return "sapient.unknown"
	}
	return strings.Join([]string{
		"sapient",
		"object",
		entityToken(report.ObjectID),
		"node",
		entityToken(msg.NodeID),
		"report",
		entityToken(report.ReportID),
	}, ".")
}

func statusForDetection(report *sapientcodec.DetectionReport) string {
	if report == nil || strings.TrimSpace(report.State) == "" {
		return "active.detection"
	}
	return "active.detection." + entityToken(report.State)
}

func wktPoint(lat, lon float64) string {
	return "POINT(" + coord(lon) + " " + coord(lat) + ")"
}

func coord(v float64) string {
	return strconv.FormatFloat(v, 'f', 7, 64)
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
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

func requestID(prefix, id string) string {
	return prefix + "-" + entityToken(id)
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

func cloneStringSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

func cloneOwnerTokens(tokens map[string]ownership.OwnerToken) map[string]ownership.OwnerToken {
	if tokens == nil {
		return nil
	}
	out := make(map[string]ownership.OwnerToken, len(tokens))
	for owner, token := range tokens {
		out[owner] = token
	}
	return out
}
