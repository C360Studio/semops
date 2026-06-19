package cop

import (
	"context"
	"time"
)

type FixtureProvider struct {
	now func() time.Time
}

func NewFixtureProvider(now func() time.Time) *FixtureProvider {
	if now == nil {
		now = time.Now
	}
	return &FixtureProvider{now: now}
}

func (p *FixtureProvider) Snapshot(context.Context) (Snapshot, error) {
	now := p.now().UTC()
	observed := now.Add(-18 * time.Second)
	hazardObserved := now.Add(-2 * time.Minute)

	snapshot := Snapshot{
		GeneratedAt: now,
		Scenario:    "phase-1-fixture",
		Feeds: []FeedHealth{
			{
				ID:          "feed.mavlink",
				Name:        "MAVLink",
				Kind:        "telemetry",
				Status:      "live",
				LastEventAt: observed,
				Message:     "Generated heartbeat and position smoke path",
			},
			{
				ID:          "feed.tak",
				Name:        "TAK/CoT",
				Kind:        "operators",
				Status:      "planned",
				LastEventAt: now.Add(-18 * time.Minute),
				Message:     "Seed replay gate pending",
			},
			{
				ID:          "feed.cap",
				Name:        "CAP",
				Kind:        "advisory",
				Status:      "planned",
				LastEventAt: now.Add(-33 * time.Minute),
				Message:     "Schema/sample gate pending",
			},
		},
		Assets: []Asset{
			{
				ID:         "c360.edge.cop.mavlink.asset.system-42",
				Label:      "MAVLink system 42",
				Kind:       "mavlink-system",
				Source:     "mavlink",
				Position:   &GeoPoint{Lat: 38.9001, Lon: -77.0002},
				Confidence: 1,
				UpdatedAt:  observed,
				Provenance: Provenance{
					Owner:     "semops.feed.asset",
					SourceRef: "raw:mavlink:fixture:0001",
					Observed:  observed,
				},
			},
		},
		Tracks: []Track{
			{
				ID:         "c360.edge.cop.mavlink.track.system-42",
				Label:      "UAS 42",
				Source:     "mavlink",
				Status:     "active.armed",
				Position:   GeoPoint{Lat: 38.9001, Lon: -77.0002},
				Velocity:   "NED_CMPS(321 -12 7)",
				Confidence: 1,
				UpdatedAt:  observed,
				Provenance: Provenance{
					Owner:     "semops.feed.mavlink",
					SourceRef: "raw:mavlink:fixture:0002",
					Observed:  observed,
				},
			},
		},
		Hazards: []Hazard{
			{
				ID:       "c360.edge.cop.cap.hazard_area.flood-watch-1",
				Label:    "Flood watch sector",
				Kind:     "flood",
				Severity: "watch",
				Geometry: []GeoPoint{
					{Lat: 38.895, Lon: -77.012},
					{Lat: 38.907, Lon: -77.011},
					{Lat: 38.908, Lon: -76.992},
					{Lat: 38.896, Lon: -76.991},
				},
				Source:     "cap",
				Confidence: 0.74,
				UpdatedAt:  hazardObserved,
				Provenance: Provenance{
					Owner:     "semops.feed.cap",
					SourceRef: "fixture:cap:flood-watch-1",
					Observed:  hazardObserved,
				},
			},
		},
		Alerts: []Alert{
			{
				ID:        "alert.mavlink.track-freshness",
				Label:     "Track freshness nominal",
				Severity:  "info",
				Status:    "active",
				EntityID:  "c360.edge.cop.mavlink.track.system-42",
				Reason:    "MAVLink position observed within freshness window",
				UpdatedAt: observed,
			},
		},
	}
	snapshot.Summary = Summary{
		ActiveTracks: len(snapshot.Tracks),
		ActiveAlerts: len(snapshot.Alerts),
		StaleFeeds:   countFeeds(snapshot.Feeds, "stale"),
	}
	return snapshot, nil
}

func countFeeds(feeds []FeedHealth, status string) int {
	var count int
	for _, feed := range feeds {
		if feed.Status == status {
			count++
		}
	}
	return count
}
