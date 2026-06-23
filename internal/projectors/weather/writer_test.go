package weather

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

func TestGraphWriterAppliesWeatherCreateAndUpdate(t *testing.T) {
	requester := &recordingRequester{
		responses: [][]byte{
			mustMutationResponse(t, true, "", ""),
			mustMutationResponse(t, true, "", ""),
		},
	}
	writer := NewGraphWriter(requester, WithWriteTimeout(25*time.Millisecond))
	plan := Plan{Mutations: []Mutation{
		{
			Kind: MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{ID: "weather-1"},
				Triples: []message.Triple{{
					Subject:   "weather-1",
					Predicate: cop.WeatherValue,
					Object:    29.4,
				}},
			},
		},
		{
			Kind: MutationUpdate,
			Update: graph.UpdateEntityWithTriplesRequest{
				Entity: &graph.EntityState{ID: "weather-1"},
				AddTriples: []message.Triple{{
					Subject:   "weather-1",
					Predicate: cop.WeatherValue,
					Object:    29.8,
				}},
			},
		},
	}}

	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply plan: %v", err)
	}
	if len(requester.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(requester.calls))
	}
	if requester.calls[0].subject != SubjectEntityCreateWithTriples ||
		requester.calls[1].subject != SubjectEntityUpdateWithTriples {
		t.Fatalf("subjects = %+v", requester.calls)
	}
	if requester.calls[0].timeout != 25*time.Millisecond ||
		requester.calls[1].timeout != 25*time.Millisecond {
		t.Fatalf("timeouts = %+v", requester.calls)
	}
	var create graph.CreateEntityWithTriplesRequest
	if err := json.Unmarshal(requester.calls[0].payload, &create); err != nil {
		t.Fatalf("decode create request: %v", err)
	}
	if create.Entity.ID != "weather-1" || len(create.Triples) != 1 {
		t.Fatalf("create request = %+v", create)
	}
	var update graph.UpdateEntityWithTriplesRequest
	if err := json.Unmarshal(requester.calls[1].payload, &update); err != nil {
		t.Fatalf("decode update request: %v", err)
	}
	if update.Entity.ID != "weather-1" || len(update.AddTriples) != 1 {
		t.Fatalf("update request = %+v", update)
	}
}

func TestGraphWriterReturnsWeatherMutationFailure(t *testing.T) {
	requester := &recordingRequester{
		responses: [][]byte{mustMutationResponse(t, false, graph.ErrorCodeEntityExists, "already exists")},
	}
	writer := NewGraphWriter(requester)
	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "weather-1"},
		},
	}}})
	if err == nil {
		t.Fatal("expected mutation failure")
	}
	var mutationErr *MutationFailureError
	if !errors.As(err, &mutationErr) {
		t.Fatalf("error = %T %v, want MutationFailureError", err, err)
	}
	if mutationErr.Kind != MutationCreate ||
		mutationErr.EntityID != "weather-1" ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists {
		t.Fatalf("mutation error = %+v", mutationErr)
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v", err)
	}
}

func TestGraphWriterHonorsWeatherCustomSubjects(t *testing.T) {
	requester := &recordingRequester{
		responses: [][]byte{mustMutationResponse(t, true, "", "")},
	}
	writer := NewGraphWriter(
		requester,
		WithMutationSubjects("weather.graph.create", "weather.graph.update"),
	)
	err := writer.Apply(context.Background(), Plan{Mutations: []Mutation{{
		Kind: MutationCreate,
		Create: graph.CreateEntityWithTriplesRequest{
			Entity: &graph.EntityState{ID: "weather-1"},
		},
	}}})
	if err != nil {
		t.Fatalf("apply plan: %v", err)
	}
	if len(requester.calls) != 1 || requester.calls[0].subject != "weather.graph.create" {
		t.Fatalf("calls = %+v", requester.calls)
	}
}

type recordedRequest struct {
	subject string
	payload []byte
	timeout time.Duration
}

type recordingRequester struct {
	calls     []recordedRequest
	responses [][]byte
}

func (r *recordingRequester) Request(
	_ context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	r.calls = append(r.calls, recordedRequest{
		subject: subject,
		payload: append([]byte(nil), data...),
		timeout: timeout,
	})
	if len(r.responses) == 0 {
		return nil, errors.New("no response queued")
	}
	resp := r.responses[0]
	r.responses = r.responses[1:]
	return resp, nil
}

func mustMutationResponse(t *testing.T, success bool, code string, msg string) []byte {
	t.Helper()
	data, err := json.Marshal(graph.CreateEntityWithTriplesResponse{
		MutationResponse: graph.MutationResponse{
			Success:   success,
			ErrorCode: code,
			Error:     msg,
		},
	})
	if err != nil {
		t.Fatalf("marshal mutation response: %v", err)
	}
	return data
}
