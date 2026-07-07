package cop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	semlinkingress "github.com/c360studio/semops/internal/ingress/semlink"
	commandprojector "github.com/c360studio/semops/internal/projectors/command"
	"github.com/c360studio/semstreams/graph"
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
		review.AuthorityDomain != DefaultAssociationReviewAuthorityDomain ||
		review.ConflictPolicy != DefaultAssociationReviewConflictPolicy ||
		review.ConflictState != DefaultAssociationReviewConflictState ||
		review.Authenticated ||
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
		snapshot.Associations[0].OperatorReview.AuthorityScope != DefaultAssociationReviewAuthorityScope ||
		snapshot.Associations[0].OperatorReview.AuthorityDomain != DefaultAssociationReviewAuthorityDomain {
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
		review.AuthorityScope != DefaultAssociationReviewAuthorityScope ||
		review.AuthorityDomain != DefaultAssociationReviewAuthorityDomain {
		t.Fatalf("review = %+v", review)
	}
}

func TestHandlerTrustedIdentityArbitratesConflictingAuthorityReviews(t *testing.T) {
	now := time.Date(2026, 6, 24, 1, 14, 0, 0, time.UTC)
	handler, err := NewHandler(
		associationReviewSnapshotProvider{snapshot: Snapshot{
			Associations: []Association{{ID: "association-1"}},
		}},
		WithClock(func() time.Time { return now }),
		WithOperatorIdentityResolver(ResolveTrustedHeaderOperatorIdentity),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	postReview := func(domain, decision string) AssociationReview {
		t.Helper()
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/cop/associations/association-1/review",
			strings.NewReader(`{"decision":"`+decision+`"}`),
		)
		req.Header.Set(OperatorAuthenticatedHeader, "true")
		req.Header.Set(OperatorIDHeader, "operator:"+domain)
		req.Header.Set(OperatorRoleHeader, AuthenticatedAssociationReviewerRole)
		req.Header.Set(OperatorAuthorityScopeHeader, AuthenticatedAssociationReviewScope)
		req.Header.Set(OperatorAuthorityDomainHeader, domain)
		rec := httptest.NewRecorder()
		handler.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
		var review AssociationReview
		if err := json.Unmarshal(rec.Body.Bytes(), &review); err != nil {
			t.Fatalf("decode review: %v", err)
		}
		return review
	}

	first := postReview("incident-command", AssociationReviewAcknowledged)
	if !first.Authenticated ||
		first.Decision != AssociationReviewAcknowledged ||
		first.AuthorityDomain != "incident-command" ||
		first.ConflictState != DefaultAssociationReviewConflictState {
		t.Fatalf("first review = %+v", first)
	}
	second := postReview("airspace-control", AssociationReviewChallenged)
	if second.Decision != AssociationReviewConflictBlocked ||
		second.ConflictState != AssociationReviewConflictBlocked ||
		second.AuthorityDomain != "airspace-control,incident-command" ||
		second.ConflictPolicy != AuthenticatedAssociationReviewPolicy ||
		!second.Authenticated {
		t.Fatalf("second review = %+v, want blocked conflict", second)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/snapshot", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d, body %s", rec.Code, rec.Body.String())
	}
	var snapshot Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.Associations[0].OperatorReview == nil ||
		snapshot.Associations[0].OperatorReview.Decision != AssociationReviewConflictBlocked {
		t.Fatalf("snapshot review = %+v", snapshot.Associations[0].OperatorReview)
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

func TestHandlerAdmitsSemLinkReadbackIntentAndWritesPlan(t *testing.T) {
	entityID := "c360.edge.cop.command.task.semlink-blue-boat-autopilot-version"
	ingress := &fakeSemLinkReadbackIngress{
		result: semlinkingress.Result{
			ClaimScope: semlinkingress.ClaimScopeCompanionIntentOnly,
			Intent: commandprojector.Intent{
				NativeID:      "semlink-blue-boat-autopilot-version",
				TargetAssetID: "c360.edge.cop.mavlink.asset.system-42",
				SourceRef:     "semlink://blue-boat/ardupilot/system-42/request-autopilot-version",
			},
			Admission: commandprojector.AdmissionResult{Accepted: true},
		},
		plan: commandprojector.Plan{Mutations: []commandprojector.Mutation{{
			Kind: commandprojector.MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{ID: entityID},
			},
		}}},
	}
	writer := &recordingCommandPlanWriter{}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, writer),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{
			"mesh_node_id":"blue-boat",
			"id":"autopilot-version",
			"target_asset_id":"c360.edge.cop.mavlink.asset.system-42",
			"vehicle_system_id":42,
			"vehicle_component_id":1,
			"action":"request_autopilot_version",
			"correlation_id":"corr-42",
			"idempotency_key":"idem-42",
			"source_ref":"semlink://blue-boat/ardupilot/system-42/request-autopilot-version",
			"observed_at":"2026-07-07T13:45:00Z",
			"ttl_seconds":45
		}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(ingress.requests) != 1 {
		t.Fatalf("ingress requests = %d, want 1", len(ingress.requests))
	}
	gotRequest := ingress.requests[0]
	if gotRequest.CompanionNodeID != "blue-boat" ||
		gotRequest.ID != "autopilot-version" ||
		gotRequest.TargetAssetID != "c360.edge.cop.mavlink.asset.system-42" ||
		gotRequest.TargetSystemID != 42 ||
		gotRequest.TargetComponentID != 1 ||
		gotRequest.TTL != 45*time.Second {
		t.Fatalf("ingress request = %+v", gotRequest)
	}
	if len(writer.plans) != 1 || len(writer.plans[0].Mutations) != 1 {
		t.Fatalf("writer plans = %+v, want one mutation plan", writer.plans)
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Accepted ||
		response.ClaimScope != semlinkingress.ClaimScopeCompanionIntentOnly ||
		response.EntityID != entityID ||
		response.NativeID != "semlink-blue-boat-autopilot-version" ||
		response.Mutations != 1 ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerAdmitsSemLinkReadbackV0FixtureThroughIngress(t *testing.T) {
	now := time.Date(2026, 7, 7, 18, 30, 1, 0, time.UTC)
	writer := &recordingCommandPlanWriter{}
	handler := newSemLinkReadbackV0Handler(t, now, writer, "c360.edge.cop.mavlink.asset.system-42")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(readSemLinkContractFixture(t, "request.accepted.json")),
	)
	setTrustedSemLinkReadbackHeadersForNode(req, "blue-boat-01")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(writer.plans) != 1 || len(writer.plans[0].Mutations) != 1 {
		t.Fatalf("writer plans = %+v, want one command-intent write", writer.plans)
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Contract != semLinkReadbackContractV0 ||
		!response.Accepted ||
		response.Status != "accepted" ||
		response.Duplicate ||
		response.CorrelationID != "corr-blue-boat-01-autopilot-version-001" ||
		response.IdempotencyKey != "idem-blue-boat-01-autopilot-version-001" ||
		response.CompanionNodeID != "blue-boat-01" ||
		response.AuthorizedCompanionNodeID != "blue-boat-01" ||
		response.AuthorityScope != SemLinkReadbackAuthorityScope ||
		response.TargetAssetID != "c360.edge.cop.mavlink.asset.system-42" ||
		response.EntityID != "c360.edge.cop.command.task.semlink-blue-boat-01-autopilot-version" ||
		response.NativeID != "semlink-blue-boat-01-autopilot-version" ||
		response.ClaimScope != semlinkingress.ClaimScopeCompanionIntentOnly ||
		response.RequestedAt != time.Date(2026, 7, 7, 18, 30, 0, 0, time.UTC) ||
		response.ExpiresAt != time.Date(2026, 7, 7, 18, 30, 30, 0, time.UTC) ||
		response.Mutations != 1 ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerCollapsesDuplicateSemLinkReadbackV0FixtureBeforeSecondWrite(t *testing.T) {
	now := time.Date(2026, 7, 7, 18, 30, 1, 0, time.UTC)
	writer := &recordingCommandPlanWriter{}
	handler := newSemLinkReadbackV0Handler(t, now, writer, "c360.edge.cop.mavlink.asset.system-42")

	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/cop/semlink/ardupilot/readback",
			strings.NewReader(readSemLinkContractFixture(t, "request.accepted.json")),
		)
		setTrustedSemLinkReadbackHeadersForNode(req, "blue-boat-01")
		rec := httptest.NewRecorder()
		handler.Routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("attempt %d status = %d, body %s", attempt+1, rec.Code, rec.Body.String())
		}
		if attempt == 0 {
			continue
		}
		var response semLinkReadbackResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode duplicate response: %v", err)
		}
		if response.Accepted ||
			response.Status != "duplicate" ||
			!response.Duplicate ||
			response.ExistingNativeID != "semlink-blue-boat-01-autopilot-version" ||
			response.Mutations != 0 ||
			response.NativeExecutionAllowed ||
			response.CompanionTransmitAllowed {
			t.Fatalf("duplicate response = %+v", response)
		}
	}
	if len(writer.plans) != 1 {
		t.Fatalf("writer plans = %d, want only the first accepted write", len(writer.plans))
	}
}

func TestHandlerRejectsStaleSemLinkReadbackV0FixtureBeforeWrite(t *testing.T) {
	now := time.Date(2026, 7, 7, 18, 30, 1, 0, time.UTC)
	writer := &recordingCommandPlanWriter{}
	handler := newSemLinkReadbackV0Handler(t, now, writer, "c360.edge.cop.mavlink.asset.system-42")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(readSemLinkContractFixture(t, "request.expired.json")),
	)
	setTrustedSemLinkReadbackHeadersForNode(req, "blue-boat-01")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none for stale readback", writer.plans)
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Accepted ||
		response.Status != "rejected" ||
		response.RejectedReason != "expired request" ||
		response.Mutations != 0 ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerRejectsUnsupportedSemLinkReadbackV0MessageBeforeIngressWrite(t *testing.T) {
	now := time.Date(2026, 7, 7, 18, 30, 1, 0, time.UTC)
	writer := &recordingCommandPlanWriter{}
	handler := newSemLinkReadbackV0Handler(t, now, writer, "c360.edge.cop.mavlink.asset.system-42")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(readSemLinkContractFixture(t, "request.rejected-unsupported-message.json")),
	)
	setTrustedSemLinkReadbackHeadersForNode(req, "blue-boat-01")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none for unsupported message", writer.plans)
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "rejected" ||
		!strings.Contains(response.RejectedReason, "unsupported requested_message_id 76") ||
		response.Mutations != 0 ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerRejectsSemLinkReadbackV0FixtureForUnbornTargetBeforeWrite(t *testing.T) {
	now := time.Date(2026, 7, 7, 18, 30, 1, 0, time.UTC)
	writer := &recordingCommandPlanWriter{}
	handler := newSemLinkReadbackV0Handler(t, now, writer)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(readSemLinkContractFixture(t, "request.accepted.json")),
	)
	setTrustedSemLinkReadbackHeadersForNode(req, "blue-boat-01")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none for unborn target", writer.plans)
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Accepted ||
		response.Status != "rejected" ||
		response.RejectedReason != "command target asset is not born" ||
		response.Mutations != 0 ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerSemLinkReadbackV0RejectsMissingAuthorityScope(t *testing.T) {
	now := time.Date(2026, 7, 7, 18, 30, 1, 0, time.UTC)
	writer := &recordingCommandPlanWriter{}
	handler := newSemLinkReadbackV0Handler(t, now, writer, "c360.edge.cop.mavlink.asset.system-42")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(readSemLinkContractFixture(t, "request.accepted.json")),
	)
	req.Header.Set(OperatorAuthenticatedHeader, "true")
	req.Header.Set(OperatorIDHeader, "operator:semlink-gateway")
	req.Header.Set(OperatorRoleHeader, SemLinkReadbackOperatorRole)
	req.Header.Set(OperatorAuthorityDomainHeader, "boat-blue")
	req.Header.Set(SemLinkCompanionNodeIDHeader, "blue-boat-01")
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none before authorized admission", writer.plans)
	}
}

func TestHandlerDoesNotWriteRejectedSemLinkReadbackIntent(t *testing.T) {
	ingress := &fakeSemLinkReadbackIngress{
		result: semlinkingress.Result{
			ClaimScope: semlinkingress.ClaimScopeCompanionIntentOnly,
			Intent: commandprojector.Intent{
				NativeID:      "semlink-blue-boat-autopilot-version",
				TargetAssetID: "c360.edge.cop.mavlink.asset.system-42",
			},
			Admission: commandprojector.AdmissionResult{
				RejectedReason: "command target asset is not born",
			},
		},
	}
	writer := &recordingCommandPlanWriter{}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, writer),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{
			"mesh_node_id":"blue-boat",
			"target_asset_id":"c360.edge.cop.mavlink.asset.system-42",
			"vehicle_system_id":42,
			"correlation_id":"corr-42",
			"idempotency_key":"idem-42"
		}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none for rejected admission", writer.plans)
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Accepted ||
		response.RejectedReason != "command target asset is not born" ||
		response.Mutations != 0 ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerReportsSemLinkReadbackWriterFailureWithoutTransmitAuthority(t *testing.T) {
	ingress := &fakeSemLinkReadbackIngress{
		result: semlinkingress.Result{
			ClaimScope: semlinkingress.ClaimScopeCompanionIntentOnly,
			Intent: commandprojector.Intent{
				NativeID:      "semlink-blue-boat-autopilot-version",
				TargetAssetID: "c360.edge.cop.mavlink.asset.system-42",
			},
			Admission: commandprojector.AdmissionResult{Accepted: true},
		},
		plan: commandprojector.Plan{Mutations: []commandprojector.Mutation{{
			Kind: commandprojector.MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{ID: "c360.edge.cop.command.task.semlink-blue-boat-autopilot-version"},
			},
		}}},
	}
	writer := &recordingCommandPlanWriter{err: errors.New("graph offline")}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, writer),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{
			"mesh_node_id":"blue-boat",
			"target_asset_id":"c360.edge.cop.mavlink.asset.system-42",
			"vehicle_system_id":42,
			"correlation_id":"corr-42",
			"idempotency_key":"idem-42"
		}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Accepted ||
		!strings.Contains(response.Error, "persist semlink readback intent") ||
		response.NativeExecutionAllowed ||
		response.CompanionTransmitAllowed {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandlerSemLinkReadbackAuthorizerRejectsUnauthenticatedCaller(t *testing.T) {
	ingress := &fakeSemLinkReadbackIngress{}
	writer := &recordingCommandPlanWriter{}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, writer),
		WithSemLinkReadbackAuthorizer(RequireTrustedSemLinkReadbackHeaders),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{"mesh_node_id":"blue-boat"}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(ingress.requests) != 0 {
		t.Fatalf("ingress requests = %+v, want none before auth", ingress.requests)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none before auth", writer.plans)
	}
}

func TestHandlerSemLinkReadbackAuthorizerAnnotatesAcceptedResponse(t *testing.T) {
	ingress := &fakeSemLinkReadbackIngress{
		result: semlinkingress.Result{
			ClaimScope: semlinkingress.ClaimScopeCompanionIntentOnly,
			Intent: commandprojector.Intent{
				NativeID:      "semlink-blue-boat-autopilot-version",
				TargetAssetID: "c360.edge.cop.mavlink.asset.system-42",
			},
			Admission: commandprojector.AdmissionResult{Accepted: true},
		},
		plan: commandprojector.Plan{Mutations: []commandprojector.Mutation{{
			Kind: commandprojector.MutationCreate,
			Create: graph.CreateEntityWithTriplesRequest{
				Entity: &graph.EntityState{ID: "c360.edge.cop.command.task.semlink-blue-boat-autopilot-version"},
			},
		}}},
	}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, &recordingCommandPlanWriter{}),
		WithSemLinkReadbackAuthorizer(RequireTrustedSemLinkReadbackHeaders),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{
			"mesh_node_id":"blue-boat",
			"target_asset_id":"c360.edge.cop.mavlink.asset.system-42",
			"vehicle_system_id":42,
			"correlation_id":"corr-42",
			"idempotency_key":"idem-42"
		}`),
	)
	setTrustedSemLinkReadbackHeaders(req)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var response semLinkReadbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AuthorizedBy != "operator:semlink-gateway" ||
		response.AuthorizedCompanionNodeID != "blue-boat" ||
		response.AuthorizedMeshNodeID != "blue-boat" ||
		response.AuthorityScope != SemLinkReadbackAuthorityScope ||
		response.AuthorityDomain != "boat-blue" ||
		!response.Authenticated {
		t.Fatalf("response caller = %+v", response)
	}
}

func TestHandlerSemLinkReadbackRejectsMeshNodeMismatch(t *testing.T) {
	ingress := &fakeSemLinkReadbackIngress{}
	writer := &recordingCommandPlanWriter{}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, writer),
		WithSemLinkReadbackAuthorizer(RequireTrustedSemLinkReadbackHeaders),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{
			"mesh_node_id":"other-boat",
			"target_asset_id":"c360.edge.cop.mavlink.asset.system-42",
			"vehicle_system_id":42,
			"correlation_id":"corr-42",
			"idempotency_key":"idem-42"
		}`),
	)
	setTrustedSemLinkReadbackHeaders(req)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(ingress.requests) != 0 {
		t.Fatalf("ingress requests = %+v, want none for mesh mismatch", ingress.requests)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("writer plans = %+v, want none for mesh mismatch", writer.plans)
	}
}

func TestHandlerSemLinkReadbackRouteFailsClosedWhenUnconfigured(t *testing.T) {
	handler, err := NewHandler(NewFixtureProvider(nil))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{"mesh_node_id":"blue-boat"}`),
	)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
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

type fakeSemLinkReadbackIngress struct {
	result   semlinkingress.Result
	plan     commandprojector.Plan
	err      error
	requests []semlinkingress.ArduPilotReadbackRequest
}

func (f *fakeSemLinkReadbackIngress) AdmitArduPilotReadback(
	_ context.Context,
	request semlinkingress.ArduPilotReadbackRequest,
) (semlinkingress.Result, commandprojector.Plan, error) {
	f.requests = append(f.requests, request)
	return f.result, f.plan, f.err
}

type recordingCommandPlanWriter struct {
	plans []commandprojector.Plan
	err   error
}

func (w *recordingCommandPlanWriter) Apply(_ context.Context, plan commandprojector.Plan) error {
	w.plans = append(w.plans, plan)
	return w.err
}

func setTrustedSemLinkReadbackHeaders(req *http.Request) {
	setTrustedSemLinkReadbackHeadersForNode(req, "blue-boat")
}

func setTrustedSemLinkReadbackHeadersForNode(req *http.Request, nodeID string) {
	req.Header.Set(OperatorAuthenticatedHeader, "true")
	req.Header.Set(OperatorIDHeader, "operator:semlink-gateway")
	req.Header.Set(OperatorRoleHeader, SemLinkReadbackOperatorRole)
	req.Header.Set(OperatorAuthorityScopeHeader, SemLinkReadbackAuthorityScope)
	req.Header.Set(OperatorAuthorityDomainHeader, "boat-blue")
	req.Header.Set(SemLinkCompanionNodeIDHeader, nodeID)
}

func readSemLinkContractFixture(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile("../../../testdata/contracts/semlink-companion-readback-v0/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(body)
}

func newSemLinkReadbackV0Handler(
	t *testing.T,
	now time.Time,
	writer *recordingCommandPlanWriter,
	targetIDs ...string,
) *Handler {
	t.Helper()
	if writer == nil {
		writer = &recordingCommandPlanWriter{}
	}
	ingress := semlinkingress.Ingress{
		Projector: commandprojector.NewGuardedProjector(
			commandprojector.NewProjector(commandprojector.Config{}),
			commandprojector.AdmissionConfig{
				Clock:          func() time.Time { return now },
				TargetResolver: commandprojector.NewStaticTargetResolver(targetIDs...),
			},
		),
		Clock: func() time.Time { return now },
	}
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithSemLinkReadbackIngress(ingress, writer),
		WithSemLinkReadbackAuthorizer(RequireTrustedSemLinkReadbackHeaders),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return handler
}
