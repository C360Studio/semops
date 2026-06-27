package cop

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMemoryAssociationReviewStoreBlocksConflictingAuthenticatedVotes(t *testing.T) {
	store := NewMemoryAssociationReviewStore()
	first := authenticatedReviewVote(
		time.Date(2026, 6, 27, 20, 0, 0, 0, time.UTC),
		"incident-command",
		AssociationReviewAcknowledged,
	)
	second := authenticatedReviewVote(
		time.Date(2026, 6, 27, 20, 1, 0, 0, time.UTC),
		"airspace-control",
		AssociationReviewChallenged,
	)

	if got, err := store.PutAssociationReview(context.Background(), first); err != nil ||
		got.Decision != AssociationReviewAcknowledged ||
		got.ConflictState != DefaultAssociationReviewConflictState {
		t.Fatalf("first review = %+v, err = %v", got, err)
	}
	got, err := store.PutAssociationReview(context.Background(), second)
	if err != nil {
		t.Fatalf("put conflicting review: %v", err)
	}

	if got.Decision != AssociationReviewConflictBlocked ||
		got.ConflictState != AssociationReviewConflictBlocked ||
		got.AuthorityDomain != "airspace-control,incident-command" ||
		!strings.Contains(got.Comment, "acknowledged=incident-command") ||
		!strings.Contains(got.Comment, "challenged=airspace-control") {
		t.Fatalf("effective review = %+v, want blocked multi-authority conflict", got)
	}
	reviews, err := store.ListAssociationReviews(context.Background())
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if len(reviews) != 1 || reviews[0].Decision != AssociationReviewConflictBlocked {
		t.Fatalf("reviews = %+v, want one blocked conflict", reviews)
	}
}

func TestMemoryAssociationReviewStoreAcceptsAuthenticatedConsensus(t *testing.T) {
	store := NewMemoryAssociationReviewStore()
	first := authenticatedReviewVote(
		time.Date(2026, 6, 27, 20, 0, 0, 0, time.UTC),
		"incident-command",
		AssociationReviewAcknowledged,
	)
	second := authenticatedReviewVote(
		time.Date(2026, 6, 27, 20, 1, 0, 0, time.UTC),
		"airspace-control",
		AssociationReviewAcknowledged,
	)

	if _, err := store.PutAssociationReview(context.Background(), first); err != nil {
		t.Fatalf("put first review: %v", err)
	}
	got, err := store.PutAssociationReview(context.Background(), second)
	if err != nil {
		t.Fatalf("put second review: %v", err)
	}

	if got.Decision != AssociationReviewAcknowledged ||
		got.ConflictState != DefaultAssociationReviewConflictState ||
		got.AuthorityDomain != "airspace-control,incident-command" ||
		!got.Authenticated ||
		!strings.Contains(got.Comment, "multi-authority consensus") {
		t.Fatalf("effective review = %+v, want authenticated consensus", got)
	}
}

func TestMemoryAssociationReviewStoreRejectsDisplayReviewAfterAuthenticatedVote(t *testing.T) {
	store := NewMemoryAssociationReviewStore()
	if _, err := store.PutAssociationReview(
		context.Background(),
		authenticatedReviewVote(time.Date(2026, 6, 27, 20, 0, 0, 0, time.UTC), "incident-command", AssociationReviewAcknowledged),
	); err != nil {
		t.Fatalf("put authenticated review: %v", err)
	}

	_, err := store.PutAssociationReview(context.Background(), AssociationReview{
		AssociationID: "association-1",
		Decision:      AssociationReviewChallenged,
		ReviewedBy:    "operator.local",
		ReviewedAt:    time.Date(2026, 6, 27, 20, 1, 0, 0, time.UTC),
	})
	if err == nil || !strings.Contains(err.Error(), "display-only association review cannot overwrite") {
		t.Fatalf("error = %v, want display overwrite rejection", err)
	}
}

func authenticatedReviewVote(reviewedAt time.Time, domain, decision string) AssociationReview {
	return AssociationReview{
		AssociationID:   "association-1",
		Decision:        decision,
		ReviewedBy:      "operator:" + domain,
		ReviewedAt:      reviewedAt,
		ReviewerRole:    AuthenticatedAssociationReviewerRole,
		AuthorityScope:  AuthenticatedAssociationReviewScope,
		AuthorityDomain: domain,
		ConflictPolicy:  AuthenticatedAssociationReviewPolicy,
		ConflictState:   DefaultAssociationReviewConflictState,
		Authenticated:   true,
	}
}
