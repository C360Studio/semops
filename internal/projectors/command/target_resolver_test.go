package command

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGraphTargetResolverFindsBornTarget(t *testing.T) {
	requester := &recordingGraphTargetRequester{
		responses: map[string][]byte{
			SubjectGraphQueryEntity: []byte(`{"id":"c360.edge.cop.mavlink.asset.system-42"}`),
		},
	}
	resolver := NewGraphTargetResolver(requester, WithTargetQueryTimeout(25*time.Millisecond))

	exists, err := resolver.TargetExists(context.Background(), " c360.edge.cop.mavlink.asset.system-42 ")
	if err != nil {
		t.Fatalf("target exists: %v", err)
	}
	if !exists {
		t.Fatal("target exists = false, want true")
	}
	if len(requester.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requester.requests))
	}
	request := requester.requests[0]
	if request.subject != SubjectGraphQueryEntity {
		t.Fatalf("subject = %q, want %q", request.subject, SubjectGraphQueryEntity)
	}
	if request.timeout != 25*time.Millisecond {
		t.Fatalf("timeout = %s, want 25ms", request.timeout)
	}
	var body map[string]string
	if err := json.Unmarshal(request.data, &body); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if body["id"] != "c360.edge.cop.mavlink.asset.system-42" {
		t.Fatalf("request id = %q", body["id"])
	}
}

func TestGraphTargetResolverTreatsMissingTargetAsNotBorn(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		err      error
	}{
		{name: "empty response", response: []byte(`{}`)},
		{name: "error envelope", response: []byte(`{"error":"entity not found"}`)},
		{name: "request not found", err: errors.New("graph query: not found")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requester := &recordingGraphTargetRequester{
				responses: map[string][]byte{SubjectGraphQueryEntity: tt.response},
				err:       tt.err,
			}
			resolver := NewGraphTargetResolver(requester)

			exists, err := resolver.TargetExists(context.Background(), "target-1")
			if err != nil {
				t.Fatalf("target exists: %v", err)
			}
			if exists {
				t.Fatal("target exists = true, want false")
			}
		})
	}
}

func TestGraphTargetResolverReportsQueryFailures(t *testing.T) {
	resolver := NewGraphTargetResolver(&recordingGraphTargetRequester{
		err: errors.New("nats unavailable"),
	})

	_, err := resolver.TargetExists(context.Background(), "target-1")
	if err == nil || !strings.Contains(err.Error(), "query command target target-1") {
		t.Fatalf("err = %v, want query failure", err)
	}
}

func TestGraphTargetResolverRequiresRequester(t *testing.T) {
	resolver := NewGraphTargetResolver(nil)

	_, err := resolver.TargetExists(context.Background(), "target-1")
	if err == nil || !strings.Contains(err.Error(), "has no requester") {
		t.Fatalf("err = %v, want requester error", err)
	}
}

type graphTargetRequest struct {
	subject string
	data    []byte
	timeout time.Duration
}

type recordingGraphTargetRequester struct {
	responses map[string][]byte
	err       error
	requests  []graphTargetRequest
}

func (r *recordingGraphTargetRequester) Request(
	_ context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	r.requests = append(r.requests, graphTargetRequest{
		subject: subject,
		data:    append([]byte(nil), data...),
		timeout: timeout,
	})
	if r.err != nil {
		return nil, r.err
	}
	if response, ok := r.responses[subject]; ok {
		return append([]byte(nil), response...), nil
	}
	return []byte(`{}`), nil
}
