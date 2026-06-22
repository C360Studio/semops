package stack

import (
	"context"
	"strings"
	"testing"
	"time"

	klvprojector "github.com/c360studio/semops/internal/projectors/klv"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
)

func TestNewKLVPlanWriterWiresNATSGraphWriter(t *testing.T) {
	now := time.Date(2026, 6, 22, 18, 0, 0, 0, time.UTC)
	retry := natsclient.RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	client := &recordingRetryRequester{}
	writer, err := NewKLVPlanWriter(KLVAdapterConfig{
		Source:       "klv:fixture",
		Org:          "c360",
		Platform:     "edge",
		OwnerTokens:  testOwnerTokens("stack-test"),
		TraceID:      "klv-stack-test",
		WriteTimeout: 25 * time.Millisecond,
		Retry:        retry,
		Clock:        func() time.Time { return now },
	}, KLVAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new KLV plan writer: %v", err)
	}

	projector := klvprojector.NewProjector(klvprojector.Config{
		OwnerTokens: testOwnerTokens("stack-test"),
		TraceID:     "klv-stack-test",
	})
	plan, err := projector.ProjectFrame(klvprojector.Frame{
		Source:               "klv:fixture",
		MediaRef:             "file:///fixtures/klv/synthetic.ts",
		PacketRef:            "klv://packet/synthetic/0/0",
		ReceivedAt:           now,
		SensorLatitude:       float64Ptr(38.9),
		SensorLongitude:      float64Ptr(-77.04),
		FrameCenterLatitude:  float64Ptr(38.91),
		FrameCenterLongitude: float64Ptr(-77.05),
	})
	if err != nil {
		t.Fatalf("project KLV frame: %v", err)
	}
	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply KLV plan: %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("requests = %d, want one KLV footprint create", len(client.calls))
	}
	call := client.calls[0]
	if call.subject != klvprojector.SubjectEntityCreateWithTriples {
		t.Fatalf("request subject = %q", call.subject)
	}
	if call.timeout != 25*time.Millisecond {
		t.Fatalf("request timeout = %s, want 25ms", call.timeout)
	}
	if call.retry != retry {
		t.Fatalf("retry = %+v, want %+v", call.retry, retry)
	}

	var create graph.CreateEntityWithTriplesRequest
	decodePayload(t, call.payload, &create)
	if create.OwnerToken != "semops.feed.klv#stack-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "klv-stack-test" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	if create.IndexingProfile != cop.KLVSensorFootprintContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	requireTriple(t, create.Triples, cop.SensorFootprintSource, "klv")
	requireTriple(t, create.Triples, cop.SensorFootprintFrameCenter, "POINT(-77.0500000 38.9100000)")
}

func TestNewKLVPlanWriterRequiresGraphDependency(t *testing.T) {
	_, err := NewKLVPlanWriter(KLVAdapterConfig{}, KLVAdapterDeps{})
	if err == nil {
		t.Fatal("expected missing graph dependency error")
	}
	if !strings.Contains(err.Error(), "NATS requester or injected plan writer") {
		t.Fatalf("error = %v", err)
	}
}

func float64Ptr(value float64) *float64 {
	return &value
}
