package mavlink

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/c360studio/semstreams/graph"
)

const (
	SubjectEntityCreateWithTriples = "graph.mutation.entity.create_with_triples"
	SubjectEntityUpdateWithTriples = "graph.mutation.entity.update_with_triples"
	DefaultWriteTimeout            = 5 * time.Second
)

type GraphRequester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type GraphWriter struct {
	requester     GraphRequester
	timeout       time.Duration
	createSubject string
	updateSubject string
}

type GraphWriterOption func(*GraphWriter)

type MutationFailureError struct {
	Operation string
	Kind      MutationKind
	EntityID  string
	ErrorCode string
	Message   string
}

func (e *MutationFailureError) Error() string {
	if e == nil {
		return ""
	}
	if e.ErrorCode != "" && e.Message != "" {
		return fmt.Sprintf("%s failed (%s): %s", e.Operation, e.ErrorCode, e.Message)
	}
	if e.ErrorCode != "" {
		return fmt.Sprintf("%s failed (%s)", e.Operation, e.ErrorCode)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s failed: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("%s failed", e.Operation)
}

func NewGraphWriter(requester GraphRequester, opts ...GraphWriterOption) *GraphWriter {
	writer := &GraphWriter{
		requester:     requester,
		timeout:       DefaultWriteTimeout,
		createSubject: SubjectEntityCreateWithTriples,
		updateSubject: SubjectEntityUpdateWithTriples,
	}
	for _, opt := range opts {
		opt(writer)
	}
	return writer
}

func WithWriteTimeout(timeout time.Duration) GraphWriterOption {
	return func(writer *GraphWriter) {
		if timeout > 0 {
			writer.timeout = timeout
		}
	}
}

func WithMutationSubjects(createSubject, updateSubject string) GraphWriterOption {
	return func(writer *GraphWriter) {
		if createSubject != "" {
			writer.createSubject = createSubject
		}
		if updateSubject != "" {
			writer.updateSubject = updateSubject
		}
	}
}

func (w *GraphWriter) Apply(ctx context.Context, plan Plan) error {
	if w == nil || w.requester == nil {
		return fmt.Errorf("mavlink graph writer has no requester")
	}
	for i, mutation := range plan.Mutations {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("apply mavlink mutation %d: %w", i, err)
		}
		if err := w.applyMutation(ctx, mutation); err != nil {
			return fmt.Errorf("apply mavlink mutation %d: %w", i, err)
		}
	}
	return nil
}

func (w *GraphWriter) applyMutation(ctx context.Context, mutation Mutation) error {
	switch mutation.Kind {
	case MutationCreate:
		return w.createWithTriples(ctx, mutation.Create)
	case MutationUpdate:
		return w.updateWithTriples(ctx, mutation.Update)
	default:
		return fmt.Errorf("unsupported mutation kind %q", mutation.Kind)
	}
}

func (w *GraphWriter) createWithTriples(ctx context.Context, req graph.CreateEntityWithTriplesRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal create_with_triples request: %w", err)
	}
	respData, err := w.requester.Request(ctx, w.createSubject, data, w.timeout)
	if err != nil {
		return fmt.Errorf("request create_with_triples: %w", err)
	}
	var resp graph.CreateEntityWithTriplesResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return fmt.Errorf("decode create_with_triples response: %w", err)
	}
	return mutationResponseError(
		"create_with_triples",
		MutationCreate,
		requestEntityID(req.Entity),
		resp.MutationResponse,
	)
}

func (w *GraphWriter) updateWithTriples(ctx context.Context, req graph.UpdateEntityWithTriplesRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal update_with_triples request: %w", err)
	}
	respData, err := w.requester.Request(ctx, w.updateSubject, data, w.timeout)
	if err != nil {
		return fmt.Errorf("request update_with_triples: %w", err)
	}
	var resp graph.UpdateEntityWithTriplesResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return fmt.Errorf("decode update_with_triples response: %w", err)
	}
	return mutationResponseError(
		"update_with_triples",
		MutationUpdate,
		requestEntityID(req.Entity),
		resp.MutationResponse,
	)
}

func mutationResponseError(
	operation string,
	kind MutationKind,
	entityID string,
	resp graph.MutationResponse,
) error {
	if resp.Success {
		return nil
	}
	return &MutationFailureError{
		Operation: operation,
		Kind:      kind,
		EntityID:  entityID,
		ErrorCode: resp.ErrorCode,
		Message:   resp.Error,
	}
}

func requestEntityID(entity *graph.EntityState) string {
	if entity == nil {
		return ""
	}
	return entity.ID
}
