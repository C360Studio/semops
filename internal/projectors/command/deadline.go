package command

import (
	"fmt"
	"strings"
	"time"
)

type DeadlineEvidence struct {
	ObservedAt time.Time
	Reason     string
	Source     string
	SourceRef  string
}

func ReconcileDeadline(current Intent, evidence DeadlineEvidence) (StatusUpdate, error) {
	if err := current.validate(); err != nil {
		return StatusUpdate{}, fmt.Errorf("current command intent: %w", err)
	}
	if evidence.ObservedAt.IsZero() {
		return StatusUpdate{}, fmt.Errorf("command deadline evidence observed_at is required")
	}
	observedAt := evidence.ObservedAt.UTC()
	if !observedAt.After(current.ExpiresAt.UTC()) {
		return StatusUpdate{}, fmt.Errorf("command deadline has not expired")
	}

	status := deadlineStatus(current.Status)
	if err := ValidateStatusTransition(current.Status, status); err != nil {
		return StatusUpdate{}, fmt.Errorf("reconcile command deadline: %w", err)
	}

	return StatusUpdate{
		NativeID:    strings.TrimSpace(current.NativeID),
		Status:      status,
		Description: deadlineDescription(status, evidence.Reason),
		ObservedAt:  observedAt,
		Source:      firstNonEmpty(evidence.Source, "command.deadline"),
		SourceRef:   strings.TrimSpace(evidence.SourceRef),
	}, nil
}

func deadlineStatus(status string) string {
	switch statusOrDefault(status) {
	case StatusRequested:
		return StatusExpired
	default:
		return StatusTimeout
	}
}

func deadlineDescription(status string, reason string) string {
	reason = strings.TrimSpace(reason)
	var description string
	switch status {
	case StatusExpired:
		description = "command deadline expired before acceptance"
	default:
		description = "command execution timed out"
	}
	if reason == "" {
		return description
	}
	return description + ": " + reason
}
