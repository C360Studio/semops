package fusion

import (
	"strings"
	"testing"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectAssociationCreatesBornFirstFusionEvidence(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 30, 0, 0, time.UTC)
	projector := NewProjector(Config{
		OwnerTokens: map[string]ownership.OwnerToken{
			cop.OwnerFusion: ownership.ExpectedOwnerToken(cop.OwnerFusion, "test"),
		},
		TraceID: "trace-fusion",
	})

	evidence := sampleEvidence(observed)
	plan, err := projector.ProjectAssociation(evidence)
	if err != nil {
		t.Fatalf("project association: %v", err)
	}
	if len(plan.Mutations) != 1 || plan.Mutations[0].Kind != MutationCreate {
		t.Fatalf("plan = %+v, want one create", plan)
	}

	create := plan.Mutations[0].Create
	if create.Entity == nil || create.Entity.ID != evidence.EntityID {
		t.Fatalf("create entity = %+v, want %s", create.Entity, evidence.EntityID)
	}
	if create.Entity.MessageType.String() != cop.FusionTrackAssociationContract().MessageType {
		t.Fatalf("message type = %s, want %s", create.Entity.MessageType.String(), cop.FusionTrackAssociationContract().MessageType)
	}
	if create.IndexingProfile != "control" {
		t.Fatalf("indexing profile = %q, want control", create.IndexingProfile)
	}
	if create.OwnerToken != cop.OwnerFusion+"#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	requireTriple(t, create.Triples, cop.AssociationKind, "track")
	requireTriple(t, create.Triples, cop.AssociationStatus, "ambiguous")
	requireTriple(t, create.Triples, cop.AssociationPrimaryTrack, evidence.PrimaryTrackID)
	requireTriple(t, create.Triples, cop.AssociationCandidateTrack, evidence.CandidateTrackID)
	requireTriple(t, create.Triples, cop.AssociationAlgorithm, evidence.Algorithm)
	requireTriple(t, create.Triples, cop.AssociationDistanceMeters, evidence.DistanceMeters)
	requireTriple(t, create.Triples, cop.AssociationTimeDeltaSeconds, evidence.TimeDelta.Seconds())
	requireTriple(t, create.Triples, cop.ProvenanceSource, "fusion.track_association")
	sourceRef := tripleValue(t, create.Triples, cop.ProvenanceSourceRef).(string)
	if !strings.Contains(sourceRef, evidence.PrimarySourceRef) || !strings.Contains(sourceRef, evidence.CandidateSourceRef) {
		t.Fatalf("source ref = %q, want both source refs", sourceRef)
	}
}

func TestProjectAssociationUpdatesWithoutRepeatingStrictTrackEdges(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 30, 0, 0, time.UTC)
	projector := NewProjector(Config{})
	evidence := sampleEvidence(observed)

	createPlan, err := projector.ProjectAssociation(evidence)
	if err != nil {
		t.Fatalf("create association: %v", err)
	}
	projector.MarkBornForPlan(createPlan)

	evidence.Status = "associated"
	evidence.Confidence = 0.91
	updatePlan, err := projector.ProjectAssociation(evidence)
	if err != nil {
		t.Fatalf("update association: %v", err)
	}
	if len(updatePlan.Mutations) != 1 || updatePlan.Mutations[0].Kind != MutationUpdate {
		t.Fatalf("plan = %+v, want one update", updatePlan)
	}
	update := updatePlan.Mutations[0].Update
	requireTriple(t, update.AddTriples, cop.AssociationStatus, "associated")
	requireTriple(t, update.AddTriples, cop.AssociationConfidence, 0.91)
	for _, triple := range update.AddTriples {
		if triple.Predicate == cop.AssociationPrimaryTrack || triple.Predicate == cop.AssociationCandidateTrack {
			t.Fatalf("update repeated strict association edge: %+v", triple)
		}
	}
}

func TestProjectAssociationRejectsMalformedEvidence(t *testing.T) {
	projector := NewProjector(Config{})
	evidence := sampleEvidence(time.Date(2026, 6, 23, 15, 30, 0, 0, time.UTC))
	evidence.CandidateTrackID = ""

	if _, err := projector.ProjectAssociation(evidence); err == nil || !strings.Contains(err.Error(), "candidate track") {
		t.Fatalf("error = %v, want candidate track validation", err)
	}
}

func TestProjectAssociationReviewCreatesBornFirstAudit(t *testing.T) {
	review := sampleReview(time.Date(2026, 6, 24, 1, 35, 0, 0, time.UTC))
	projector := NewProjector(Config{
		OwnerTokens: map[string]ownership.OwnerToken{
			cop.OwnerFusion: ownership.ExpectedOwnerToken(cop.OwnerFusion, "lease-123"),
		},
		TraceID: "trace-review",
	})

	plan, err := projector.ProjectAssociationReview(review)
	if err != nil {
		t.Fatalf("project association review: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want review birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	wantID := AssociationReviewEntityID(review.Org, review.Platform, review.AssociationID)
	if create.Entity.ID != wantID {
		t.Fatalf("review entity id = %q, want %q", create.Entity.ID, wantID)
	}
	if create.Entity.MessageType.String() != cop.FusionAssociationReviewContract().MessageType {
		t.Fatalf("message type = %s, want %s", create.Entity.MessageType.String(), cop.FusionAssociationReviewContract().MessageType)
	}
	if create.OwnerToken != "semops.fusion.structural#lease-123" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	requireTriple(t, create.Triples, cop.AssociationReviewAssociation, review.AssociationID)
	requireTriple(t, create.Triples, cop.AssociationReviewDecision, "challenged")
	requireTriple(t, create.Triples, cop.AssociationReviewReviewedBy, "operator:lead")
	requireTriple(t, create.Triples, cop.AssociationReviewReviewedAt, review.ReviewedAt)
	requireTriple(t, create.Triples, cop.AssociationReviewReviewerRole, cop.AssociationReviewerRoleUnverified)
	requireTriple(t, create.Triples, cop.AssociationReviewAuthorityScope, cop.AssociationReviewScopeDisplayOnly)
	requireTriple(t, create.Triples, cop.AssociationReviewConflictPolicy, cop.AssociationReviewConflictLatestDisplayOnly)
	requireTriple(t, create.Triples, cop.AssociationReviewComment, "TAK point stale")
	requireTriple(t, create.Triples, cop.ProvenanceSource, "operator.association_review")
}

func TestProjectAssociationReviewUpdatesWithoutRepeatingStrictAssociationEdge(t *testing.T) {
	review := sampleReview(time.Date(2026, 6, 24, 1, 40, 0, 0, time.UTC))
	projector := NewProjector(Config{})
	createPlan, err := projector.ProjectAssociationReview(review)
	if err != nil {
		t.Fatalf("create association review: %v", err)
	}
	create := requireCreate(t, createPlan.Mutations[0])
	projector.MarkBornForAssociationReview(review, create.Entity.ID)

	review.Decision = "acknowledged"
	review.Comment = ""
	updatePlan, err := projector.ProjectAssociationReview(review)
	if err != nil {
		t.Fatalf("update association review: %v", err)
	}
	if len(updatePlan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want review update", len(updatePlan.Mutations))
	}
	update := requireUpdate(t, updatePlan.Mutations[0])
	requireTriple(t, update.AddTriples, cop.AssociationReviewDecision, "acknowledged")
	requireTriple(t, update.AddTriples, cop.AssociationReviewAuthorityScope, cop.AssociationReviewScopeDisplayOnly)
	requireTriple(t, update.AddTriples, cop.AssociationReviewComment, "")
	for _, triple := range update.AddTriples {
		if triple.Predicate == cop.AssociationReviewAssociation {
			t.Fatalf("update repeated strict association edge: %+v", triple)
		}
	}
}

func TestProjectAssociationReviewRejectsMalformedEvidence(t *testing.T) {
	projector := NewProjector(Config{})
	review := sampleReview(time.Date(2026, 6, 24, 1, 45, 0, 0, time.UTC))
	review.AssociationID = ""

	if _, err := projector.ProjectAssociationReview(review); err == nil || !strings.Contains(err.Error(), "association id") {
		t.Fatalf("error = %v, want association id validation", err)
	}
}

func sampleEvidence(observed time.Time) fusionassociation.Evidence {
	return fusionassociation.Evidence{
		EntityID:           "c360.edge.cop.fusion.association.mavlink-to-adsb",
		PrimaryTrackID:     "c360.edge.cop.mavlink.track.system-42",
		CandidateTrackID:   "c360.edge.cop.adsb.track.a1b2c3",
		Status:             "ambiguous",
		Confidence:         0.875,
		Algorithm:          fusionassociation.DefaultAlgorithm,
		DistanceMeters:     31,
		TimeDelta:          2 * time.Second,
		ObservedAt:         observed,
		Reasons:            []string{"sources=mavlink,adsb", "distance_meters=31", "ambiguous_with=candidate-b"},
		PrimarySourceRef:   "mavlink://raw/udp/0001",
		CandidateSourceRef: "adsb://opensky/state/0001",
	}
}

func sampleReview(reviewedAt time.Time) AssociationReviewEvidence {
	return AssociationReviewEvidence{
		Org:            "c360",
		Platform:       "edge",
		AssociationID:  "c360.edge.cop.fusion.association.mavlink-to-tak",
		Decision:       "challenged",
		ReviewedBy:     "operator:lead",
		ReviewedAt:     reviewedAt,
		ReviewerRole:   cop.AssociationReviewerRoleUnverified,
		AuthorityScope: cop.AssociationReviewScopeDisplayOnly,
		ConflictPolicy: cop.AssociationReviewConflictLatestDisplayOnly,
		Comment:        "TAK point stale",
	}
}

func requireCreate(t *testing.T, mutation Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationUpdate {
		t.Fatalf("mutation kind = %q, want update", mutation.Kind)
	}
	return mutation.Update
}

func requireTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
	t.Helper()
	got := tripleValue(t, triples, predicate)
	if got != want {
		t.Fatalf("%s = %#v, want %#v", predicate, got, want)
	}
}

func tripleValue(t *testing.T, triples []message.Triple, predicate string) any {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return triple.Object
		}
	}
	t.Fatalf("missing triple %s in %+v", predicate, triples)
	return nil
}
