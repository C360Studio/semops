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
	AssociationReviewAcknowledged           = "acknowledged"
	AssociationReviewChallenged             = "challenged"
	AssociationReviewConflictBlocked        = copmodel.AssociationReviewConflictStateBlocked
	DefaultAssociationReviewer              = "operator.local"
	DefaultAssociationReviewerRole          = copmodel.AssociationReviewerRoleUnverified
	DefaultAssociationReviewAuthorityScope  = copmodel.AssociationReviewScopeDisplayOnly
	DefaultAssociationReviewAuthorityDomain = "local-display"
	DefaultAssociationReviewConflictPolicy  = copmodel.AssociationReviewConflictLatestDisplayOnly
	DefaultAssociationReviewConflictState   = copmodel.AssociationReviewConflictStateNone
	AuthenticatedAssociationReviewerRole    = copmodel.AssociationReviewerRoleAuthenticated
	AuthenticatedAssociationReviewScope     = copmodel.AssociationReviewScopeAssociationReview
	AuthenticatedAssociationReviewPolicy    = copmodel.AssociationReviewConflictMultiAuthority
	MaxAssociationReviewCommentLen          = 512
)

type AssociationReview struct {
	AssociationID   string    `json:"association_id"`
	Decision        string    `json:"decision"`
	ReviewedBy      string    `json:"reviewed_by"`
	ReviewedAt      time.Time `json:"reviewed_at"`
	ReviewerRole    string    `json:"reviewer_role"`
	AuthorityScope  string    `json:"authority_scope"`
	AuthorityDomain string    `json:"authority_domain"`
	ConflictPolicy  string    `json:"conflict_policy"`
	ConflictState   string    `json:"conflict_state"`
	Authenticated   bool      `json:"authenticated"`
	Comment         string    `json:"comment,omitempty"`
}

type AssociationReviewStore interface {
	PutAssociationReview(context.Context, AssociationReview) (AssociationReview, error)
	ListAssociationReviews(context.Context) ([]AssociationReview, error)
}

type MemoryAssociationReviewStore struct {
	mu                 sync.RWMutex
	reviews            map[string]AssociationReview
	authenticatedVotes map[string]map[string]AssociationReview
}

func NewMemoryAssociationReviewStore() *MemoryAssociationReviewStore {
	return &MemoryAssociationReviewStore{
		reviews:            make(map[string]AssociationReview),
		authenticatedVotes: make(map[string]map[string]AssociationReview),
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
	return s.putAssociationReviewLocked(review)
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

func (s *MemoryAssociationReviewStore) PreviewAssociationReview(
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	return previewAssociationReview(review, s.authenticatedVotes)
}

func (s *MemoryAssociationReviewStore) putAssociationReviewLocked(review AssociationReview) (AssociationReview, error) {
	if review.Authenticated {
		votes := s.authenticatedVotes[review.AssociationID]
		if votes == nil {
			votes = make(map[string]AssociationReview)
			s.authenticatedVotes[review.AssociationID] = votes
		}
		votes[review.AuthorityDomain] = review
		effective := arbitrateAuthenticatedAssociationReviews(votes)
		s.reviews[review.AssociationID] = effective
		return effective, nil
	}
	if len(s.authenticatedVotes[review.AssociationID]) > 0 {
		return AssociationReview{}, fmt.Errorf(
			"display-only association review cannot overwrite authenticated authority review for %s",
			review.AssociationID,
		)
	}
	s.reviews[review.AssociationID] = review
	return review, nil
}

func previewAssociationReview(
	review AssociationReview,
	existingVotes map[string]map[string]AssociationReview,
) (AssociationReview, error) {
	if review.Authenticated {
		votes := cloneAssociationReviewVotes(existingVotes[review.AssociationID])
		votes[review.AuthorityDomain] = review
		return arbitrateAuthenticatedAssociationReviews(votes), nil
	}
	if len(existingVotes[review.AssociationID]) > 0 {
		return AssociationReview{}, fmt.Errorf(
			"display-only association review cannot overwrite authenticated authority review for %s",
			review.AssociationID,
		)
	}
	return review, nil
}

func arbitrateAuthenticatedAssociationReviews(votes map[string]AssociationReview) AssociationReview {
	domains := sortedAssociationReviewDomains(votes)
	if len(domains) == 0 {
		return AssociationReview{}
	}
	latest := votes[domains[0]]
	decisions := make(map[string][]string)
	for _, domain := range domains {
		vote := votes[domain]
		decisions[vote.Decision] = append(decisions[vote.Decision], domain)
		if vote.ReviewedAt.After(latest.ReviewedAt) ||
			(vote.ReviewedAt.Equal(latest.ReviewedAt) && domain < latest.AuthorityDomain) {
			latest = vote
		}
	}
	if len(decisions) == 1 {
		latest.AuthorityDomain = strings.Join(domains, ",")
		latest.ConflictState = DefaultAssociationReviewConflictState
		latest.ConflictPolicy = AuthenticatedAssociationReviewPolicy
		latest.Authenticated = true
		if len(domains) > 1 {
			latest.Comment = fmt.Sprintf("multi-authority consensus: %s across %s", latest.Decision, latest.AuthorityDomain)
		}
		return latest
	}
	return AssociationReview{
		AssociationID:   latest.AssociationID,
		Decision:        AssociationReviewConflictBlocked,
		ReviewedBy:      "multi-authority",
		ReviewedAt:      latest.ReviewedAt,
		ReviewerRole:    AuthenticatedAssociationReviewerRole,
		AuthorityScope:  AuthenticatedAssociationReviewScope,
		AuthorityDomain: strings.Join(domains, ","),
		ConflictPolicy:  AuthenticatedAssociationReviewPolicy,
		ConflictState:   AssociationReviewConflictBlocked,
		Authenticated:   true,
		Comment:         authenticatedConflictComment(decisions),
	}
}

func authenticatedConflictComment(decisions map[string][]string) string {
	parts := make([]string, 0, len(decisions))
	for _, decision := range sortedAssociationReviewDecisions(decisions) {
		domains := append([]string(nil), decisions[decision]...)
		sort.Strings(domains)
		parts = append(parts, decision+"="+strings.Join(domains, ","))
	}
	return "conflicting association review decisions across authority domains: " + strings.Join(parts, " ")
}

func sortedAssociationReviewDomains(votes map[string]AssociationReview) []string {
	domains := make([]string, 0, len(votes))
	for domain := range votes {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	return domains
}

func sortedAssociationReviewDecisions(decisions map[string][]string) []string {
	out := make([]string, 0, len(decisions))
	for decision := range decisions {
		out = append(out, decision)
	}
	sort.Strings(out)
	return out
}

func cloneAssociationReviewVotes(votes map[string]AssociationReview) map[string]AssociationReview {
	cloned := make(map[string]AssociationReview, len(votes)+1)
	for domain, review := range votes {
		cloned[domain] = review
	}
	return cloned
}

func validateAssociationReview(review AssociationReview) error {
	if strings.TrimSpace(review.AssociationID) == "" {
		return fmt.Errorf("association review association_id is required")
	}
	switch normalizeAssociationReviewDecision(review.Decision) {
	case AssociationReviewAcknowledged, AssociationReviewChallenged, AssociationReviewConflictBlocked:
	default:
		return fmt.Errorf(
			"association review decision must be %q, %q, or %q",
			AssociationReviewAcknowledged,
			AssociationReviewChallenged,
			AssociationReviewConflictBlocked,
		)
	}
	if strings.TrimSpace(review.ReviewedBy) == "" {
		return fmt.Errorf("association review reviewed_by is required")
	}
	if review.ReviewedAt.IsZero() {
		return fmt.Errorf("association review reviewed_at is required")
	}
	if review.Authenticated {
		if review.ReviewerRole != AuthenticatedAssociationReviewerRole {
			return fmt.Errorf("authenticated association review reviewer_role must be %q", AuthenticatedAssociationReviewerRole)
		}
		if review.AuthorityScope != AuthenticatedAssociationReviewScope {
			return fmt.Errorf("authenticated association review authority_scope must be %q", AuthenticatedAssociationReviewScope)
		}
		if strings.TrimSpace(review.AuthorityDomain) == "" || review.AuthorityDomain == DefaultAssociationReviewAuthorityDomain {
			return fmt.Errorf("authenticated association review authority_domain is required")
		}
		if review.ConflictPolicy != AuthenticatedAssociationReviewPolicy {
			return fmt.Errorf("authenticated association review conflict_policy must be %q", AuthenticatedAssociationReviewPolicy)
		}
	} else {
		if review.ReviewerRole != DefaultAssociationReviewerRole {
			return fmt.Errorf("association review reviewer_role must be %q", DefaultAssociationReviewerRole)
		}
		if review.AuthorityScope != DefaultAssociationReviewAuthorityScope {
			return fmt.Errorf("association review authority_scope must be %q", DefaultAssociationReviewAuthorityScope)
		}
		if review.AuthorityDomain != DefaultAssociationReviewAuthorityDomain {
			return fmt.Errorf("association review authority_domain must be %q", DefaultAssociationReviewAuthorityDomain)
		}
		if review.ConflictPolicy != DefaultAssociationReviewConflictPolicy {
			return fmt.Errorf("association review conflict_policy must be %q", DefaultAssociationReviewConflictPolicy)
		}
	}
	if review.Decision == AssociationReviewConflictBlocked {
		if !review.Authenticated {
			return fmt.Errorf("association review blocked conflicts require authenticated authority review")
		}
		if review.ConflictState != AssociationReviewConflictBlocked {
			return fmt.Errorf("association review conflict_state must be %q for blocked conflicts", AssociationReviewConflictBlocked)
		}
	} else if review.ConflictState != DefaultAssociationReviewConflictState {
		return fmt.Errorf("association review conflict_state must be %q", DefaultAssociationReviewConflictState)
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
		if review.Authenticated {
			review.ReviewerRole = AuthenticatedAssociationReviewerRole
		} else {
			review.ReviewerRole = DefaultAssociationReviewerRole
		}
	}
	review.AuthorityScope = strings.TrimSpace(review.AuthorityScope)
	if review.AuthorityScope == "" {
		if review.Authenticated {
			review.AuthorityScope = AuthenticatedAssociationReviewScope
		} else {
			review.AuthorityScope = DefaultAssociationReviewAuthorityScope
		}
	}
	review.AuthorityDomain = strings.TrimSpace(review.AuthorityDomain)
	if review.AuthorityDomain == "" && !review.Authenticated {
		review.AuthorityDomain = DefaultAssociationReviewAuthorityDomain
	}
	review.ConflictPolicy = strings.TrimSpace(review.ConflictPolicy)
	if review.ConflictPolicy == "" {
		if review.Authenticated {
			review.ConflictPolicy = AuthenticatedAssociationReviewPolicy
		} else {
			review.ConflictPolicy = DefaultAssociationReviewConflictPolicy
		}
	}
	review.ConflictState = strings.TrimSpace(review.ConflictState)
	if review.ConflictState == "" {
		review.ConflictState = DefaultAssociationReviewConflictState
	}
	review.Comment = strings.TrimSpace(review.Comment)
	return review
}

func normalizeAssociationReviewDecision(decision string) string {
	return strings.ToLower(strings.TrimSpace(decision))
}
