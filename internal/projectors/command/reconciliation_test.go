package command

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
)

func TestReconcileNativeMAVLinkAcceptedProjectsStatusOnlyUpdate(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusRequested

	statusUpdate, err := ReconcileNativeStatus(current, NativeStatusEvidence{
		NativeID:     current.NativeID,
		Protocol:     "mavlink",
		NativeStatus: "accepted",
		Detail:       "command=400 progress=100 target=255/1",
		ObservedAt:   current.ObservedAt.Add(3 * time.Second),
		Source:       "mavlink.command_ack",
		SourceRef:    "mavlink://raw/ack/1",
	})
	if err != nil {
		t.Fatalf("reconcile native status: %v", err)
	}
	if statusUpdate.Status != StatusAccepted {
		t.Fatalf("status update = %q, want %q", statusUpdate.Status, StatusAccepted)
	}

	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test"), TraceID: "native-status"})
	plan, err := projector.ProjectStatusUpdate(statusUpdate)
	if err != nil {
		t.Fatalf("project status update: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want status update only", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.command.task.csapi-command-123" {
		t.Fatalf("status update entity id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.CommandIntentContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", update.IndexingProfile)
	}
	if update.OwnerToken != "semops.command.intent#test" {
		t.Fatalf("owner token = %q", update.OwnerToken)
	}
	if update.TraceID != "native-status" {
		t.Fatalf("trace id = %q", update.TraceID)
	}
	if hasPredicate(update.AddTriples, cop.TaskTarget) {
		t.Fatal("status update must not repeat strict target edge")
	}
	if hasPredicate(update.AddTriples, cop.TaskDesired) || hasPredicate(update.AddTriples, cop.TaskAuthority) {
		t.Fatalf("status update must not rewrite desired-state or authority: %+v", update.AddTriples)
	}
	requireTriple(t, update.AddTriples, cop.TaskNativeID, current.NativeID)
	requireTriple(t, update.AddTriples, cop.TaskStatus, StatusAccepted)
	requireTriple(t, update.AddTriples, cop.TaskDescription, "native mavlink status: accepted - command=400 progress=100 target=255/1")
	requireTriple(t, update.AddTriples, cop.ProvenanceSource, "mavlink.command_ack")
	requireTriple(t, update.AddTriples, cop.ProvenanceObservedAt, statusUpdate.ObservedAt)
	requireTriple(t, update.AddTriples, cop.ProvenanceSourceRef, "mavlink://raw/ack/1")
}

func TestReconcileNativeMAVLinkStatusMapping(t *testing.T) {
	tests := []struct {
		name         string
		current      string
		nativeStatus string
		want         string
	}{
		{name: "accepted", current: StatusRequested, nativeStatus: "accepted", want: StatusAccepted},
		{name: "in progress", current: StatusAccepted, nativeStatus: "in_progress", want: StatusExecuting},
		{name: "denied", current: StatusAccepted, nativeStatus: "denied", want: StatusRejected},
		{name: "temporarily rejected", current: StatusAccepted, nativeStatus: "temporarily_rejected", want: StatusRejected},
		{name: "unsupported", current: StatusAccepted, nativeStatus: "unsupported", want: StatusRejected},
		{name: "failed", current: StatusExecuting, nativeStatus: "failed", want: StatusFailed},
		{name: "cancelled", current: StatusCancelRequested, nativeStatus: "cancelled", want: StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := sampleIntent()
			current.Status = tt.current
			update, err := ReconcileNativeStatus(current, NativeStatusEvidence{
				Protocol:     "mavlink.command_ack",
				NativeStatus: tt.nativeStatus,
				ObservedAt:   current.ObservedAt.Add(time.Second),
			})
			if err != nil {
				t.Fatalf("reconcile %s: %v", tt.nativeStatus, err)
			}
			if update.Status != tt.want {
				t.Fatalf("mapped status = %q, want %q", update.Status, tt.want)
			}
		})
	}
}

func TestReconcileNativeStatusRejectsUnsafeTransition(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusRequested

	_, err := ReconcileNativeStatus(current, NativeStatusEvidence{
		Protocol:     "mavlink",
		NativeStatus: "in_progress",
		ObservedAt:   current.ObservedAt.Add(time.Second),
	})
	if err == nil {
		t.Fatal("expected unsafe transition error")
	}
	if !strings.Contains(err.Error(), "requested") || !strings.Contains(err.Error(), "executing") {
		t.Fatalf("error = %v, want requested-to-executing rejection", err)
	}
}

func TestReconcileNativeStatusRejectsUnknownOrMismatchedEvidence(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted

	_, err := ReconcileNativeStatus(current, NativeStatusEvidence{
		NativeID:     "other-command",
		Protocol:     "mavlink",
		NativeStatus: "accepted",
		ObservedAt:   current.ObservedAt.Add(time.Second),
	})
	if err == nil || !strings.Contains(err.Error(), "native_id") {
		t.Fatalf("native mismatch error = %v", err)
	}

	_, err = ReconcileNativeStatus(current, NativeStatusEvidence{
		Protocol:     "mavlink",
		NativeStatus: "unknown(99)",
		ObservedAt:   current.ObservedAt.Add(time.Second),
	})
	if err == nil || !strings.Contains(err.Error(), "unknown(99)") {
		t.Fatalf("unknown native status error = %v", err)
	}

	_, err = ReconcileNativeStatus(current, NativeStatusEvidence{
		Protocol:     "mavlink",
		NativeStatus: "accepted",
	})
	if err == nil || !strings.Contains(err.Error(), "observed_at") {
		t.Fatalf("missing observed_at error = %v", err)
	}
}

func TestReconcileCanonicalNativeStatusForFutureDrivers(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted

	update, err := ReconcileNativeStatus(current, NativeStatusEvidence{
		Protocol:     "tak",
		NativeStatus: StatusExecuting,
		ObservedAt:   current.ObservedAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("reconcile canonical status: %v", err)
	}
	if update.Status != StatusExecuting {
		t.Fatalf("status = %q, want %q", update.Status, StatusExecuting)
	}
}
