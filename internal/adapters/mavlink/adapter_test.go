package mavlink

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestAdapterIngestFrameCapturesProjectsAndWrites(t *testing.T) {
	now := time.Date(2026, 6, 17, 14, 0, 0, 0, time.UTC)
	writer := &recordingPlanWriter{}
	adapter := newTestAdapter(t, writer, func() time.Time { return now })

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}

	result, err := adapter.IngestFrame(context.Background(), frame)
	if err != nil {
		t.Fatalf("ingest frame: %v", err)
	}
	if result.PacketsDecoded != 1 {
		t.Fatalf("packets decoded = %d, want 1", result.PacketsDecoded)
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want asset birth + track birth", result.Mutations)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans written = %d, want 1", len(writer.plans))
	}

	plan := writer.plans[0]
	trackCreate := requireCreate(t, plan.Mutations[1])
	requireTriple(t, trackCreate.Triples, cop.ProvenanceSourceRef, result.RawRef)

	snapshot := adapter.RawLane().Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("raw records = %d, want 1", len(snapshot))
	}
	if snapshot[0].Ref != result.RawRef {
		t.Fatalf("raw ref = %q, want %q", snapshot[0].Ref, result.RawRef)
	}

	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.FramesReceived != 1 || health.FramesCaptured != 1 || health.PacketsDecoded != 1 {
		t.Fatalf("health counts = frames %d captured %d packets %d",
			health.FramesReceived, health.FramesCaptured, health.PacketsDecoded)
	}
	if health.GraphMutations != 2 {
		t.Fatalf("graph mutations = %d, want 2", health.GraphMutations)
	}
	if health.LastGraphWriteAt != now {
		t.Fatalf("last graph write = %s, want %s", health.LastGraphWriteAt, now)
	}
}

func TestAdapterCapturesCommandAckAndWritesControlTask(t *testing.T) {
	writer := &recordingPlanWriter{}
	adapter := newTestAdapter(t, writer, time.Now)

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateCommandAck(mavcodec.CommandAckMessage{
		Command:           mavcodec.CommandComponentArmDisarm,
		Result:            mavcodec.MAVResultAccepted,
		TargetSystemID:    255,
		TargetComponentID: 1,
	})
	if err != nil {
		t.Fatalf("generate command ack: %v", err)
	}

	result, err := adapter.IngestFrame(context.Background(), frame)
	if err != nil {
		t.Fatalf("ingest command ack: %v", err)
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want source asset + command task create", result.Mutations)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans written = %d, want 1", len(writer.plans))
	}
	taskCreate := requireCreate(t, writer.plans[0].Mutations[1])
	if taskCreate.Entity.ID != "c360.edge.cop.mavlink.task.system-42-command-400-target-255-1" {
		t.Fatalf("task id = %q", taskCreate.Entity.ID)
	}
	if taskCreate.IndexingProfile != cop.MAVLinkCommandTaskContract().IndexingProfile {
		t.Fatalf("task indexing profile = %q", taskCreate.IndexingProfile)
	}
	requireTriple(t, taskCreate.Triples, cop.TaskTarget, "c360.edge.cop.mavlink.asset.system-42")
	requireTriple(t, taskCreate.Triples, cop.TaskStatus, "accepted")
	requireTriple(t, taskCreate.Triples, cop.ProvenanceSourceRef, result.RawRef)
	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.ProjectionDrops != 0 {
		t.Fatalf("projection drops = %d, want 0", health.ProjectionDrops)
	}
	if health.GraphMutations != 2 {
		t.Fatalf("graph mutations = %d, want source asset + command task create", health.GraphMutations)
	}
}

func TestAdapterCapturesInvalidFrameWithoutGraphWrite(t *testing.T) {
	writer := &recordingPlanWriter{}
	adapter := newTestAdapter(t, writer, time.Now)

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{MavlinkVersion: mavcodec.Version2})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	frame[len(frame)-1] ^= 0xff

	result, err := adapter.IngestFrame(context.Background(), frame)
	if err == nil {
		t.Fatal("expected invalid frame error")
	}
	if result.RawRef == "" {
		t.Fatal("invalid frame should still be captured in raw lane")
	}
	if len(writer.plans) != 0 {
		t.Fatalf("plans written = %d, want none for invalid frame", len(writer.plans))
	}

	health := adapter.Health()
	if health.Ready {
		t.Fatal("health ready = true after invalid frame")
	}
	if health.ParseErrors != 1 || health.FramesCaptured != 1 {
		t.Fatalf("health parse/captured = %d/%d, want 1/1", health.ParseErrors, health.FramesCaptured)
	}
	if !strings.Contains(health.LastError, "expected exactly one valid MAVLink packet") {
		t.Fatalf("last error = %q", health.LastError)
	}
}

func TestAdapterRecordsWriterFailure(t *testing.T) {
	writer := &recordingPlanWriter{err: errors.New("graph unavailable")}
	adapter := newTestAdapter(t, writer, time.Now)

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{MavlinkVersion: mavcodec.Version2})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}

	result, err := adapter.IngestFrame(context.Background(), frame)
	if err == nil {
		t.Fatal("expected writer failure")
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want attempted plan mutations", result.Mutations)
	}
	health := adapter.Health()
	if health.Ready {
		t.Fatal("health ready = true after writer failure")
	}
	if health.WriteErrors != 1 {
		t.Fatalf("write errors = %d, want 1", health.WriteErrors)
	}
	if health.GraphMutations != 0 {
		t.Fatalf("graph mutations = %d, want no committed graph mutations", health.GraphMutations)
	}
}

func TestAdapterReconcilesExistingBirthsAfterRestart(t *testing.T) {
	writer := &birthConflictPlanWriter{
		conflicts: map[string]struct{}{
			"c360.edge.cop.mavlink.asset.system-42": {},
			"c360.edge.cop.mavlink.track.system-42": {},
		},
	}
	now := time.Date(2026, 6, 17, 15, 0, 0, 0, time.UTC)
	adapter := newTestAdapter(t, writer, func() time.Time { return now })

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateGlobalPosition(mavcodec.PositionMessage{
		Lat: 389000001,
		Lon: -770000002,
		Vx:  321,
		Vy:  -12,
		Vz:  7,
	})
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}

	result, err := adapter.IngestFrame(context.Background(), frame)
	if err != nil {
		t.Fatalf("ingest after restart: %v", err)
	}
	if result.Mutations != 1 {
		t.Fatalf("mutations = %d, want reconciled update", result.Mutations)
	}
	if len(writer.plans) != 3 {
		t.Fatalf("plans = %d, want asset conflict, track conflict, update", len(writer.plans))
	}
	firstCreate := requireCreate(t, writer.plans[0].Mutations[0])
	if firstCreate.Entity.ID != "c360.edge.cop.mavlink.asset.system-42" {
		t.Fatalf("first create = %q", firstCreate.Entity.ID)
	}
	secondCreate := requireCreate(t, writer.plans[1].Mutations[0])
	if secondCreate.Entity.ID != "c360.edge.cop.mavlink.track.system-42" {
		t.Fatalf("second create = %q", secondCreate.Entity.ID)
	}
	update := requireUpdate(t, writer.plans[2].Mutations[0])
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0000002 38.9000001)")
	if hasPredicate(update.AddTriples, cop.TrackSource) {
		t.Fatal("reconciled update must not repeat strict source edge")
	}

	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.WriteErrors != 0 {
		t.Fatalf("write errors = %d, want 0", health.WriteErrors)
	}
	if health.GraphMutations != 1 {
		t.Fatalf("graph mutations = %d, want final update only", health.GraphMutations)
	}
}

func newTestAdapter(t *testing.T, writer PlanWriter, clock func() time.Time) *Adapter {
	t.Helper()
	adapter, err := NewAdapter(Config{
		Source: "udp:14550",
		RawLane: mavcodec.NewRawLane(mavcodec.RawLaneConfig{
			Source:     "udp:14550",
			MaxRecords: 16,
			MaxBytes:   4096,
			Clock:      clock,
		}),
		Projector: mavprojector.NewProjector(mavprojector.Config{
			OwnerTokens: testOwnerTokens("adapter-test"),
			TraceID:     "adapter-test",
		}),
		Writer: writer,
		Clock:  clock,
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return adapter
}

type recordingPlanWriter struct {
	plans []mavprojector.Plan
	err   error
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

type birthConflictPlanWriter struct {
	plans     []mavprojector.Plan
	conflicts map[string]struct{}
}

func (w *birthConflictPlanWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.plans = append(w.plans, plan)
	for _, mutation := range plan.Mutations {
		if mutation.Kind != mavprojector.MutationCreate || mutation.Create.Entity == nil {
			continue
		}
		entityID := mutation.Create.Entity.ID
		if _, ok := w.conflicts[entityID]; !ok {
			continue
		}
		delete(w.conflicts, entityID)
		return &mavprojector.MutationFailureError{
			Operation: "create_with_triples",
			Kind:      mavprojector.MutationCreate,
			EntityID:  entityID,
			ErrorCode: graph.ErrorCodeEntityExists,
			Message:   "entity already exists",
		}
	}
	return nil
}

func requireCreate(t *testing.T, mutation mavprojector.Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != mavprojector.MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	if mutation.Create.Entity == nil {
		t.Fatal("create entity is nil")
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation mavprojector.Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != mavprojector.MutationUpdate {
		t.Fatalf("mutation kind = %q, want update", mutation.Kind)
	}
	if mutation.Update.Entity == nil {
		t.Fatal("update entity is nil")
	}
	return mutation.Update
}

func requireTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
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

func hasPredicate(triples []message.Triple, predicate string) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return true
		}
	}
	return false
}

func testOwnerTokens(incarnation string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerAsset:   ownership.ExpectedOwnerToken(cop.OwnerAsset, incarnation),
		cop.OwnerMAVLink: ownership.ExpectedOwnerToken(cop.OwnerMAVLink, incarnation),
	}
}
