package command

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
)

func TestReconcileDeadlineExpiresUnacceptedCommand(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusRequested

	statusUpdate, err := ReconcileDeadline(current, DeadlineEvidence{
		ObservedAt: current.ExpiresAt.Add(time.Second),
		Reason:     "no acceptance before ttl",
		Source:     "command.deadline.test",
		SourceRef:  "timer://command/csapi-command-123",
	})
	if err != nil {
		t.Fatalf("reconcile deadline: %v", err)
	}
	if statusUpdate.Status != StatusExpired {
		t.Fatalf("status = %q, want %q", statusUpdate.Status, StatusExpired)
	}

	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	plan, err := projector.ProjectStatusUpdate(statusUpdate)
	if err != nil {
		t.Fatalf("project deadline status: %v", err)
	}
	update := requireUpdate(t, plan.Mutations[0])
	if hasPredicate(update.AddTriples, cop.TaskTarget) {
		t.Fatal("deadline update must not repeat strict target edge")
	}
	if hasPredicate(update.AddTriples, cop.TaskDesired) || hasPredicate(update.AddTriples, cop.TaskAuthority) {
		t.Fatalf("deadline update must not rewrite desired-state or authority: %+v", update.AddTriples)
	}
	requireTriple(t, update.AddTriples, cop.TaskStatus, StatusExpired)
	requireTriple(t, update.AddTriples, cop.TaskDescription, "command deadline expired before acceptance: no acceptance before ttl")
	requireTriple(t, update.AddTriples, cop.ProvenanceSource, "command.deadline.test")
	requireTriple(t, update.AddTriples, cop.ProvenanceSourceRef, "timer://command/csapi-command-123")
}

func TestReconcileDeadlineTimesOutActiveCommands(t *testing.T) {
	for _, status := range []string{StatusAccepted, StatusExecuting, StatusCancelRequested} {
		t.Run(status, func(t *testing.T) {
			current := sampleIntent()
			current.Status = status

			update, err := ReconcileDeadline(current, DeadlineEvidence{
				ObservedAt: current.ExpiresAt.Add(time.Second),
				Reason:     "native status window elapsed",
			})
			if err != nil {
				t.Fatalf("reconcile deadline: %v", err)
			}
			if update.Status != StatusTimeout {
				t.Fatalf("status = %q, want %q", update.Status, StatusTimeout)
			}
			if update.Description != "command execution timed out: native status window elapsed" {
				t.Fatalf("description = %q", update.Description)
			}
		})
	}
}

func TestReconcileDeadlineRejectsEarlyOrTerminalCommands(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted

	_, err := ReconcileDeadline(current, DeadlineEvidence{ObservedAt: current.ExpiresAt})
	if err == nil || !strings.Contains(err.Error(), "not expired") {
		t.Fatalf("early deadline error = %v", err)
	}

	current.Status = StatusSucceeded
	_, err = ReconcileDeadline(current, DeadlineEvidence{ObservedAt: current.ExpiresAt.Add(time.Second)})
	if err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("terminal deadline error = %v", err)
	}
}

func TestReconcileDeadlineRejectsMismatchedEvidence(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted

	_, err := ReconcileDeadline(current, DeadlineEvidence{
		NativeID:   "other-command",
		ObservedAt: current.ExpiresAt.Add(time.Second),
	})
	if err == nil || !strings.Contains(err.Error(), "native_id") {
		t.Fatalf("native mismatch error = %v", err)
	}
}

func TestReconcileDeadlineRequiresObservedAt(t *testing.T) {
	_, err := ReconcileDeadline(sampleIntent(), DeadlineEvidence{})
	if err == nil {
		t.Fatal("expected missing observed_at error")
	}
	if !strings.Contains(err.Error(), "observed_at") {
		t.Fatalf("error = %v, want observed_at validation", err)
	}
}
