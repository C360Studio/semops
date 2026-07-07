package semlink

import (
	"context"
	"strings"
	"testing"
	"time"

	commandprojector "github.com/c360studio/semops/internal/projectors/command"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/message"
)

func TestIngressAdmitsSemLinkArduPilotReadbackAsIntentOnly(t *testing.T) {
	now := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-1"
	ingress := testIngress(now, targetID)

	result, plan, err := ingress.AdmitArduPilotReadback(context.Background(), ArduPilotReadbackRequest{
		MeshNodeID:       "blueboat-01",
		ID:               "autopilot-version",
		TargetAssetID:    targetID,
		VehicleSystemID:  1,
		VehicleComponent: 1,
		CorrelationID:    "semlink:blueboat-01:req-1",
		IdempotencyKey:   "semlink-blueboat-01-autopilot-version",
		TTL:              30 * time.Second,
	})
	if err != nil {
		t.Fatalf("admit SemLink readback: %v", err)
	}
	if !result.Admission.Accepted || result.Admission.RejectedReason != "" {
		t.Fatalf("admission = %+v, want accepted", result.Admission)
	}
	if result.ClaimScope != ClaimScopeCompanionIntentOnly ||
		result.NativeExecutionAllowed ||
		result.CompanionTransmitAllowed {
		t.Fatalf("result posture = %+v", result)
	}
	if result.Intent.NativeID != "semlink-autopilot-version" ||
		result.Intent.Source != commandprojector.SemLinkCompanionSource ||
		result.Intent.Authority != commandprojector.SemLinkCompanionAuthority ||
		result.Intent.SourceRef != "semlink://blueboat-01/ardupilot/system-1/request-autopilot-version" {
		t.Fatalf("intent = %+v", result.Intent)
	}

	triples := requireCreateTriples(t, plan)
	requireTriple(t, triples, cop.TaskTarget, targetID)
	requireTriple(t, triples, cop.TaskAuthority, commandprojector.SemLinkCompanionAuthority)
	requireTriple(t, triples, cop.TaskLocalOverridePolicy, commandprojector.LocalOverrideNotRequired)
	requireTriple(t, triples, cop.ProvenanceSource, commandprojector.SemLinkCompanionSource)
	requireTriple(t, triples, cop.ProvenanceSourceRef, "semlink://blueboat-01/ardupilot/system-1/request-autopilot-version")
	if !strings.Contains(result.Intent.DesiredState, `"message":"AUTOPILOT_VERSION"`) {
		t.Fatalf("desired state = %s", result.Intent.DesiredState)
	}
}

func TestIngressDerivesCanonicalMAVLinkTargetFromSemLinkRoute(t *testing.T) {
	now := time.Date(2026, 7, 7, 16, 15, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-42"
	ingress := testIngress(now, targetID)

	result, plan, err := ingress.AdmitArduPilotReadback(context.Background(), ArduPilotReadbackRequest{
		MeshNodeID:      "blueboat-01",
		ID:              "autopilot-version",
		VehicleSystemID: 42,
		CorrelationID:   "semlink:blueboat-01:req-42",
		IdempotencyKey:  "semlink-blueboat-01-autopilot-version-42",
		TTL:             30 * time.Second,
	})
	if err != nil {
		t.Fatalf("admit SemLink readback: %v", err)
	}
	if !result.Admission.Accepted {
		t.Fatalf("admission = %+v, want accepted", result.Admission)
	}
	if result.Intent.TargetAssetID != targetID {
		t.Fatalf("target asset = %q, want %q", result.Intent.TargetAssetID, targetID)
	}
	triples := requireCreateTriples(t, plan)
	requireTriple(t, triples, cop.TaskTarget, targetID)
}

func TestIngressRejectsUnsafeSemLinkCompanionActionBeforeProjection(t *testing.T) {
	now := time.Date(2026, 7, 7, 16, 30, 0, 0, time.UTC)
	_, plan, err := testIngress(now, "c360.edge.cop.mavlink.asset.system-1").AdmitArduPilotReadback(
		context.Background(),
		ArduPilotReadbackRequest{
			MeshNodeID:      "blueboat-01",
			ID:              "arm",
			TargetAssetID:   "c360.edge.cop.mavlink.asset.system-1",
			VehicleSystemID: 1,
			Action:          "arm",
			CorrelationID:   "semlink:arm",
			IdempotencyKey:  "semlink-arm",
			TTL:             time.Minute,
		},
	)
	if err == nil || !strings.Contains(err.Error(), commandprojector.SemLinkActionRequestAutopilotVersion) {
		t.Fatalf("error = %v, want MVP allowlist rejection", err)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want no projection", len(plan.Mutations))
	}
}

func TestIngressRejectsUnbornSemLinkTargetBeforeProjection(t *testing.T) {
	now := time.Date(2026, 7, 7, 17, 0, 0, 0, time.UTC)
	result, plan, err := testIngress(now).AdmitArduPilotReadback(context.Background(), ArduPilotReadbackRequest{
		MeshNodeID:      "blueboat-01",
		ID:              "autopilot-version",
		TargetAssetID:   "c360.edge.cop.mavlink.asset.system-1",
		VehicleSystemID: 1,
		CorrelationID:   "semlink:req",
		IdempotencyKey:  "semlink-req",
		TTL:             time.Minute,
	})
	if err != nil {
		t.Fatalf("admit missing target: %v", err)
	}
	if result.Admission.Accepted || !strings.Contains(result.Admission.RejectedReason, "target asset") {
		t.Fatalf("admission = %+v, want target rejection", result.Admission)
	}
	if result.NativeExecutionAllowed || result.CompanionTransmitAllowed {
		t.Fatalf("rejected result still allowed execution: %+v", result)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want no projection", len(plan.Mutations))
	}
}

func testIngress(now time.Time, targetIDs ...string) Ingress {
	return Ingress{
		Projector: commandprojector.NewGuardedProjector(
			commandprojector.NewProjector(commandprojector.Config{}),
			commandprojector.AdmissionConfig{
				Clock:          func() time.Time { return now },
				TargetResolver: commandprojector.NewStaticTargetResolver(targetIDs...),
			},
		),
		Clock: func() time.Time { return now },
	}
}

func requireCreateTriples(t *testing.T, plan commandprojector.Plan) []message.Triple {
	t.Helper()
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want one command-intent create", len(plan.Mutations))
	}
	mutation := plan.Mutations[0]
	if mutation.Kind != commandprojector.MutationCreate || mutation.Create.Entity == nil {
		t.Fatalf("mutation = %+v, want create", mutation)
	}
	return mutation.Create.Triples
}

func requireTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate != predicate {
			continue
		}
		if triple.Object != want {
			t.Fatalf("%s object = %#v, want %#v", predicate, triple.Object, want)
		}
		return
	}
	t.Fatalf("missing predicate %q in %+v", predicate, triples)
}
