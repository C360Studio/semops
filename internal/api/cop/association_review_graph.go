package cop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	fusionprojector "github.com/c360studio/semops/internal/projectors/fusion"
	"github.com/c360studio/semstreams/graph"
)

type AssociationReviewGraphStoreConfig struct {
	Org       string
	Platform  string
	Projector *fusionprojector.Projector
	Writer    AssociationReviewPlanWriter
}

type AssociationReviewPlanWriter interface {
	Apply(context.Context, fusionprojector.Plan) error
}

type GraphAssociationReviewStore struct {
	local AssociationReviewStore
	cfg   AssociationReviewGraphStoreConfig
}

func NewGraphAssociationReviewStore(
	local AssociationReviewStore,
	cfg AssociationReviewGraphStoreConfig,
) (*GraphAssociationReviewStore, error) {
	if local == nil {
		local = NewMemoryAssociationReviewStore()
	}
	if strings.TrimSpace(cfg.Org) == "" {
		return nil, fmt.Errorf("association review graph store org is required")
	}
	if strings.TrimSpace(cfg.Platform) == "" {
		return nil, fmt.Errorf("association review graph store platform is required")
	}
	if cfg.Projector == nil {
		return nil, fmt.Errorf("association review graph store projector is required")
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("association review graph store writer is required")
	}
	return &GraphAssociationReviewStore{local: local, cfg: cfg}, nil
}

func (s *GraphAssociationReviewStore) PutAssociationReview(
	ctx context.Context,
	review AssociationReview,
) (AssociationReview, error) {
	if s == nil {
		return AssociationReview{}, fmt.Errorf("association review graph store is nil")
	}
	review = normalizeAssociationReview(review)
	if err := validateAssociationReview(review); err != nil {
		return AssociationReview{}, err
	}
	evidence := fusionprojector.AssociationReviewEvidence{
		Org:            s.cfg.Org,
		Platform:       s.cfg.Platform,
		AssociationID:  review.AssociationID,
		Decision:       review.Decision,
		ReviewedBy:     review.ReviewedBy,
		ReviewedAt:     review.ReviewedAt,
		ReviewerRole:   review.ReviewerRole,
		AuthorityScope: review.AuthorityScope,
		ConflictPolicy: review.ConflictPolicy,
		Comment:        review.Comment,
	}
	if err := s.writeReview(ctx, evidence); err != nil {
		return AssociationReview{}, err
	}
	return s.local.PutAssociationReview(ctx, review)
}

func (s *GraphAssociationReviewStore) ListAssociationReviews(ctx context.Context) ([]AssociationReview, error) {
	if s == nil {
		return nil, fmt.Errorf("association review graph store is nil")
	}
	return s.local.ListAssociationReviews(ctx)
}

func (s *GraphAssociationReviewStore) writeReview(
	ctx context.Context,
	evidence fusionprojector.AssociationReviewEvidence,
) error {
	plan, err := s.cfg.Projector.ProjectAssociationReview(evidence)
	if err != nil {
		return fmt.Errorf("project association review audit: %w", err)
	}
	if err := s.cfg.Writer.Apply(ctx, plan); err != nil {
		entityID, ok := associationReviewAlreadyExists(err)
		if !ok || !s.cfg.Projector.MarkBornForAssociationReview(evidence, entityID) {
			return fmt.Errorf("write association review audit: %w", err)
		}
		updatePlan, projectErr := s.cfg.Projector.ProjectAssociationReview(evidence)
		if projectErr != nil {
			return fmt.Errorf("reproject association review audit after birth reconciliation: %w", projectErr)
		}
		if applyErr := s.cfg.Writer.Apply(ctx, updatePlan); applyErr != nil {
			return fmt.Errorf("write association review audit update: %w", applyErr)
		}
		return nil
	}
	s.cfg.Projector.MarkBornForPlan(plan)
	s.cfg.Projector.MarkBornForAssociationReview(
		evidence,
		fusionprojector.AssociationReviewEntityID(evidence.Org, evidence.Platform, evidence.AssociationID),
	)
	return nil
}

func associationReviewAlreadyExists(err error) (string, bool) {
	var mutationErr *fusionprojector.MutationFailureError
	if !errors.As(err, &mutationErr) {
		return "", false
	}
	if mutationErr.Kind != fusionprojector.MutationCreate ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists ||
		mutationErr.EntityID == "" {
		return "", false
	}
	return mutationErr.EntityID, true
}

func NewAssociationReviewGraphWriter(
	requester fusionprojector.GraphRequester,
	timeout time.Duration,
) *fusionprojector.GraphWriter {
	return fusionprojector.NewGraphWriter(requester, fusionprojector.WithWriteTimeout(timeout))
}
