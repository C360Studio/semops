package adsb

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestAdapterIngestCapturesReplayProjectsAndReportsHealth(t *testing.T) {
	now := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	records, err := adsbcodec.OpenSkyFixtureRecords(now)
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	storePath := filepath.Join(t.TempDir(), "adsb.jsonl")
	writer := &recordingWriter{}
	adapter, err := NewAdapter(Config{
		Source: "opensky-fixture",
		RawLane: adsbcodec.NewRawLane(adsbcodec.RawLaneConfig{
			Source: "opensky-fixture",
			Clock:  fixedClock(now),
		}),
		Replay: adsbcodec.NewReplayStore(storePath),
		Projector: adsbprojector.NewProjector(adsbprojector.Config{
			Org:         "c360",
			Platform:    "adapter",
			OwnerTokens: testOwnerTokens("adapter-test"),
			TraceID:     "adsb-adapter-test",
		}),
		Writer: writer,
		Clock:  fixedClock(now),
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	first, err := adapter.IngestSnapshot(context.Background(), records[0].RawJSON)
	if err != nil {
		t.Fatalf("ingest first snapshot: %v", err)
	}
	if first.RawRef != "adsb://raw/opensky-fixture/00000001" ||
		first.StatesDecoded != 2 ||
		first.Mutations != 2 {
		t.Fatalf("first result = %+v", first)
	}
	if len(writer.plans) != 1 ||
		writer.plans[0].Mutations[0].Kind != adsbprojector.MutationCreate ||
		writer.plans[0].Mutations[1].Kind != adsbprojector.MutationCreate {
		t.Fatalf("first writer plans = %+v", writer.plans)
	}

	second, err := adapter.IngestSnapshot(context.Background(), records[1].RawJSON)
	if err != nil {
		t.Fatalf("ingest second snapshot: %v", err)
	}
	if second.RawRef != "adsb://raw/opensky-fixture/00000002" ||
		second.StatesDecoded != 2 ||
		second.Mutations != 2 {
		t.Fatalf("second result = %+v", second)
	}
	if len(writer.plans) != 2 ||
		writer.plans[1].Mutations[0].Kind != adsbprojector.MutationUpdate ||
		writer.plans[1].Mutations[1].Kind != adsbprojector.MutationCreate {
		t.Fatalf("second writer plans = %+v", writer.plans)
	}

	loaded, err := adsbcodec.LoadReplay(storePath)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(loaded) != 2 ||
		loaded[0].Ref != first.RawRef ||
		loaded[1].Ref != second.RawRef {
		t.Fatalf("loaded replay = %+v", loaded)
	}
	health := adapter.Health()
	if !health.Ready ||
		health.SnapshotsReceived != 2 ||
		health.SnapshotsCaptured != 2 ||
		health.SnapshotsDecoded != 2 ||
		health.StatesDecoded != 4 ||
		health.GraphMutations != 4 ||
		health.LastRawRef != second.RawRef ||
		health.LastICAO24 != "b7c8d9" {
		t.Fatalf("health = %+v", health)
	}
}

func TestAdapterCapturesMalformedSnapshotsBeforeParseFailure(t *testing.T) {
	now := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	storePath := filepath.Join(t.TempDir(), "adsb.jsonl")
	writer := &recordingWriter{}
	adapter, err := NewAdapter(Config{
		Source:    "opensky-fixture",
		RawLane:   adsbcodec.NewRawLane(adsbcodec.RawLaneConfig{Source: "opensky-fixture", Clock: fixedClock(now)}),
		Replay:    adsbcodec.NewReplayStore(storePath),
		Projector: adsbprojector.NewProjector(adsbprojector.Config{OwnerTokens: testOwnerTokens("adapter-test")}),
		Writer:    writer,
		Clock:     fixedClock(now),
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	result, err := adapter.IngestSnapshot(context.Background(), []byte(`{"states":[]}`))
	if err == nil || !strings.Contains(err.Error(), "time") {
		t.Fatalf("ingest malformed error = %v", err)
	}
	if result.RawRef != "adsb://raw/opensky-fixture/00000001" {
		t.Fatalf("malformed result = %+v", result)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer should not be called for malformed snapshots: %+v", writer.plans)
	}
	loaded, err := adsbcodec.LoadReplay(storePath)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Ref != result.RawRef {
		t.Fatalf("loaded replay = %+v", loaded)
	}
	health := adapter.Health()
	if health.Ready ||
		health.ParseErrors != 1 ||
		health.SnapshotsCaptured != 1 ||
		health.LastRawRef != result.RawRef ||
		health.LastError == "" {
		t.Fatalf("health = %+v", health)
	}
}

func TestAdapterRequiresWriterWhenProjectorConfigured(t *testing.T) {
	_, err := NewAdapter(Config{Projector: adsbprojector.NewProjector(adsbprojector.Config{})})
	if err == nil || !strings.Contains(err.Error(), "plan writer") {
		t.Fatalf("adapter error = %v, want writer requirement", err)
	}
}

func TestAdapterReconcilesAlreadyBornADSBBirths(t *testing.T) {
	now := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	records, err := adsbcodec.OpenSkyFixtureRecords(now)
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	writer := &recordingWriter{
		errs: []error{&adsbprojector.MutationFailureError{
			Operation: "create_with_triples",
			Kind:      adsbprojector.MutationCreate,
			EntityID:  "c360.adapter.cop.adsb.track.a1b2c3",
			ErrorCode: graph.ErrorCodeEntityExists,
			Message:   "already exists",
		}},
	}
	adapter, err := NewAdapter(Config{
		RawLane: adsbcodec.NewRawLane(adsbcodec.RawLaneConfig{Clock: fixedClock(now)}),
		Projector: adsbprojector.NewProjector(adsbprojector.Config{
			Org:         "c360",
			Platform:    "adapter",
			OwnerTokens: testOwnerTokens("adapter-test"),
		}),
		Writer: writer,
		Clock:  fixedClock(now),
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	result, err := adapter.IngestSnapshot(context.Background(), records[0].RawJSON)
	if err != nil {
		t.Fatalf("ingest snapshot: %v", err)
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want reprojected update+create count", result.Mutations)
	}
	if len(writer.plans) != 2 {
		t.Fatalf("writer plans = %d, want initial and reconciled plans", len(writer.plans))
	}
	if writer.plans[0].Mutations[0].Kind != adsbprojector.MutationCreate ||
		writer.plans[1].Mutations[0].Kind != adsbprojector.MutationUpdate ||
		writer.plans[1].Mutations[1].Kind != adsbprojector.MutationCreate {
		t.Fatalf("writer plans = %+v", writer.plans)
	}
	if health := adapter.Health(); !health.Ready || health.WriteErrors != 0 || health.GraphMutations != 2 {
		t.Fatalf("health = %+v", health)
	}
}

type recordingWriter struct {
	plans []adsbprojector.Plan
	errs  []error
	err   error
}

func (w *recordingWriter) Apply(_ context.Context, plan adsbprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.errs) > 0 {
		err := w.errs[0]
		w.errs = w.errs[1:]
		return err
	}
	return w.err
}

func testOwnerTokens(incarnation string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerADSB: ownership.ExpectedOwnerToken(cop.OwnerADSB, incarnation),
	}
}

func fixedClock(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

var _ PlanWriter = (*recordingWriter)(nil)
var _ error = (*adsbprojector.MutationFailureError)(nil)
