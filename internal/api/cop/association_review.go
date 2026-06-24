package cop

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	copmodel "github.com/c360studio/semops/pkg/cop"
)

const (
	AssociationReviewAcknowledged          = "acknowledged"
	AssociationReviewChallenged            = "challenged"
	DefaultAssociationReviewer             = "operator.local"
	DefaultAssociationReviewerRole         = copmodel.AssociationReviewerRoleUnverified
	DefaultAssociationReviewAuthorityScope = copmodel.AssociationReviewScopeDisplayOnly
	DefaultAssociationReviewConflictPolicy = copmodel.AssociationReviewConflictLatestDisplayOnly
	MaxAssociationReviewCommentLen         = 512
)

type AssociationReview struct {
	AssociationID  string    `json:"association_id"`
	Decision       string    `json:"decision"`
	ReviewedBy     string    `json:"reviewed_by"`
	ReviewedAt     time.Time `json:"reviewed_at"`
	ReviewerRole   string    `json:"reviewer_role"`
	AuthorityScope string    `json:"authority_scope"`
	ConflictPolicy string    `json:"conflict_policy"`
	Comment        string    `json:"comment,omitempty"`
}

type AssociationReviewStore interface {
	PutAssociationReview(context.Context, AssociationReview) (AssociationReview, error)
	ListAssociationReviews(context.Context) ([]AssociationReview, error)
}

type MemoryAssociationReviewStore struct {
	mu      sync.RWMutex
	reviews map[string]AssociationReview
}

func NewMemoryAssociationReviewStore() *MemoryAssociationReviewStore {
	return &MemoryAssociationReviewStore{
		reviews: make(map[string]AssociationReview),
	}
}

func (s *MemoryAssociationReviewStore) PutAssociationReview(
	_ context.Context,
	review AssociationReview,
) (AssociationReview, error) {
	if s == nil {
		return AssociationReview{}, fmt.Errorf("association review store is nil")
	}
	review = normalizeAssociationReview(review)
	if err := validateAssociationReview(review); err != nil {
		return AssociationReview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reviews[review.AssociationID] = review
	return review, nil
}

func (s *MemoryAssociationReviewStore) ListAssociationReviews(context.Context) ([]AssociationReview, error) {
	if s == nil {
		return nil, fmt.Errorf("association review store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	reviews := make([]AssociationReview, 0, len(s.reviews))
	for _, review := range s.reviews {
		reviews = append(reviews, review)
	}
	sort.Slice(reviews, func(i, j int) bool {
		return reviews[i].AssociationID < reviews[j].AssociationID
	})
	return reviews, nil
}

func validateAssociationReview(review AssociationReview) error {
	if strings.TrimSpace(review.AssociationID) == "" {
		return fmt.Errorf("association review association_id is required")
	}
	switch normalizeAssociationReviewDecision(review.Decision) {
	case AssociationReviewAcknowledged, AssociationReviewChallenged:
	default:
		return fmt.Errorf("association review decision must be %q or %q", AssociationReviewAcknowledged, AssociationReviewChallenged)
	}
	if strings.TrimSpace(review.ReviewedBy) == "" {
		return fmt.Errorf("association review reviewed_by is required")
	}
	if review.ReviewedAt.IsZero() {
		return fmt.Errorf("association review reviewed_at is required")
	}
	if review.ReviewerRole != DefaultAssociationReviewerRole {
		return fmt.Errorf("association review reviewer_role must be %q", DefaultAssociationReviewerRole)
	}
	if review.AuthorityScope != DefaultAssociationReviewAuthorityScope {
		return fmt.Errorf("association review authority_scope must be %q", DefaultAssociationReviewAuthorityScope)
	}
	if review.ConflictPolicy != DefaultAssociationReviewConflictPolicy {
		return fmt.Errorf("association review conflict_policy must be %q", DefaultAssociationReviewConflictPolicy)
	}
	if len(review.Comment) > MaxAssociationReviewCommentLen {
		return fmt.Errorf("association review comment exceeds %d characters", MaxAssociationReviewCommentLen)
	}
	return nil
}

func normalizeAssociationReview(review AssociationReview) AssociationReview {
	review.AssociationID = strings.TrimSpace(review.AssociationID)
	review.Decision = normalizeAssociationReviewDecision(review.Decision)
	review.ReviewedBy = strings.TrimSpace(review.ReviewedBy)
	if review.ReviewedBy == "" {
		review.ReviewedBy = DefaultAssociationReviewer
	}
	review.ReviewerRole = strings.TrimSpace(review.ReviewerRole)
	if review.ReviewerRole == "" {
		review.ReviewerRole = DefaultAssociationReviewerRole
	}
	review.AuthorityScope = strings.TrimSpace(review.AuthorityScope)
	if review.AuthorityScope == "" {
		review.AuthorityScope = DefaultAssociationReviewAuthorityScope
	}
	review.ConflictPolicy = strings.TrimSpace(review.ConflictPolicy)
	if review.ConflictPolicy == "" {
		review.ConflictPolicy = DefaultAssociationReviewConflictPolicy
	}
	review.Comment = strings.TrimSpace(review.Comment)
	return review
}

func normalizeAssociationReviewDecision(decision string) string {
	return strings.ToLower(strings.TrimSpace(decision))
}
