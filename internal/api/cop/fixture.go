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
	takObserved := now.Add(-46 * time.Second)
	hazardObserved := now.Add(-2 * time.Minute)
	klvObserved := now.Add(-75 * time.Second)
	sensorAltitude := 1250.5
	sensorAzimuth := 90.25
	sensorElevation := -12.5
	frameCenterElevation := 932.2

	snapshot := Snapshot{
		GeneratedAt: now,
		Scenario:    "phase-1-fixture",
		Diagnostics: SnapshotDiagnostics{
			Discovery: []DiscoveryDiagnostic{},
		},
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
				Status:      "live",
				LastEventAt: takObserved,
				Message:     "Seed replay track, task, and GeoChat smoke path",
			},
			{
				ID:          "feed.cap",
				Name:        "CAP",
				Kind:        "advisory",
				Status:      "planned",
				LastEventAt: now.Add(-33 * time.Minute),
				Message:     "Schema/sample gate pending",
			},
			{
				ID:          "feed.klv",
				Name:        "KLV",
				Kind:        "sensor-footprint",
				Status:      "live",
				LastEventAt: klvObserved,
				Message:     "Graph-backed KLV sensor/footprint proof",
			},
			{
				ID:          "feed.weather",
				Name:        "Weather",
				Kind:        "tactical-weather",
				Status:      "planned",
				LastEventAt: now.Add(-52 * time.Minute),
				Message:     "Weather graph proof pending",
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
			{
				ID:         "c360.edge.cop.tak.track.android-alpha",
				Label:      "ANDROID-ALPHA",
				Source:     "tak-cot",
				Status:     "active.operator",
				Position:   GeoPoint{Lat: 38.892, Lon: -77.035},
				Velocity:   "",
				Confidence: 1,
				UpdatedAt:  takObserved,
				Provenance: Provenance{
					Owner:     "semops.feed.tak",
					SourceRef: "cot://fixture/0001",
					Observed:  takObserved,
				},
			},
		},
		Tasks: []Task{
			{
				ID:          "c360.edge.cop.tak.task.marker-north-gate",
				Label:       "North Gate",
				Kind:        "marker",
				Source:      "tak-cot",
				Status:      "active.marker",
				Position:    &GeoPoint{Lat: 38.894, Lon: -77.038},
				Description: "checkpoint",
				Confidence:  1,
				UpdatedAt:   takObserved,
				Provenance: Provenance{
					Owner:     "semops.feed.tak",
					SourceRef: "cot://fixture/0003",
					Observed:  takObserved,
				},
			},
		},
		Advisories: []Advisory{
			{
				ID:         "c360.edge.cop.tak.advisory.chat-alpha-1",
				Label:      "GeoChat ANDROID-ALPHA",
				Kind:       "geochat",
				Source:     "tak-cot",
				Status:     "active.geochat",
				Text:       "hold at checkpoint",
				Sender:     "ANDROID-ALPHA",
				Position:   &GeoPoint{Lat: 38.892, Lon: -77.035},
				Confidence: 1,
				UpdatedAt:  takObserved,
				Provenance: Provenance{
					Owner:     "semops.feed.tak",
					SourceRef: "cot://fixture/0004",
					Observed:  takObserved,
				},
			},
		},
		Hazards: []Hazard{
			{
				ID:       "c360.edge.cop.cap.hazard_area.flood-watch-1",
				Label:    "Flood watch sector",
				Kind:     "flood",
				Severity: "watch",
				Status:   "active",
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
		SensorFootprints: []SensorFootprint{
			{
				ID:             "c360.edge.cop.klv.sensor_footprint.object-semops-klv-deterministic-001-ts",
				Label:          "TEST-UAS-01 sensor footprint",
				Source:         "klv",
				Status:         "active.footprint-polygon",
				SensorPosition: GeoPoint{Lat: 38.9022, Lon: -77.0254},
				FrameCenter:    GeoPoint{Lat: 38.8956, Lon: -77.0108},
				Ray:            []GeoPoint{{Lat: 38.9022, Lon: -77.0254}, {Lat: 38.8956, Lon: -77.0108}},
				Footprint: []GeoPoint{
					{Lat: 38.8971, Lon: -77.0136},
					{Lat: 38.8968, Lon: -77.0079},
					{Lat: 38.8939, Lon: -77.0075},
					{Lat: 38.8936, Lon: -77.0132},
				},
				SensorAltitudeMeters:       &sensorAltitude,
				SensorAzimuthDegrees:       &sensorAzimuth,
				SensorElevationDegrees:     &sensorElevation,
				FrameCenterElevationMeters: &frameCenterElevation,
				MediaRef:                   "object://semops/klv/deterministic-001.ts",
				PacketRef:                  "klv://packet/deterministic/00000001",
				FrameTime:                  klvObserved,
				PlatformDesignation:        "TEST-UAS-01",
				ClaimPosture:               "sensor/frame-center and footprint polygon graph readback; tested MISB ST 0601 offset-corner subset; no STANAG conformance",
				DecodedFields: []string{
					"media_ref",
					"packet_ref",
					"observed_at",
					"platform_designation",
					"sensor_position",
					"sensor_altitude_meters",
					"sensor_azimuth_degrees",
					"sensor_elevation_degrees",
					"frame_center",
					"frame_center_elevation_meters",
					"footprint_polygon",
				},
				Warnings:   []string{},
				Confidence: 0.82,
				UpdatedAt:  klvObserved,
				Provenance: Provenance{
					Owner:     "semops.feed.klv",
					SourceRef: "klv://packet/deterministic/00000001",
					Observed:  klvObserved,
				},
			},
		},
		Weather: []WeatherObservation{},
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
		ActiveTracks:           len(snapshot.Tracks),
		ActiveTasks:            len(snapshot.Tasks),
		ActiveAdvisories:       len(snapshot.Advisories),
		ActiveSensorFootprints: len(snapshot.SensorFootprints),
		ActiveWeather:          len(snapshot.Weather),
		ActiveAlerts:           len(snapshot.Alerts),
		StaleFeeds:             countFeeds(snapshot.Feeds, "stale"),
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
