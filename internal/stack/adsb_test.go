package stack

import (
	"context"
	"strings"
	"testing"
	"time"

	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
)

func TestNewADSBAdapterWiresNATSGraphWriter(t *testing.T) {
	now := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	retry := natsclient.RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	records, err := adsbcodec.OpenSkyFixtureRecords(now)
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	client := &recordingRetryRequester{}
	adapter, err := NewADSBAdapter(ADSBAdapterConfig{
		Source:        "opensky:fixture",
		Org:           "c360",
		Platform:      "edge",
		OwnerTokens:   testOwnerTokens("stack-test"),
		TraceID:       "adsb-stack-test",
		RawMaxRecords: 2,
		RawMaxBytes:   4096,
		WriteTimeout:  25 * time.Millisecond,
		Retry:         retry,
		Clock:         func() time.Time { return now },
	}, ADSBAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	if _, err := adapter.IngestSnapshot(context.Background(), records[0].RawJSON); err != nil {
		t.Fatalf("ingest snapshot: %v", err)
	}
	if len(client.calls) != 2 {
		t.Fatalf("requests = %d, want two ADS-B track creates", len(client.calls))
	}
	for i, call := range client.calls {
		if call.subject != adsbprojector.SubjectEntityCreateWithTriples {
			t.Fatalf("request %d subject = %q", i, call.subject)
		}
		if call.timeout != 25*time.Millisecond {
			t.Fatalf("request %d timeout = %s, want 25ms", i, call.timeout)
		}
		if call.retry != retry {
			t.Fatalf("request %d retry = %+v, want %+v", i, call.retry, retry)
		}
	}

	var trackCreate graph.CreateEntityWithTriplesRequest
	decodePayload(t, client.calls[0].payload, &trackCreate)
	if trackCreate.Entity.ID != "c360.edge.cop.adsb.track.a1b2c3" {
		t.Fatalf("track id = %q", trackCreate.Entity.ID)
	}
	if trackCreate.OwnerToken != "semops.feed.adsb#stack-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	if trackCreate.TraceID != "adsb-stack-test" {
		t.Fatalf("track trace id = %q", trackCreate.TraceID)
	}
	if trackCreate.IndexingProfile != cop.ADSBTrackContract().IndexingProfile {
		t.Fatalf("track indexing profile = %q", trackCreate.IndexingProfile)
	}
	if hasPredicate(trackCreate.Triples, cop.TrackSource) {
		t.Fatal("ADS-B track creates must not emit track source association edges")
	}
	requireTriple(t, trackCreate.Triples, cop.TrackPosition, "POINT(-77.0400000 38.9000000)")

	if records := adapter.RawLane().Snapshot(); len(records) != 1 {
		t.Fatalf("raw records = %d, want 1", len(records))
	}
	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.GraphMutations != 2 {
		t.Fatalf("graph mutations = %d, want 2", health.GraphMutations)
	}
}

func TestNewADSBAdapterSupportsWriterInjection(t *testing.T) {
	records, err := adsbcodec.OpenSkyFixtureRecords(time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	writer := &recordingADSBPlanWriter{}
	adapter, err := NewADSBAdapter(ADSBAdapterConfig{
		Source: "fixture",
	}, ADSBAdapterDeps{Writer: writer})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	result, err := adapter.IngestSnapshot(context.Background(), records[0].RawJSON)
	if err != nil {
		t.Fatalf("ingest snapshot: %v", err)
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want two ADS-B track births", result.Mutations)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	if got := adapter.Health().Source; got != "fixture" {
		t.Fatalf("health source = %q, want fixture", got)
	}
}

func TestNewADSBAdapterRequiresGraphDependency(t *testing.T) {
	_, err := NewADSBAdapter(ADSBAdapterConfig{}, ADSBAdapterDeps{})
	if err == nil {
		t.Fatal("expected missing graph dependency error")
	}
	if !strings.Contains(err.Error(), "NATS requester or injected plan writer") {
		t.Fatalf("error = %v", err)
	}
}

type recordingADSBPlanWriter struct {
	plans []adsbprojector.Plan
}

func (w *recordingADSBPlanWriter) Apply(_ context.Context, plan adsbprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}
