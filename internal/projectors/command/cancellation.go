package command

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CancellationRequest struct {
	NativeID            string
	TargetAssetID       string
	Authority           string
	Priority            int
	ExpiresAt           time.Time
	CorrelationID       string
	IdempotencyKey      string
	RequestedBy         string
	LocalOverridePolicy string
	Reason              string
	ObservedAt          time.Time
	Source              string
	SourceRef           string
}

type cancellationDesiredState struct {
	Command        string `json:"command"`
	TargetNativeID string `json:"target_native_id"`
	Reason         string `json:"reason,omitempty"`
}

func BuildCancellationIntent(current Intent, request CancellationRequest) (Intent, error) {
	if err := current.validate(); err != nil {
		return Intent{}, fmt.Errorf("current command intent: %w", err)
	}
	if request.NativeID != "" && strings.TrimSpace(request.NativeID) != strings.TrimSpace(current.NativeID) {
		return Intent{}, fmt.Errorf("cancellation native_id %q does not match current command %q", request.NativeID, current.NativeID)
	}
	if request.TargetAssetID != "" && strings.TrimSpace(request.TargetAssetID) != strings.TrimSpace(current.TargetAssetID) {
		return Intent{}, fmt.Errorf("cancellation target asset %q does not match current command target %q", request.TargetAssetID, current.TargetAssetID)
	}
	if err := ValidateStatusTransition(current.Status, StatusCancelRequested); err != nil {
		return Intent{}, fmt.Errorf("cancel command intent: %w", err)
	}

	desired, err := json.Marshal(cancellationDesiredState{
		Command:        "cancel",
		TargetNativeID: strings.TrimSpace(current.NativeID),
		Reason:         strings.TrimSpace(request.Reason),
	})
	if err != nil {
		return Intent{}, fmt.Errorf("encode cancellation desired state: %w", err)
	}

	intent := current
	intent.Status = StatusCancelRequested
	intent.DesiredState = string(desired)
	intent.Description = cancellationDescription(request.Reason)
	intent.Authority = strings.TrimSpace(request.Authority)
	intent.Priority = request.Priority
	intent.ExpiresAt = request.ExpiresAt.UTC()
	intent.CorrelationID = strings.TrimSpace(request.CorrelationID)
	intent.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	intent.RequestedBy = strings.TrimSpace(request.RequestedBy)
	intent.LocalOverridePolicy = normalizeLocalOverridePolicy(request.LocalOverridePolicy)
	intent.ObservedAt = request.ObservedAt.UTC()
	intent.Source = firstNonEmpty(request.Source, "command.cancel")
	intent.SourceRef = strings.TrimSpace(request.SourceRef)
	if err := intent.validate(); err != nil {
		return Intent{}, fmt.Errorf("cancellation request: %w", err)
	}
	return intent, nil
}

func cancellationDescription(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "cancel requested"
	}
	return "cancel requested: " + reason
}
