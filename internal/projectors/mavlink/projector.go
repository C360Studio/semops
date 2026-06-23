package mavlink

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type graphTriple = message.Triple

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

type Projector struct {
	cfg        Config
	bornAssets map[string]struct{}
	bornTracks map[string]struct{}
	bornTasks  map[string]struct{}
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
		cfg:        cfg,
		bornAssets: make(map[string]struct{}),
		bornTracks: make(map[string]struct{}),
		bornTasks:  make(map[string]struct{}),
	}
}

func (p *Projector) ProjectPackets(packets []*mavcodec.Packet) (Plan, error) {
	bornAssets := cloneStringSet(p.bornAssets)
	bornTracks := cloneStringSet(p.bornTracks)
	bornTasks := cloneStringSet(p.bornTasks)
	var plan Plan
	for _, packet := range packets {
		next, err := p.projectPacket(packet, bornAssets, bornTracks, bornTasks)
		if err != nil {
			return Plan{}, err
		}
		plan.Mutations = append(plan.Mutations, next.Mutations...)
	}
	return plan, nil
}

func (p *Projector) ProjectPacket(packet *mavcodec.Packet) (Plan, error) {
	return p.projectPacket(
		packet,
		cloneStringSet(p.bornAssets),
		cloneStringSet(p.bornTracks),
		cloneStringSet(p.bornTasks),
	)
}

func (p *Projector) projectPacket(
	packet *mavcodec.Packet,
	bornAssets map[string]struct{},
	bornTracks map[string]struct{},
	bornTasks map[string]struct{},
) (Plan, error) {
	if packet == nil {
		return Plan{}, nil
	}
	if packet.MessageID == mavcodec.MessageIDCommandAck {
		return p.projectCommandTask(packet, bornAssets, bornTasks), nil
	}

	trackTriples := p.trackTriples(packet)
	if len(trackTriples) == 0 {
		return Plan{}, nil
	}

	var plan Plan
	assetID := p.sourceAssetID(packet.SystemID)
	trackID := p.trackID(packet.SystemID)
	now := observedAt(packet)

	if _, ok := bornAssets[assetID]; !ok {
		plan.Mutations = append(plan.Mutations, p.sourceAssetBirthMutation(assetID, packet))
		bornAssets[assetID] = struct{}{}
	}

	if _, ok := bornTracks[trackID]; !ok {
		trackTriples = append(trackTriples, triple(trackID, cop.TrackSource, assetID, packet, p.cfg.Confidence))
		plan.Mutations = append(plan.Mutations, Mutation{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          trackID,
					MessageType: mavlinkTrackMessageType(),
					UpdatedAt:   now,
				},
				Triples:         trackTriples,
				IndexingProfile: cop.MAVLinkTrackContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerMAVLink),
				TraceID:         p.cfg.TraceID,
				RequestID:       requestID("create-track", packet.SystemID),
			},
		})
		bornTracks[trackID] = struct{}{}
		return plan, nil
	}

	plan.Mutations = append(plan.Mutations, Mutation{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: trackID,
			},
			AddTriples:      trackTriples,
			IndexingProfile: cop.MAVLinkTrackContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerMAVLink),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("update-track", packet.SystemID),
		},
	})
	return plan, nil
}

func (p *Projector) projectCommandTask(
	packet *mavcodec.Packet,
	bornAssets map[string]struct{},
	bornTasks map[string]struct{},
) Plan {
	taskTriples := p.commandTaskTriples(packet)
	if len(taskTriples) == 0 {
		return Plan{}
	}

	var plan Plan
	assetID := p.sourceAssetID(packet.SystemID)
	taskID := p.commandTaskID(packet)
	now := observedAt(packet)

	if _, ok := bornAssets[assetID]; !ok {
		plan.Mutations = append(plan.Mutations, p.sourceAssetBirthMutation(assetID, packet))
		bornAssets[assetID] = struct{}{}
	}

	if _, ok := bornTasks[taskID]; !ok {
		taskTriples = append(taskTriples, triple(taskID, cop.TaskTarget, assetID, packet, p.cfg.Confidence))
		plan.Mutations = append(plan.Mutations, Mutation{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{
					ID:          taskID,
					MessageType: mavlinkCommandTaskMessageType(),
					UpdatedAt:   now,
				},
				Triples:         taskTriples,
				IndexingProfile: cop.MAVLinkCommandTaskContract().IndexingProfile,
				OwnerToken:      p.ownerToken(cop.OwnerMAVLink),
				TraceID:         p.cfg.TraceID,
				RequestID:       commandRequestID("create-command-task", packet),
			},
		})
		bornTasks[taskID] = struct{}{}
		return plan
	}

	plan.Mutations = append(plan.Mutations, Mutation{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID: taskID,
			},
			AddTriples:      taskTriples,
			IndexingProfile: cop.MAVLinkCommandTaskContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerMAVLink),
			TraceID:         p.cfg.TraceID,
			RequestID:       commandRequestID("update-command-task", packet),
		},
	})
	return plan
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

func (p *Projector) MarkBornForPacket(packet *mavcodec.Packet, entityID string) bool {
	if p == nil || packet == nil || entityID == "" {
		return false
	}
	switch entityID {
	case p.sourceAssetID(packet.SystemID):
		p.bornAssets[entityID] = struct{}{}
		return true
	case p.trackID(packet.SystemID):
		p.bornTracks[entityID] = struct{}{}
		return true
	default:
		if packet.MessageID == mavcodec.MessageIDCommandAck && entityID == p.commandTaskID(packet) {
			p.bornTasks[entityID] = struct{}{}
			return true
		}
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
	case cop.MAVLinkTrackContract().MessageType:
		if _, ok := p.bornTracks[entity.ID]; ok {
			return false
		}
		p.bornTracks[entity.ID] = struct{}{}
		return true
	case cop.MAVLinkCommandTaskContract().MessageType:
		if _, ok := p.bornTasks[entity.ID]; ok {
			return false
		}
		p.bornTasks[entity.ID] = struct{}{}
		return true
	default:
		return false
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

func (p *Projector) trackTriples(packet *mavcodec.Packet) []message.Triple {
	trackID := p.trackID(packet.SystemID)
	base := []message.Triple{
		triple(trackID, cop.TrackNativeID, nativeID(packet), packet, p.cfg.Confidence),
		triple(trackID, cop.TrackObservedAt, observedAt(packet), packet, p.cfg.Confidence),
		triple(trackID, cop.ProvenanceSource, "mavlink", packet, p.cfg.Confidence),
		triple(trackID, cop.ProvenanceConfidence, p.cfg.Confidence, packet, p.cfg.Confidence),
		triple(trackID, cop.ProvenanceObservedAt, observedAt(packet), packet, p.cfg.Confidence),
	}
	if packet.SourceRef != "" {
		base = append(base, triple(trackID, cop.ProvenanceSourceRef, packet.SourceRef, packet, p.cfg.Confidence))
	}

	switch packet.MessageID {
	case mavcodec.MessageIDHeartbeat:
		if status, ok := heartbeatStatus(packet); ok {
			return append(base, triple(trackID, cop.TrackStatus, status, packet, p.cfg.Confidence))
		}
	case mavcodec.MessageIDGlobalPositionInt:
		lat, latOK := field[int32](packet, "lat")
		lon, lonOK := field[int32](packet, "lon")
		vx, vxOK := field[int16](packet, "vx")
		vy, vyOK := field[int16](packet, "vy")
		vz, vzOK := field[int16](packet, "vz")
		hasSpecificField := false
		if latOK && lonOK {
			base = append(base, triple(trackID, cop.TrackPosition, wktPoint(lat, lon), packet, p.cfg.Confidence))
			hasSpecificField = true
		}
		if vxOK && vyOK && vzOK {
			base = append(base, triple(trackID, cop.TrackVelocity, nedVelocity(vx, vy, vz), packet, p.cfg.Confidence))
			hasSpecificField = true
		}
		if !hasSpecificField {
			return nil
		}
		return base
	case mavcodec.MessageIDAttitude:
		hasSpecificField := false
		if roll, ok := field[float32](packet, "roll"); ok {
			base = append(base, triple(trackID, cop.TrackRoll, roundedSignal(roll), packet, p.cfg.Confidence))
			hasSpecificField = true
		}
		if pitch, ok := field[float32](packet, "pitch"); ok {
			base = append(base, triple(trackID, cop.TrackPitch, roundedSignal(pitch), packet, p.cfg.Confidence))
			hasSpecificField = true
		}
		if yaw, ok := field[float32](packet, "yaw"); ok {
			base = append(base, triple(trackID, cop.TrackYaw, roundedSignal(yaw), packet, p.cfg.Confidence))
			hasSpecificField = true
		}
		if !hasSpecificField {
			return nil
		}
		return base
	case mavcodec.MessageIDBatteryStatus:
		if remaining, ok := field[int8](packet, "battery_remaining"); ok {
			return append(base, triple(trackID, cop.TrackBattery, int64(remaining), packet, p.cfg.Confidence))
		}
	}

	return nil
}

func (p *Projector) commandTaskTriples(packet *mavcodec.Packet) []message.Triple {
	command, commandOK := field[uint16](packet, "command")
	result, resultOK := field[uint8](packet, "result")
	if !commandOK || !resultOK {
		return nil
	}
	targetSystem, _ := field[uint8](packet, "target_system")
	targetComponent, _ := field[uint8](packet, "target_component")
	progress, _ := field[uint8](packet, "progress")
	resultParam2, _ := field[int32](packet, "result_param2")
	taskID := p.commandTaskID(packet)
	status := mavcodec.MAVResultString(result)
	description := fmt.Sprintf(
		"command=%d result=%s progress=%d target=%d/%d result_param2=%d",
		command,
		status,
		progress,
		targetSystem,
		targetComponent,
		resultParam2,
	)

	triples := []message.Triple{
		triple(taskID, cop.TaskName, fmt.Sprintf("MAVLink command %d ACK", command), packet, p.cfg.Confidence),
		triple(taskID, cop.TaskKind, "mavlink.command_ack", packet, p.cfg.Confidence),
		triple(taskID, cop.TaskStatus, status, packet, p.cfg.Confidence),
		triple(taskID, cop.TaskDescription, description, packet, p.cfg.Confidence),
		triple(taskID, cop.TaskNativeID, commandTaskNativeID(packet), packet, p.cfg.Confidence),
		triple(taskID, cop.ProvenanceSource, "mavlink", packet, p.cfg.Confidence),
		triple(taskID, cop.ProvenanceConfidence, p.cfg.Confidence, packet, p.cfg.Confidence),
		triple(taskID, cop.ProvenanceObservedAt, observedAt(packet), packet, p.cfg.Confidence),
	}
	if packet.SourceRef != "" {
		triples = append(triples, triple(taskID, cop.ProvenanceSourceRef, packet.SourceRef, packet, p.cfg.Confidence))
	}
	return triples
}

func (p *Projector) sourceAssetTriples(assetID string, packet *mavcodec.Packet) []message.Triple {
	triples := []message.Triple{
		triple(assetID, cop.AssetName, fmt.Sprintf("MAVLink system %d", packet.SystemID), packet, p.cfg.Confidence),
		triple(assetID, cop.AssetKind, "mavlink-system", packet, p.cfg.Confidence),
		triple(assetID, cop.AssetSource, "mavlink", packet, p.cfg.Confidence),
		triple(assetID, cop.AssetNativeID, fmt.Sprintf("mavlink.system.%d", packet.SystemID), packet, p.cfg.Confidence),
		triple(assetID, cop.ProvenanceSource, "mavlink", packet, p.cfg.Confidence),
		triple(assetID, cop.ProvenanceConfidence, p.cfg.Confidence, packet, p.cfg.Confidence),
		triple(assetID, cop.ProvenanceObservedAt, observedAt(packet), packet, p.cfg.Confidence),
	}
	if packet.SourceRef != "" {
		triples = append(triples, triple(assetID, cop.ProvenanceSourceRef, packet.SourceRef, packet, p.cfg.Confidence))
	}
	return triples
}

func (p *Projector) sourceAssetBirthMutation(assetID string, packet *mavcodec.Packet) Mutation {
	return Mutation{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{
				ID:          assetID,
				MessageType: sourceAssetMessageType(),
				UpdatedAt:   observedAt(packet),
			},
			Triples:         p.sourceAssetTriples(assetID, packet),
			IndexingProfile: cop.SourceAssetContract().IndexingProfile,
			OwnerToken:      p.ownerToken(cop.OwnerAsset),
			TraceID:         p.cfg.TraceID,
			RequestID:       requestID("create-source-asset", packet.SystemID),
		},
	}
}

func (p *Projector) sourceAssetID(systemID uint8) string {
	return message.EntityID{
		Org:      p.cfg.Org,
		Platform: p.cfg.Platform,
		Domain:   "cop",
		System:   "mavlink",
		Type:     cop.EntityAsset,
		Instance: fmt.Sprintf("system-%d", systemID),
	}.Key()
}

func (p *Projector) trackID(systemID uint8) string {
	return message.EntityID{
		Org:      p.cfg.Org,
		Platform: p.cfg.Platform,
		Domain:   "cop",
		System:   "mavlink",
		Type:     cop.EntityTrack,
		Instance: fmt.Sprintf("system-%d", systemID),
	}.Key()
}

func (p *Projector) commandTaskID(packet *mavcodec.Packet) string {
	command, _ := field[uint16](packet, "command")
	targetSystem, _ := field[uint8](packet, "target_system")
	targetComponent, _ := field[uint8](packet, "target_component")
	return message.EntityID{
		Org:      p.cfg.Org,
		Platform: p.cfg.Platform,
		Domain:   "cop",
		System:   "mavlink",
		Type:     cop.EntityTask,
		Instance: fmt.Sprintf(
			"system-%d-command-%d-target-%d-%d",
			packet.SystemID,
			command,
			targetSystem,
			targetComponent,
		),
	}.Key()
}

func (p *Projector) ownerToken(owner string) string {
	return p.cfg.OwnerTokens[owner].Wire()
}

func heartbeatStatus(packet *mavcodec.Packet) (string, bool) {
	status, ok := field[uint8](packet, "system_status")
	if !ok {
		return "", false
	}
	mode, _ := field[uint8](packet, "base_mode")

	state := "unknown"
	switch status {
	case mavcodec.StateStandby:
		state = "standby"
	case mavcodec.StateActive:
		state = "active"
	}
	armed := "disarmed"
	if mode&mavcodec.ModeFlagSafetyArmed != 0 {
		armed = "armed"
	}
	return state + "." + armed, true
}

func nativeID(packet *mavcodec.Packet) string {
	return fmt.Sprintf("mavlink.system.%d.component.%d", packet.SystemID, packet.ComponentID)
}

func commandTaskNativeID(packet *mavcodec.Packet) string {
	command, _ := field[uint16](packet, "command")
	targetSystem, _ := field[uint8](packet, "target_system")
	targetComponent, _ := field[uint8](packet, "target_component")
	return fmt.Sprintf(
		"mavlink.system.%d.component.%d.command.%d.target.%d.%d",
		packet.SystemID,
		packet.ComponentID,
		command,
		targetSystem,
		targetComponent,
	)
}

func field[T any](packet *mavcodec.Packet, name string) (T, bool) {
	var zero T
	if packet.ParsedFields == nil {
		return zero, false
	}
	value, ok := packet.ParsedFields[name]
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	return typed, ok
}

func observedAt(packet *mavcodec.Packet) time.Time {
	if packet.Timestamp.IsZero() {
		return time.Now().UTC()
	}
	return packet.Timestamp.UTC()
}

func triple(subject, predicate string, object any, packet *mavcodec.Packet, confidence float64) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "mavlink",
		Timestamp:  observedAt(packet),
		Confidence: confidence,
	}
}

func wktPoint(latE7, lonE7 int32) string {
	return "POINT(" + coord(float64(lonE7)/1e7) + " " + coord(float64(latE7)/1e7) + ")"
}

func coord(v float64) string {
	return strconv.FormatFloat(v, 'f', 7, 64)
}

func nedVelocity(vx, vy, vz int16) string {
	return fmt.Sprintf("NED_CMPS(%d %d %d)", vx, vy, vz)
}

func roundedSignal(v float32) float64 {
	return math.Round(float64(v)*100) / 100
}

func requestID(prefix string, systemID uint8) string {
	return fmt.Sprintf("%s-system-%d", prefix, systemID)
}

func commandRequestID(prefix string, packet *mavcodec.Packet) string {
	command, _ := field[uint16](packet, "command")
	targetSystem, _ := field[uint8](packet, "target_system")
	targetComponent, _ := field[uint8](packet, "target_component")
	return fmt.Sprintf(
		"%s-system-%d-command-%d-target-%d-%d",
		prefix,
		packet.SystemID,
		command,
		targetSystem,
		targetComponent,
	)
}

func sourceAssetMessageType() message.Type {
	return messageType(cop.SourceAssetContract().MessageType)
}

func mavlinkTrackMessageType() message.Type {
	return messageType(cop.MAVLinkTrackContract().MessageType)
}

func mavlinkCommandTaskMessageType() message.Type {
	return messageType(cop.MAVLinkCommandTaskContract().MessageType)
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
