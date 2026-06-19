package cot

import (
	"testing"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectorBirthsSourceAssetBeforeTAKTrack(t *testing.T) {
	event := cotcodec.SeedEvents(seedTime())[0]
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "scenario-tak-001",
	})

	plan, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000001")
	if err != nil {
		t.Fatalf("project operator event: %v", err)
	}
	if len(plan.Mutations) != 2 {
		t.Fatalf("mutations = %d, want asset birth + track birth", len(plan.Mutations))
	}

	assetCreate := requireCreate(t, plan.Mutations[0])
	trackCreate := requireCreate(t, plan.Mutations[1])

	if assetCreate.Entity.ID != "c360.edge.cop.tak.asset.android-alpha" {
		t.Fatalf("asset id = %q", assetCreate.Entity.ID)
	}
	if assetCreate.Entity.MessageType.Key() != cop.SourceAssetContract().MessageType {
		t.Fatalf("asset message type = %q", assetCreate.Entity.MessageType.Key())
	}
	if assetCreate.IndexingProfile != cop.SourceAssetContract().IndexingProfile {
		t.Fatalf("asset indexing profile = %q", assetCreate.IndexingProfile)
	}
	if assetCreate.OwnerToken != "semops.feed.asset#test" {
		t.Fatalf("asset owner token = %q", assetCreate.OwnerToken)
	}
	if hasPredicate(assetCreate.Triples, cop.TrackSource) {
		t.Fatal("source asset birth must not emit track-source foreign edges")
	}
	requireTriple(t, assetCreate.Triples, cop.AssetName, "ALPHA")
	requireTriple(t, assetCreate.Triples, cop.AssetNativeID, "cot.uid.ANDROID-ALPHA")
	requireTriple(t, assetCreate.Triples, cop.ProvenanceSourceRef, "cot://raw/tak-unit/00000001")

	if trackCreate.Entity.ID != "c360.edge.cop.tak.track.android-alpha" {
		t.Fatalf("track id = %q", trackCreate.Entity.ID)
	}
	if trackCreate.Entity.MessageType.Key() != cop.TAKTrackContract().MessageType {
		t.Fatalf("track message type = %q", trackCreate.Entity.MessageType.Key())
	}
	if trackCreate.IndexingProfile != cop.TAKTrackContract().IndexingProfile {
		t.Fatalf("track indexing profile = %q", trackCreate.IndexingProfile)
	}
	if trackCreate.OwnerToken != "semops.feed.tak#test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	requireTriple(t, trackCreate.Triples, cop.TrackSource, assetCreate.Entity.ID)
	requireTriple(t, trackCreate.Triples, cop.TrackNativeID, "cot.uid.ANDROID-ALPHA")
	requireTriple(t, trackCreate.Triples, cop.TrackStatus, "active.operator")
	requireTriple(t, trackCreate.Triples, cop.TrackPosition, "POINT(-77.0350000 38.8920000)")
	requireTriple(t, trackCreate.Triples, cop.ProvenanceSourceRef, "cot://raw/tak-unit/00000001")
}

func TestProjectorUpdatesKnownTrackWithoutRebirth(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	event := cotcodec.Event{
		UID:       "AIRCRAFT-1",
		Type:      cotcodec.TypeAirTrack,
		Time:      seedTime(),
		Stale:     seedTime().Add(2 * time.Minute),
		Point:     &cotcodec.Point{Lat: 38.91, Lon: -77.02},
		Callsign:  "EAGLE-1",
		HasTrack:  true,
		CourseDeg: 271.456,
		SpeedMPS:  17.234,
	}

	birthPlan, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000010")
	if err != nil {
		t.Fatalf("project first air track: %v", err)
	}
	if marked := projector.MarkBornForPlan(birthPlan); marked != 2 {
		t.Fatalf("marked births = %d, want asset + track", marked)
	}

	event.Point = &cotcodec.Point{Lat: 38.92, Lon: -77.03}
	event.CourseDeg = 90
	event.SpeedMPS = 12.5
	plan, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000011")
	if err != nil {
		t.Fatalf("project second air track: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track update only", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.tak.track.aircraft-1" {
		t.Fatalf("update entity id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.TAKTrackContract().IndexingProfile {
		t.Fatalf("update indexing profile = %q", update.IndexingProfile)
	}
	if update.OwnerToken != "semops.feed.tak#test" {
		t.Fatalf("update owner token = %q", update.OwnerToken)
	}
	if hasPredicate(update.AddTriples, cop.TrackSource) {
		t.Fatal("track updates must not re-emit source foreign edge after born-first create")
	}
	requireTriple(t, update.AddTriples, cop.TrackStatus, "active.air-track")
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0300000 38.9200000)")
	requireTriple(t, update.AddTriples, cop.TrackVelocity, "COURSE_SPEED_MPS(90.00 12.50)")
}

func TestProjectorProjectsMarkersToControlAndGeoChatToContent(t *testing.T) {
	events := cotcodec.SeedEvents(seedTime())
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})

	plan, err := projector.ProjectEvents([]SourceEvent{
		{Event: events[2], SourceRef: "cot://raw/tak-unit/00000003"},
		{Event: events[3], SourceRef: "cot://raw/tak-unit/00000004"},
	})
	if err != nil {
		t.Fatalf("project marker/chat: %v", err)
	}
	if len(plan.Mutations) != 2 {
		t.Fatalf("mutations = %d, want task create + advisory create", len(plan.Mutations))
	}

	taskCreate := requireCreate(t, plan.Mutations[0])
	if taskCreate.Entity.ID != "c360.edge.cop.tak.task.marker-north-gate" {
		t.Fatalf("task id = %q", taskCreate.Entity.ID)
	}
	if taskCreate.Entity.MessageType.Key() != cop.TAKTaskContract().MessageType {
		t.Fatalf("task message type = %q", taskCreate.Entity.MessageType.Key())
	}
	if taskCreate.IndexingProfile != "control" {
		t.Fatalf("task indexing profile = %q", taskCreate.IndexingProfile)
	}
	if taskCreate.OwnerToken != "semops.feed.tak#test" {
		t.Fatalf("task owner token = %q", taskCreate.OwnerToken)
	}
	requireTriple(t, taskCreate.Triples, cop.TaskName, "North Gate")
	requireTriple(t, taskCreate.Triples, cop.TaskKind, "marker")
	requireTriple(t, taskCreate.Triples, cop.TaskDescription, "checkpoint")
	requireTriple(t, taskCreate.Triples, cop.TaskPosition, "POINT(-77.0380000 38.8940000)")
	requireTriple(t, taskCreate.Triples, cop.ProvenanceSourceRef, "cot://raw/tak-unit/00000003")
	if hasPredicate(taskCreate.Triples, cop.TrackSource) {
		t.Fatal("TAK task projection must not emit track-source edges")
	}

	advisoryCreate := requireCreate(t, plan.Mutations[1])
	if advisoryCreate.Entity.ID != "c360.edge.cop.tak.advisory.chat-alpha-1" {
		t.Fatalf("advisory id = %q", advisoryCreate.Entity.ID)
	}
	if advisoryCreate.Entity.MessageType.Key() != cop.TAKAdvisoryContract().MessageType {
		t.Fatalf("advisory message type = %q", advisoryCreate.Entity.MessageType.Key())
	}
	if advisoryCreate.IndexingProfile != "content" {
		t.Fatalf("advisory indexing profile = %q", advisoryCreate.IndexingProfile)
	}
	if advisoryCreate.OwnerToken != "semops.feed.tak#test" {
		t.Fatalf("advisory owner token = %q", advisoryCreate.OwnerToken)
	}
	requireTriple(t, advisoryCreate.Triples, cop.AdvisoryText, "hold at checkpoint")
	requireTriple(t, advisoryCreate.Triples, cop.AdvisoryKind, "geochat")
	requireTriple(t, advisoryCreate.Triples, cop.AdvisorySender, "ANDROID-ALPHA")
	requireTriple(t, advisoryCreate.Triples, cop.AdvisoryPosition, "POINT(-77.0350000 38.8920000)")
	requireTriple(t, advisoryCreate.Triples, cop.ProvenanceSourceRef, "cot://raw/tak-unit/00000004")
	if hasPredicate(advisoryCreate.Triples, cop.TrackSource) {
		t.Fatal("TAK advisory projection must not emit track-source edges")
	}
}

func TestProjectorDoesNotCommitBirthStateUntilMarked(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	event := cotcodec.SeedEvents(seedTime())[0]

	first, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000001")
	if err != nil {
		t.Fatalf("project first event: %v", err)
	}
	second, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000001")
	if err != nil {
		t.Fatalf("project second event: %v", err)
	}
	if len(first.Mutations) != 2 || len(second.Mutations) != 2 {
		t.Fatalf("unmarked projections = %d/%d mutations, want repeated birth proposals", len(first.Mutations), len(second.Mutations))
	}

	if marked := projector.MarkBornForPlan(first); marked != 2 {
		t.Fatalf("marked births = %d, want asset + track", marked)
	}
	third, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000002")
	if err != nil {
		t.Fatalf("project third event: %v", err)
	}
	if len(third.Mutations) != 1 {
		t.Fatalf("marked projection mutations = %d, want track update", len(third.Mutations))
	}
	if third.Mutations[0].Kind != MutationUpdate {
		t.Fatalf("marked projection kind = %q, want update", third.Mutations[0].Kind)
	}
}

func TestProjectorCanSeedBornStateForRestartReconciliation(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	event := cotcodec.SeedEvents(seedTime())[0]

	if !projector.MarkBornForEvent(event, "c360.edge.cop.tak.asset.android-alpha") {
		t.Fatal("expected asset born-state seed")
	}
	plan, err := projector.ProjectEvent(event, "cot://raw/tak-unit/00000001")
	if err != nil {
		t.Fatalf("project with seeded asset: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track birth only", len(plan.Mutations))
	}
	trackCreate := requireCreate(t, plan.Mutations[0])
	requireTriple(t, trackCreate.Triples, cop.TrackSource, "c360.edge.cop.tak.asset.android-alpha")

	if !projector.MarkBornForEvent(event, "c360.edge.cop.tak.track.android-alpha") {
		t.Fatal("expected track born-state seed")
	}
	plan, err = projector.ProjectEvent(event, "cot://raw/tak-unit/00000002")
	if err != nil {
		t.Fatalf("project with seeded track: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track update only", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if hasPredicate(update.AddTriples, cop.TrackSource) {
		t.Fatal("update after seeded birth must not repeat strict source edge")
	}
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-77.0350000 38.8920000)")
}

func TestProjectorIgnoresUnsupportedEventsWithoutBirth(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	plan, err := projector.ProjectEvent(cotcodec.Event{
		UID:  "ALERT-1",
		Type: cotcodec.TypeAlert,
		Time: seedTime(),
	}, "cot://raw/tak-unit/00000099")
	if err != nil {
		t.Fatalf("project unsupported alert: %v", err)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want no-op", len(plan.Mutations))
	}
}

func seedTime() time.Time {
	return time.Date(2026, 6, 19, 14, 20, 0, 0, time.UTC)
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
		cop.OwnerAsset: ownership.ExpectedOwnerToken(cop.OwnerAsset, incarnation),
		cop.OwnerTAK:   ownership.ExpectedOwnerToken(cop.OwnerTAK, incarnation),
	}
}
