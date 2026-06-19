package cot

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestAdapterIngestCapturesReplayAndHealth(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 20, 0, 0, time.UTC)
	events := cotcodec.SeedEvents(now)
	raw, err := cotcodec.Marshal(events[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	storePath := filepath.Join(t.TempDir(), "cot.jsonl")
	seen := make(chan IngestResult, 1)
	adapter, err := NewAdapter(Config{
		Source: "tak:unit",
		Replay: cotcodec.NewReplayStore(storePath),
		Clock:  func() time.Time { return now },
		OnEvent: func(result IngestResult) {
			seen <- result
		},
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), raw)
	if err != nil {
		t.Fatalf("ingest event: %v", err)
	}
	if result.RawRef != "cot://raw/tak-unit/00000001" || result.Event.UID != "ANDROID-ALPHA" {
		t.Fatalf("result = %+v", result)
	}
	select {
	case got := <-seen:
		if got.Event.UID != "ANDROID-ALPHA" {
			t.Fatalf("seen uid = %q", got.Event.UID)
		}
	default:
		t.Fatal("expected OnEvent notification")
	}

	health := adapter.Health()
	if !health.Ready || health.EventsReceived != 1 || health.EventsCaptured != 1 || health.EventsDecoded != 1 {
		t.Fatalf("health = %+v", health)
	}
	if health.LastUID != "ANDROID-ALPHA" || health.LastType != cotcodec.TypeOperatorPosition || health.LastRawRef != result.RawRef {
		t.Fatalf("last health fields = %+v", health)
	}

	records, err := cotcodec.LoadReplay(storePath)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(records) != 1 || records[0].UID != "ANDROID-ALPHA" {
		t.Fatalf("replay records = %+v", records)
	}
}

func TestAdapterCapturesMalformedInputBeforeRejecting(t *testing.T) {
	adapter, err := NewAdapter(Config{Source: "tak:unit"})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), []byte("<event"))
	if err == nil {
		t.Fatal("expected malformed CoT rejection")
	}
	if result.RawRef == "" {
		t.Fatal("malformed input should still be replay-addressable")
	}
	health := adapter.Health()
	if health.Ready || health.EventsReceived != 1 || health.EventsCaptured != 1 || health.ParseErrors != 1 {
		t.Fatalf("health = %+v", health)
	}
	if len(adapter.RawLane().Snapshot()) != 1 {
		t.Fatalf("raw lane records = %d, want 1", len(adapter.RawLane().Snapshot()))
	}
}

func TestAdapterReportsReplayAppendErrors(t *testing.T) {
	adapter, err := NewAdapter(Config{
		Source: "tak:unit",
		Replay: failingReplayAppender{},
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(time.Now())[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}

	if _, err := adapter.IngestEvent(context.Background(), raw); err == nil {
		t.Fatal("expected replay append error")
	}
	health := adapter.Health()
	if health.ReplayErrors != 1 || health.Ready {
		t.Fatalf("health = %+v", health)
	}
}

func TestAdapterIngestCapturesProjectsAndWrites(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 20, 0, 0, time.UTC)
	writer := &recordingPlanWriter{}
	seen := make(chan IngestResult, 1)
	adapter := newGraphAdapter(t, writer, func() time.Time { return now }, func(result IngestResult) {
		seen <- result
	})
	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(now)[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), raw)
	if err != nil {
		t.Fatalf("ingest event: %v", err)
	}
	if result.RawRef != "cot://raw/tak-unit/00000001" {
		t.Fatalf("raw ref = %q", result.RawRef)
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want source asset + track", result.Mutations)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans written = %d, want 1", len(writer.plans))
	}

	plan := writer.plans[0]
	trackCreate := requireCreate(t, plan.Mutations[1])
	requireTriple(t, trackCreate.Triples, cop.ProvenanceSourceRef, result.RawRef)
	requireTriple(t, trackCreate.Triples, cop.TrackSource, "c360.edge.cop.tak.asset.android-alpha")

	select {
	case got := <-seen:
		if got.Mutations != 2 || got.Event.UID != "ANDROID-ALPHA" {
			t.Fatalf("seen result = %+v", got)
		}
	default:
		t.Fatal("expected OnEvent notification after graph write")
	}

	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.GraphMutations != 2 {
		t.Fatalf("graph mutations = %d, want 2", health.GraphMutations)
	}
	if health.LastGraphWriteAt != now {
		t.Fatalf("last graph write = %s, want %s", health.LastGraphWriteAt, now)
	}
}

func TestAdapterIgnoresUnsupportedEventsWithoutGraphWrite(t *testing.T) {
	writer := &recordingPlanWriter{}
	adapter := newGraphAdapter(t, writer, time.Now, nil)
	raw, err := cotcodec.Marshal(cotcodec.Event{
		UID:  "ALERT-1",
		Type: cotcodec.TypeAlert,
		Time: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal alert: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), raw)
	if err != nil {
		t.Fatalf("ingest unsupported alert: %v", err)
	}
	if result.Mutations != 0 {
		t.Fatalf("mutations = %d, want 0", result.Mutations)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("plans written = %d, want 0", len(writer.plans))
	}
	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.ProjectionDrops != 1 {
		t.Fatalf("projection drops = %d, want unsupported event drop", health.ProjectionDrops)
	}
}

func TestAdapterRecordsWriterFailure(t *testing.T) {
	writer := &recordingPlanWriter{err: errors.New("graph unavailable")}
	adapter := newGraphAdapter(t, writer, time.Now, nil)
	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(time.Now())[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), raw)
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
			"c360.edge.cop.tak.asset.android-alpha": {},
			"c360.edge.cop.tak.track.android-alpha": {},
		},
	}
	now := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	adapter := newGraphAdapter(t, writer, func() time.Time { return now }, nil)
	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(now)[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), raw)
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
	if firstCreate.Entity.ID != "c360.edge.cop.tak.asset.android-alpha" {
		t.Fatalf("first create = %q", firstCreate.Entity.ID)
	}
	secondCreate := requireCreate(t, writer.plans[1].Mutations[0])
	if secondCreate.Entity.ID != "c360.edge.cop.tak.track.android-alpha" {
		t.Fatalf("second create = %q", secondCreate.Entity.ID)
	}
	update := requireUpdate(t, writer.plans[2].Mutations[0])
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0350000 38.8920000)")
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

func TestAdapterRequiresWriterWhenProjectorConfigured(t *testing.T) {
	_, err := NewAdapter(Config{Projector: cotprojector.NewProjector(cotprojector.Config{})})
	if err == nil {
		t.Fatal("expected missing graph writer error")
	}
}

func newGraphAdapter(
	t *testing.T,
	writer PlanWriter,
	clock func() time.Time,
	onEvent func(IngestResult),
) *Adapter {
	t.Helper()
	adapter, err := NewAdapter(Config{
		Source: "tak:unit",
		RawLane: cotcodec.NewRawLane(cotcodec.RawLaneConfig{
			Source:     "tak:unit",
			MaxRecords: 16,
			MaxBytes:   4096,
			Clock:      clock,
		}),
		Projector: cotprojector.NewProjector(cotprojector.Config{
			OwnerTokens: testOwnerTokens("adapter-test"),
			TraceID:     "adapter-test",
		}),
		Writer:  writer,
		Clock:   clock,
		OnEvent: onEvent,
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return adapter
}

type recordingPlanWriter struct {
	plans []cotprojector.Plan
	err   error
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan cotprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

type birthConflictPlanWriter struct {
	plans     []cotprojector.Plan
	conflicts map[string]struct{}
}

func (w *birthConflictPlanWriter) Apply(_ context.Context, plan cotprojector.Plan) error {
	w.plans = append(w.plans, plan)
	for _, mutation := range plan.Mutations {
		if mutation.Kind != cotprojector.MutationCreate || mutation.Create.Entity == nil {
			continue
		}
		entityID := mutation.Create.Entity.ID
		if _, ok := w.conflicts[entityID]; !ok {
			continue
		}
		delete(w.conflicts, entityID)
		return &cotprojector.MutationFailureError{
			Operation: "create_with_triples",
			Kind:      cotprojector.MutationCreate,
			EntityID:  entityID,
			ErrorCode: graph.ErrorCodeEntityExists,
			Message:   "entity already exists",
		}
	}
	return nil
}

func requireCreate(t *testing.T, mutation cotprojector.Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != cotprojector.MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	if mutation.Create.Entity == nil {
		t.Fatal("create entity is nil")
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation cotprojector.Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != cotprojector.MutationUpdate {
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
		cop.OwnerAsset: ownership.ExpectedOwnerToken(cop.OwnerAsset, incarnation),
		cop.OwnerTAK:   ownership.ExpectedOwnerToken(cop.OwnerTAK, incarnation),
	}
}

type failingReplayAppender struct{}

func (failingReplayAppender) Append(cotcodec.RawEventRecord) error {
	return errors.New("disk full")
}
