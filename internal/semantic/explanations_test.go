package semantic_test

import (
	"context"
	"strings"
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
	"github.com/c360studio/semops/internal/semantic"
)

func TestBuildTrackTranslationCarriesTaskEvidenceAndTrajectory(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	snapshot, err := copapi.NewFixtureProvider(func() time.Time { return now }).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("build fixture snapshot: %v", err)
	}

	set := semantic.Build(snapshot)

	if got, want := set.Algorithm, semantic.AlgorithmDeterministicV1; got != want {
		t.Fatalf("set algorithm = %q, want %q", got, want)
	}
	if set.ClaimPosture != semantic.ClaimPostureReadOnly {
		t.Fatalf("set claim posture = %q", set.ClaimPosture)
	}

	explanation := findExplanation(t, set.Items, semantic.KindTrackTranslation, "c360.edge.cop.mavlink.track.system-42")
	if got, want := explanation.TrajectoryRef, "semops://semantic/trajectory/phase-1-fixture/semantic-track-translation/c360-edge-cop-mavlink-track-system-42"; got != want {
		t.Fatalf("trajectory ref = %q, want %q", got, want)
	}
	if explanation.Task.Kind != semantic.TaskTrackTranslation {
		t.Fatalf("task kind = %q", explanation.Task.Kind)
	}
	if got, want := explanation.Task.InputRef, "cop://snapshot/phase-1-fixture/tracks/c360.edge.cop.mavlink.track.system-42"; got != want {
		t.Fatalf("task input ref = %q, want %q", got, want)
	}
	if explanation.Task.Prompt == "" {
		t.Fatalf("task did not record prompt/input ref: %+v", explanation.Task)
	}
	for _, needle := range []string{
		"UAS 42",
		"active.armed",
		"38.9001, -77.0002",
		"raw:mavlink:fixture:0002",
		"NED_CMPS(321 -12 7)",
		"Track freshness nominal",
	} {
		if !strings.Contains(explanation.Output, needle) {
			t.Fatalf("track output missing %q: %s", needle, explanation.Output)
		}
	}
	assertHasEvidenceRole(t, explanation.Evidence, "source_track")
	assertHasEvidenceRole(t, explanation.Evidence, "alert")
}

func TestBuildAssociationAnomalyPreservesSourceTrackEvidence(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	snapshot, err := copapi.NewFixtureProvider(func() time.Time { return now }).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("build fixture snapshot: %v", err)
	}
	distance := 42.0
	delta := 1.75
	snapshot.Associations = append(snapshot.Associations, copapi.Association{
		ID:               "c360.edge.cop.fusion.association.mavlink-system-42-to-tak-alpha",
		Label:            "UAS 42 / ANDROID-ALPHA candidate",
		Kind:             "track-to-track",
		Source:           "fusion",
		Status:           "candidate.ambiguous",
		PrimaryTrackID:   "c360.edge.cop.mavlink.track.system-42",
		CandidateTrackID: "c360.edge.cop.tak.track.android-alpha",
		Algorithm:        "semops.association.geotemporal.v1",
		Reason:           "close geotemporal match but no identity authority",
		DistanceMeters:   &distance,
		TimeDeltaSeconds: &delta,
		ClaimPosture:     "candidate only; no identity merge; operator review required",
		Confidence:       0.62,
		UpdatedAt:        now.Add(-10 * time.Second),
		Provenance: copapi.Provenance{
			Owner:     "semops.fusion.structural",
			SourceRef: "fusion://association/fixture/0001",
			Observed:  now.Add(-10 * time.Second),
		},
	})

	set := semantic.Build(snapshot)

	explanation := findExplanation(
		t,
		set.Items,
		semantic.KindAssociationAnomaly,
		"c360.edge.cop.fusion.association.mavlink-system-42-to-tak-alpha",
	)
	if explanation.Severity != "watch" {
		t.Fatalf("severity = %q, want watch", explanation.Severity)
	}
	if explanation.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", explanation.Status)
	}
	if explanation.Task.Kind != semantic.TaskAssociationAnomaly {
		t.Fatalf("task kind = %q", explanation.Task.Kind)
	}
	for _, needle := range []string{
		"UAS 42 / ANDROID-ALPHA candidate",
		"confidence 0.62",
		"distance 42 m",
		"time delta 1.75 s",
		"no identity authority",
		"Source tracks remain separate.",
	} {
		if !strings.Contains(explanation.Output, needle) {
			t.Fatalf("association output missing %q: %s", needle, explanation.Output)
		}
	}
	assertHasEvidenceRole(t, explanation.Evidence, "association")
	assertHasEvidenceRole(t, explanation.Evidence, "primary_track")
	assertHasEvidenceRole(t, explanation.Evidence, "candidate_track")
}

func TestBuildEveryExplanationRecordsPromptEvidenceOutputAndTrajectory(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	snapshot, err := copapi.NewFixtureProvider(func() time.Time { return now }).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("build fixture snapshot: %v", err)
	}

	set := semantic.Build(snapshot)

	if len(set.Items) == 0 {
		t.Fatal("expected semantic explanations")
	}
	for _, explanation := range set.Items {
		if explanation.Task.Prompt == "" {
			t.Fatalf("%s missing task prompt", explanation.ID)
		}
		if explanation.Output == "" {
			t.Fatalf("%s missing output", explanation.ID)
		}
		if explanation.TrajectoryRef == "" {
			t.Fatalf("%s missing trajectory ref", explanation.ID)
		}
		if len(explanation.Evidence) == 0 {
			t.Fatalf("%s missing source evidence", explanation.ID)
		}
		if explanation.Algorithm != semantic.AlgorithmDeterministicV1 {
			t.Fatalf("%s algorithm = %q", explanation.ID, explanation.Algorithm)
		}
		if explanation.ClaimPosture != semantic.ClaimPostureReadOnly {
			t.Fatalf("%s claim posture = %q", explanation.ID, explanation.ClaimPosture)
		}
	}
}

func findExplanation(t *testing.T, explanations []semantic.Explanation, kind, entityID string) semantic.Explanation {
	t.Helper()
	for _, explanation := range explanations {
		if explanation.Kind == kind && explanation.EntityID == entityID {
			return explanation
		}
	}
	t.Fatalf("missing explanation kind=%q entity_id=%q in %+v", kind, entityID, explanations)
	return semantic.Explanation{}
}

func assertHasEvidenceRole(t *testing.T, evidence []semantic.EvidenceRef, role string) {
	t.Helper()
	for _, ref := range evidence {
		if ref.Role == role {
			if ref.EntityID == "" || ref.SourceRef == "" || ref.Owner == "" {
				t.Fatalf("evidence role %q is incomplete: %+v", role, ref)
			}
			return
		}
	}
	t.Fatalf("missing evidence role %q in %+v", role, evidence)
}
