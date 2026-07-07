package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c360studio/semstreams/graph"
)

const (
	SubjectGraphQueryEntity   = "graph.query.entity"
	DefaultTargetQueryTimeout = 2 * time.Second
)

type GraphTargetResolver struct {
	requester GraphRequester
	subject   string
	timeout   time.Duration
}

type GraphTargetResolverOption func(*GraphTargetResolver)

func NewGraphTargetResolver(requester GraphRequester, opts ...GraphTargetResolverOption) *GraphTargetResolver {
	resolver := &GraphTargetResolver{
		requester: requester,
		subject:   SubjectGraphQueryEntity,
		timeout:   DefaultTargetQueryTimeout,
	}
	for _, opt := range opts {
		opt(resolver)
	}
	return resolver
}

func WithTargetQuerySubject(subject string) GraphTargetResolverOption {
	return func(resolver *GraphTargetResolver) {
		if strings.TrimSpace(subject) != "" {
			resolver.subject = strings.TrimSpace(subject)
		}
	}
}

func WithTargetQueryTimeout(timeout time.Duration) GraphTargetResolverOption {
	return func(resolver *GraphTargetResolver) {
		if timeout > 0 {
			resolver.timeout = timeout
		}
	}
}

func (r *GraphTargetResolver) TargetExists(ctx context.Context, targetAssetID string) (bool, error) {
	targetAssetID = strings.TrimSpace(targetAssetID)
	if targetAssetID == "" {
		return false, nil
	}
	if r == nil || r.requester == nil {
		return false, fmt.Errorf("command graph target resolver has no requester")
	}
	body, err := json.Marshal(map[string]string{"id": targetAssetID})
	if err != nil {
		return false, fmt.Errorf("marshal command target query: %w", err)
	}
	response, err := r.requester.Request(ctx, r.subject, body, r.timeout)
	if err != nil {
		if isGraphTargetNotFound(err.Error()) {
			return false, nil
		}
		return false, fmt.Errorf("query command target %s: %w", targetAssetID, err)
	}
	return graphTargetExists(targetAssetID, response)
}

func graphTargetExists(targetAssetID string, response []byte) (bool, error) {
	var envelope struct {
		ID      string            `json:"id"`
		Entity  graph.EntityState `json:"entity"`
		Data    graph.EntityState `json:"data"`
		Error   string            `json:"error"`
		Message string            `json:"message"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		return false, fmt.Errorf("decode command target %s: %w", targetAssetID, err)
	}
	if envelope.Error != "" || envelope.Message != "" {
		text := strings.TrimSpace(envelope.Error + " " + envelope.Message)
		if isGraphTargetNotFound(text) {
			return false, nil
		}
		return false, fmt.Errorf("query command target %s: %s", targetAssetID, text)
	}
	for _, id := range []string{envelope.ID, envelope.Entity.ID, envelope.Data.ID} {
		if strings.TrimSpace(id) != "" {
			return true, nil
		}
	}
	return false, nil
}

func isGraphTargetNotFound(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "not found") ||
		strings.Contains(value, "not_found") ||
		strings.Contains(value, "notfound")
}
