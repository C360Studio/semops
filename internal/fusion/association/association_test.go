package association

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semstreams/message"
)

func TestAssociateTracksEmitsFusionEvidenceForCloseMAVLinkAndADSB(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	results := Associate(
		[]TrackObservation{{
			ID:         "c360.edge.cop.mavlink.track.system-42",
			Source:     "mavlink",
			NativeID:   "mavlink.system.42.component.1",
			Position:   GeoPoint{Lat: 30.2672, Lon: -97.7431},
			ObservedAt: observed,
			Confidence: 0.96,
			SourceRef:  "mavlink://raw/udp/0001",
		}},
		[]TrackObservation{{
			ID:         "c360.edge.cop.adsb.track.a1b2c3",
			Source:     "adsb",
			NativeID:   "adsb.icao24.a1b2c3.callsign.n42cx.source.ads-b",
			Position:   GeoPoint{Lat: 30.2674, Lon: -97.7429},
			ObservedAt: observed.Add(2 * time.Second),
			Confidence: 0.88,
			SourceRef:  "adsb://opensky/state/0001",
		}},
		Config{Org: "c360", Platform: "edge-demo"},
	)

	if len(results) != 1 {
		t.Fatalf("associations = %d, want 1: %+v", len(results), results)
	}
	got := results[0]
	if got.Status != "associated" {
		t.Fatalf("status = %q, want associated", got.Status)
	}
	if got.Algorithm != DefaultAlgorithm {
		t.Fatalf("algorithm = %q, want %q", got.Algorithm, DefaultAlgorithm)
	}
	if got.Confidence < 0.85 {
		t.Fatalf("confidence = %.3f, want >= 0.85", got.Confidence)
	}
	if got.DistanceMeters <= 0 || got.DistanceMeters > 40 {
		t.Fatalf("distance = %.1f, want small positive distance", got.DistanceMeters)
	}
	if got.TimeDelta != 2*time.Second {
		t.Fatalf("time delta = %s, want 2s", got.TimeDelta)
	}
	if got.ObservedAt != observed.Add(2*time.Second) {
		t.Fatalf("observed = %s, want latest source observation", got.ObservedAt)
	}
	if !strings.HasPrefix(got.EntityID, "c360.edge-demo.cop.fusion.association.") {
		t.Fatalf("entity ID = %q, want fusion association ID", got.EntityID)
	}
	parsedEntityID, err := message.ParseEntityID(got.EntityID)
	if err != nil {
		t.Fatalf("fusion association entity ID should remain SemStreams-parseable: %v", err)
	}
	if parsedEntityID.System != "fusion" || parsedEntityID.Type != "association" {
		t.Fatalf("parsed entity ID = %+v, want fusion association", parsedEntityID)
	}
	if strings.Contains(parsedEntityID.Instance, ".") {
		t.Fatalf("association instance = %q, want single entity-id segment", parsedEntityID.Instance)
	}
	if got.PrimarySourceRef != "mavlink://raw/udp/0001" || got.CandidateSourceRef != "adsb://opensky/state/0001" {
		t.Fatalf("source refs not preserved: %+v", got)
	}
	if !hasReasonPrefix(got.Reasons, "sources=mavlink,adsb") {
		t.Fatalf("reasons missing source pair: %+v", got.Reasons)
	}
}

func TestAssociateTracksMarksAmbiguousWhenCandidatesAreClose(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	results := Associate(
		[]TrackObservation{track("mav", "mavlink", 30.2672, -97.7431, observed, 0.9)},
		[]TrackObservation{
			track("adsb-a", "adsb", 30.2673, -97.7430, observed.Add(time.Second), 0.9),
			track("adsb-b", "adsb", 30.26735, -97.74295, observed.Add(time.Second), 0.9),
		},
		Config{AmbiguityMargin: 0.10},
	)

	if len(results) != 1 {
		t.Fatalf("associations = %d, want 1", len(results))
	}
	if results[0].Status != "ambiguous" {
		t.Fatalf("status = %q, want ambiguous: %+v", results[0].Status, results[0])
	}
	if !hasReasonPrefix(results[0].Reasons, "ambiguous_with=") {
		t.Fatalf("reasons missing ambiguity evidence: %+v", results[0].Reasons)
	}
}

func TestAssociateTracksUsesSourcePriorityAsTieBreaker(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	results := Associate(
		[]TrackObservation{track("mav", "mavlink", 30.2672, -97.7431, observed, 0.9)},
		[]TrackObservation{
			track("adsb", "adsb", 30.2673, -97.7430, observed, 0.9),
			track("tak", "tak", 30.2673, -97.7430, observed, 0.9),
		},
		Config{
			SourcePriority:  []string{"tak", "adsb"},
			AmbiguityMargin: 0.01,
		},
	)

	if len(results) != 1 {
		t.Fatalf("associations = %d, want 1", len(results))
	}
	if results[0].CandidateTrackID != "c360.edge.cop.tak.track.tak" {
		t.Fatalf("candidate = %q, want TAK priority winner", results[0].CandidateTrackID)
	}
}

func TestAssociateTracksRejectsObservationsOutsideStaleWindow(t *testing.T) {
	now := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	results := Associate(
		[]TrackObservation{track("mav", "mavlink", 30.2672, -97.7431, now.Add(-5*time.Second), 0.9)},
		[]TrackObservation{
			track("stale", "adsb", 30.26721, -97.74311, now.Add(-20*time.Second), 0.9),
		},
		Config{
			ReferenceTime:     now,
			MaxObservationAge: 10 * time.Second,
			MaxTimeDelta:      time.Minute,
		},
	)

	if len(results) != 0 {
		t.Fatalf("associations = %+v, want stale candidate filtered", results)
	}
}

func TestAssociateTracksRejectsFarOrStaleCandidates(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	results := Associate(
		[]TrackObservation{track("mav", "mavlink", 30.2672, -97.7431, observed, 0.9)},
		[]TrackObservation{
			track("far", "adsb", 31.2672, -97.7431, observed, 0.9),
			track("stale", "adsb", 30.26721, -97.74311, observed.Add(-time.Minute), 0.9),
		},
		Config{},
	)

	if len(results) != 0 {
		t.Fatalf("associations = %+v, want none", results)
	}
}

func TestAssociateTracksDoesNotAssociateSameSource(t *testing.T) {
	observed := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	results := Associate(
		[]TrackObservation{track("mav-a", "mavlink", 30.2672, -97.7431, observed, 0.9)},
		[]TrackObservation{track("mav-b", "mavlink", 30.26721, -97.74311, observed.Add(time.Second), 0.9)},
		Config{},
	)

	if len(results) != 0 {
		t.Fatalf("associations = %+v, want no same-source association", results)
	}
}

func track(id, source string, lat, lon float64, observedAt time.Time, confidence float64) TrackObservation {
	return TrackObservation{
		ID:         "c360.edge.cop." + source + ".track." + id,
		Source:     source,
		NativeID:   source + "." + id,
		Position:   GeoPoint{Lat: lat, Lon: lon},
		ObservedAt: observedAt,
		Confidence: confidence,
		SourceRef:  source + "://fixture/" + id,
	}
}

func hasReasonPrefix(reasons []string, prefix string) bool {
	for _, reason := range reasons {
		if strings.HasPrefix(reason, prefix) {
			return true
		}
	}
	return false
}
