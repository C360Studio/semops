package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
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

type failingTargetResolver struct {
	err error
}

func (r failingTargetResolver) TargetExists(context.Context, string) (bool, error) {
	return false, r.err
}
