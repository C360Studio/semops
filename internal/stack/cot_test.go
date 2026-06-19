package stack

import (
	"context"
	"strings"
	"testing"
	"time"

	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
)

func TestNewCoTAdapterWiresNATSGraphWriter(t *testing.T) {
	now := time.Date(2026, 6, 19, 16, 0, 0, 0, time.UTC)
	retry := natsclient.RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	client := &recordingRetryRequester{}
	adapter, err := NewCoTAdapter(CoTAdapterConfig{
		Source:        "udp:cot",
		Org:           "c360",
		Platform:      "edge",
		OwnerTokens:   testOwnerTokens("stack-test"),
		TraceID:       "cot-stack-test",
		RawMaxRecords: 4,
		RawMaxBytes:   4096,
		WriteTimeout:  25 * time.Millisecond,
		Retry:         retry,
		Clock:         func() time.Time { return now },
	}, CoTAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	events := cotcodec.SeedEvents(now)
	operatorRaw, err := cotcodec.Marshal(events[0])
	if err != nil {
		t.Fatalf("marshal operator: %v", err)
	}
	markerRaw, err := cotcodec.Marshal(events[2])
	if err != nil {
		t.Fatalf("marshal marker: %v", err)
	}

	if _, err := adapter.IngestEvent(context.Background(), operatorRaw); err != nil {
		t.Fatalf("ingest operator: %v", err)
	}
	if _, err := adapter.IngestEvent(context.Background(), markerRaw); err != nil {
		t.Fatalf("ingest marker: %v", err)
	}

	if len(client.calls) != 3 {
		t.Fatalf("requests = %d, want asset create, track create, task create", len(client.calls))
	}
	for i, call := range client.calls {
		if call.subject != cotprojector.SubjectEntityCreateWithTriples {
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
	decodePayload(t, client.calls[1].payload, &trackCreate)
	if trackCreate.Entity.ID != "c360.edge.cop.tak.track.android-alpha" {
		t.Fatalf("track id = %q", trackCreate.Entity.ID)
	}
	if trackCreate.OwnerToken != "semops.feed.tak#stack-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	if trackCreate.TraceID != "cot-stack-test" {
		t.Fatalf("track trace id = %q", trackCreate.TraceID)
	}
	requireTriple(t, trackCreate.Triples, cop.TrackSource, "c360.edge.cop.tak.asset.android-alpha")

	var taskCreate graph.CreateEntityWithTriplesRequest
	decodePayload(t, client.calls[2].payload, &taskCreate)
	if taskCreate.Entity.ID != "c360.edge.cop.tak.task.marker-north-gate" {
		t.Fatalf("task id = %q", taskCreate.Entity.ID)
	}
	if taskCreate.IndexingProfile != cop.TAKTaskContract().IndexingProfile {
		t.Fatalf("task indexing profile = %q", taskCreate.IndexingProfile)
	}
	requireTriple(t, taskCreate.Triples, cop.TaskName, "North Gate")

	if records := adapter.RawLane().Snapshot(); len(records) != 2 {
		t.Fatalf("raw records = %d, want 2", len(records))
	}
	health := adapter.Health()
	if !health.Ready {
		t.Fatalf("health ready = false, last error %q", health.LastError)
	}
	if health.GraphMutations != 3 {
		t.Fatalf("graph mutations = %d, want 3", health.GraphMutations)
	}
}

func TestNewCoTAdapterDefaultsToSemStreamsRetryConfig(t *testing.T) {
	client := &recordingRetryRequester{}
	adapter, err := NewCoTAdapter(CoTAdapterConfig{
		Source: "udp:cot",
	}, CoTAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(time.Now())[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	if _, err := adapter.IngestEvent(context.Background(), raw); err != nil {
		t.Fatalf("ingest event: %v", err)
	}
	if len(client.calls) == 0 {
		t.Fatal("expected graph requests")
	}
	if client.calls[0].retry != natsclient.DefaultRetryConfig() {
		t.Fatalf("retry = %+v, want default %+v", client.calls[0].retry, natsclient.DefaultRetryConfig())
	}
}

func TestNewCoTAdapterSupportsWriterInjection(t *testing.T) {
	writer := &recordingCoTPlanWriter{}
	adapter, err := NewCoTAdapter(CoTAdapterConfig{
		Source: "fixture",
	}, CoTAdapterDeps{Writer: writer})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(time.Now())[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	result, err := adapter.IngestEvent(context.Background(), raw)
	if err != nil {
		t.Fatalf("ingest event: %v", err)
	}
	if result.Mutations != 2 {
		t.Fatalf("mutations = %d, want asset birth + track birth", result.Mutations)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	if got := adapter.Health().Source; got != "fixture" {
		t.Fatalf("health source = %q, want fixture", got)
	}
}

func TestNewCoTAdapterRequiresGraphDependency(t *testing.T) {
	_, err := NewCoTAdapter(CoTAdapterConfig{}, CoTAdapterDeps{})
	if err == nil {
		t.Fatal("expected missing graph dependency error")
	}
	if !strings.Contains(err.Error(), "NATS requester or injected plan writer") {
		t.Fatalf("error = %v", err)
	}
}

type recordingCoTPlanWriter struct {
	plans []cotprojector.Plan
}

func (w *recordingCoTPlanWriter) Apply(_ context.Context, plan cotprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}
