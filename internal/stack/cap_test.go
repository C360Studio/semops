package stack

import (
	"context"
	"strings"
	"testing"
	"time"

	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
)

func TestNewCAPPlanWriterWiresNATSGraphWriter(t *testing.T) {
	now := time.Date(2026, 6, 20, 18, 0, 0, 0, time.UTC)
	retry := natsclient.RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	client := &recordingRetryRequester{}
	writer, err := NewCAPPlanWriter(CAPAdapterConfig{
		Source:       "cap:http",
		Org:          "c360",
		Platform:     "edge",
		OwnerTokens:  testOwnerTokens("stack-test"),
		TraceID:      "cap-stack-test",
		WriteTimeout: 25 * time.Millisecond,
		Retry:        retry,
		Clock:        func() time.Time { return now },
	}, CAPAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new CAP plan writer: %v", err)
	}

	record := mustCAPRecord(t, now)
	alert, err := record.Alert()
	if err != nil {
		t.Fatalf("parse CAP record: %v", err)
	}
	projector := capprojector.NewProjector(capprojector.Config{
		OwnerTokens: testOwnerTokens("stack-test"),
		TraceID:     "cap-stack-test",
	})
	plan, err := projector.ProjectAlert(alert, record.Ref)
	if err != nil {
		t.Fatalf("project CAP alert: %v", err)
	}
	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply CAP plan: %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("requests = %d, want one CAP hazard create", len(client.calls))
	}
	call := client.calls[0]
	if call.subject != capprojector.SubjectEntityCreateWithTriples {
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
	if create.OwnerToken != "semops.feed.cap#stack-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "cap-stack-test" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	requireTriple(t, create.Triples, cop.HazardSource, "cap")
}

func TestNewCAPPlanWriterRequiresGraphDependency(t *testing.T) {
	_, err := NewCAPPlanWriter(CAPAdapterConfig{}, CAPAdapterDeps{})
	if err == nil {
		t.Fatal("expected missing graph dependency error")
	}
	if !strings.Contains(err.Error(), "NATS requester or injected plan writer") {
		t.Fatalf("error = %v", err)
	}
}

func mustCAPRecord(t *testing.T, start time.Time) capcodec.RawAlertRecord {
	t.Helper()
	records, err := capcodec.LifecycleFixtureRecords(start)
	if err != nil {
		t.Fatalf("load CAP fixture records: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("CAP fixture records empty")
	}
	return records[0]
}
