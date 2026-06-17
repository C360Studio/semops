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

func TestAdapterCapturesCommandAckWithoutGraphWrite(t *testing.T) {
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
	if result.Mutations != 0 {
		t.Fatalf("mutations = %d, want 0 for command ack current-state projection", result.Mutations)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("plans written = %d, want 0", len(writer.plans))
	}
	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.ProjectionDrops != 1 {
		t.Fatalf("projection drops = %d, want unsupported packet drop", health.ProjectionDrops)
	}
	if health.GraphMutations != 0 {
		t.Fatalf("graph mutations = %d, want 0", health.GraphMutations)
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
			OwnerTokenSuffix: "adapter-test",
			TraceID:          "adapter-test",
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
