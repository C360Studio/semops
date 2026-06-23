package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/message"
)

func TestGuardedProjectorAdmitsResolvedFreshIntent(t *testing.T) {
	intent := sampleIntent()
	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:          func() time.Time { return intent.ObservedAt },
			TargetResolver: NewStaticTargetResolver(intent.TargetAssetID),
		},
	)

	result, plan, err := guarded.ProjectIntent(context.Background(), intent)
	if err != nil {
		t.Fatalf("guarded project intent: %v", err)
	}
	if !result.Accepted || result.RejectedReason != "" {
		t.Fatalf("admission result = %+v, want accepted", result)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want admitted command intent create", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	requireTriple(t, create.Triples, cop.TaskTarget, intent.TargetAssetID)
}

func TestGuardedProjectorRejectsUnresolvedTargetBeforeProjection(t *testing.T) {
	intent := sampleIntent()
	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:          func() time.Time { return intent.ObservedAt },
			TargetResolver: NewStaticTargetResolver(),
		},
	)

	result, plan, err := guarded.ProjectIntent(context.Background(), intent)
	if err != nil {
		t.Fatalf("guarded project intent: %v", err)
	}
	if result.Accepted || !strings.Contains(result.RejectedReason, "target asset") {
		t.Fatalf("admission result = %+v, want target rejection", result)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want none for unresolved target", len(plan.Mutations))
	}
}

func TestGuardedProjectorRejectsExpiredIntentBeforeProjection(t *testing.T) {
	intent := sampleIntent()
	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:          func() time.Time { return intent.ExpiresAt.Add(time.Second) },
			TargetResolver: NewStaticTargetResolver(intent.TargetAssetID),
		},
	)

	result, plan, err := guarded.ProjectIntent(context.Background(), intent)
	if err != nil {
		t.Fatalf("guarded project intent: %v", err)
	}
	if result.Accepted || !strings.Contains(result.RejectedReason, "expired") {
		t.Fatalf("admission result = %+v, want expired rejection", result)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want none for expired intent", len(plan.Mutations))
	}
}

func TestGuardedProjectorCollapsesDuplicateIdempotencyBeforeProjection(t *testing.T) {
	intent := sampleIntent()
	store := NewMemoryIdempotencyStore()
	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:            func() time.Time { return intent.ObservedAt },
			TargetResolver:   NewStaticTargetResolver(intent.TargetAssetID),
			IdempotencyStore: store,
		},
	)

	first, firstPlan, err := guarded.ProjectIntent(context.Background(), intent)
	if err != nil {
		t.Fatalf("first guarded project intent: %v", err)
	}
	if !first.Accepted || len(firstPlan.Mutations) != 1 {
		t.Fatalf("first result/plan = %+v/%+v, want accepted create", first, firstPlan)
	}

	duplicate := intent
	duplicate.NativeID = "csapi-command-duplicate"
	second, secondPlan, err := guarded.ProjectIntent(context.Background(), duplicate)
	if err != nil {
		t.Fatalf("duplicate guarded project intent: %v", err)
	}
	if second.Accepted || !second.Duplicate || second.ExistingNativeID != intent.NativeID {
		t.Fatalf("duplicate result = %+v, want collapse to %q", second, intent.NativeID)
	}
	if len(secondPlan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want none for duplicate idempotency key", len(secondPlan.Mutations))
	}
}

func TestGuardedProjectorPropagatesAdmissionInfrastructureErrors(t *testing.T) {
	intent := sampleIntent()
	want := errors.New("resolver unavailable")
	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:          func() time.Time { return intent.ObservedAt },
			TargetResolver: failingTargetResolver{err: want},
		},
	)

	_, _, err := guarded.ProjectIntent(context.Background(), intent)
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want resolver error", err)
	}
}

func TestGuardedProjectorBatchProjectsArbitratedIntentStatuses(t *testing.T) {
	remote := sampleIntent()
	remote.NativeID = "remote-high-priority"
	remote.CorrelationID = "csapi:req-remote"
	remote.IdempotencyKey = "idem-remote"
	remote.Authority = "upstream.federated"
	remote.Priority = 100

	local := sampleIntent()
	local.NativeID = "local-low-priority"
	local.CorrelationID = "csapi:req-local"
	local.IdempotencyKey = "idem-local"
	local.Authority = "local.operator"
	local.Priority = 10
	local.ObservedAt = remote.ObservedAt.Add(time.Second)

	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:          func() time.Time { return remote.ObservedAt },
			TargetResolver: NewStaticTargetResolver(remote.TargetAssetID),
		},
	)

	result, plan, err := guarded.ProjectIntents(context.Background(), []Intent{remote, local})
	if err != nil {
		t.Fatalf("guarded project intents: %v", err)
	}
	if len(result.Admissions) != 2 || !result.Admissions[0].Accepted || !result.Admissions[1].Accepted {
		t.Fatalf("admissions = %+v, want both admitted before arbitration", result.Admissions)
	}
	requireOutcome(t, result.Arbitration, "local-low-priority", ArbitrationAccepted, "")
	requireOutcome(t, result.Arbitration, "remote-high-priority", ArbitrationSuperseded, "local-low-priority")
	if candidates := result.NativeExecutionCandidates(); len(candidates) != 1 ||
		candidates[0].NativeID != "local-low-priority" {
		t.Fatalf("execution candidates = %+v, want local command only", candidates)
	}
	requirePlanStatus(t, plan, "local-low-priority", StatusAccepted)
	requirePlanStatus(t, plan, "remote-high-priority", StatusSuperseded)
}

func TestGuardedProjectorBatchSkipsRejectedAndDuplicateBeforeArbitration(t *testing.T) {
	valid := sampleIntent()
	valid.NativeID = "valid-command"
	valid.IdempotencyKey = "idem-shared"

	duplicate := sampleIntent()
	duplicate.NativeID = "duplicate-command"
	duplicate.IdempotencyKey = "idem-shared"

	unresolved := sampleIntent()
	unresolved.NativeID = "unresolved-command"
	unresolved.IdempotencyKey = "idem-unresolved"
	unresolved.TargetAssetID = "c360.edge.cop.mavlink.asset.missing"

	guarded := NewGuardedProjector(
		NewProjector(Config{OwnerTokens: testOwnerTokens("test")}),
		AdmissionConfig{
			Clock:          func() time.Time { return valid.ObservedAt },
			TargetResolver: NewStaticTargetResolver(valid.TargetAssetID),
		},
	)

	result, plan, err := guarded.ProjectIntents(context.Background(), []Intent{valid, duplicate, unresolved})
	if err != nil {
		t.Fatalf("guarded project intents: %v", err)
	}
	if len(result.Admissions) != 3 {
		t.Fatalf("admissions = %d, want one per input", len(result.Admissions))
	}
	if !result.Admissions[0].Accepted {
		t.Fatalf("first admission = %+v, want accepted", result.Admissions[0])
	}
	if !result.Admissions[1].Duplicate || result.Admissions[1].ExistingNativeID != "valid-command" {
		t.Fatalf("duplicate admission = %+v, want duplicate collapse", result.Admissions[1])
	}
	if result.Admissions[2].Accepted || !strings.Contains(result.Admissions[2].RejectedReason, "target asset") {
		t.Fatalf("unresolved admission = %+v, want target rejection", result.Admissions[2])
	}
	if len(result.Arbitration.Decisions) != 1 {
		t.Fatalf("arbitration decisions = %+v, want only admitted command", result.Arbitration.Decisions)
	}
	requirePlanStatus(t, plan, "valid-command", StatusAccepted)
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want only accepted command projection", len(plan.Mutations))
	}
}

type failingTargetResolver struct {
	err error
}

func (r failingTargetResolver) TargetExists(context.Context, string) (bool, error) {
	return false, r.err
}

func requirePlanStatus(t *testing.T, plan Plan, nativeID string, status string) {
	t.Helper()
	for _, mutation := range plan.Mutations {
		var triples []message.Triple
		switch mutation.Kind {
		case MutationCreate:
			triples = mutation.Create.Triples
		case MutationUpdate:
			triples = mutation.Update.AddTriples
		default:
			continue
		}
		if !hasTripleValue(triples, cop.TaskNativeID, nativeID) {
			continue
		}
		requireTriple(t, triples, cop.TaskStatus, status)
		return
	}
	t.Fatalf("missing plan status for native id %q in %+v", nativeID, plan.Mutations)
}

func hasTripleValue(triples []message.Triple, predicate string, value any) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate && triple.Object == value {
			return true
		}
	}
	return false
}
