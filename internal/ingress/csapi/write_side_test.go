package csapi

import (
	"context"
	"strings"
	"testing"
	"time"

	commandprojector "github.com/c360studio/semops/internal/projectors/command"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/message"
)

func TestIngressAdmitsCSAPICommandAsCommandIntentOnly(t *testing.T) {
	now := time.Date(2026, 6, 27, 16, 0, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-42"
	ingress := testIngress(now, targetID)

	result, plan, err := ingress.AdmitCommand(context.Background(), CommandRequest{
		ID:                     "route-42",
		TargetAssetID:          targetID,
		Name:                   "Route MAVLink system 42",
		Kind:                   "mavlink.goto",
		Description:            "CS API requested route change",
		DesiredState:           `{"command":"goto","lat":38.9,"lon":-77.04}`,
		Authority:              "incident.commander",
		AuthorityAuthenticated: true,
		Priority:               75,
		TTL:                    2 * time.Minute,
		CorrelationID:          "csapi:req-route-42",
		IdempotencyKey:         "idem-route-42",
		RequestedBy:            "csapi:federated",
		LocalOverridePolicy:    commandprojector.LocalOverrideAcknowledged,
		ObservedAt:             now,
	})
	if err != nil {
		t.Fatalf("admit command: %v", err)
	}
	if !result.Admission.Accepted || result.Admission.RejectedReason != "" {
		t.Fatalf("admission = %+v, want accepted", result.Admission)
	}
	if result.Surface != SurfaceCommand ||
		result.ClaimScope != ClaimScopeCommandIntentOnly ||
		result.NativeExecutionAllowed ||
		result.UpstreamStatusPublicationAllowed {
		t.Fatalf("result posture = %+v", result)
	}
	if result.Intent.NativeID != "csapi-command-route-42" ||
		result.Intent.ExpiresAt != now.Add(2*time.Minute) ||
		result.Intent.SourceRef != "csapi://commands/route-42" {
		t.Fatalf("intent = %+v", result.Intent)
	}
	triples := requireCreateTriples(t, plan)
	requireTriple(t, triples, cop.TaskTarget, targetID)
	requireTriple(t, triples, cop.TaskAuthority, "incident.commander")
	requireTriple(t, triples, cop.TaskLocalOverridePolicy, commandprojector.LocalOverrideAcknowledged)
	requireTriple(t, triples, cop.ProvenanceSource, SourceCSAPI)
	requireTriple(t, triples, cop.ProvenanceSourceRef, "csapi://commands/route-42")
}

func TestIngressAdmitsControlStreamCommandAsCommandIntentOnly(t *testing.T) {
	now := time.Date(2026, 6, 27, 16, 30, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-42"
	ingress := testIngress(now, targetID)

	result, plan, err := ingress.AdmitControlStream(context.Background(), ControlStreamRequest{
		StreamID: "vehicle-42/gimbal",
		Command: CommandRequest{
			ID:                     "point-7",
			TargetAssetID:          targetID,
			DesiredState:           `{"command":"point_gimbal","azimuth":90}`,
			Authority:              "local.operator",
			AuthorityAuthenticated: true,
			Priority:               40,
			Deadline:               now.Add(45 * time.Second),
			CorrelationID:          "csapi:req-point-7",
			IdempotencyKey:         "idem-point-7",
			RequestedBy:            "operator:coby",
			LocalOverridePolicy:    commandprojector.LocalOverrideAcknowledged,
			ObservedAt:             now,
		},
	})
	if err != nil {
		t.Fatalf("admit controlstream: %v", err)
	}
	if !result.Admission.Accepted {
		t.Fatalf("admission = %+v, want accepted", result.Admission)
	}
	if result.Surface != SurfaceControlStream ||
		result.Intent.NativeID != "csapi-controlstream-vehicle-42-gimbal-point-7" ||
		result.Intent.Kind != "csapi.controlstream.command" ||
		result.Intent.SourceRef != "csapi://controlstreams/vehicle-42-gimbal/commands/point-7" ||
		result.NativeExecutionAllowed {
		t.Fatalf("result/intent = %+v / %+v", result, result.Intent)
	}
	requireTriple(t, requireCreateTriples(t, plan), cop.TaskKind, "csapi.controlstream.command")
}

func TestIngressRejectsCSAPICommandBeforeProjection(t *testing.T) {
	now := time.Date(2026, 6, 27, 17, 0, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-42"

	tests := []struct {
		name    string
		request CommandRequest
		ingress Ingress
		want    string
	}{
		{
			name:    "expired ttl",
			request: validCommandRequest(now.Add(-2*time.Minute), targetID, "expired"),
			ingress: testIngress(now, targetID),
			want:    "expired",
		},
		{
			name:    "unresolved target",
			request: validCommandRequest(now, targetID, "missing-target"),
			ingress: testIngress(now),
			want:    "target asset",
		},
		{
			name: "missing local override",
			request: func() CommandRequest {
				req := validCommandRequest(now, targetID, "missing-override")
				req.LocalOverridePolicy = ""
				return req
			}(),
			ingress: testIngress(now, targetID),
			want:    "local_override_policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, plan, err := tt.ingress.AdmitCommand(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("admit command: %v", err)
			}
			if result.Admission.Accepted || !strings.Contains(result.Admission.RejectedReason, tt.want) {
				t.Fatalf("admission = %+v, want rejection containing %q", result.Admission, tt.want)
			}
			if len(plan.Mutations) != 0 {
				t.Fatalf("mutations = %d, want no projection", len(plan.Mutations))
			}
		})
	}
}

func TestIngressRejectsUnauthenticatedAuthorityBeforeAdmission(t *testing.T) {
	now := time.Date(2026, 6, 27, 17, 15, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-42"
	request := validCommandRequest(now, targetID, "unauthenticated")
	request.AuthorityAuthenticated = false

	_, plan, err := testIngress(now, targetID).AdmitCommand(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "authenticated") {
		t.Fatalf("error = %v, want authenticated authority rejection", err)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want no projection", len(plan.Mutations))
	}
}

func TestIngressCollapsesDuplicateIdempotencyBeforeProjection(t *testing.T) {
	now := time.Date(2026, 6, 27, 17, 30, 0, 0, time.UTC)
	targetID := "c360.edge.cop.mavlink.asset.system-42"
	ingress := testIngress(now, targetID)
	first := validCommandRequest(now, targetID, "route-42")

	firstResult, firstPlan, err := ingress.AdmitCommand(context.Background(), first)
	if err != nil {
		t.Fatalf("admit first command: %v", err)
	}
	if !firstResult.Admission.Accepted || len(firstPlan.Mutations) != 1 {
		t.Fatalf("first result/plan = %+v / %+v, want accepted projection", firstResult, firstPlan)
	}

	duplicate := validCommandRequest(now.Add(time.Second), targetID, "route-42-duplicate")
	duplicate.IdempotencyKey = first.IdempotencyKey
	duplicateResult, duplicatePlan, err := ingress.AdmitCommand(context.Background(), duplicate)
	if err != nil {
		t.Fatalf("admit duplicate command: %v", err)
	}
	if duplicateResult.Admission.Accepted ||
		!duplicateResult.Admission.Duplicate ||
		duplicateResult.Admission.ExistingNativeID != "csapi-command-route-42" {
		t.Fatalf("duplicate admission = %+v", duplicateResult.Admission)
	}
	if len(duplicatePlan.Mutations) != 0 {
		t.Fatalf("duplicate mutations = %d, want none", len(duplicatePlan.Mutations))
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

func validCommandRequest(observedAt time.Time, targetID string, id string) CommandRequest {
	return CommandRequest{
		ID:                     id,
		TargetAssetID:          targetID,
		Kind:                   "mavlink.goto",
		DesiredState:           `{"command":"goto","lat":38.9,"lon":-77.04}`,
		Authority:              "local.operator",
		AuthorityAuthenticated: true,
		Priority:               80,
		TTL:                    time.Minute,
		CorrelationID:          "csapi:req-" + id,
		IdempotencyKey:         "idem-" + id,
		RequestedBy:            "operator:coby",
		LocalOverridePolicy:    commandprojector.LocalOverrideAcknowledged,
		ObservedAt:             observedAt,
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
