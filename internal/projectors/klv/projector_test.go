package klv

import (
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectorCreatesKLVSensorFootprintWithoutForeignEdges(t *testing.T) {
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "scenario-klv-001",
	})

	plan, err := projector.ProjectFrame(sampleFrame())
	if err != nil {
		t.Fatalf("project KLV frame: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want sensor-footprint birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	if create.Entity.ID != "c360.edge.cop.klv.sensor_footprint.object-semops-klv-deterministic-001-ts" {
		t.Fatalf("sensor-footprint id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.KLVSensorFootprintContract().MessageType {
		t.Fatalf("message type = %q", create.Entity.MessageType.Key())
	}
	if create.Entity.UpdatedAt != sampleFrameTime() {
		t.Fatalf("updated_at = %s, want %s", create.Entity.UpdatedAt, sampleFrameTime())
	}
	if create.IndexingProfile != cop.KLVSensorFootprintContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.OwnerToken != "semops.feed.klv#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "scenario-klv-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}

	requireTriple(t, create.Triples, cop.SensorFootprintMediaRef, "object://semops/klv/deterministic-001.ts")
	requireTriple(t, create.Triples, cop.SensorFootprintPacketRef, "klv://packet/deterministic/00000001")
	requireTriple(t, create.Triples, cop.SensorFootprintSensorPosition, "POINT(-117.1234560 34.1234560)")
	requireTriple(t, create.Triples, cop.SensorFootprintFrameCenter, "POINT(-117.1202220 34.1250010)")
	requireTriple(t, create.Triples, cop.SensorFootprintSensorAzimuth, 87.5)
	requireTriple(t, create.Triples, cop.SensorFootprintSensorElevation, -12.25)
	requireTriple(t, create.Triples, cop.ProvenanceSource, "klv")
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "klv://packet/deterministic/00000001")
}

func TestProjectorUpdatesKnownKLVSensorFootprint(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	first, err := projector.ProjectFrame(sampleFrame())
	if err != nil {
		t.Fatalf("project first frame: %v", err)
	}
	if marked := projector.MarkBornForPlan(first); marked != 1 {
		t.Fatalf("marked births = %d, want sensor footprint", marked)
	}

	frame := sampleFrame()
	lat, lon := 34.126, -117.121
	frame.FrameCenterLatitude = &lat
	frame.FrameCenterLongitude = &lon
	frame.PacketRef = "klv://packet/deterministic/00000002"
	frame.FrameTime = sampleFrameTime().Add(time.Second)
	plan, err := projector.ProjectFrame(frame)
	if err != nil {
		t.Fatalf("project update frame: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want sensor-footprint update", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.klv.sensor_footprint.object-semops-klv-deterministic-001-ts" {
		t.Fatalf("update entity id = %q", update.Entity.ID)
	}
	if update.OwnerToken != "semops.feed.klv#test" {
		t.Fatalf("update owner token = %q", update.OwnerToken)
	}
	requireTriple(t, update.AddTriples, cop.SensorFootprintPacketRef, "klv://packet/deterministic/00000002")
	requireTriple(t, update.AddTriples, cop.SensorFootprintFrameCenter, "POINT(-117.1210000 34.1260000)")
}

func TestProjectorCanSeedBornStateForRestartReconciliation(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	if !projector.MarkBornForFrame(sampleFrame(), "c360.edge.cop.klv.sensor_footprint.object-semops-klv-deterministic-001-ts") {
		t.Fatal("expected sensor-footprint born-state seed")
	}
	plan, err := projector.ProjectFrame(sampleFrame())
	if err != nil {
		t.Fatalf("project after born seed: %v", err)
	}
	if len(plan.Mutations) != 1 || plan.Mutations[0].Kind != MutationUpdate {
		t.Fatalf("plan = %#v, want update after born seed", plan)
	}
}

func sampleFrame() Frame {
	sensorLat, sensorLon := 34.123456, -117.123456
	sensorAlt := 1420.25
	azimuth, elevation := 87.5, -12.25
	centerLat, centerLon := 34.125001, -117.120222
	centerElevation := 905.5
	return Frame{
		Source:                     "klv:decode",
		MediaRef:                   "object://semops/klv/deterministic-001.ts",
		PacketRef:                  "klv://packet/deterministic/00000001",
		ReceivedAt:                 sampleFrameTime().Add(2 * time.Second),
		FrameTime:                  sampleFrameTime(),
		PlatformDesignation:        "SYNTHETIC-UAS-1",
		SensorLatitude:             &sensorLat,
		SensorLongitude:            &sensorLon,
		SensorAltitudeMeters:       &sensorAlt,
		SensorAzimuthDegrees:       &azimuth,
		SensorElevationDegrees:     &elevation,
		FrameCenterLatitude:        &centerLat,
		FrameCenterLongitude:       &centerLon,
		FrameCenterElevationMeters: &centerElevation,
	}
}

func sampleFrameTime() time.Time {
	return time.Date(2026, 6, 22, 17, 59, 58, 123456000, time.UTC)
}

func requireCreate(t *testing.T, mutation Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationUpdate {
		t.Fatalf("mutation kind = %q, want update", mutation.Kind)
	}
	return mutation.Update
}

func requireTriple(t *testing.T, triples []message.Triple, predicate string, object any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate && triple.Object == object {
			return
		}
	}
	t.Fatalf("missing triple %s=%v in %#v", predicate, object, triples)
}

func testOwnerTokens(incarnation string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerKLV: ownership.ExpectedOwnerToken(cop.OwnerKLV, incarnation),
	}
}
