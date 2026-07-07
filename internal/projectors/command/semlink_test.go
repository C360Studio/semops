package command

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
)

func TestSemLinkCompanionRequestBuildsGovernedArduPilotReadbackIntent(t *testing.T) {
	observedAt := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)
	targetAssetID := "c360.edge.cop.mavlink.asset.system-1"

	intent, err := NewSemLinkArduPilotReadbackIntent(SemLinkCompanionRequest{
		MeshNodeID:       "blueboat-01",
		NativeID:         "blueboat-01-autopilot-version",
		TargetAssetID:    targetAssetID,
		VehicleSystemID:  1,
		VehicleComponent: 1,
		Action:           SemLinkActionRequestAutopilotVersion,
		CorrelationID:    "semlink:blueboat-01:req-1",
		IdempotencyKey:   "semlink-blueboat-01-autopilot-version",
		SourceRef:        "semlink://blueboat-01/ardupilot/system-1/request-autopilot-version",
		ObservedAt:       observedAt,
		TTL:              30 * time.Second,
	})
	if err != nil {
		t.Fatalf("build SemLink readback intent: %v", err)
	}

	if intent.Kind != "mavlink.request_message" ||
		intent.Authority != SemLinkCompanionAuthority ||
		intent.Source != SemLinkCompanionSource ||
		intent.RequestedBy != "semlink:blueboat-01" ||
		intent.LocalOverridePolicy != LocalOverrideNotRequired {
		t.Fatalf("intent posture = %+v", intent)
	}
	if intent.TargetAssetID != targetAssetID {
		t.Fatalf("target asset = %q", intent.TargetAssetID)
	}
	if intent.ExpiresAt != observedAt.Add(30*time.Second) {
		t.Fatalf("expires_at = %s", intent.ExpiresAt)
	}
	for _, want := range []string{
		`"command":"MAV_CMD_REQUEST_MESSAGE"`,
		`"message":"AUTOPILOT_VERSION"`,
		`"mavlink_command":512`,
		`"message_id":148`,
		`"vehicle_system_id":1`,
		`"vehicle_component_id":1`,
	} {
		if !strings.Contains(intent.DesiredState, want) {
			t.Fatalf("desired state missing %q: %s", want, intent.DesiredState)
		}
	}

	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("semlink"), TraceID: "semlink-companion"})
	guarded := NewGuardedProjector(projector, AdmissionConfig{
		Clock:          func() time.Time { return observedAt },
		TargetResolver: NewStaticTargetResolver(targetAssetID),
	})
	admission, plan, err := guarded.ProjectIntent(context.Background(), intent)
	if err != nil {
		t.Fatalf("project SemLink intent: %v", err)
	}
	if !admission.Accepted {
		t.Fatalf("admission = %+v, want accepted", admission)
	}
	create := requireCreate(t, plan.Mutations[0])
	if create.IndexingProfile != cop.CommandIntentContract().IndexingProfile ||
		create.OwnerToken != "semops.command.intent#semlink" {
		t.Fatalf("create contract = profile %q token %q", create.IndexingProfile, create.OwnerToken)
	}
	requireTriple(t, create.Triples, cop.TaskTarget, targetAssetID)
	requireTriple(t, create.Triples, cop.TaskDesired, intent.DesiredState)
	requireTriple(t, create.Triples, cop.TaskAuthority, SemLinkCompanionAuthority)
	requireTriple(t, create.Triples, cop.TaskLocalOverridePolicy, LocalOverrideNotRequired)
	requireTriple(t, create.Triples, cop.ProvenanceSource, SemLinkCompanionSource)

	statusUpdate, err := ReconcileNativeStatus(intent, NativeStatusEvidence{
		NativeID:     intent.NativeID,
		Protocol:     "mavlink.command_ack",
		NativeStatus: "accepted",
		Detail:       "request_message AUTOPILOT_VERSION accepted",
		ObservedAt:   observedAt.Add(time.Second),
		Source:       "mavlink.command_ack",
		SourceRef:    "mavlink://raw/blueboat-01/ack-1",
	})
	if err != nil {
		t.Fatalf("reconcile MAVLink ACK: %v", err)
	}
	statusPlan, err := projector.ProjectStatusUpdate(statusUpdate)
	if err != nil {
		t.Fatalf("project status update: %v", err)
	}
	update := requireUpdate(t, statusPlan.Mutations[0])
	requireTriple(t, update.AddTriples, cop.TaskStatus, StatusAccepted)
	if hasPredicate(update.AddTriples, cop.TaskDesired) ||
		hasPredicate(update.AddTriples, cop.TaskAuthority) ||
		hasPredicate(update.AddTriples, cop.TaskTarget) {
		t.Fatalf("ACK update rewrote desired state, authority, or target: %+v", update.AddTriples)
	}
}

func TestSemLinkCompanionRequestRejectsUnsafeArduPilotActions(t *testing.T) {
	for _, action := range []string{"arm", "set_mode", "mission_upload", "offboard_setpoint"} {
		t.Run(action, func(t *testing.T) {
			_, err := NewSemLinkArduPilotReadbackIntent(SemLinkCompanionRequest{
				MeshNodeID:      "blueboat-01",
				NativeID:        "unsafe-" + action,
				TargetAssetID:   "c360.edge.cop.mavlink.asset.system-1",
				VehicleSystemID: 1,
				Action:          action,
				CorrelationID:   "semlink:unsafe:" + action,
				IdempotencyKey:  "semlink-unsafe-" + action,
				ObservedAt:      time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC),
				TTL:             time.Minute,
			})
			if err == nil {
				t.Fatal("expected unsafe SemLink companion action rejection")
			}
			if !strings.Contains(err.Error(), SemLinkActionRequestAutopilotVersion) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
