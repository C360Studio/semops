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
		WithGraphDiscovery(false),
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
		WithGraphDiscovery(false),
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
		WithGraphDiscovery(false),
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
		hazard.Status != "active" ||
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

func TestGraphProviderMapsCAPHazardLifecycleStatus(t *testing.T) {
	now := time.Date(2026, 6, 19, 16, 0, 0, 0, time.UTC)
	hazardID := capprojector.EntityID("c360", "edge-cap", "nws-demo-flood-warning")
	tests := []struct {
		name      string
		evidence  copmodel.HazardEvidenceDocument
		observed  time.Time
		freshness time.Duration
		want      string
	}{
		{
			name: "update",
			evidence: capLifecycleEvidence(copmodel.HazardEvidenceDocument{
				MessageType: "Update",
				Status:      "Actual",
			}),
			observed:  now.Add(-30 * time.Second),
			freshness: 2 * time.Minute,
			want:      "active.update",
		},
		{
			name: "cancel",
			evidence: capLifecycleEvidence(copmodel.HazardEvidenceDocument{
				MessageType: "Cancel",
				Status:      "Actual",
			}),
			observed:  now.Add(-30 * time.Second),
			freshness: 2 * time.Minute,
			want:      "cancelled",
		},
		{
			name: "expired",
			evidence: capLifecycleEvidence(copmodel.HazardEvidenceDocument{
				MessageType: "Alert",
				Status:      "Actual",
				Expires:     now.Add(-1 * time.Minute).Format(time.RFC3339Nano),
			}),
			observed:  now.Add(-30 * time.Second),
			freshness: 2 * time.Minute,
			want:      "expired",
		},
		{
			name: "stale",
			evidence: capLifecycleEvidence(copmodel.HazardEvidenceDocument{
				MessageType: "Alert",
				Status:      "Actual",
			}),
			observed:  now.Add(-5 * time.Minute),
			freshness: 2 * time.Minute,
			want:      "stale",
		},
		{
			name: "test status",
			evidence: capLifecycleEvidence(copmodel.HazardEvidenceDocument{
				MessageType: "Alert",
				Status:      "Test",
			}),
			observed:  now.Add(-30 * time.Second),
			freshness: 2 * time.Minute,
			want:      "nonoperational.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hazard, ok := hazardFromEntity(capHazardEntity(t, hazardID, tt.observed, tt.evidence), now, tt.freshness)
			if !ok {
				t.Fatalf("hazard did not map")
			}
			if hazard.Status != tt.want {
				t.Fatalf("status = %q, want %q", hazard.Status, tt.want)
			}
		})
	}
}

func TestGraphProviderUsesLatestCAPHazardEvidence(t *testing.T) {
	now := time.Date(2026, 6, 19, 16, 0, 0, 0, time.UTC)
	older := now.Add(-90 * time.Second)
	newer := now.Add(-30 * time.Second)
	hazardID := capprojector.EntityID("c360", "edge-cap", "nws-demo-flood-warning")
	entity := capHazardEntity(
		t,
		hazardID,
		older,
		capLifecycleEvidence(copmodel.HazardEvidenceDocument{
			MessageType: "Alert",
			Status:      "Actual",
			Severity:    "Moderate",
		}),
	)
	newerEvidence := capLifecycleEvidence(copmodel.HazardEvidenceDocument{
		MessageType: "Update",
		Status:      "Actual",
		Severity:    "Severe",
	})
	newerJSON, err := json.Marshal(newerEvidence)
	if err != nil {
		t.Fatalf("marshal newer evidence: %v", err)
	}
	entity.Triples = append(entity.Triples,
		testTriple(hazardID, copmodel.HazardEvidence, string(newerJSON), newer),
		testTriple(hazardID, copmodel.ProvenanceObservedAt, newer, newer),
		testTriple(hazardID, copmodel.ProvenanceSourceRef, "cap://nws/demo/flood-update", newer),
	)

	hazard, ok := hazardFromEntity(entity, now, 2*time.Minute)
	if !ok {
		t.Fatalf("hazard did not map")
	}
	if hazard.Status != "active.update" ||
		hazard.Severity != "severe" ||
		hazard.UpdatedAt != newer ||
		hazard.Provenance.SourceRef != "cap://nws/demo/flood-update" {
		t.Fatalf("hazard = %+v", hazard)
	}
}

func TestGraphProviderDiscoversCOPEntitiesByPrefix(t *testing.T) {
	now := time.Date(2026, 6, 19, 17, 0, 0, 0, time.UTC)
	observed := now.Add(-15 * time.Second)
	platform := "edge-discover"
	mavAssetID := "c360.edge-discover.cop.mavlink.asset.system-77"
	mavTrackID := "c360.edge-discover.cop.mavlink.track.system-77"
	takTaskID := cotprojector.EntityID("c360", platform, copmodel.EntityTask, "MARKER-EVAC")
	takAdvisoryID := cotprojector.EntityID("c360", platform, copmodel.EntityAdvisory, "CHAT-EVAC")
	hazardID := capprojector.EntityID("c360", platform, "nws-demo-flood-warning")
	evidenceJSON, err := json.Marshal(copmodel.HazardEvidenceDocument{
		Identifier: "nws-demo-flood-warning",
		Event:      "Flood Warning",
		Severity:   "Severe",
		AreaDesc:   "Evacuation Zone",
		Polygons: [][]copmodel.HazardEvidencePoint{{
			{Lat: 38.8900, Lon: -77.0500},
			{Lat: 38.9050, Lon: -77.0440},
			{Lat: 38.9030, Lon: -77.0200},
			{Lat: 38.8860, Lon: -77.0280},
		}},
	})
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	requester := &fakeGraphSnapshotRequester{
		prefixEntities: map[string][]graph.EntityState{
			graphEntityPrefix("c360", platform, "mavlink", copmodel.EntityAsset): {{
				ID:        mavAssetID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(mavAssetID, copmodel.AssetName, "MAVLink system 77", observed),
					testTriple(mavAssetID, copmodel.AssetSource, "mavlink", observed),
					testTriple(mavAssetID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
			graphEntityPrefix("c360", platform, "mavlink", copmodel.EntityTrack): {{
				ID:        mavTrackID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(mavTrackID, copmodel.TrackNativeID, "mavlink.system.77.component.1", observed),
					testTriple(mavTrackID, copmodel.TrackStatus, "active.armed", observed),
					testTriple(mavTrackID, copmodel.TrackPosition, "POINT(-77.0400000 38.9000000)", observed),
					testTriple(mavTrackID, copmodel.TrackSource, mavAssetID, observed),
					testTriple(mavTrackID, copmodel.TrackObservedAt, observed, observed),
					testTriple(mavTrackID, copmodel.ProvenanceSource, "mavlink", observed),
					testTriple(mavTrackID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
			graphEntityPrefix("c360", platform, "tak", copmodel.EntityTask): {{
				ID:        takTaskID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(takTaskID, copmodel.TaskName, "Evacuation Marker", observed),
					testTriple(takTaskID, copmodel.TaskKind, "marker", observed),
					testTriple(takTaskID, copmodel.TaskStatus, "active.marker", observed),
					testTriple(takTaskID, copmodel.TaskPosition, "POINT(-77.0380000 38.8940000)", observed),
					testTriple(takTaskID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(takTaskID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
			graphEntityPrefix("c360", platform, "tak", copmodel.EntityAdvisory): {{
				ID:        takAdvisoryID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(takAdvisoryID, copmodel.AdvisoryText, "evacuate north route", observed),
					testTriple(takAdvisoryID, copmodel.AdvisoryKind, "geochat", observed),
					testTriple(takAdvisoryID, copmodel.AdvisoryStatus, "active.geochat", observed),
					testTriple(takAdvisoryID, copmodel.AdvisorySender, "ANDROID-EVAC", observed),
					testTriple(takAdvisoryID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(takAdvisoryID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
			graphEntityPrefix("c360", platform, "cap", copmodel.EntityHazardArea): {{
				ID:        hazardID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(hazardID, copmodel.HazardEvidence, string(evidenceJSON), observed),
					testTriple(hazardID, copmodel.HazardSource, "cap", observed),
					testTriple(hazardID, copmodel.ProvenanceSource, "cap", observed),
					testTriple(hazardID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithGraphDiscovery(true),
		WithGraphDiscoveryLimit(25),
		WithGraphDiscoveryScopes([]GraphDiscoveryScope{{Org: "c360", Platform: platform}}),
		WithMAVLinkSystems("c360", platform, []int{42}),
		WithCoTUIDs("c360", platform, []string{"SEED-NOT-USED"}),
		WithCAPAlertIDs("c360", platform, []string{"seed-not-used"}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Summary.ActiveTracks != 1 ||
		snapshot.Summary.ActiveTasks != 1 ||
		snapshot.Summary.ActiveAdvisories != 1 ||
		len(snapshot.Hazards) != 1 {
		t.Fatalf("snapshot summary/entities = %+v hazards=%+v", snapshot.Summary, snapshot.Hazards)
	}
	if snapshot.Tracks[0].ID != mavTrackID || snapshot.Tracks[0].Label != "UAS 77" {
		t.Fatalf("track = %+v", snapshot.Tracks[0])
	}
	if snapshot.Tasks[0].ID != takTaskID || snapshot.Advisories[0].ID != takAdvisoryID {
		t.Fatalf("tasks/advisories = %+v / %+v", snapshot.Tasks, snapshot.Advisories)
	}
	if snapshot.Hazards[0].ID != hazardID || snapshot.Hazards[0].Source != "cap" {
		t.Fatalf("hazard = %+v", snapshot.Hazards[0])
	}
	if len(requester.prefixRequests) != 7 {
		t.Fatalf("prefix requests = %+v", requester.prefixRequests)
	}
	for _, subject := range requester.subjects {
		if subject == SubjectGraphQueryEntity {
			t.Fatalf("discovery should avoid seed entity lookups, subjects = %+v", requester.subjects)
		}
	}
	if requester.prefixRequests[0].limit != 25 {
		t.Fatalf("discovery limit = %d", requester.prefixRequests[0].limit)
	}
	if len(snapshot.Diagnostics.Discovery) != 7 {
		t.Fatalf("discovery diagnostics = %+v", snapshot.Diagnostics.Discovery)
	}
	taskDiagnostic, ok := findDiscoveryDiagnostic(snapshot.Diagnostics.Discovery, "tak", copmodel.EntityTask)
	if !ok {
		t.Fatalf("missing TAK task discovery diagnostic: %+v", snapshot.Diagnostics.Discovery)
	}
	if taskDiagnostic.Org != "c360" ||
		taskDiagnostic.Platform != platform ||
		taskDiagnostic.Family != "cot" ||
		taskDiagnostic.Count != 1 ||
		taskDiagnostic.Limit != 25 ||
		taskDiagnostic.AtLimit {
		t.Fatalf("TAK task diagnostic = %+v", taskDiagnostic)
	}
	trackDiagnostic, ok := findDiscoveryDiagnostic(snapshot.Diagnostics.Discovery, "tak", copmodel.EntityTrack)
	if !ok {
		t.Fatalf("missing TAK track discovery diagnostic: %+v", snapshot.Diagnostics.Discovery)
	}
	if trackDiagnostic.Count != 0 || trackDiagnostic.AtLimit {
		t.Fatalf("TAK track diagnostic = %+v", trackDiagnostic)
	}
}

func TestGraphProviderFallsBackPerFeedWhenDiscoveryIsPartial(t *testing.T) {
	now := time.Date(2026, 6, 19, 17, 30, 0, 0, time.UTC)
	observed := now.Add(-20 * time.Second)
	platform := "edge-partial"
	mavAssetID := "c360.edge-partial.cop.mavlink.asset.system-77"
	mavTrackID := "c360.edge-partial.cop.mavlink.track.system-77"
	takTaskID := cotprojector.EntityID("c360", platform, copmodel.EntityTask, "MARKER-EVAC")
	takAdvisoryID := cotprojector.EntityID("c360", platform, copmodel.EntityAdvisory, "CHAT-EVAC")
	hazardID := capprojector.EntityID("c360", platform, "nws-demo-flood-warning")
	requester := &fakeGraphSnapshotRequester{
		prefixEntities: map[string][]graph.EntityState{
			graphEntityPrefix("c360", platform, "mavlink", copmodel.EntityAsset): {{
				ID:        mavAssetID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(mavAssetID, copmodel.AssetName, "MAVLink system 77", observed),
					testTriple(mavAssetID, copmodel.AssetSource, "mavlink", observed),
					testTriple(mavAssetID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
			graphEntityPrefix("c360", platform, "mavlink", copmodel.EntityTrack): {{
				ID:        mavTrackID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(mavTrackID, copmodel.TrackNativeID, "mavlink.system.77.component.1", observed),
					testTriple(mavTrackID, copmodel.TrackStatus, "active.armed", observed),
					testTriple(mavTrackID, copmodel.TrackPosition, "POINT(-77.0400000 38.9000000)", observed),
					testTriple(mavTrackID, copmodel.TrackSource, mavAssetID, observed),
					testTriple(mavTrackID, copmodel.TrackObservedAt, observed, observed),
					testTriple(mavTrackID, copmodel.ProvenanceSource, "mavlink", observed),
					testTriple(mavTrackID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			}},
		},
		entities: map[string]graph.EntityState{
			takTaskID: {
				ID:        takTaskID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(takTaskID, copmodel.TaskName, "Evacuation Marker", observed),
					testTriple(takTaskID, copmodel.TaskKind, "marker", observed),
					testTriple(takTaskID, copmodel.TaskStatus, "active.marker", observed),
					testTriple(takTaskID, copmodel.TaskPosition, "POINT(-77.0380000 38.8940000)", observed),
					testTriple(takTaskID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(takTaskID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			},
			takAdvisoryID: {
				ID:        takAdvisoryID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(takAdvisoryID, copmodel.AdvisoryText, "evacuate north route", observed),
					testTriple(takAdvisoryID, copmodel.AdvisoryKind, "geochat", observed),
					testTriple(takAdvisoryID, copmodel.AdvisoryStatus, "active.geochat", observed),
					testTriple(takAdvisoryID, copmodel.AdvisorySender, "ANDROID-EVAC", observed),
					testTriple(takAdvisoryID, copmodel.ProvenanceSource, "tak-cot", observed),
					testTriple(takAdvisoryID, copmodel.ProvenanceObservedAt, observed, observed),
				},
			},
			hazardID: capHazardEntity(t, hazardID, observed, capLifecycleEvidence(copmodel.HazardEvidenceDocument{
				Identifier: "nws-demo-flood-warning",
			})),
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithGraphDiscoveryScopes([]GraphDiscoveryScope{{Org: "c360", Platform: platform}}),
		WithMAVLinkSystems("c360", platform, []int{77}),
		WithCoTUIDs("c360", platform, []string{"MARKER-EVAC", "CHAT-EVAC"}),
		WithCAPAlertIDs("c360", platform, []string{"nws-demo-flood-warning"}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Summary.ActiveTracks != 1 || snapshot.Summary.ActiveTasks != 1 ||
		snapshot.Summary.ActiveAdvisories != 1 || len(snapshot.Hazards) != 1 {
		t.Fatalf("snapshot summary/entities = %+v hazards=%+v", snapshot.Summary, snapshot.Hazards)
	}
	if hasString(requester.entityRequests, mavAssetID) || hasString(requester.entityRequests, mavTrackID) {
		t.Fatalf("MAVLink discovery should avoid MAVLink seed lookups, entity requests = %+v", requester.entityRequests)
	}
	if !hasString(requester.entityRequests, takTaskID) || !hasString(requester.entityRequests, hazardID) {
		t.Fatalf("partial discovery should fall back for CoT/CAP seeds, entity requests = %+v", requester.entityRequests)
	}
}

func TestGraphProviderReportsDiscoveryLimitPressureAndErrors(t *testing.T) {
	now := time.Date(2026, 6, 19, 18, 0, 0, 0, time.UTC)
	observed := now.Add(-10 * time.Second)
	platform := "edge-pressure"
	firstTrackID := "c360.edge-pressure.cop.mavlink.track.system-41"
	secondTrackID := "c360.edge-pressure.cop.mavlink.track.system-42"
	trackPrefix := graphEntityPrefix("c360", platform, "mavlink", copmodel.EntityTrack)
	takAdvisoryPrefix := graphEntityPrefix("c360", platform, "tak", copmodel.EntityAdvisory)
	requester := &fakeGraphSnapshotRequester{
		prefixEntities: map[string][]graph.EntityState{
			trackPrefix: {
				{
					ID:        firstTrackID,
					UpdatedAt: observed,
					Triples: []message.Triple{
						testTriple(firstTrackID, copmodel.TrackNativeID, "mavlink.system.41.component.1", observed),
						testTriple(firstTrackID, copmodel.TrackStatus, "active.armed", observed),
						testTriple(firstTrackID, copmodel.TrackPosition, "POINT(-77.0400000 38.9000000)", observed),
						testTriple(firstTrackID, copmodel.TrackObservedAt, observed, observed),
						testTriple(firstTrackID, copmodel.ProvenanceSource, "mavlink", observed),
						testTriple(firstTrackID, copmodel.ProvenanceObservedAt, observed, observed),
					},
				},
				{
					ID:        secondTrackID,
					UpdatedAt: observed,
					Triples: []message.Triple{
						testTriple(secondTrackID, copmodel.TrackNativeID, "mavlink.system.42.component.1", observed),
						testTriple(secondTrackID, copmodel.TrackStatus, "active.armed", observed),
						testTriple(secondTrackID, copmodel.TrackPosition, "POINT(-77.0410000 38.9010000)", observed),
						testTriple(secondTrackID, copmodel.TrackObservedAt, observed, observed),
						testTriple(secondTrackID, copmodel.ProvenanceSource, "mavlink", observed),
						testTriple(secondTrackID, copmodel.ProvenanceObservedAt, observed, observed),
					},
				},
			},
		},
		prefixErrors: map[string]error{
			takAdvisoryPrefix: errors.New("temporary prefix index unavailable"),
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithGraphDiscoveryLimit(2),
		WithGraphDiscoveryScopes([]GraphDiscoveryScope{{Org: "c360", Platform: platform}}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	trackDiagnostic, ok := findDiscoveryDiagnostic(snapshot.Diagnostics.Discovery, "mavlink", copmodel.EntityTrack)
	if !ok {
		t.Fatalf("missing MAVLink track diagnostic: %+v", snapshot.Diagnostics.Discovery)
	}
	if trackDiagnostic.Count != 2 || trackDiagnostic.Limit != 2 || !trackDiagnostic.AtLimit {
		t.Fatalf("MAVLink track diagnostic = %+v", trackDiagnostic)
	}
	advisoryDiagnostic, ok := findDiscoveryDiagnostic(snapshot.Diagnostics.Discovery, "tak", copmodel.EntityAdvisory)
	if !ok {
		t.Fatalf("missing TAK advisory diagnostic: %+v", snapshot.Diagnostics.Discovery)
	}
	if advisoryDiagnostic.Count != 0 ||
		advisoryDiagnostic.Limit != 2 ||
		advisoryDiagnostic.AtLimit ||
		advisoryDiagnostic.Error == "" {
		t.Fatalf("TAK advisory diagnostic = %+v", advisoryDiagnostic)
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
		WithGraphDiscovery(false),
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
	entities       map[string]graph.EntityState
	prefixEntities map[string][]graph.EntityState
	prefixErrors   map[string]error
	subjects       []string
	prefixRequests []recordedPrefixRequest
	entityRequests []string
}

type recordedPrefixRequest struct {
	prefix string
	limit  int
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
	if subject == SubjectGraphQueryPrefix {
		var req struct {
			Prefix string `json:"prefix"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, err
		}
		r.prefixRequests = append(r.prefixRequests, recordedPrefixRequest{prefix: req.Prefix, limit: req.Limit})
		if err := r.prefixErrors[req.Prefix]; err != nil {
			return nil, err
		}
		entities := append([]graph.EntityState(nil), r.prefixEntities[req.Prefix]...)
		if req.Limit > 0 && len(entities) > req.Limit {
			entities = entities[:req.Limit]
		}
		return json.Marshal(map[string][]graph.EntityState{"entities": entities})
	}
	var req map[string]string
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	entityID := req["id"]
	r.entityRequests = append(r.entityRequests, entityID)
	entity, ok := r.entities[entityID]
	if !ok {
		return nil, errors.New("not found: " + entityID)
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

func capHazardEntity(
	t *testing.T,
	hazardID string,
	observed time.Time,
	evidence copmodel.HazardEvidenceDocument,
) graph.EntityState {
	t.Helper()
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	return graph.EntityState{
		ID:        hazardID,
		UpdatedAt: observed,
		Triples: []message.Triple{
			testTriple(hazardID, copmodel.HazardEvidence, string(evidenceJSON), observed),
			testTriple(hazardID, copmodel.HazardSource, "cap", observed),
			testTriple(hazardID, copmodel.ProvenanceSource, "cap", observed),
			testTriple(hazardID, copmodel.ProvenanceObservedAt, observed, observed),
			testTriple(hazardID, copmodel.ProvenanceSourceRef, "cap://nws/demo/flood-warning", observed),
		},
	}
}

func capLifecycleEvidence(base copmodel.HazardEvidenceDocument) copmodel.HazardEvidenceDocument {
	if base.Identifier == "" {
		base.Identifier = "nws-demo-flood-warning"
	}
	if base.Event == "" {
		base.Event = "Flood Warning"
	}
	if base.Severity == "" {
		base.Severity = "Severe"
	}
	if base.AreaDesc == "" {
		base.AreaDesc = "River Corridor"
	}
	if len(base.Polygons) == 0 {
		base.Polygons = [][]copmodel.HazardEvidencePoint{{
			{Lat: 38.8900, Lon: -77.0500},
			{Lat: 38.9050, Lon: -77.0440},
			{Lat: 38.9030, Lon: -77.0200},
			{Lat: 38.8860, Lon: -77.0280},
		}}
	}
	return base
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func findDiscoveryDiagnostic(
	diagnostics []DiscoveryDiagnostic,
	source string,
	entityType string,
) (DiscoveryDiagnostic, bool) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Source == source && diagnostic.EntityType == entityType {
			return diagnostic, true
		}
	}
	return DiscoveryDiagnostic{}, false
}
