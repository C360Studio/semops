package cot

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
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
	Org      string
	Platform string
	// OwnerTokens are typed write-lease credentials minted by the SemStreams
	// ownership registry/bind path. The projector serializes them only at the
	// graph mutation request boundary.
	OwnerTokens map[string]ownership.OwnerToken
	TraceID     string
	Confidence  float64
}

type SourceEvent struct {
	Event     cotcodec.Event
	SourceRef string
}

type Projector struct {
	cfg            Config
	bornAssets     map[string]struct{}
	bornTracks     map[string]struct{}
	bornTasks      map[string]struct{}
	bornAdvisories map[string]struct{}
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
		cfg:            cfg,
		bornAssets:     make(map[string]struct{}),
		bornTracks:     make(map[string]struct{}),
		bornTasks:      make(map[string]struct{}),
		bornAdvisories: make(map[string]struct{}),
	}
}

func (p *Projector) ProjectEvents(events []SourceEvent) (Plan, error) {
	bornAssets := cloneStringSet(p.bornAssets)
	bornTracks := cloneStringSet(p.bornTracks)
	bornTasks := cloneStringSet(p.bornTasks)
	bornAdvisories := cloneStringSet(p.bornAdvisories)
	var plan Plan
	for _, event := range events {
		next, err := p.projectEvent(event, bornAssets, bornTracks, bornTasks, bornAdvisories)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) ProjectEvent(event cotcodec.Event, sourceRef string) (Plan, error) {
	return p.projectEvent(
		SourceEvent{Event: event, SourceRef: sourceRef},
		cloneStringSet(p.bornAssets),
		cloneStringSet(p.bornTracks),
		cloneStringSet(p.bornTasks),
		cloneStringSet(p.bornAdvisories),
	)
}

func (p *Projector) projectEvent(
	source SourceEvent,
	bornAssets map[string]struct{},
	bornTracks map[string]struct{},
	bornTasks map[string]struct{},
	bornAdvisories map[string]struct{},
) (Plan, error) {
	event := source.Event
	if err := event.Validate(); err != nil {
		return Plan{}, err
	}

	switch {
	case cotcodec.IsGeoChatType(event.Type):
		return p.projectAdvisory(event, source.SourceRef, bornAdvisories), nil
	case cotcodec.IsMarkerType(event.Type):
		return p.projectTask(event, source.SourceRef, bornTasks), nil
	case cotcodec.IsOperatorType(event.Type) || cotcodec.IsAirTrackType(event.Type):
		return p.projectTrack(event, source.SourceRef, bornAssets, bornTracks), nil
	default:
		return Plan{}, nil
	}
}

func (p *Projector) projectTrack(
	event cotcodec.Event,
	sourceRef string,
	bornAssets map[string]struct{},
	bornTracks map[string]struct{},
) Plan {
	trackTriples := p.trackTriples(event, sourceRef)
	if len(trackTriples) == 0 {
		return Plan{}
	}

	var plan Plan
	assetID := p.sourceAssetID(event.UID)
	trackID := p.trackID(event.UID)
	now := observedAt(event)

	if _, ok := bornAssets[assetID]; !ok {
		plan.Mutations = append(plan.Mutations, Mutation{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          assetID,
					MessageType: sourceAssetMessageType(),
					UpdatedAt:   now,
				},
				Triples:         p.sourceAssetTriples(assetID, event, sourceRef),
				IndexingProfile: cop.SourceAssetContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerAsset),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-source-asset", event.UID),
			},
		})
		bornAssets[assetID] = struct{}{}
	}

	if _, ok := bornTracks[trackID]; !ok {
		trackTriples = append(trackTriples, p.triple(trackID, cop.TrackSource, assetID, event))
		plan.Mutations = append(plan.Mutations, Mutation{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          trackID,
					MessageType: takTrackMessageType(),
					UpdatedAt:   now,
				},
				Triples:         trackTriples,
				IndexingProfile: cop.TAKTrackContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerTAK),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-track", event.UID),
			},
		})
		bornTracks[trackID] = struct{}{}
		return plan
	}

	plan.Mutations = append(plan.Mutations, Mutation{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: trackID,
			},
			AddTriples:      trackTriples,
			IndexingProfile: cop.TAKTrackContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerTAK),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-track", event.UID),
		},
	})
	return plan
}

func (p *Projector) projectTask(event cotcodec.Event, sourceRef string, bornTasks map[string]struct{}) Plan {
	taskTriples := p.taskTriples(event, sourceRef)
	if len(taskTriples) == 0 {
		return Plan{}
	}

	taskID := p.taskID(event.UID)
	now := observedAt(event)
	if _, ok := bornTasks[taskID]; !ok {
		bornTasks[taskID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          taskID,
					MessageType: takTaskMessageType(),
					UpdatedAt:   now,
				},
				Triples:         taskTriples,
				IndexingProfile: cop.TAKTaskContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerTAK),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-task", event.UID),
			},
		}}}
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: taskID,
			},
			AddTriples:      taskTriples,
			IndexingProfile: cop.TAKTaskContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerTAK),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-task", event.UID),
		},
	}}}
}

func (p *Projector) projectAdvisory(event cotcodec.Event, sourceRef string, bornAdvisories map[string]struct{}) Plan {
	advisoryTriples := p.advisoryTriples(event, sourceRef)
	if len(advisoryTriples) == 0 {
		return Plan{}
	}

	advisoryID := p.advisoryID(event.UID)
	now := observedAt(event)
	if _, ok := bornAdvisories[advisoryID]; !ok {
		bornAdvisories[advisoryID] = struct{}{}
		return Plan{Mutations: []Mutation{{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          advisoryID,
					MessageType: takAdvisoryMessageType(),
					UpdatedAt:   now,
				},
				Triples:         advisoryTriples,
				IndexingProfile: cop.TAKAdvisoryContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerTAK),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-advisory", event.UID),
			},
		}}}
	}

	return Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: advisoryID,
			},
			AddTriples:      advisoryTriples,
			IndexingProfile: cop.TAKAdvisoryContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerTAK),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-advisory", event.UID),
		},
	}}}
}

func (p *Projector) MarkBornForPlan(plan Plan) int {
	if p == nil {
		return 0
	}
	var marked int
	for _, mutation := range plan.Mutations {
		if mutation.Kind != MutationCreate {
			continue
		}
		if p.markBornEntity(mutation.Create.Entity) {
			marked++
		}
	}
	return marked
}

func (p *Projector) MarkBornForEvent(event cotcodec.Event, entityID string) bool {
	if p == nil || entityID == "" {
		return false
	}
	switch entityID {
	case p.sourceAssetID(event.UID):
		p.bornAssets[entityID] = struct{}{}
		return true
	case p.trackID(event.UID):
		p.bornTracks[entityID] = struct{}{}
		return true
	case p.taskID(event.UID):
		p.bornTasks[entityID] = struct{}{}
		return true
	case p.advisoryID(event.UID):
		p.bornAdvisories[entityID] = struct{}{}
		return true
	default:
		return false
	}
}

func (p *Projector) markBornEntity(entity *graph.EntityState) bool {
	if entity == nil || entity.ID == "" {
		return false
	}
	switch entity.MessageType.Key() {
	case cop.SourceAssetContract().MessageType:
		if _, ok := p.bornAssets[entity.ID]; ok {
			return false
		}
		p.bornAssets[entity.ID] = struct{}{}
		return true
	case cop.TAKTrackContract().MessageType:
		if _, ok := p.bornTracks[entity.ID]; ok {
			return false
		}
		p.bornTracks[entity.ID] = struct{}{}
		return true
	case cop.TAKTaskContract().MessageType:
		if _, ok := p.bornTasks[entity.ID]; ok {
			return false
		}
		p.bornTasks[entity.ID] = struct{}{}
		return true
	case cop.TAKAdvisoryContract().MessageType:
		if _, ok := p.bornAdvisories[entity.ID]; ok {
			return false
		}
		p.bornAdvisories[entity.ID] = struct{}{}
		return true
	default:
		return false
	}
}

func (p *Projector) trackTriples(event cotcodec.Event, sourceRef string) []message.Triple {
	if event.Point == nil && !event.HasTrack {
		return nil
	}
	trackID := p.trackID(event.UID)
	base := p.provenanceTriples(trackID, event, sourceRef)
	base = append(base,
		p.triple(trackID, cop.TrackNativeID, nativeID(event.UID), event),
		p.triple(trackID, cop.TrackObservedAt, observedAt(event), event),
		p.triple(trackID, cop.TrackStatus, statusForEvent(event, trackKind(event)), event),
	)
	if event.Point != nil {
		base = append(base, p.triple(trackID, cop.TrackPosition, wktPoint(event.Point), event))
	}
	if event.HasTrack {
		base = append(base, p.triple(trackID, cop.TrackVelocity, courseSpeed(event), event))
	}
	return base
}

func (p *Projector) sourceAssetTriples(assetID string, event cotcodec.Event, sourceRef string) []message.Triple {
	triples := p.provenanceTriples(assetID, event, sourceRef)
	triples = append(triples,
		p.triple(assetID, cop.AssetName, firstNonEmpty(event.Callsign, event.UID), event),
		p.triple(assetID, cop.AssetKind, "tak-cot-source", event),
		p.triple(assetID, cop.AssetSource, "tak-cot", event),
		p.triple(assetID, cop.AssetNativeID, nativeID(event.UID), event),
	)
	return triples
}

func (p *Projector) taskTriples(event cotcodec.Event, sourceRef string) []message.Triple {
	if event.Point == nil && strings.TrimSpace(event.Callsign) == "" && strings.TrimSpace(event.Remarks) == "" {
		return nil
	}
	taskID := p.taskID(event.UID)
	triples := p.provenanceTriples(taskID, event, sourceRef)
	triples = append(triples,
		p.triple(taskID, cop.TaskNativeID, nativeID(event.UID), event),
		p.triple(taskID, cop.TaskName, firstNonEmpty(event.Callsign, event.UID), event),
		p.triple(taskID, cop.TaskKind, "marker", event),
		p.triple(taskID, cop.TaskStatus, statusForEvent(event, "marker"), event),
	)
	if event.Point != nil {
		triples = append(triples, p.triple(taskID, cop.TaskPosition, wktPoint(event.Point), event))
	}
	if strings.TrimSpace(event.Remarks) != "" {
		triples = append(triples, p.triple(taskID, cop.TaskDescription, strings.TrimSpace(event.Remarks), event))
	}
	return triples
}

func (p *Projector) advisoryTriples(event cotcodec.Event, sourceRef string) []message.Triple {
	text := advisoryText(event)
	if text == "" {
		return nil
	}
	advisoryID := p.advisoryID(event.UID)
	triples := p.provenanceTriples(advisoryID, event, sourceRef)
	triples = append(triples,
		p.triple(advisoryID, cop.AdvisoryNativeID, nativeID(event.UID), event),
		p.triple(advisoryID, cop.AdvisoryText, text, event),
		p.triple(advisoryID, cop.AdvisoryKind, "geochat", event),
		p.triple(advisoryID, cop.AdvisoryStatus, statusForEvent(event, "geochat"), event),
	)
	if sender := firstNonEmpty(event.SenderUID, event.Callsign); sender != "" {
		triples = append(triples, p.triple(advisoryID, cop.AdvisorySender, sender, event))
	}
	if event.Point != nil {
		triples = append(triples, p.triple(advisoryID, cop.AdvisoryPosition, wktPoint(event.Point), event))
	}
	return triples
}

func (p *Projector) provenanceTriples(subject string, event cotcodec.Event, sourceRef string) []message.Triple {
	triples := []message.Triple{
		p.triple(subject, cop.ProvenanceSource, "tak-cot", event),
		p.triple(subject, cop.ProvenanceConfidence, p.cfg.Confidence, event),
		p.triple(subject, cop.ProvenanceObservedAt, observedAt(event), event),
	}
	if sourceRef != "" {
		triples = append(triples, p.triple(subject, cop.ProvenanceSourceRef, sourceRef, event))
	}
	return triples
}

func (p *Projector) sourceAssetID(uid string) string {
	return p.entityID(cop.EntityAsset, uid)
}

func (p *Projector) trackID(uid string) string {
	return p.entityID(cop.EntityTrack, uid)
}

func (p *Projector) taskID(uid string) string {
	return p.entityID(cop.EntityTask, uid)
}

func (p *Projector) advisoryID(uid string) string {
	return p.entityID(cop.EntityAdvisory, uid)
}

func (p *Projector) entityID(entityType, uid string) string {
	return EntityID(p.cfg.Org, p.cfg.Platform, entityType, uid)
}

func EntityID(org, platform, entityType, uid string) string {
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
		System:   "tak",
		Type:     entityType,
		Instance: entityToken(uid),
	}.Key()
}

func (p *Projector) ownerToken(owner string) string {
	return p.cfg.OwnerTokens[owner].Wire()
}

func (p *Projector) triple(subject, predicate string, object any, event cotcodec.Event) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "tak-cot",
		Timestamp:  observedAt(event),
		Confidence: p.cfg.Confidence,
	}
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

func observedAt(event cotcodec.Event) time.Time {
	if event.Time.IsZero() {
		return time.Now().UTC()
	}
	return event.Time.UTC()
}

func nativeID(uid string) string {
	return "cot.uid." + strings.TrimSpace(uid)
}

func wktPoint(point *cotcodec.Point) string {
	if point == nil {
		return ""
	}
	return "POINT(" + coord(point.Lon) + " " + coord(point.Lat) + ")"
}

func coord(v float64) string {
	return strconv.FormatFloat(v, 'f', 7, 64)
}

func courseSpeed(event cotcodec.Event) string {
	return fmt.Sprintf(
		"COURSE_SPEED_MPS(%s %s)",
		strconv.FormatFloat(math.Round(event.CourseDeg*100)/100, 'f', 2, 64),
		strconv.FormatFloat(math.Round(event.SpeedMPS*100)/100, 'f', 2, 64),
	)
}

func statusForEvent(event cotcodec.Event, kind string) string {
	state := "active"
	if !event.Stale.IsZero() && observedAt(event).After(event.Stale.UTC()) {
		state = "stale"
	}
	if kind == "" {
		return state
	}
	return state + "." + kind
}

func trackKind(event cotcodec.Event) string {
	if cotcodec.IsAirTrackType(event.Type) {
		return "air-track"
	}
	return "operator"
}

func advisoryText(event cotcodec.Event) string {
	return firstNonEmpty(event.ChatText, event.Remarks)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
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

func requestID(prefix, uid string) string {
	return prefix + "-" + entityToken(uid)
}

func sourceAssetMessageType() message.Type {
	return messageType(cop.SourceAssetContract().MessageType)
}

func takTrackMessageType() message.Type {
	return messageType(cop.TAKTrackContract().MessageType)
}

func takTaskMessageType() message.Type {
	return messageType(cop.TAKTaskContract().MessageType)
}

func takAdvisoryMessageType() message.Type {
	return messageType(cop.TAKAdvisoryContract().MessageType)
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
