package command

import (
	"strings"
	"testing"
	"time"
)

func TestArbitratorPrefersLocalAuthorityOverHigherRemotePriority(t *testing.T) {
	remote := sampleIntent()
	remote.NativeID = "remote-high-priority"
	remote.Authority = "upstream.federated"
	remote.Priority = 100

	local := sampleIntent()
	local.NativeID = "local-low-priority"
	local.Authority = "local.operator"
	local.Priority = 10
	local.ObservedAt = remote.ObservedAt.Add(time.Second)

	result, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{remote, local})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}

	requireOutcome(t, result, "local-low-priority", ArbitrationAccepted, "")
	requireOutcome(t, result, "remote-high-priority", ArbitrationSuperseded, "local-low-priority")
	candidates := result.NativeExecutionCandidates()
	if len(candidates) != 1 || candidates[0].NativeID != "local-low-priority" || candidates[0].Status != StatusAccepted {
		t.Fatalf("execution candidates = %+v, want accepted local command", candidates)
	}
}

func TestArbitratorUsesAuthorityRankBeforePriorityForNonLocalCommands(t *testing.T) {
	automation := sampleIntent()
	automation.NativeID = "automation-high-priority"
	automation.Authority = "automation"
	automation.Priority = 100

	commander := sampleIntent()
	commander.NativeID = "commander-lower-priority"
	commander.Authority = "incident.commander"
	commander.Priority = 60

	result, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{automation, commander})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}

	requireOutcome(t, result, "commander-lower-priority", ArbitrationAccepted, "")
	requireOutcome(t, result, "automation-high-priority", ArbitrationSuperseded, "commander-lower-priority")
}

func TestArbitratorUsesPriorityWithinSameAuthority(t *testing.T) {
	low := sampleIntent()
	low.NativeID = "operator-low-priority"
	low.Authority = "local.operator"
	low.Priority = 25

	high := sampleIntent()
	high.NativeID = "operator-high-priority"
	high.Authority = "local.operator"
	high.Priority = 90

	result, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{low, high})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}

	requireOutcome(t, result, "operator-high-priority", ArbitrationAccepted, "")
	requireOutcome(t, result, "operator-low-priority", ArbitrationSuperseded, "operator-high-priority")
}

func TestArbitratorKeepsIndependentTargetsSeparate(t *testing.T) {
	first := sampleIntent()
	first.NativeID = "first-target"

	second := sampleIntent()
	second.NativeID = "second-target"
	second.TargetAssetID = "c360.edge.cop.mavlink.asset.system-99"

	result, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{first, second})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}

	requireOutcome(t, result, "first-target", ArbitrationAccepted, "")
	requireOutcome(t, result, "second-target", ArbitrationAccepted, "")
	if candidates := result.NativeExecutionCandidates(); len(candidates) != 2 {
		t.Fatalf("execution candidates = %d, want one per target", len(candidates))
	}
}

func TestArbitratorDeterministicTieBreaksByObservationThenNativeID(t *testing.T) {
	later := sampleIntent()
	later.NativeID = "later-command"
	later.Priority = 50
	later.ObservedAt = later.ObservedAt.Add(time.Second)

	earlier := sampleIntent()
	earlier.NativeID = "earlier-command"
	earlier.Priority = 50

	result, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{later, earlier})
	if err != nil {
		t.Fatalf("arbitrate by observed_at: %v", err)
	}
	requireOutcome(t, result, "earlier-command", ArbitrationAccepted, "")
	requireOutcome(t, result, "later-command", ArbitrationSuperseded, "earlier-command")

	zeta := sampleIntent()
	zeta.NativeID = "zeta-command"
	alpha := sampleIntent()
	alpha.NativeID = "alpha-command"
	alpha.ObservedAt = zeta.ObservedAt

	result, err = NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{zeta, alpha})
	if err != nil {
		t.Fatalf("arbitrate by native id: %v", err)
	}
	requireOutcome(t, result, "alpha-command", ArbitrationAccepted, "")
	requireOutcome(t, result, "zeta-command", ArbitrationSuperseded, "alpha-command")
}

func TestArbitratorIgnoresTerminalCommandIntents(t *testing.T) {
	terminal := sampleIntent()
	terminal.NativeID = "cancelled-command"
	terminal.Status = "cancelled"
	terminal.Priority = 100

	active := sampleIntent()
	active.NativeID = "active-command"
	active.Priority = 1

	result, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{terminal, active})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}

	requireOutcome(t, result, "cancelled-command", ArbitrationIgnored, "")
	requireOutcome(t, result, "active-command", ArbitrationAccepted, "")
	if updates := result.StatusUpdates(); len(updates) != 1 || updates[0].NativeID != "active-command" {
		t.Fatalf("status updates = %+v, want only active decision", updates)
	}
}

func TestArbitratorRejectsMalformedIntent(t *testing.T) {
	intent := sampleIntent()
	intent.Authority = ""

	_, err := NewArbitrator(ArbitrationConfig{}).Arbitrate([]Intent{intent})
	if err == nil {
		t.Fatal("expected malformed intent error")
	}
	if !strings.Contains(err.Error(), "authority") {
		t.Fatalf("error = %v, want authority validation", err)
	}
}

func requireOutcome(t *testing.T, result ArbitrationResult, nativeID string, outcome ArbitrationOutcome, winner string) {
	t.Helper()
	for _, decision := range result.Decisions {
		if decision.Intent.NativeID != nativeID {
			continue
		}
		if decision.Outcome != outcome {
			t.Fatalf("%s outcome = %q, want %q", nativeID, decision.Outcome, outcome)
		}
		if decision.WinnerNativeID != winner {
			t.Fatalf("%s winner = %q, want %q", nativeID, decision.WinnerNativeID, winner)
		}
		return
	}
	t.Fatalf("missing decision for %q in %+v", nativeID, result.Decisions)
}
