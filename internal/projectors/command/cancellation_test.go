package command

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
)

func TestBuildCancellationIntentProjectsCancelRequestedUpdate(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted

	cancelIntent, err := BuildCancellationIntent(current, CancellationRequest{
		NativeID:       current.NativeID,
		TargetAssetID:  current.TargetAssetID,
		Authority:      "local.operator",
		Priority:       95,
		ExpiresAt:      current.ObservedAt.Add(30 * time.Second),
		CorrelationID:  "cancel:req-123",
		IdempotencyKey: "cancel-idem-123",
		RequestedBy:    "operator:lead",
		Reason:         "airspace conflict",
		ObservedAt:     current.ObservedAt.Add(5 * time.Second),
		Source:         "local-ui",
		SourceRef:      "ui://commands/cancel/123",
	})
	if err != nil {
		t.Fatalf("build cancellation: %v", err)
	}

	if cancelIntent.NativeID != current.NativeID || cancelIntent.TargetAssetID != current.TargetAssetID {
		t.Fatalf("cancel intent target identity = %q/%q, want current command", cancelIntent.NativeID, cancelIntent.TargetAssetID)
	}
	if cancelIntent.Kind != current.Kind {
		t.Fatalf("cancel intent kind = %q, want original kind %q", cancelIntent.Kind, current.Kind)
	}
	if cancelIntent.Status != StatusCancelRequested {
		t.Fatalf("cancel intent status = %q, want %q", cancelIntent.Status, StatusCancelRequested)
	}
	if cancelIntent.DesiredState != `{"command":"cancel","target_native_id":"csapi-command-123","reason":"airspace conflict"}` {
		t.Fatalf("cancel desired state = %s", cancelIntent.DesiredState)
	}

	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	birthPlan, err := projector.ProjectIntent(current)
	if err != nil {
		t.Fatalf("project birth: %v", err)
	}
	projector.MarkBornForPlan(birthPlan)
	updatePlan, err := projector.ProjectIntent(cancelIntent)
	if err != nil {
		t.Fatalf("project cancel request: %v", err)
	}
	update := requireUpdate(t, updatePlan.Mutations[0])
	if hasPredicate(update.AddTriples, cop.TaskTarget) {
		t.Fatal("cancellation update must not repeat strict target edge")
	}
	requireTriple(t, update.AddTriples, cop.TaskStatus, StatusCancelRequested)
	requireTriple(t, update.AddTriples, cop.TaskDesired, cancelIntent.DesiredState)
	requireTriple(t, update.AddTriples, cop.TaskAuthority, "local.operator")
	requireTriple(t, update.AddTriples, cop.TaskPriority, int64(95))
	requireTriple(t, update.AddTriples, cop.TaskCorrelation, "cancel:req-123")
	requireTriple(t, update.AddTriples, cop.TaskIdempotency, "cancel-idem-123")
	requireTriple(t, update.AddTriples, cop.TaskRequestedBy, "operator:lead")
	requireTriple(t, update.AddTriples, cop.ProvenanceSource, "local-ui")
	requireTriple(t, update.AddTriples, cop.ProvenanceSourceRef, "ui://commands/cancel/123")
}

func TestBuildCancellationIntentRejectsTerminalCurrentCommand(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusSucceeded

	_, err := BuildCancellationIntent(current, validCancellationRequest(current))
	if err == nil {
		t.Fatal("expected terminal cancellation rejection")
	}
	if !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("error = %v, want terminal transition rejection", err)
	}
}

func TestBuildCancellationIntentRejectsMismatchedIdentity(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted

	request := validCancellationRequest(current)
	request.NativeID = "other-command"
	_, err := BuildCancellationIntent(current, request)
	if err == nil || !strings.Contains(err.Error(), "native_id") {
		t.Fatalf("native mismatch error = %v", err)
	}

	request = validCancellationRequest(current)
	request.TargetAssetID = "c360.edge.cop.mavlink.asset.other-system"
	_, err = BuildCancellationIntent(current, request)
	if err == nil || !strings.Contains(err.Error(), "target asset") {
		t.Fatalf("target mismatch error = %v", err)
	}
}

func TestBuildCancellationIntentValidatesRequestFields(t *testing.T) {
	current := sampleIntent()
	current.Status = StatusAccepted
	request := validCancellationRequest(current)
	request.IdempotencyKey = ""

	_, err := BuildCancellationIntent(current, request)
	if err == nil {
		t.Fatal("expected request validation error")
	}
	if !strings.Contains(err.Error(), "idempotency_key") {
		t.Fatalf("error = %v, want idempotency validation", err)
	}
}

func validCancellationRequest(current Intent) CancellationRequest {
	return CancellationRequest{
		NativeID:       current.NativeID,
		TargetAssetID:  current.TargetAssetID,
		Authority:      "local.operator",
		Priority:       90,
		ExpiresAt:      current.ObservedAt.Add(time.Minute),
		CorrelationID:  "cancel:req",
		IdempotencyKey: "cancel:idem",
		RequestedBy:    "operator:test",
		Reason:         "test",
		ObservedAt:     current.ObservedAt.Add(time.Second),
		Source:         "local-ui",
		SourceRef:      "ui://cancel/test",
	}
}
