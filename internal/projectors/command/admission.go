package command

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TargetResolver interface {
	TargetExists(ctx context.Context, targetAssetID string) (bool, error)
}

type IdempotencyStore interface {
	Reserve(ctx context.Context, key string, nativeID string) (IdempotencyReservation, error)
}

type IdempotencyReservation struct {
	Duplicate        bool
	ExistingNativeID string
}

type AdmissionConfig struct {
	Clock            func() time.Time
	TargetResolver   TargetResolver
	IdempotencyStore IdempotencyStore
	Arbitrator       *Arbitrator
}

type AdmissionResult struct {
	Accepted         bool
	RejectedReason   string
	Duplicate        bool
	ExistingNativeID string
}

type BatchProjectionResult struct {
	Admissions  []AdmissionResult
	Arbitration ArbitrationResult
}

type GuardedProjector struct {
	projector *Projector
	cfg       AdmissionConfig
}

func NewGuardedProjector(projector *Projector, cfg AdmissionConfig) *GuardedProjector {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.IdempotencyStore == nil {
		cfg.IdempotencyStore = NewMemoryIdempotencyStore()
	}
	return &GuardedProjector{projector: projector, cfg: cfg}
}

func (g *GuardedProjector) ProjectIntent(ctx context.Context, intent Intent) (AdmissionResult, Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if g == nil || g.projector == nil {
		return AdmissionResult{}, Plan{}, fmt.Errorf("command guarded projector is nil")
	}
	result, err := g.admit(ctx, intent)
	if err != nil {
		return AdmissionResult{}, Plan{}, err
	}
	if !result.Accepted {
		return result, Plan{}, nil
	}
	plan, err := g.projector.ProjectIntent(intent)
	if err != nil {
		return AdmissionResult{}, Plan{}, err
	}
	return result, plan, nil
}

func (g *GuardedProjector) ProjectIntents(ctx context.Context, intents []Intent) (BatchProjectionResult, Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if g == nil || g.projector == nil {
		return BatchProjectionResult{}, Plan{}, fmt.Errorf("command guarded projector is nil")
	}

	admissions := make([]AdmissionResult, 0, len(intents))
	admitted := make([]Intent, 0, len(intents))
	for _, intent := range intents {
		result, err := g.admit(ctx, intent)
		if err != nil {
			return BatchProjectionResult{}, Plan{}, err
		}
		admissions = append(admissions, result)
		if result.Accepted {
			admitted = append(admitted, intent)
		}
	}
	if len(admitted) == 0 {
		return BatchProjectionResult{Admissions: admissions}, Plan{}, nil
	}

	arbitrator := g.cfg.Arbitrator
	if arbitrator == nil {
		arbitrator = NewArbitrator(ArbitrationConfig{})
	}
	arbitration, err := arbitrator.Arbitrate(admitted)
	if err != nil {
		return BatchProjectionResult{}, Plan{}, err
	}
	plan, err := g.projector.ProjectIntents(arbitration.StatusUpdates())
	if err != nil {
		return BatchProjectionResult{}, Plan{}, err
	}
	return BatchProjectionResult{
		Admissions:  admissions,
		Arbitration: arbitration,
	}, plan, nil
}

func (r BatchProjectionResult) NativeExecutionCandidates() []Intent {
	return r.Arbitration.NativeExecutionCandidates()
}

func (g *GuardedProjector) admit(ctx context.Context, intent Intent) (AdmissionResult, error) {
	if err := intent.validate(); err != nil {
		return AdmissionResult{RejectedReason: err.Error()}, nil
	}
	if !intent.ExpiresAt.After(g.cfg.Clock().UTC()) {
		return AdmissionResult{RejectedReason: "command intent is expired"}, nil
	}
	if g.cfg.TargetResolver == nil {
		return AdmissionResult{}, fmt.Errorf("command admission requires target resolver")
	}
	exists, err := g.cfg.TargetResolver.TargetExists(ctx, intent.TargetAssetID)
	if err != nil {
		return AdmissionResult{}, fmt.Errorf("resolve command target: %w", err)
	}
	if !exists {
		return AdmissionResult{RejectedReason: "command target asset is not born"}, nil
	}

	reservation, err := g.cfg.IdempotencyStore.Reserve(ctx, intent.IdempotencyKey, intent.NativeID)
	if err != nil {
		return AdmissionResult{}, fmt.Errorf("reserve command idempotency key: %w", err)
	}
	if reservation.Duplicate {
		return AdmissionResult{
			RejectedReason:   "duplicate command idempotency key",
			Duplicate:        true,
			ExistingNativeID: reservation.ExistingNativeID,
		}, nil
	}
	return AdmissionResult{Accepted: true}, nil
}

type StaticTargetResolver struct {
	targets map[string]struct{}
}

func NewStaticTargetResolver(targetIDs ...string) *StaticTargetResolver {
	targets := make(map[string]struct{}, len(targetIDs))
	for _, targetID := range targetIDs {
		if targetID != "" {
			targets[targetID] = struct{}{}
		}
	}
	return &StaticTargetResolver{targets: targets}
}

func (r *StaticTargetResolver) TargetExists(_ context.Context, targetAssetID string) (bool, error) {
	if r == nil {
		return false, nil
	}
	_, ok := r.targets[targetAssetID]
	return ok, nil
}

type MemoryIdempotencyStore struct {
	mu     sync.Mutex
	claims map[string]string
}

func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{claims: make(map[string]string)}
}

func (s *MemoryIdempotencyStore) Reserve(_ context.Context, key string, nativeID string) (IdempotencyReservation, error) {
	if s == nil {
		return IdempotencyReservation{}, fmt.Errorf("idempotency store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claims == nil {
		s.claims = make(map[string]string)
	}
	if existing, ok := s.claims[key]; ok {
		return IdempotencyReservation{Duplicate: true, ExistingNativeID: existing}, nil
	}
	s.claims[key] = nativeID
	return IdempotencyReservation{}, nil
}
