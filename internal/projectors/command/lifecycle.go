package command

import (
	"fmt"
	"strings"
)

const (
	StatusRequested       = "requested"
	StatusAccepted        = "accepted"
	StatusExecuting       = "executing"
	StatusCancelRequested = "cancel_requested"
	StatusCancelled       = "cancelled"
	StatusSuperseded      = "superseded"
	StatusSucceeded       = "succeeded"
	StatusFailed          = "failed"
	StatusRejected        = "rejected"
	StatusExpired         = "expired"
	StatusTimeout         = "timeout"
	StatusDuplicate       = "duplicate"
)

var knownStatuses = map[string]struct{}{
	StatusRequested:       {},
	StatusAccepted:        {},
	StatusExecuting:       {},
	StatusCancelRequested: {},
	StatusCancelled:       {},
	StatusSuperseded:      {},
	StatusSucceeded:       {},
	StatusFailed:          {},
	StatusRejected:        {},
	StatusExpired:         {},
	StatusTimeout:         {},
	StatusDuplicate:       {},
}

var terminalStatuses = map[string]struct{}{
	StatusCancelled:  {},
	StatusDuplicate:  {},
	StatusExpired:    {},
	StatusFailed:     {},
	StatusRejected:   {},
	StatusSucceeded:  {},
	StatusSuperseded: {},
	StatusTimeout:    {},
}

var allowedTransitions = map[string]map[string]struct{}{
	StatusRequested: {
		StatusRequested:       {},
		StatusAccepted:        {},
		StatusRejected:        {},
		StatusExpired:         {},
		StatusDuplicate:       {},
		StatusCancelled:       {},
		StatusSuperseded:      {},
		StatusCancelRequested: {},
	},
	StatusAccepted: {
		StatusAccepted:        {},
		StatusExecuting:       {},
		StatusSucceeded:       {},
		StatusFailed:          {},
		StatusTimeout:         {},
		StatusRejected:        {},
		StatusCancelRequested: {},
		StatusCancelled:       {},
		StatusSuperseded:      {},
	},
	StatusExecuting: {
		StatusExecuting:       {},
		StatusSucceeded:       {},
		StatusFailed:          {},
		StatusTimeout:         {},
		StatusCancelRequested: {},
		StatusCancelled:       {},
	},
	StatusCancelRequested: {
		StatusCancelRequested: {},
		StatusCancelled:       {},
		StatusFailed:          {},
		StatusTimeout:         {},
	},
}

func statusOrDefault(status string) string {
	if normalized := normalizeStatus(status); normalized != "" {
		return normalized
	}
	return StatusRequested
}

func validateStatus(status string) error {
	normalized := statusOrDefault(status)
	if _, ok := knownStatuses[normalized]; ok {
		return nil
	}
	return fmt.Errorf("command intent status %q is not supported", strings.TrimSpace(status))
}

func isTerminalStatus(status string) bool {
	_, ok := terminalStatuses[statusOrDefault(status)]
	return ok
}

func nativeExecutionEligible(status string) bool {
	return statusOrDefault(status) == StatusAccepted
}

func ValidateStatusTransition(from string, to string) error {
	from = statusOrDefault(from)
	to = statusOrDefault(to)
	if err := validateStatus(from); err != nil {
		return fmt.Errorf("from status: %w", err)
	}
	if err := validateStatus(to); err != nil {
		return fmt.Errorf("to status: %w", err)
	}
	if isTerminalStatus(from) {
		if from == to {
			return nil
		}
		return fmt.Errorf("command status transition from terminal %q to %q is not allowed", from, to)
	}
	allowed, ok := allowedTransitions[from]
	if !ok {
		return fmt.Errorf("command status transition from %q to %q is not allowed", from, to)
	}
	if _, ok := allowed[to]; ok {
		return nil
	}
	return fmt.Errorf("command status transition from %q to %q is not allowed", from, to)
}

func normalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}
