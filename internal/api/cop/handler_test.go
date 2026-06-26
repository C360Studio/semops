package cop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerServesSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	handler, err := NewHandler(NewFixtureProvider(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/snapshot", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var snapshot Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.GeneratedAt != now {
		t.Fatalf("generated at = %s, want %s", snapshot.GeneratedAt, now)
	}
	if snapshot.Summary.ActiveTracks != 2 || len(snapshot.Tracks) != 2 {
		t.Fatalf("tracks summary/list = %d/%d, want 2/2", snapshot.Summary.ActiveTracks, len(snapshot.Tracks))
	}
	if snapshot.Summary.ActiveTasks != 1 || snapshot.Summary.ActiveAdvisories != 1 {
		t.Fatalf("TAK summary = %+v", snapshot.Summary)
	}
	if snapshot.Tracks[0].Position.Lat == 0 || snapshot.Tracks[0].Position.Lon == 0 {
		t.Fatalf("track position missing: %+v", snapshot.Tracks[0].Position)
	}
	if snapshot.Tracks[0].Provenance.Owner != "semops.feed.mavlink" {
		t.Fatalf("track owner = %q", snapshot.Tracks[0].Provenance.Owner)
	}
}

func TestHandlerReportsProviderFailure(t *testing.T) {
	handler, err := NewHandler(failingProvider{})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/snapshot", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandlerServesRuntimeSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 21, 17, 30, 0, 0, time.UTC)
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithClock(func() time.Time { return now }),
		WithRuntimeProvider(runtimeProviderStub{}),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/runtime", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var snapshot RuntimeSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode runtime snapshot: %v", err)
	}
	if snapshot.GeneratedAt != now {
		t.Fatalf("generated at = %s, want %s", snapshot.GeneratedAt, now)
	}
	if snapshot.Feeds == nil || snapshot.Components == nil {
		t.Fatalf("runtime snapshot slices should be JSON arrays: %+v", snapshot)
	}
}

func TestHandlerReviewsAssociationAndOverlaysSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 24, 1, 10, 0, 0, time.UTC)
	provider := associationReviewSnapshotProvider{snapshot: Snapshot{
		GeneratedAt: now.Add(-1 * time.Minute),
		Associations: []Association{{
			ID:               "c360.edge.cop.fusion.association.mavlink-to-tak",
			Label:            "Candidate association UAS 42 -> ANDROID-ALPHA",
			Kind:             "track",
			Source:           "fusion",
			Status:           "associated",
			PrimaryTrackID:   "c360.edge.cop.mavlink.track.system-42",
			CandidateTrackID: "c360.edge.cop.tak.track.android-alpha",
			Confidence:       0.91,
			UpdatedAt:        now.Add(-2 * time.Minute),
		}},
	}}
	handler, err := NewHandler(provider, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/associations/c360.edge.cop.fusion.association.mavlink-to-tak/review",
		strings.NewReader(`{"decision":"challenged","reviewed_by":"operator:lead","comment":"TAK point is stale"}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var review AssociationReview
	if err := json.Unmarshal(rec.Body.Bytes(), &review); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if review.Decision != AssociationReviewChallenged ||
		review.ReviewedBy != "operator:lead" ||
		review.ReviewedAt != now ||
		review.ReviewerRole != DefaultAssociationReviewerRole ||
		review.AuthorityScope != DefaultAssociationReviewAuthorityScope ||
		review.ConflictPolicy != DefaultAssociationReviewConflictPolicy ||
		review.Comment != "TAK point is stale" {
		t.Fatalf("review = %+v", review)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/cop/snapshot", nil)
	rec = httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d, body %s", rec.Code, rec.Body.String())
	}
	var snapshot Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if len(snapshot.Associations) != 1 || snapshot.Associations[0].OperatorReview == nil {
		t.Fatalf("snapshot association review missing: %+v", snapshot.Associations)
	}
	if snapshot.Associations[0].OperatorReview.Decision != AssociationReviewChallenged ||
		snapshot.Associations[0].OperatorReview.AuthorityScope != DefaultAssociationReviewAuthorityScope {
		t.Fatalf("snapshot review = %+v", snapshot.Associations[0].OperatorReview)
	}
}

func TestHandlerReviewUsesOperatorIdentityHeader(t *testing.T) {
	now := time.Date(2026, 6, 24, 1, 12, 0, 0, time.UTC)
	handler, err := NewHandler(
		associationReviewSnapshotProvider{snapshot: Snapshot{
			Associations: []Association{{ID: "association-1"}},
		}},
		WithClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/associations/association-1/review",
		strings.NewReader(`{"decision":"acknowledged","reviewed_by":"operator:body"}`),
	)
	req.Header.Set(OperatorIDHeader, "operator:incident-command")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var review AssociationReview
	if err := json.Unmarshal(rec.Body.Bytes(), &review); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if review.ReviewedBy != "operator:incident-command" ||
		review.ReviewerRole != DefaultAssociationReviewerRole ||
		review.AuthorityScope != DefaultAssociationReviewAuthorityScope {
		t.Fatalf("review = %+v", review)
	}
}

func TestHandlerRejectsReviewAuthorityEscalation(t *testing.T) {
	handler, err := NewHandler(associationReviewSnapshotProvider{snapshot: Snapshot{
		Associations: []Association{{ID: "association-1"}},
	}})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/associations/association-1/review",
		strings.NewReader(`{"decision":"acknowledged","reviewed_by":"operator:lead"}`),
	)
	req.Header.Set(OperatorAuthorityScopeHeader, "command.execute")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "display-only reviews") {
		t.Fatalf("body = %s, want display-only rejection", rec.Body.String())
	}
}

func TestHandlerRejectsInvalidAssociationReviewDecision(t *testing.T) {
	handler, err := NewHandler(associationReviewSnapshotProvider{snapshot: Snapshot{
		Associations: []Association{{ID: "association-1"}},
	}})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/associations/association-1/review",
		strings.NewReader(`{"decision":"merged"}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRejectsReviewForUnknownAssociation(t *testing.T) {
	handler, err := NewHandler(associationReviewSnapshotProvider{snapshot: Snapshot{
		Associations: []Association{{ID: "association-1"}},
	}})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/associations/association-2/review",
		strings.NewReader(`{"decision":"acknowledged"}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerHealthz(t *testing.T) {
	handler, err := NewHandler(NewFixtureProvider(nil))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

type failingProvider struct{}

func (failingProvider) Snapshot(context.Context) (Snapshot, error) {
	return Snapshot{}, errors.New("snapshot unavailable")
}

type associationReviewSnapshotProvider struct {
	snapshot Snapshot
}

func (p associationReviewSnapshotProvider) Snapshot(context.Context) (Snapshot, error) {
	return p.snapshot, nil
}
