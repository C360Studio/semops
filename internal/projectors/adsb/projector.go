package adsb

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
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

type SourceState struct {
	State     adsbcodec.StateVector
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
		cfg.Confidence = 0.85
	}
	cfg.OwnerTokens = cloneOwnerTokens(cfg.OwnerTokens)
	return &Projector{
		cfg:        cfg,
		bornTracks: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectState(state adsbcodec.StateVector, sourceRef string) (Plan, error) {
	return p.projectState(SourceState{State: state, SourceRef: sourceRef}, cloneStringSet(p.bornTracks))
}

func (p *Projector) ProjectStates(states []SourceState) (Plan, error) {
	bornTracks := cloneStringSet(p.bornTracks)
	var plan Plan
	for _, state := range states {
		next, err := p.projectState(state, bornTracks)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) projectState(source SourceState, bornTracks map[string]struct{}) (Plan, error) {
	state := source.State
	if strings.TrimSpace(state.ICAO24) == "" {
		return Plan{}, fmt.Errorf("adsb state icao24 is required")
	}
	if state.LastContact.IsZero() {
		return Plan{}, fmt.Errorf("adsb state last_contact is required")
	}

	trackID := p.trackID(state.ICAO24)
	triples := p.trackTriples(trackID, state, source.SourceRef)
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
					MessageType: adsbTrackMessageType(),
					UpdatedAt:   observedAt(state),
				},
				Triples:         triples,
				IndexingProfile: cop.ADSBTrackContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerADSB),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-track", state.ICAO24),
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
			IndexingProfile: cop.ADSBTrackContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerADSB),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-track", state.ICAO24),
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

func (p *Projector) MarkBornForState(state adsbcodec.StateVector, entityID string) bool {
	if p == nil || entityID == "" || entityID != p.trackID(state.ICAO24) {
		return false
	}
	p.bornTracks[entityID] = struct{}{}
	return true
}

func (p *Projector) trackTriples(trackID string, state adsbcodec.StateVector, sourceRef string) []message.Triple {
	triples := []message.Triple{
		p.triple(trackID, cop.TrackNativeID, nativeID(state), state),
		p.triple(trackID, cop.TrackStatus, statusForState(state), state),
		p.triple(trackID, cop.TrackObservedAt, observedAt(state), state),
		p.triple(trackID, cop.ProvenanceSource, "adsb", state),
		p.triple(trackID, cop.ProvenanceConfidence, p.confidenceForState(state), state),
		p.triple(trackID, cop.ProvenanceObservedAt, observedAt(state), state),
	}
	if state.HasPosition() {
		triples = append(triples, p.triple(trackID, cop.TrackPosition, wktPoint(*state.Latitude, *state.Longitude), state))
	}
	if velocity := velocityForState(state); velocity != "" {
		triples = append(triples, p.triple(trackID, cop.TrackVelocity, velocity, state))
	}
	if sourceRef != "" {
		triples = append(triples, p.triple(trackID, cop.ProvenanceSourceRef, sourceRef, state))
	}
	return triples
}

func (p *Projector) triple(subject, predicate string, object any, state adsbcodec.StateVector) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "adsb",
		Timestamp:  observedAt(state),
		Confidence: p.confidenceForState(state),
	}
}

func (p *Projector) trackID(icao24 string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, icao24)
}

func EntityID(org, platform, icao24 string) string {
	return strings.Join([]string{
		entityToken(org),
		entityToken(platform),
		"cop",
		"adsb",
		cop.EntityTrack,
		entityToken(icao24),
	}, ".")
}

func (p *Projector) ownerToken(owner string) string {
	if p == nil || p.cfg.OwnerTokens == nil {
		return ""
	}
	return p.cfg.OwnerTokens[owner].Wire()
}

func (p *Projector) confidenceForState(state adsbcodec.StateVector) float64 {
	base := p.cfg.Confidence
	if !state.HasPositionSource {
		return math.Min(base, 0.5)
	}
	switch state.PositionSource {
	case adsbcodec.PositionSourceADSB:
		return base
	case adsbcodec.PositionSourceASTERIX:
		return math.Min(base, 0.8)
	case adsbcodec.PositionSourceMLAT:
		return math.Min(base, 0.75)
	case adsbcodec.PositionSourceFLARM:
		return math.Min(base, 0.65)
	default:
		return math.Min(base, 0.5)
	}
}

func observedAt(state adsbcodec.StateVector) time.Time {
	if state.TimePosition != nil {
		return state.TimePosition.UTC()
	}
	if !state.LastContact.IsZero() {
		return state.LastContact.UTC()
	}
	return time.Now().UTC()
}

func nativeID(state adsbcodec.StateVector) string {
	native := "adsb.icao24." + entityToken(state.ICAO24)
	if state.Callsign != nil {
		native += ".callsign." + entityToken(*state.Callsign)
	}
	native += ".source." + state.PositionSourceLabel()
	return native
}

func statusForState(state adsbcodec.StateVector) string {
	if state.OnGround {
		return "active.aircraft.ground"
	}
	return "active.aircraft"
}

func wktPoint(lat, lon float64) string {
	return "POINT(" + coord(lon) + " " + coord(lat) + ")"
}

func coord(v float64) string {
	return strconv.FormatFloat(v, 'f', 7, 64)
}

func velocityForState(state adsbcodec.StateVector) string {
	if state.VelocityMPS == nil && state.TrueTrackDeg == nil && state.VerticalRateMPS == nil {
		return ""
	}
	return fmt.Sprintf(
		"AIR_MOTION_MPS(%s %s %s)",
		optionalSignal(state.VelocityMPS),
		optionalSignal(state.TrueTrackDeg),
		optionalSignal(state.VerticalRateMPS),
	)
}

func optionalSignal(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return strconv.FormatFloat(math.Round(*value*100)/100, 'f', 2, 64)
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

func requestID(prefix, icao24 string) string {
	return prefix + "-" + entityToken(icao24)
}

func adsbTrackMessageType() message.Type {
	return messageType(cop.ADSBTrackContract().MessageType)
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
