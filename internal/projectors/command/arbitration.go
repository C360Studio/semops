package command

import (
	"fmt"
	"strings"
)

type ArbitrationOutcome string

const (
	ArbitrationAccepted   ArbitrationOutcome = "accepted"
	ArbitrationSuperseded ArbitrationOutcome = "superseded"
	ArbitrationIgnored    ArbitrationOutcome = "ignored"
)

type ArbitrationConfig struct {
	AuthorityRanks         map[string]int
	LocalAuthorities       []string
	LocalAuthorityPrefixes []string
}

type Arbitrator struct {
	cfg ArbitrationConfig
}

type ArbitrationResult struct {
	Decisions []ArbitratedIntent
}

type ArbitratedIntent struct {
	Intent         Intent
	Outcome        ArbitrationOutcome
	Status         string
	Reason         string
	WinnerNativeID string
}

func NewArbitrator(cfg ArbitrationConfig) *Arbitrator {
	cfg.AuthorityRanks = cloneIntMapWithDefaults(cfg.AuthorityRanks, defaultAuthorityRanks())
	cfg.LocalAuthorities = normalizedListWithDefaults(cfg.LocalAuthorities, []string{
		"local.operator",
		"local.safety",
	})
	cfg.LocalAuthorityPrefixes = normalizedListWithDefaults(cfg.LocalAuthorityPrefixes, []string{
		"local.",
	})
	return &Arbitrator{cfg: cfg}
}

func (a *Arbitrator) Arbitrate(intents []Intent) (ArbitrationResult, error) {
	if a == nil {
		a = NewArbitrator(ArbitrationConfig{})
	}

	decisions := make([]ArbitratedIntent, len(intents))
	winnersByTarget := make(map[string]int)
	active := make(map[int]struct{}, len(intents))

	for index, intent := range intents {
		if err := intent.validate(); err != nil {
			return ArbitrationResult{}, fmt.Errorf("arbitrate command intent %d: %w", index, err)
		}
		decisions[index] = ArbitratedIntent{
			Intent:  intent,
			Outcome: ArbitrationIgnored,
			Status:  strings.TrimSpace(intent.Status),
			Reason:  "terminal command intent ignored by arbitration",
		}
		if isTerminalStatus(intent.Status) {
			continue
		}

		active[index] = struct{}{}
		targetID := strings.TrimSpace(intent.TargetAssetID)
		if current, ok := winnersByTarget[targetID]; !ok || a.better(intent, intents[current]) {
			winnersByTarget[targetID] = index
		}
	}

	for index := range active {
		winnerIndex := winnersByTarget[strings.TrimSpace(intents[index].TargetAssetID)]
		if index == winnerIndex {
			decisions[index] = ArbitratedIntent{
				Intent:  intents[index],
				Outcome: ArbitrationAccepted,
				Status:  StatusAccepted,
				Reason:  "selected by command authority arbitration",
			}
			continue
		}

		winner := intents[winnerIndex]
		decisions[index] = ArbitratedIntent{
			Intent:         intents[index],
			Outcome:        ArbitrationSuperseded,
			Status:         StatusSuperseded,
			Reason:         "superseded by higher-ranked command intent",
			WinnerNativeID: strings.TrimSpace(winner.NativeID),
		}
	}

	return ArbitrationResult{Decisions: decisions}, nil
}

func (r ArbitrationResult) NativeExecutionCandidates() []Intent {
	out := make([]Intent, 0, len(r.Decisions))
	for _, decision := range r.Decisions {
		if decision.Outcome != ArbitrationAccepted ||
			!nativeExecutionEligible(decision.Status) ||
			!localOverrideCleared(decision.Intent.LocalOverridePolicy) {
			continue
		}
		intent := decision.Intent
		intent.Status = decision.Status
		out = append(out, intent)
	}
	return out
}

func (r ArbitrationResult) StatusUpdates() []Intent {
	out := make([]Intent, 0, len(r.Decisions))
	for _, decision := range r.Decisions {
		if decision.Outcome == ArbitrationIgnored {
			continue
		}
		intent := decision.Intent
		intent.Status = decision.Status
		out = append(out, intent)
	}
	return out
}

func (a *Arbitrator) better(candidate Intent, incumbent Intent) bool {
	candidateLocal := a.isLocalAuthority(candidate.Authority)
	incumbentLocal := a.isLocalAuthority(incumbent.Authority)
	if candidateLocal != incumbentLocal {
		return candidateLocal
	}

	candidateOverride := localOverrideRank(candidate.LocalOverridePolicy)
	incumbentOverride := localOverrideRank(incumbent.LocalOverridePolicy)
	if candidateOverride != incumbentOverride {
		return candidateOverride > incumbentOverride
	}

	candidateRank := a.authorityRank(candidate.Authority)
	incumbentRank := a.authorityRank(incumbent.Authority)
	if candidateRank != incumbentRank {
		return candidateRank > incumbentRank
	}

	if candidate.Priority != incumbent.Priority {
		return candidate.Priority > incumbent.Priority
	}

	candidateObserved := candidate.ObservedAt.UTC()
	incumbentObserved := incumbent.ObservedAt.UTC()
	if !candidateObserved.Equal(incumbentObserved) {
		if candidateObserved.IsZero() {
			return false
		}
		if incumbentObserved.IsZero() {
			return true
		}
		return candidateObserved.Before(incumbentObserved)
	}

	return entityToken(candidate.NativeID) < entityToken(incumbent.NativeID)
}

func (a *Arbitrator) isLocalAuthority(authority string) bool {
	normalized := normalizeAuthority(authority)
	for _, local := range a.cfg.LocalAuthorities {
		if normalized == local {
			return true
		}
	}
	for _, prefix := range a.cfg.LocalAuthorityPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func (a *Arbitrator) authorityRank(authority string) int {
	if rank, ok := a.cfg.AuthorityRanks[normalizeAuthority(authority)]; ok {
		return rank
	}
	return 100
}

func normalizeAuthority(authority string) string {
	return strings.ToLower(strings.TrimSpace(authority))
}

func defaultAuthorityRanks() map[string]int {
	return map[string]int{
		"local.safety":        1000,
		"local.operator":      900,
		"incident.commander":  850,
		"federated.emergency": 800,
		"upstream.federated":  500,
		"cs-api":              500,
		"automation":          300,
		"demo.fixture":        100,
	}
}

func cloneIntMapWithDefaults(values map[string]int, defaults map[string]int) map[string]int {
	out := make(map[string]int, len(defaults)+len(values))
	for key, value := range defaults {
		out[key] = value
	}
	for key, value := range values {
		out[normalizeAuthority(key)] = value
	}
	return out
}

func normalizedListWithDefaults(values []string, defaults []string) []string {
	if len(values) == 0 {
		values = defaults
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalized := normalizeAuthority(value); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
