package cop

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	copmodel "github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

func TestGraphProviderMapsMAVLinkEntities(t *testing.T) {
	now := time.Date(2026, 6, 19, 13, 0, 0, 0, time.UTC)
	observed := now.Add(-12 * time.Second)
	assetID := "c360.edge-compose.cop.mavlink.asset.system-42"
	trackID := "c360.edge-compose.cop.mavlink.track.system-42"
	requester := &fakeGraphSnapshotRequester{
		entities: map[string]graph.EntityState{
			assetID: {
				ID:        assetID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(assetID, copmodel.AssetName, "MAVLink system 42", observed),
					testTriple(assetID, copmodel.AssetKind, "mavlink-system", observed),
					testTriple(assetID, copmodel.AssetSource, "mavlink", observed),
					testTriple(assetID, copmodel.ProvenanceConfidence, 0.98, observed),
					testTriple(assetID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(assetID, copmodel.ProvenanceSourceRef, "mavlink://raw/udp/00000001", observed),
				},
			},
			trackID: {
				ID:        trackID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(trackID, copmodel.TrackNativeID, "mavlink.system.42.component.7", observed),
					testTriple(trackID, copmodel.TrackStatus, "active.armed", observed),
					testTriple(trackID, copmodel.TrackPosition, "POINT(-77.0000002 38.9000001)", observed),
					testTriple(trackID, copmodel.TrackVelocity, "NED_CMPS(321 -12 7)", observed),
					testTriple(trackID, copmodel.TrackSource, assetID, observed),
					testTriple(trackID, copmodel.TrackObservedAt, observed, observed),
					testTriple(trackID, copmodel.ProvenanceSource, "mavlink", observed),
					testTriple(trackID, copmodel.ProvenanceConfidence, 0.98, observed),
					testTriple(trackID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(trackID, copmodel.ProvenanceSourceRef, "mavlink://raw/udp/00000002", observed),
				},
			},
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithMAVLinkSystems("c360", "edge-compose", []int{42}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Scenario != "phase-1-live-graph" {
		t.Fatalf("scenario = %q", snapshot.Scenario)
	}
	if snapshot.Summary.ActiveTracks != 1 || len(snapshot.Tracks) != 1 {
		t.Fatalf("track summary/list = %d/%d", snapshot.Summary.ActiveTracks, len(snapshot.Tracks))
	}
	track := snapshot.Tracks[0]
	if track.ID != trackID || track.Label != "UAS 42" || track.Status != "active.armed" {
		t.Fatalf("track = %+v", track)
	}
	if track.Position.Lat != 38.9000001 || track.Position.Lon != -77.0000002 {
		t.Fatalf("track position = %+v", track.Position)
	}
	if track.Provenance.Owner != copmodel.OwnerMAVLink || track.Provenance.SourceRef != "mavlink://raw/udp/00000002" {
		t.Fatalf("track provenance = %+v", track.Provenance)
	}
	if len(snapshot.Assets) != 1 || snapshot.Assets[0].Position == nil {
		t.Fatalf("asset missing position copied from source track: %+v", snapshot.Assets)
	}
	if snapshot.Assets[0].Position.Lat != track.Position.Lat || snapshot.Assets[0].Position.Lon != track.Position.Lon {
		t.Fatalf("asset position = %+v, want track position %+v", snapshot.Assets[0].Position, track.Position)
	}
	if snapshot.Feeds[0].Status != "live" {
		t.Fatalf("MAVLink feed status = %q", snapshot.Feeds[0].Status)
	}
	if got := requester.subjects; len(got) != 3 ||
		got[0] != SubjectGraphQueryEntity ||
		got[1] != SubjectGraphQueryEntity ||
		got[2] != SubjectGraphQueryEntity {
		t.Fatalf("query subjects = %+v", got)
	}
}

func TestGraphProviderMapsTAKCoTEntities(t *testing.T) {
	now := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	observed := now.Add(-20 * time.Second)
	platform := "edge-cot"
	trackID := cotprojector.EntityID("c360", platform, copmodel.EntityTrack, "ANDROID-ALPHA")
	assetID := cotprojector.EntityID("c360", platform, copmodel.EntityAsset, "ANDROID-ALPHA")
	taskID := cotprojector.EntityID("c360", platform, copmodel.EntityTask, "MARKER-NORTH-GATE")
	advisoryID := cotprojector.EntityID("c360", platform, copmodel.EntityAdvisory, "CHAT-ALPHA-1")
	requester := &fakeGraphSnapshotRequester{
		entities: map[string]graph.EntityState{
			assetID: {
				ID:        assetID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(assetID, copmodel.AssetName, "ALPHA", observed),
					testTriple(assetID, copmodel.AssetKind, "tak-cot-source", observed),
					testTriple(assetID, copmodel.AssetSource, "tak-cot", observed),
					testTriple(assetID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(assetID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(assetID, copmodel.ProvenanceSourceRef, "cot://raw/tak-unit/00000001", observed),
				},
			},
			trackID: {
				ID:        trackID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(trackID, copmodel.TrackNativeID, "cot.uid.ANDROID-ALPHA", observed),
					testTriple(trackID, copmodel.TrackStatus, "active.operator", observed),
					testTriple(trackID, copmodel.TrackPosition, "POINT(-77.0350000 38.8920000)", observed),
					testTriple(trackID, copmodel.TrackSource, assetID, observed),
					testTriple(trackID, copmodel.TrackObservedAt, observed, observed),
					testTriple(trackID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(trackID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(trackID, copmodel.ProvenanceSourceRef, "cot://raw/tak-unit/00000001", observed),
				},
			},
			taskID: {
				ID:        taskID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(taskID, copmodel.TaskNativeID, "cot.uid.MARKER-NORTH-GATE", observed),
					testTriple(taskID, copmodel.TaskName, "North Gate", observed),
					testTriple(taskID, copmodel.TaskKind, "marker", observed),
					testTriple(taskID, copmodel.TaskStatus, "active.marker", observed),
					testTriple(taskID, copmodel.TaskPosition, "POINT(-77.0380000 38.8940000)", observed),
					testTriple(taskID, copmodel.TaskDescription, "checkpoint", observed),
					testTriple(taskID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(taskID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(taskID, copmodel.ProvenanceSourceRef, "cot://raw/tak-unit/00000003", observed),
				},
			},
			advisoryID: {
				ID:        advisoryID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(advisoryID, copmodel.AdvisoryNativeID, "cot.uid.CHAT-ALPHA-1", observed),
					testTriple(advisoryID, copmodel.AdvisoryText, "hold at checkpoint", observed),
					testTriple(advisoryID, copmodel.AdvisoryKind, "geochat", observed),
					testTriple(advisoryID, copmodel.AdvisoryStatus, "active.geochat", observed),
					testTriple(advisoryID, copmodel.AdvisorySender, "ANDROID-ALPHA", observed),
					testTriple(advisoryID, copmodel.AdvisoryPosition, "POINT(-77.0350000 38.8920000)", observed),
					testTriple(advisoryID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(advisoryID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(advisoryID, copmodel.ProvenanceSourceRef, "cot://raw/tak-unit/00000004", observed),
				},
			},
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithCoTUIDs("c360", platform, []string{"ANDROID-ALPHA", "MARKER-NORTH-GATE", "CHAT-ALPHA-1"}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Summary.ActiveTracks != 1 || snapshot.Summary.ActiveTasks != 1 ||
		snapshot.Summary.ActiveAdvisories != 1 {
		t.Fatalf("summary = %+v", snapshot.Summary)
	}
	if got := snapshot.Tracks[0]; got.ID != trackID || got.Label != "ANDROID-ALPHA" ||
		got.Source != "tak-cot" || got.Provenance.Owner != copmodel.OwnerTAK {
		t.Fatalf("track = %+v", got)
	}
	if got := snapshot.Tasks[0]; got.ID != taskID || got.Label != "North Gate" ||
		got.Description != "checkpoint" || got.Position == nil || got.Position.Lat != 38.894 {
		t.Fatalf("task = %+v", got)
	}
	if got := snapshot.Advisories[0]; got.ID != advisoryID || got.Text != "hold at checkpoint" ||
		got.Sender != "ANDROID-ALPHA" || got.Position == nil {
		t.Fatalf("advisory = %+v", got)
	}
	if snapshot.Feeds[1].ID != "feed.tak" || snapshot.Feeds[1].Status != "live" {
		t.Fatalf("TAK feed = %+v", snapshot.Feeds[1])
	}
}

func TestGraphProviderMapsCAPHazardEvidence(t *testing.T) {
	now := time.Date(2026, 6, 19, 16, 0, 0, 0, time.UTC)
	observed := now.Add(-30 * time.Second)
	platform := "edge-cap"
	identifier := "nws-demo-flood-warning"
	hazardID := capprojector.EntityID("c360", platform, identifier)
	evidence := copmodel.HazardEvidenceDocument{
		Identifier:  identifier,
		MessageType: "Alert",
		Status:      "Actual",
		Event:       "Flood Warning",
		Urgency:     "Immediate",
		Severity:    "Severe",
		Certainty:   "Likely",
		AreaDesc:    "River Corridor",
		Sender:      "w-nws.webmaster@noaa.gov",
		SenderName:  "NWS Demo",
		Sent:        observed.Format(time.RFC3339Nano),
		Polygons: [][]copmodel.HazardEvidencePoint{{
			{Lat: 38.8900, Lon: -77.0500},
			{Lat: 38.9050, Lon: -77.0440},
			{Lat: 38.9030, Lon: -77.0200},
			{Lat: 38.8860, Lon: -77.0280},
		}},
	}
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	requester := &fakeGraphSnapshotRequester{
		entities: map[string]graph.EntityState{
			hazardID: {
				ID:        hazardID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(hazardID, copmodel.HazardAdvisoryText, "Flood Warning for River Corridor", observed),
					testTriple(hazardID, copmodel.HazardEvidence, string(evidenceJSON), observed),
					testTriple(hazardID, copmodel.HazardSource, "cap", observed),
					testTriple(hazardID, copmodel.ProvenanceSource, "cap", observed),
					testTriple(hazardID, copmodel.ProvenanceConfidence, 0.82, observed),
					testTriple(hazardID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(hazardID, copmodel.ProvenanceSourceRef, "cap://nws/demo/flood-warning", observed),
				},
			},
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithCAPAlertIDs("c360", platform, []string{identifier}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if len(snapshot.Hazards) != 1 {
		t.Fatalf("hazards = %+v", snapshot.Hazards)
	}
	hazard := snapshot.Hazards[0]
	if hazard.ID != hazardID ||
		hazard.Label != "Flood Warning: River Corridor" ||
		hazard.Kind != "cap-flood-warning" ||
		hazard.Severity != "severe" ||
		hazard.Source != "cap" {
		t.Fatalf("hazard = %+v", hazard)
	}
	if len(hazard.Geometry) != 4 || hazard.Geometry[0].Lat != 38.8900 || hazard.Geometry[0].Lon != -77.0500 {
		t.Fatalf("hazard geometry = %+v", hazard.Geometry)
	}
	if hazard.Provenance.Owner != copmodel.OwnerCAP ||
		hazard.Provenance.SourceRef != "cap://nws/demo/flood-warning" ||
		hazard.Provenance.Observed != observed {
		t.Fatalf("hazard provenance = %+v", hazard.Provenance)
	}
	if snapshot.Feeds[2].ID != "feed.cap" || snapshot.Feeds[2].Status != "live" {
		t.Fatalf("CAP feed = %+v", snapshot.Feeds[2])
	}
}

func TestGraphProviderDowngradesStaleTAKStateAtReadTime(t *testing.T) {
	now := time.Date(2026, 6, 19, 15, 30, 0, 0, time.UTC)
	observed := now.Add(-5 * time.Minute)
	taskID := cotprojector.EntityID("c360", "edge", copmodel.EntityTask, "MARKER-NORTH-GATE")
	requester := &fakeGraphSnapshotRequester{
		entities: map[string]graph.EntityState{
			taskID: {
				ID:        taskID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(taskID, copmodel.TaskName, "North Gate", observed),
					testTriple(taskID, copmodel.TaskKind, "marker", observed),
					testTriple(taskID, copmodel.TaskStatus, "active.marker", observed),
					testTriple(taskID, copmodel.TaskPosition, "POINT(-77.0380000 38.8940000)", observed),
					testTriple(taskID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(taskID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			},
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithFeedFreshnessWindow(2*time.Minute),
		WithCoTUIDs("c360", "edge", []string{"MARKER-NORTH-GATE"}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snapshot.Tasks) != 1 || snapshot.Tasks[0].Status != "stale.marker" {
		t.Fatalf("tasks = %+v", snapshot.Tasks)
	}
	if snapshot.Feeds[1].Status != "stale" || snapshot.Feeds[1].LastEventAt != observed {
		t.Fatalf("TAK feed = %+v", snapshot.Feeds[1])
	}
}

func TestGraphProviderFallsBackWhenNoGraphEntitiesExist(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 0, 0, 0, time.UTC)
	provider, err := NewGraphProvider(
		legacyNotFoundRequester{},
		WithGraphNow(func() time.Time { return now }),
		WithGraphFallback(staticSnapshotProvider{snapshot: Snapshot{
			GeneratedAt: now,
			Scenario:    "fixture-fallback",
			Summary:     Summary{ActiveTracks: 1},
		}}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.Scenario != "fixture-fallback" {
		t.Fatalf("scenario = %q", snapshot.Scenario)
	}
}

type fakeGraphSnapshotRequester struct {
	entities map[string]graph.EntityState
	subjects []string
}

func (r *fakeGraphSnapshotRequester) Request(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	return r.request(ctx, subject, data, timeout)
}

func (r *fakeGraphSnapshotRequester) RequestClassified(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	return r.request(ctx, subject, data, timeout)
}

func (r *fakeGraphSnapshotRequester) request(
	_ context.Context,
	subject string,
	data []byte,
	_ time.Duration,
) ([]byte, error) {
	r.subjects = append(r.subjects, subject)
	var req map[string]string
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	entity, ok := r.entities[req["id"]]
	if !ok {
		return nil, errors.New("not found: " + req["id"])
	}
	return json.Marshal(entity)
}

type legacyNotFoundRequester struct{}

func (legacyNotFoundRequester) Request(
	context.Context,
	string,
	[]byte,
	time.Duration,
) ([]byte, error) {
	return []byte("error: not found: no such entity"), nil
}

type staticSnapshotProvider struct {
	snapshot Snapshot
}

func (p staticSnapshotProvider) Snapshot(context.Context) (Snapshot, error) {
	return p.snapshot, nil
}

func testTriple(subject, predicate string, object any, observed time.Time) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "test",
		Timestamp:  observed,
		Confidence: 0.98,
	}
}
