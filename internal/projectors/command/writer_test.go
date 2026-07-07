package command

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/pkg/errs"
)

func TestGraphWriterAppliesCommandPlanAndMarksCreatesBorn(t *testing.T) {
	intent := sampleIntent()
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("writer"),
		TraceID:     "command-writer",
	})
	plan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project intent: %v", err)
	}

	requester := &recordingRequester{}
	writer := NewGraphWriter(requester, WithProjector(projector), WithWriteTimeout(25*time.Millisecond))
	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply plan: %v", err)
	}

	if len(requester.calls) != 1 {
		t.Fatalf("requests = %d, want command intent create", len(requester.calls))
	}
	if requester.calls[0].subject != SubjectEntityCreateWithTriples {
		t.Fatalf("subject = %q", requester.calls[0].subject)
	}
	if requester.calls[0].timeout != 25*time.Millisecond {
		t.Fatalf("timeout = %s", requester.calls[0].timeout)
	}
	var create graph.CreateEntityWithTriplesRequest
	decodePayload(t, requester.calls[0].payload, &create)
	if create.Entity.ID != "c360.edge.cop.command.task.csapi-command-123" {
		t.Fatalf("entity id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.CommandIntentContract().MessageType {
		t.Fatalf("message type = %q", create.Entity.MessageType.Key())
	}
	if create.IndexingProfile != cop.CommandIntentContract().IndexingProfile ||
		create.OwnerToken != "semops.command.intent#writer" ||
		create.TraceID != "command-writer" {
		t.Fatalf("create request = %+v", create)
	}
	requireTriple(t, create.Triples, cop.TaskTarget, intent.TargetAssetID)

	nextPlan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project intent after apply: %v", err)
	}
	if len(nextPlan.Mutations) != 1 || nextPlan.Mutations[0].Kind != MutationUpdate {
		t.Fatalf("next plan = %+v, want update after create was marked born", nextPlan)
	}
}

func TestGraphWriterDoesNotMarkBornWhenApplyFails(t *testing.T) {
	intent := sampleIntent()
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("writer")})
	plan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project intent: %v", err)
	}
	writer := NewGraphWriter(&recordingRequester{failOnCall: 1}, WithProjector(projector))

	if err := writer.Apply(context.Background(), plan); err == nil {
		t.Fatal("expected apply failure")
	}
	nextPlan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project intent after failed apply: %v", err)
	}
	if len(nextPlan.Mutations) != 1 || nextPlan.Mutations[0].Kind != MutationCreate {
		t.Fatalf("next plan = %+v, want create because failed apply was not marked born", nextPlan)
	}
}

func TestGraphWriterAppliesUpdateMutation(t *testing.T) {
	intent := sampleIntent()
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("writer")})
	createPlan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project create: %v", err)
	}
	projector.MarkBornForPlan(createPlan)
	updatePlan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project update: %v", err)
	}

	requester := &recordingRequester{}
	writer := NewGraphWriter(requester)
	if err := writer.Apply(context.Background(), updatePlan); err != nil {
		t.Fatalf("apply update: %v", err)
	}
	if len(requester.calls) != 1 || requester.calls[0].subject != SubjectEntityUpdateWithTriples {
		t.Fatalf("calls = %+v, want one update", requester.calls)
	}
	var update graph.UpdateEntityWithTriplesRequest
	decodePayload(t, requester.calls[0].payload, &update)
	if update.Entity.ID != "c360.edge.cop.command.task.csapi-command-123" {
		t.Fatalf("update entity = %q", update.Entity.ID)
	}
	if update.OwnerToken != "semops.command.intent#writer" {
		t.Fatalf("update owner token = %q", update.OwnerToken)
	}
	requireTriple(t, update.AddTriples, cop.TaskStatus, StatusRequested)
	if hasPredicate(update.AddTriples, cop.TaskTarget) {
		t.Fatal("command intent updates must not repeat the strict target edge")
	}
}

func TestGraphWriterSurfacesClassifiedMutationFailure(t *testing.T) {
	requester := &recordingRequester{
		createErr: classifiedMutationError(
			graph.ErrorCodeEntityExists,
			"c360.edge.cop.command.task.csapi-command-123",
			"entity already exists",
		),
	}
	writer := NewGraphWriter(requester)

	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "c360.edge.cop.command.task.csapi-command-123"},
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
		mutationErr.EntityID != "c360.edge.cop.command.task.csapi-command-123" ||
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
			Entity: &graph.EntityState{ID: "c360.edge.cop.command.task.csapi-command-123"},
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
