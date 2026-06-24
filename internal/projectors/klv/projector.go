package klv

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

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

type Frame struct {
	Source                     string
	MediaRef                   string
	PacketRef                  string
	ReceivedAt                 time.Time
	FrameTime                  time.Time
	PlatformDesignation        string
	SensorLatitude             *float64
	SensorLongitude            *float64
	SensorAltitudeMeters       *float64
	SensorAzimuthDegrees       *float64
	SensorElevationDegrees     *float64
	FrameCenterLatitude        *float64
	FrameCenterLongitude       *float64
	FrameCenterElevationMeters *float64
	FootprintWKT               string
}

type Projector struct {
	cfg            Config
	bornFootprints map[string]struct{}
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
		cfg.Confidence = 0.8
	}
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:            cfg,
		bornFootprints: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectFrame(frame Frame) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("klv projector is nil")
	}
	if strings.TrimSpace(frame.MediaRef) == "" {
		return Plan{}, fmt.Errorf("klv frame media_ref is required")
	}
	if strings.TrimSpace(frame.PacketRef) == "" {
		return Plan{}, fmt.Errorf("klv frame packet_ref is required")
	}
	if observedAt(frame).IsZero() {
		return Plan{}, fmt.Errorf("klv frame observed time is required")
	}

	footprintID := p.footprintID(frame.MediaRef)
	triples := p.footprintTriples(footprintID, frame)
	if len(triples) == 0 {
		return Plan{}, nil
	}

	if _, ok := p.bornFootprints[footprintID]; !ok {
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          footprintID,
					MessageType: messageType(cop.KLVSensorFootprintContract().MessageType),
					UpdatedAt:   observedAt(frame),
				},
				Triples:         triples,
				IndexingProfile: cop.KLVSensorFootprintContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerKLV),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-sensor-footprint", frame.MediaRef),
			},
		}}}, nil
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: footprintID,
			},
			AddTriples:      triples,
			IndexingProfile: cop.KLVSensorFootprintContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerKLV),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-sensor-footprint", frame.MediaRef),
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
		if _, ok := p.bornFootprints[mutation.Create.Entity.ID]; ok {
			continue
		}
		p.bornFootprints[mutation.Create.Entity.ID] = struct{}{}
		marked++
	}
	return marked
}

func (p *Projector) MarkBornForFrame(frame Frame, entityID string) bool {
	if p == nil || entityID == "" || entityID != p.footprintID(frame.MediaRef) {
		return false
	}
	p.bornFootprints[entityID] = struct{}{}
	return true
}

func (p *Projector) footprintTriples(footprintID string, frame Frame) []message.Triple {
	when := observedAt(frame)
	triples := []message.Triple{
		p.triple(footprintID, cop.SensorFootprintNativeID, nativeID(frame), when),
		p.triple(footprintID, cop.SensorFootprintSource, "klv", when),
		p.triple(footprintID, cop.SensorFootprintMediaRef, frame.MediaRef, when),
		p.triple(footprintID, cop.SensorFootprintPacketRef, frame.PacketRef, when),
		p.triple(footprintID, cop.SensorFootprintObservedAt, when, when),
		p.triple(footprintID, cop.ProvenanceSource, "klv", when),
		p.triple(footprintID, cop.ProvenanceConfidence, p.cfg.Confidence, when),
		p.triple(footprintID, cop.ProvenanceObservedAt, when, when),
		p.triple(footprintID, cop.ProvenanceSourceRef, frame.PacketRef, when),
	}
	if frame.PlatformDesignation != "" {
		triples = append(triples, p.triple(footprintID, cop.SensorFootprintPlatformDesignation, frame.PlatformDesignation, when))
	}
	if frame.SensorLatitude != nil && frame.SensorLongitude != nil {
		triples = append(triples, p.triple(
			footprintID,
			cop.SensorFootprintSensorPosition,
			wktPoint(*frame.SensorLatitude, *frame.SensorLongitude),
			when,
		))
	}
	if frame.SensorAltitudeMeters != nil {
		triples = append(triples, p.triple(footprintID, cop.SensorFootprintSensorAltitude, *frame.SensorAltitudeMeters, when))
	}
	if frame.SensorAzimuthDegrees != nil {
		triples = append(triples, p.triple(footprintID, cop.SensorFootprintSensorAzimuth, *frame.SensorAzimuthDegrees, when))
	}
	if frame.SensorElevationDegrees != nil {
		triples = append(triples, p.triple(footprintID, cop.SensorFootprintSensorElevation, *frame.SensorElevationDegrees, when))
	}
	if frame.FrameCenterLatitude != nil && frame.FrameCenterLongitude != nil {
		triples = append(triples, p.triple(
			footprintID,
			cop.SensorFootprintFrameCenter,
			wktPoint(*frame.FrameCenterLatitude, *frame.FrameCenterLongitude),
			when,
		))
	}
	if frame.FrameCenterElevationMeters != nil {
		triples = append(triples, p.triple(footprintID, cop.SensorFootprintFrameCenterElevation, *frame.FrameCenterElevationMeters, when))
	}
	if strings.TrimSpace(frame.FootprintWKT) != "" {
		triples = append(triples, p.triple(footprintID, cop.SensorFootprintGeometry, strings.TrimSpace(frame.FootprintWKT), when))
	}
	return triples
}

func (p *Projector) triple(subject string, predicate string, object any, when time.Time) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "klv",
		Timestamp:  when.UTC(),
		Confidence: p.cfg.Confidence,
	}
}

func (p *Projector) footprintID(mediaRef string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, mediaRef)
}

func EntityID(org, platform, mediaRef string) string {
	return strings.Join([]string{
		entityToken(org),
		entityToken(platform),
		"cop",
		"klv",
		cop.EntitySensorFootprint,
		entityToken(mediaRef),
	}, ".")
}

func (p *Projector) ownerToken(owner string) string {
	if p == nil || p.cfg.OwnerTokens == nil {
		return ""
	}
	return p.cfg.OwnerTokens[owner].Wire()
}

func observedAt(frame Frame) time.Time {
	if !frame.FrameTime.IsZero() {
		return frame.FrameTime.UTC()
	}
	if !frame.ReceivedAt.IsZero() {
		return frame.ReceivedAt.UTC()
	}
	return time.Time{}
}

func nativeID(frame Frame) string {
	return "klv.media." + entityToken(frame.MediaRef) + ".packet." + entityToken(frame.PacketRef)
}

func wktPoint(lat, lon float64) string {
	return "POINT(" + coord(lon) + " " + coord(lat) + ")"
}

func coord(v float64) string {
	return strconv.FormatFloat(round(v, 7), 'f', 7, 64)
}

func round(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
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

func requestID(prefix, mediaRef string) string {
	return prefix + "-" + entityToken(mediaRef)
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
