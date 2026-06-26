package cot

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/pkg/errs"
)

func TestGraphWriterAppliesProjectionPlanInOrder(t *testing.T) {
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("writer-test"),
		TraceID:     "tak-writer",
	})
	event := cotcodec.SeedEvents(seedTime())[0]
	plan, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000001")
	if err != nil {
		t.Fatalf("project event: %v", err)
	}

	requester := &recordingRequester{}
	writer := NewGraphWriter(requester, WithWriteTimeout(25*time.Millisecond))
	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply plan: %v", err)
	}

	if len(requester.calls) != 2 {
		t.Fatalf("requests = %d, want asset create + track create", len(requester.calls))
	}
	if requester.calls[0].subject != SubjectEntityCreateWithTriples ||
		requester.calls[1].subject != SubjectEntityCreateWithTriples {
		t.Fatalf("subjects = %#v", requester.calls)
	}
	for i, call := range requester.calls {
		if call.timeout != 25*time.Millisecond {
			t.Fatalf("request %d timeout = %s", i, call.timeout)
		}
	}

	var assetCreate graph.CreateEntityWithTriplesRequest
	decodePayload(t, requester.calls[0].payload, &assetCreate)
	if assetCreate.Entity.ID != "c360.edge.cop.tak.asset.android-alpha" {
		t.Fatalf("asset id = %q", assetCreate.Entity.ID)
	}
	if assetCreate.OwnerToken != "semops.feed.asset#writer-test" {
		t.Fatalf("asset owner token = %q", assetCreate.OwnerToken)
	}

	var trackCreate graph.CreateEntityWithTriplesRequest
	decodePayload(t, requester.calls[1].payload, &trackCreate)
	if trackCreate.Entity.ID != "c360.edge.cop.tak.track.android-alpha" {
		t.Fatalf("track id = %q", trackCreate.Entity.ID)
	}
	if trackCreate.OwnerToken != "semops.feed.tak#writer-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	requireTriple(t, trackCreate.Triples, cop.TrackSource, assetCreate.Entity.ID)
}

func TestGraphWriterStopsOnFirstTransportFailure(t *testing.T) {
	requester := &recordingRequester{failOnCall: 2}
	writer := NewGraphWriter(requester)
	plan := Plan{Mutations: []Mutation{
		{Kind: MutationCreate, Create: graph.CreateEntityWithTriplesRequest{Entity: &graph.EntityState{ID: "asset"}}},
		{Kind: MutationCreate, Create: graph.CreateEntityWithTriplesRequest{Entity: &graph.EntityState{ID: "track"}}},
		{Kind: MutationUpdate, Update: graph.UpdateEntityWithTriplesRequest{Entity: &graph.EntityState{ID: "track"}}},
	}}

	err := writer.Apply(context.Background(), plan)
	if err == nil {
		t.Fatal("expected apply failure")
	}
	if !strings.Contains(err.Error(), "network split") {
		t.Fatalf("error = %v, want transport failure", err)
	}
	if len(requester.calls) != 2 {
		t.Fatalf("requests = %d, want stop after failing request", len(requester.calls))
	}
}

func TestGraphWriterReportsCreateConflictAsTypedMutationFailure(t *testing.T) {
	requester := &recordingRequester{
		createErr: classifiedMutationError(
			graph.ErrorCodeEntityExists,
			"c360.edge.cop.tak.asset.android-alpha",
			"entity already exists",
		),
	}
	writer := NewGraphWriter(requester)

	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "c360.edge.cop.tak.asset.android-alpha"},
		},
	}}})
	if err == nil {
		t.Fatal("expected create conflict")
	}
	var mutationErr *MutationFailureError
	if !errors.As(err, &mutationErr) {
		t.Fatalf("error = %T, want MutationFailureError", err)
	}
	if mutationErr.Kind != MutationCreate ||
		mutationErr.EntityID != "c360.edge.cop.tak.asset.android-alpha" ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists {
		t.Fatalf("mutation error = %+v", mutationErr)
	}
}

func TestGraphWriterHonorsCanceledContextBeforeSending(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	requester := &recordingRequester{}
	writer := NewGraphWriter(requester)

	err := writer.Apply(ctx, Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "c360.edge.cop.tak.track.android-alpha"},
		},
	}}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if len(requester.calls) != 0 {
		t.Fatalf("requests = %d, want none after canceled context", len(requester.calls))
	}
}

type recordedRequest struct {
	subject string
	payload []byte
	timeout time.Duration
}

type recordingRequester struct {
	calls          []recordedRequest
	failOnCall     int
	createResponse []byte
	updateResponse []byte
	createErr      error
	updateErr      error
}

func (r *recordingRequester) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	payload := append([]byte(nil), data...)
	r.calls = append(r.calls, recordedRequest{subject: subject, payload: payload, timeout: timeout})
	if r.failOnCall > 0 && len(r.calls) == r.failOnCall {
		return nil, errors.New("network split")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch subject {
	case SubjectEntityCreateWithTriples:
		if r.createErr != nil {
			return nil, r.createErr
		}
		if len(r.createResponse) != 0 {
			return r.createResponse, nil
		}
		return mustJSONBytes(graph.CreateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{},
		}), nil
	case SubjectEntityUpdateWithTriples:
		if r.updateErr != nil {
			return nil, r.updateErr
		}
		if len(r.updateResponse) != 0 {
			return r.updateResponse, nil
		}
		return mustJSONBytes(graph.UpdateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{},
		}), nil
	default:
		return nil, errors.New("unexpected subject")
	}
}

func decodePayload(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test json: %v", err)
	}
	return data
}

func mustJSONBytes(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func classifiedMutationError(code string, entityID string, message string) error {
	return errs.ClassifiedCodeDetail(
		errs.ErrorInvalid,
		code,
		map[string]any{"entity": entityID},
		errors.New(message),
	)
}
