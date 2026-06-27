package command

import (
	"fmt"
	"strings"
)

const (
	LocalOverrideAcknowledged = "acknowledged"
	LocalOverrideNotRequired  = "not_required"
)

func validateLocalOverridePolicy(policy string) error {
	switch normalizeLocalOverridePolicy(policy) {
	case LocalOverrideAcknowledged, LocalOverrideNotRequired:
		return nil
	default:
		return fmt.Errorf("command intent local_override_policy %q is not supported", strings.TrimSpace(policy))
	}
}

func localOverrideCleared(policy string) bool {
	switch normalizeLocalOverridePolicy(policy) {
	case LocalOverrideAcknowledged, LocalOverrideNotRequired:
		return true
	default:
		return false
	}
}

func localOverrideRank(policy string) int {
	switch normalizeLocalOverridePolicy(policy) {
	case LocalOverrideAcknowledged:
		return 2
	case LocalOverrideNotRequired:
		return 1
	default:
		return 0
	}
}

func normalizeLocalOverridePolicy(policy string) string {
	return strings.ToLower(strings.TrimSpace(policy))
}
