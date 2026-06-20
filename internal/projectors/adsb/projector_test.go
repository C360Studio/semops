package adsb

import (
	"testing"
	"time"

	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectorCreatesADSBTrackWithoutAssociationEdges(t *testing.T) {
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "scenario-adsb-001",
	})

	plan, err := projector.ProjectState(sampleState(), "opensky://states/a1b2c3/1781965785")
	if err != nil {
		t.Fatalf("project ADS-B state: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	if create.Entity.ID != "c360.edge.cop.adsb.track.a1b2c3" {
		t.Fatalf("track id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.ADSBTrackContract().MessageType {
		t.Fatalf("track message type = %q", create.Entity.MessageType.Key())
	}
	if create.Entity.UpdatedAt != sampleObservedAt() {
		t.Fatalf("track updated_at = %s, want %s", create.Entity.UpdatedAt, sampleObservedAt())
	}
	if create.IndexingProfile != cop.ADSBTrackContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.OwnerToken != "semops.feed.adsb#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "scenario-adsb-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	if hasPredicate(create.Triples, cop.TrackSource) {
		t.Fatal("ADS-B feed projection must not emit association/source foreign edges")
	}

	requireTriple(t, create.Triples, cop.TrackNativeID, "adsb.icao24.a1b2c3.callsign.n123ab.source.ads-b")
	requireTriple(t, create.Triples, cop.TrackStatus, "active.aircraft")
	requireTriple(t, create.Triples, cop.TrackObservedAt, sampleObservedAt())
	requireTriple(t, create.Triples, cop.TrackPosition, "POINT(-77.0400000 38.9000000)")
	requireTriple(t, create.Triples, cop.TrackVelocity, "AIR_MOTION_MPS(71.50 180.25 -1.20)")
	requireTriple(t, create.Triples, cop.ProvenanceSource, "adsb")
	requireTriple(t, create.Triples, cop.ProvenanceConfidence, 0.85)
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "opensky://states/a1b2c3/1781965785")
}

func TestProjectorUpdatesKnownTrackWithoutRebirth(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	state := sampleState()

	birthPlan, err := projector.ProjectState(state, "opensky://states/a1b2c3/1781965785")
	if err != nil {
		t.Fatalf("project first ADS-B state: %v", err)
	}
	if marked := projector.MarkBornForPlan(birthPlan); marked != 1 {
		t.Fatalf("marked births = %d, want track", marked)
	}

	lat, lon := 38.901, -77.041
	velocity := 75.0
	state.Latitude = &lat
	state.Longitude = &lon
	state.VelocityMPS = &velocity
	state.TimePosition = ptrTime(sampleObservedAt().Add(10 * time.Second))
	state.LastContact = sampleObservedAt().Add(15 * time.Second)
	plan, err := projector.ProjectState(state, "opensky://states/a1b2c3/1781965795")
	if err != nil {
		t.Fatalf("project second ADS-B state: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track update", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.adsb.track.a1b2c3" {
		t.Fatalf("update entity id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.ADSBTrackContract().IndexingProfile {
		t.Fatalf("update indexing profile = %q", update.IndexingProfile)
	}
	if update.OwnerToken != "semops.feed.adsb#test" {
		t.Fatalf("update owner token = %q", update.OwnerToken)
	}
	if hasPredicate(update.AddTriples, cop.TrackSource) {
		t.Fatal("ADS-B updates must not emit association/source foreign edges")
	}
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0410000 38.9010000)")
	requireTriple(t, update.AddTriples, cop.TrackVelocity, "AIR_MOTION_MPS(75.00 180.25 -1.20)")
}

func TestProjectorPreservesMissingPositionWithoutFakeCoordinates(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	state := sampleState()
	state.Latitude = nil
	state.Longitude = nil
	state.TimePosition = nil
	state.PositionSource = 0
	state.HasPositionSource = false

	plan, err := projector.ProjectState(state, "opensky://states/a1b2c3/no-position")
	if err != nil {
		t.Fatalf("project missing-position ADS-B state: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want partial track birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	if hasPredicate(create.Triples, cop.TrackPosition) {
		t.Fatalf("missing-position state emitted fake position: %+v", create.Triples)
	}
	requireTriple(t, create.Triples, cop.TrackNativeID, "adsb.icao24.a1b2c3.callsign.n123ab.source.unknown")
	requireTriple(t, create.Triples, cop.ProvenanceConfidence, 0.5)
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "opensky://states/a1b2c3/no-position")
}

func TestProjectorCanSeedBornStateForRestartReconciliation(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	state := sampleState()

	if !projector.MarkBornForState(state, "c360.edge.cop.adsb.track.a1b2c3") {
		t.Fatal("expected track born-state seed")
	}
	plan, err := projector.ProjectState(state, "opensky://states/a1b2c3/after-restart")
	if err != nil {
		t.Fatalf("project after born seed: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track update only", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.adsb.track.a1b2c3" {
		t.Fatalf("update entity id = %q", update.Entity.ID)
	}
	if hasPredicate(update.AddTriples, cop.TrackSource) {
		t.Fatal("seeded ADS-B update must not emit association/source foreign edges")
	}
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0400000 38.9000000)")
}

func sampleState() adsbcodec.StateVector {
	callsign := "N123AB"
	lat, lon := 38.9, -77.04
	velocity, track, verticalRate := 71.5, 180.25, -1.2
	observed := sampleObservedAt()
	return adsbcodec.StateVector{
		ICAO24:            "A1B2C3",
		Callsign:          &callsign,
		OriginCountry:     "United States",
		TimePosition:      &observed,
		LastContact:       observed.Add(7 * time.Second),
		Longitude:         &lon,
		Latitude:          &lat,
		OnGround:          false,
		VelocityMPS:       &velocity,
		TrueTrackDeg:      &track,
		VerticalRateMPS:   &verticalRate,
		PositionSource:    adsbcodec.PositionSourceADSB,
		HasPositionSource: true,
	}
}

func sampleObservedAt() time.Time {
	return time.Date(2026, 6, 20, 14, 29, 45, 0, time.UTC)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func requireCreate(t *testing.T, mutation Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	if mutation.Create.Entity == nil {
		t.Fatal("create entity is nil")
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationUpdate {
		t.Fatalf("mutation kind = %q, want update", mutation.Kind)
	}
	if mutation.Update.Entity == nil {
		t.Fatal("update entity is nil")
	}
	return mutation.Update
}

func requireTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate {
			if triple.Object != want {
				t.Fatalf("%s object = %#v, want %#v", predicate, triple.Object, want)
			}
			return
		}
	}
	t.Fatalf("missing predicate %q in %+v", predicate, triples)
}

func hasPredicate(triples []message.Triple, predicate string) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return true
		}
	}
	return false
}

func testOwnerTokens(incarnation string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerADSB: ownership.ExpectedOwnerToken(cop.OwnerADSB, incarnation),
	}
}
