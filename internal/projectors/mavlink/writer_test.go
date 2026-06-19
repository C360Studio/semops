package mavlink

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
)

func TestGraphWriterAppliesProjectionPlanInOrder(t *testing.T) {
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("writer-test"),
		TraceID:     "scenario-writer",
	})
	heartbeat := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateHeartbeat(mavcodec.HeartbeatMessage{
			BaseMode:       mavcodec.ModeFlagSafetyArmed,
			SystemStatus:   mavcodec.StateActive,
			MavlinkVersion: mavcodec.Version2,
		})
	})
	position := parseGeneratedPacket(t, func(g *mavcodec.Generator) ([]byte, error) {
		return g.GenerateGlobalPosition(mavcodec.PositionMessage{
			Lat: 389000001,
			Lon: -770000002,
			Vx:  321,
			Vy:  -12,
			Vz:  7,
		})
	})
	plan, err := projector.ProjectPackets([]*mavcodec.Packet{heartbeat, position})
	if err != nil {
		t.Fatalf("project packets: %v", err)
	}

	requester := &recordingRequester{}
	writer := NewGraphWriter(requester, WithWriteTimeout(25*time.Millisecond))
	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply plan: %v", err)
	}

	if len(requester.calls) != 3 {
		t.Fatalf("requests = %d, want asset create, track create, track update", len(requester.calls))
	}
	if requester.calls[0].subject != SubjectEntityCreateWithTriples {
		t.Fatalf("request 0 subject = %q", requester.calls[0].subject)
	}
	if requester.calls[1].subject != SubjectEntityCreateWithTriples {
		t.Fatalf("request 1 subject = %q", requester.calls[1].subject)
	}
	if requester.calls[2].subject != SubjectEntityUpdateWithTriples {
		t.Fatalf("request 2 subject = %q", requester.calls[2].subject)
	}
	for i, call := range requester.calls {
		if call.timeout != 25*time.Millisecond {
			t.Fatalf("request %d timeout = %s", i, call.timeout)
		}
	}

	var assetCreate graph.CreateEntityWithTriplesRequest
	decodePayload(t, requester.calls[0].payload, &assetCreate)
	if assetCreate.Entity.ID != "c360.edge.cop.mavlink.asset.system-42" {
		t.Fatalf("asset id = %q", assetCreate.Entity.ID)
	}
	if assetCreate.Entity.MessageType.Key() != cop.SourceAssetContract().MessageType {
		t.Fatalf("asset message type = %q", assetCreate.Entity.MessageType.Key())
	}
	if assetCreate.OwnerToken != "semops.feed.asset#writer-test" {
		t.Fatalf("asset owner token = %q", assetCreate.OwnerToken)
	}

	var trackCreate graph.CreateEntityWithTriplesRequest
	decodePayload(t, requester.calls[1].payload, &trackCreate)
	if trackCreate.Entity.ID != "c360.edge.cop.mavlink.track.system-42" {
		t.Fatalf("track id = %q", trackCreate.Entity.ID)
	}
	if trackCreate.Entity.MessageType.Key() != cop.MAVLinkTrackContract().MessageType {
		t.Fatalf("track message type = %q", trackCreate.Entity.MessageType.Key())
	}
	if trackCreate.OwnerToken != "semops.feed.mavlink#writer-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	requireTriple(t, trackCreate.Triples, cop.TrackSource, assetCreate.Entity.ID)

	var trackUpdate graph.UpdateEntityWithTriplesRequest
	decodePayload(t, requester.calls[2].payload, &trackUpdate)
	if trackUpdate.Entity.ID != "c360.edge.cop.mavlink.track.system-42" {
		t.Fatalf("update id = %q", trackUpdate.Entity.ID)
	}
	if trackUpdate.OwnerToken != "semops.feed.mavlink#writer-test" {
		t.Fatalf("update owner token = %q", trackUpdate.OwnerToken)
	}
	requireTriple(t, trackUpdate.AddTriples, cop.TrackPosition, "POINT(-77.0000002 38.9000001)")
	if hasPredicate(trackUpdate.AddTriples, cop.TrackSource) {
		t.Fatal("track updates must not repeat the strict source foreign edge")
	}
}

func TestGraphWriterTreatsDegradedSuccessAsCommitted(t *testing.T) {
	requester := &recordingRequester{
		createResponse: mustJSON(t, graph.CreateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{
				Success:  true,
				Degraded: true,
				Error:    "post-write read-back failed",
			},
		}),
	}
	writer := NewGraphWriter(requester)

	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "c360.edge.cop.mavlink.asset.system-42"},
		},
	}}})
	if err != nil {
		t.Fatalf("degraded success should be accepted as committed: %v", err)
	}
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

func TestGraphWriterSurfacesMutationFailureResponse(t *testing.T) {
	requester := &recordingRequester{
		updateResponse: mustJSON(t, graph.UpdateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{
				Success:   false,
				ErrorCode: graph.ErrorCodeOwnerLeaseStale,
				Error:     "owner lease stale",
			},
		}),
	}
	writer := NewGraphWriter(requester)

	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationUpdate,
		Update: graph.UpdateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "c360.edge.cop.mavlink.track.system-42"},
		},
	}}})
	if err == nil {
		t.Fatal("expected mutation response failure")
	}
	if !strings.Contains(err.Error(), graph.ErrorCodeOwnerLeaseStale) {
		t.Fatalf("error = %v, want owner lease stale code", err)
	}
	var mutationErr *MutationFailureError
	if !errors.As(err, &mutationErr) {
		t.Fatalf("error = %T, want MutationFailureError", err)
	}
	if mutationErr.Kind != MutationUpdate ||
		mutationErr.EntityID != "c360.edge.cop.mavlink.track.system-42" ||
		mutationErr.ErrorCode != graph.ErrorCodeOwnerLeaseStale {
		t.Fatalf("mutation error = %+v", mutationErr)
	}
}

func TestGraphWriterReportsCreateConflictAsTypedMutationFailure(t *testing.T) {
	requester := &recordingRequester{
		createResponse: mustJSON(t, graph.CreateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{
				Success:   false,
				ErrorCode: graph.ErrorCodeEntityExists,
				Error:     "entity already exists",
			},
		}),
	}
	writer := NewGraphWriter(requester)

	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "c360.edge.cop.mavlink.asset.system-42"},
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
		mutationErr.EntityID != "c360.edge.cop.mavlink.asset.system-42" ||
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
			Entity: &graph.EntityState{ID: "c360.edge.cop.mavlink.track.system-42"},
		},
	}}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if len(requester.calls) != 0 {
		t.Fatalf("requests = %d, want none after canceled context", len(requester.calls))
	}
}

func TestGraphWriterRejectsUnknownMutationKind(t *testing.T) {
	requester := &recordingRequester{}
	writer := NewGraphWriter(requester)

	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationKind("teleport"),
	}}})
	if err == nil {
		t.Fatal("expected unsupported mutation kind failure")
	}
	if len(requester.calls) != 0 {
		t.Fatalf("requests = %d, want none for unsupported mutation", len(requester.calls))
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
		if len(r.createResponse) != 0 {
			return r.createResponse, nil
		}
		return mustJSONBytes(graph.CreateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{Success: true},
		}), nil
	case SubjectEntityUpdateWithTriples:
		if len(r.updateResponse) != 0 {
			return r.updateResponse, nil
		}
		return mustJSONBytes(graph.UpdateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{Success: true},
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
