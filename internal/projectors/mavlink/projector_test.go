package mavlink

import (
	"testing"
	"time"

	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
)

func TestProjectorBirthsSourceAssetBeforeTrackWithStrictForeignEdge(t *testing.T) {
	packet := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateHeartbeat(mavcodec.HeartbeatMessage{
			VehicleType:    mavcodec.TypeQuadrotor,
			Autopilot:      mavcodec.AutopilotPX4,
			BaseMode:       mavcodec.ModeFlagSafetyArmed,
			SystemStatus:   mavcodec.StateActive,
			MavlinkVersion: mavcodec.Version2,
		})
	})

	projector := NewProjector(Config{
		Org:              "c360",
		Platform:         "edge",
		OwnerTokenSuffix: "test",
		TraceID:          "scenario-001",
	})
	plan, err := projector.ProjectPacket(packet)
	if err != nil {
		t.Fatalf("project heartbeat: %v", err)
	}

	if len(plan.Mutations) != 2 {
		t.Fatalf("mutations = %d, want asset birth + track birth", len(plan.Mutations))
	}

	assetCreate := requireCreate(t, plan.Mutations[0])
	trackCreate := requireCreate(t, plan.Mutations[1])

	if assetCreate.Entity.ID != "c360.edge.cop.mavlink.asset.system-42" {
		t.Fatalf("asset id = %q", assetCreate.Entity.ID)
	}
	if assetCreate.Entity.MessageType.Key() != cop.SourceAssetContract().MessageType {
		t.Fatalf("asset message type = %q", assetCreate.Entity.MessageType.Key())
	}
	if assetCreate.IndexingProfile != cop.SourceAssetContract().IndexingProfile {
		t.Fatalf("asset indexing profile = %q", assetCreate.IndexingProfile)
	}
	if assetCreate.OwnerToken != "semops.feed.asset#test" {
		t.Fatalf("asset owner token = %q", assetCreate.OwnerToken)
	}
	if hasPredicate(assetCreate.Triples, cop.TrackSource) {
		t.Fatal("source asset birth must not emit track-source foreign edges")
	}

	if trackCreate.Entity.ID != "c360.edge.cop.mavlink.track.system-42" {
		t.Fatalf("track id = %q", trackCreate.Entity.ID)
	}
	if trackCreate.Entity.MessageType.Key() != cop.MAVLinkTrackContract().MessageType {
		t.Fatalf("track message type = %q", trackCreate.Entity.MessageType.Key())
	}
	if trackCreate.IndexingProfile != cop.MAVLinkTrackContract().IndexingProfile {
		t.Fatalf("track indexing profile = %q", trackCreate.IndexingProfile)
	}
	if trackCreate.OwnerToken != "semops.feed.mavlink#test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}

	requireTriple(t, trackCreate.Triples, cop.TrackSource, assetCreate.Entity.ID)
	requireTriple(t, trackCreate.Triples, cop.TrackNativeID, "mavlink.system.42.component.7")
	requireTriple(t, trackCreate.Triples, cop.TrackStatus, "active.armed")
}

func TestProjectorUpdatesKnownTrackWithoutRebirth(t *testing.T) {
	projector := NewProjector(Config{
		Org:              "c360",
		Platform:         "edge",
		OwnerTokenSuffix: "test",
		TraceID:          "scenario-001",
	})

	heartbeat := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateHeartbeat(mavcodec.HeartbeatMessage{MavlinkVersion: mavcodec.Version2})
	})
	if _, err := projector.ProjectPacket(heartbeat); err != nil {
		t.Fatalf("project heartbeat: %v", err)
	}

	position := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateGlobalPosition(mavcodec.PositionMessage{
			TimeBootMs:  12345,
			Lat:         389000001,
			Lon:         -770000002,
			Alt:         120000,
			RelativeAlt: 45000,
			Vx:          321,
			Vy:          -12,
			Vz:          7,
			Hdg:         27000,
		})
	})
	plan, err := projector.ProjectPacket(position)
	if err != nil {
		t.Fatalf("project position: %v", err)
	}

	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track update only", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.mavlink.track.system-42" {
		t.Fatalf("update entity id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.MAVLinkTrackContract().IndexingProfile {
		t.Fatalf("update indexing profile = %q", update.IndexingProfile)
	}
	if update.OwnerToken != "semops.feed.mavlink#test" {
		t.Fatalf("update owner token = %q", update.OwnerToken)
	}
	if hasPredicate(update.AddTriples, cop.TrackSource) {
		t.Fatal("track updates must not re-emit source foreign edge after born-first create")
	}
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0000002 38.9000001)")
	requireTriple(t, update.AddTriples, cop.TrackVelocity, "NED_CMPS(321 -12 7)")
}

func TestProjectorMapsAttitudeAndBatterySignals(t *testing.T) {
	projector := NewProjector(Config{OwnerTokenSuffix: "test"})

	attitude := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateAttitude(mavcodec.AttitudeMessage{
			TimeBootMs: 100,
			Roll:       0.11,
			Pitch:      -0.05,
			Yaw:        1.57,
		})
	})
	plan, err := projector.ProjectPacket(attitude)
	if err != nil {
		t.Fatalf("project attitude: %v", err)
	}
	trackCreate := requireCreate(t, plan.Mutations[1])
	requireTriple(t, trackCreate.Triples, cop.TrackRoll, float64(0.11))
	requireTriple(t, trackCreate.Triples, cop.TrackPitch, float64(-0.05))
	requireTriple(t, trackCreate.Triples, cop.TrackYaw, float64(1.57))

	battery := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateBatteryStatus(mavcodec.BatteryMessage{BatteryRemaining: 85})
	})
	plan, err = projector.ProjectPacket(battery)
	if err != nil {
		t.Fatalf("project battery: %v", err)
	}
	update := requireUpdate(t, plan.Mutations[0])
	requireTriple(t, update.AddTriples, cop.TrackBattery, int64(85))
}

func TestProjectorIncludesRawSourceReferenceWithoutRawEntityBirth(t *testing.T) {
	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	parser := mavcodec.NewParser()
	packets, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("parse heartbeat: %v", err)
	}
	if len(packets) != 1 {
		t.Fatalf("packets = %d, want 1", len(packets))
	}
	lane := mavcodec.NewRawLane(mavcodec.RawLaneConfig{
		Source:     "udp:14550",
		MaxRecords: 8,
		MaxBytes:   1024,
	})
	record, err := lane.Capture(frame, packets[0])
	if err != nil {
		t.Fatalf("capture raw frame: %v", err)
	}

	projector := NewProjector(Config{OwnerTokenSuffix: "test"})
	plan, err := projector.ProjectPacket(packets[0])
	if err != nil {
		t.Fatalf("project heartbeat: %v", err)
	}
	if len(plan.Mutations) != 2 {
		t.Fatalf("mutations = %d, want asset birth + track birth only", len(plan.Mutations))
	}
	assetCreate := requireCreate(t, plan.Mutations[0])
	trackCreate := requireCreate(t, plan.Mutations[1])
	requireTriple(t, assetCreate.Triples, cop.ProvenanceSourceRef, record.Ref)
	requireTriple(t, trackCreate.Triples, cop.ProvenanceSourceRef, record.Ref)
	for _, mutation := range plan.Mutations {
		if mutation.Kind != MutationCreate && mutation.Kind != MutationUpdate {
			t.Fatalf("unexpected mutation kind %q", mutation.Kind)
		}
	}
}

func TestProjectorIgnoresUnsupportedMessagesWithoutBirth(t *testing.T) {
	projector := NewProjector(Config{})
	plan, err := projector.ProjectPacket(&mavcodec.Packet{
		MessageID:   999,
		SystemID:    42,
		ComponentID: 7,
		Timestamp:   time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("project unsupported packet: %v", err)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want no-op", len(plan.Mutations))
	}
}

func parseGeneratedPacket(t *testing.T, generate func(*mavcodec.Generator) ([]byte, error)) *mavcodec.Packet {
	t.Helper()

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generate(generator)
	if err != nil {
		t.Fatalf("generate frame: %v", err)
	}
	parser := mavcodec.NewParser()
	packets, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("parse frame: %v", err)
	}
	if len(packets) != 1 {
		t.Fatalf("packets = %d, want 1", len(packets))
	}
	packets[0].Timestamp = time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	return packets[0]
}

func requireCreate(t *testing.T, mutation Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	if mutation.Create.Entity == nil {
		t.Fatal("create entity is nil")
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationUpdate {
		t.Fatalf("mutation kind = %q, want update", mutation.Kind)
	}
	if mutation.Update.Entity == nil {
		t.Fatal("update entity is nil")
	}
	return mutation.Update
}

func requireTriple(t *testing.T, triples []graphTriple, predicate string, want any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate {
			if triple.Object != want {
				t.Fatalf("%s object = %#v, want %#v", predicate, triple.Object, want)
			}
			return
		}
	}
	t.Fatalf("missing predicate %q in %+v", predicate, triples)
}

func hasPredicate(triples []graphTriple, predicate string) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return true
		}
	}
	return false
}
