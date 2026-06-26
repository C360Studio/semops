package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestNewMAVLinkAdapterWiresNATSGraphWriter(t *testing.T) {
	now := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	retry := natsclient.RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	client := &recordingRetryRequester{}
	adapter, err := NewMAVLinkAdapter(MAVLinkAdapterConfig{
		Source:        "udp:14550",
		Org:           "c360",
		Platform:      "edge",
		OwnerTokens:   testOwnerTokens("stack-test"),
		TraceID:       "mavlink-stack-test",
		RawMaxRecords: 2,
		RawMaxBytes:   4096,
		WriteTimeout:  25 * time.Millisecond,
		Retry:         retry,
		Clock:         func() time.Time { return now },
	}, MAVLinkAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	generator := mavcodec.NewGenerator(42, 7)
	heartbeat, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	position, err := generator.GenerateGlobalPosition(mavcodec.PositionMessage{
		Lat: 389000001,
		Lon: -770000002,
		Vx:  321,
		Vy:  -12,
		Vz:  7,
	})
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}

	if _, err := adapter.IngestFrame(context.Background(), heartbeat); err != nil {
		t.Fatalf("ingest heartbeat: %v", err)
	}
	if _, err := adapter.IngestFrame(context.Background(), position); err != nil {
		t.Fatalf("ingest position: %v", err)
	}

	if len(client.calls) != 3 {
		t.Fatalf("requests = %d, want asset create, track create, track update", len(client.calls))
	}
	wantSubjects := []string{
		mavprojector.SubjectEntityCreateWithTriples,
		mavprojector.SubjectEntityCreateWithTriples,
		mavprojector.SubjectEntityUpdateWithTriples,
	}
	for i, call := range client.calls {
		if call.subject != wantSubjects[i] {
			t.Fatalf("request %d subject = %q, want %q", i, call.subject, wantSubjects[i])
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
	if trackCreate.OwnerToken != "semops.feed.mavlink#stack-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	if trackCreate.TraceID != "mavlink-stack-test" {
		t.Fatalf("track trace id = %q", trackCreate.TraceID)
	}
	requireTriple(t, trackCreate.Triples, cop.TrackSource, "c360.edge.cop.mavlink.asset.system-42")

	var trackUpdate graph.UpdateEntityWithTriplesRequest
	decodePayload(t, client.calls[2].payload, &trackUpdate)
	if trackUpdate.OwnerToken != "semops.feed.mavlink#stack-test" {
		t.Fatalf("update owner token = %q", trackUpdate.OwnerToken)
	}
	requireTriple(t, trackUpdate.AddTriples, cop.TrackPosition, "POINT(-77.0000002 38.9000001)")
	if hasPredicate(trackUpdate.AddTriples, cop.TrackSource) {
		t.Fatal("track updates must not repeat strict source foreign edge")
	}

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

func TestNewMAVLinkAdapterDefaultsToSemStreamsRetryConfig(t *testing.T) {
	client := &recordingRetryRequester{}
	adapter, err := NewMAVLinkAdapter(MAVLinkAdapterConfig{
		Source: "serial:/dev/ttyS0",
	}, MAVLinkAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	if _, err := adapter.IngestFrame(context.Background(), frame); err != nil {
		t.Fatalf("ingest frame: %v", err)
	}
	if len(client.calls) == 0 {
		t.Fatal("expected graph requests")
	}
	if client.calls[0].retry != natsclient.DefaultRetryConfig() {
		t.Fatalf("retry = %+v, want default %+v", client.calls[0].retry, natsclient.DefaultRetryConfig())
	}
}

func TestNewMAVLinkAdapterSupportsWriterInjection(t *testing.T) {
	writer := &recordingPlanWriter{}
	adapter, err := NewMAVLinkAdapter(MAVLinkAdapterConfig{
		Source: "fixture",
	}, MAVLinkAdapterDeps{Writer: writer})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	generator := mavcodec.NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	result, err := adapter.IngestFrame(context.Background(), frame)
	if err != nil {
		t.Fatalf("ingest frame: %v", err)
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

func TestNewMAVLinkAdapterRequiresGraphDependency(t *testing.T) {
	_, err := NewMAVLinkAdapter(MAVLinkAdapterConfig{}, MAVLinkAdapterDeps{})
	if err == nil {
		t.Fatal("expected missing graph dependency error")
	}
	if !strings.Contains(err.Error(), "NATS requester or injected plan writer") {
		t.Fatalf("error = %v", err)
	}
}

type recordedRetryRequest struct {
	subject string
	payload []byte
	timeout time.Duration
	retry   natsclient.RetryConfig
}

type recordingRetryRequester struct {
	calls []recordedRetryRequest
}

func (r *recordingRetryRequester) RequestWithRetry(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
	retry natsclient.RetryConfig,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.calls = append(r.calls, recordedRetryRequest{
		subject: subject,
		payload: append([]byte(nil), data...),
		timeout: timeout,
		retry:   retry,
	})

	switch subject {
	case mavprojector.SubjectEntityCreateWithTriples:
		return mustJSON(successCreateResponse()), nil
	case mavprojector.SubjectEntityUpdateWithTriples:
		return mustJSON(successUpdateResponse()), nil
	default:
		return nil, fmt.Errorf("unexpected subject %q", subject)
	}
}

func (r *recordingRetryRequester) RequestWithRetryClassified(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
	retry natsclient.RetryConfig,
) ([]byte, error) {
	return r.RequestWithRetry(ctx, subject, data, timeout, retry)
}

type recordingPlanWriter struct {
	plans []mavprojector.Plan
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return nil
}

func successCreateResponse() graph.CreateEntityWithTriplesResponse {
	return graph.CreateEntityWithTriplesResponse{
		MutationResponse: graph.MutationResponse{},
	}
}

func successUpdateResponse() graph.UpdateEntityWithTriplesResponse {
	return graph.UpdateEntityWithTriplesResponse{
		MutationResponse: graph.MutationResponse{},
	}
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func decodePayload(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
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
		cop.OwnerTAK:     ownership.ExpectedOwnerToken(cop.OwnerTAK, incarnation),
		cop.OwnerCAP:     ownership.ExpectedOwnerToken(cop.OwnerCAP, incarnation),
		cop.OwnerADSB:    ownership.ExpectedOwnerToken(cop.OwnerADSB, incarnation),
		cop.OwnerKLV:     ownership.ExpectedOwnerToken(cop.OwnerKLV, incarnation),
		cop.OwnerWeather: ownership.ExpectedOwnerToken(cop.OwnerWeather, incarnation),
	}
}
