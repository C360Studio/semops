package fusion

import (
	"strings"
	"testing"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	"github.com/c360studio/semops/pkg/cop"
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
