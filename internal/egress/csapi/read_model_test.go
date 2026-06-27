package csapi

import (
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
)

func TestProjectSnapshotBuildsReadSideCSAPIResourceFamilies(t *testing.T) {
	observed := time.Date(2026, 6, 26, 16, 30, 0, 0, time.UTC)
	assetID := "c360.edge.cop.mavlink.asset.system-42"
	trackID := "c360.edge.cop.mavlink.track.system-42"
	footprintID := "c360.edge.cop.klv.sensor_footprint.frame-1"
	weatherID := "c360.edge.cop.weather.weather_observation.open-meteo-temperature"

	altitude := 1250.5
	snapshot := copapi.Snapshot{
		GeneratedAt: observed.Add(2 * time.Second),
		Assets: []copapi.Asset{{
			ID:        assetID,
			Label:     "MAVLink system 42",
			Kind:      "mavlink-system",
			Source:    "mavlink",
			Position:  &copapi.GeoPoint{Lat: 38.9001, Lon: -77.0002},
			UpdatedAt: observed,
			Provenance: copapi.Provenance{
				Owner:     "semops.feed.asset",
				SourceRef: "mavlink://raw/asset",
				Observed:  observed,
			},
		}},
		Tracks: []copapi.Track{{
			ID:         trackID,
			Label:      "UAS 42",
			Source:     "mavlink",
			Status:     "active.armed",
			Position:   copapi.GeoPoint{Lat: 38.9001, Lon: -77.0002},
			Velocity:   "NED_CMPS(321 -12 7)",
			Confidence: 0.99,
			UpdatedAt:  observed,
			Provenance: copapi.Provenance{
				Owner:     "semops.feed.mavlink",
				SourceRef: "mavlink://raw/track",
				Observed:  observed,
			},
		}},
		SensorFootprints: []copapi.SensorFootprint{{
			ID:                   footprintID,
			Label:                "TEST-UAS-01 sensor footprint",
			Source:               "klv",
			Status:               "active.footprint-polygon",
			SensorPosition:       copapi.GeoPoint{Lat: 38.9022, Lon: -77.0254},
			FrameCenter:          copapi.GeoPoint{Lat: 38.8956, Lon: -77.0108},
			SensorAltitudeMeters: &altitude,
			Ray: []copapi.GeoPoint{
				{Lat: 38.9022, Lon: -77.0254},
				{Lat: 38.8956, Lon: -77.0108},
			},
			Footprint: []copapi.GeoPoint{
				{Lat: 38.8971, Lon: -77.0136},
				{Lat: 38.8968, Lon: -77.0079},
				{Lat: 38.8939, Lon: -77.0075},
			},
			MediaRef:            "object://semops/klv/demo.ts",
			PacketRef:           "klv://packet/demo/0001",
			FrameTime:           observed.Add(time.Second),
			PlatformDesignation: "TEST-UAS-01",
			ClaimPosture:        "read-side KLV sensor footprint evidence",
			DecodedFields:       []string{"sensor_position", "frame_center", "footprint_polygon"},
			Confidence:          0.82,
			UpdatedAt:           observed.Add(time.Second),
			Provenance: copapi.Provenance{
				Owner:     "semops.feed.klv",
				SourceRef: "klv://packet/demo/0001",
				Observed:  observed.Add(time.Second),
			},
		}},
		Weather: []copapi.WeatherObservation{{
			ID:               weatherID,
			Label:            "Temperature at LZ",
			Source:           "weather",
			Status:           "fresh",
			Provider:         "open-meteo",
			QueryShape:       "position",
			QueryGeometryWKT: "POINT(-77.0400000 38.9000000)",
			Position:         &copapi.GeoPoint{Lat: 38.9, Lon: -77.04},
			ValidTime:        observed.Add(5 * time.Minute),
			ModelTime:        observed,
			Variable:         "temperature_2m",
			Value:            29.4,
			Unit:             "degC",
			ClaimPosture:     "fixture-backed tactical weather evidence",
			Confidence:       0.73,
			UpdatedAt:        observed,
			Provenance: copapi.Provenance{
				Owner:     "semops.feed.weather",
				SourceRef: "weather://fixture/open-meteo/temperature",
				Observed:  observed,
			},
		}},
		Alerts: []copapi.Alert{{
			ID:        "alert.mavlink.track-freshness",
			Label:     "Track freshness nominal",
			Severity:  "info",
			Status:    "active",
			EntityID:  trackID,
			Reason:    "MAVLink position observed within freshness window",
			UpdatedAt: observed,
		}},
	}

	catalog := ProjectSnapshot(snapshot)
	if catalog.ClaimScope != ClaimScopeReadSideEgress {
		t.Fatalf("claim scope = %q, want %q", catalog.ClaimScope, ClaimScopeReadSideEgress)
	}
	if catalog.GeneratedAt != snapshot.GeneratedAt {
		t.Fatalf("generated_at = %s, want %s", catalog.GeneratedAt, snapshot.GeneratedAt)
	}
	if _, ok := findSystem(catalog, assetID); !ok {
		t.Fatalf("missing asset-backed system %q in %#v", assetID, catalog.Systems)
	}

	trackStream, ok := findDatastream(catalog, trackID+"/datastreams/position")
	if !ok {
		t.Fatalf("missing track position datastream in %#v", catalog.Datastreams)
	}
	if trackStream.SystemID != assetID || trackStream.ObservedProperty != "position" {
		t.Fatalf("track datastream = %+v, want system %q position", trackStream, assetID)
	}
	trackObservation, ok := findObservation(catalog, trackID+"/observations/current-position")
	if !ok {
		t.Fatalf("missing track observation in %#v", catalog.Observations)
	}
	if trackObservation.SystemID != assetID ||
		trackObservation.Geometry == nil ||
		trackObservation.Geometry.Type != "Point" ||
		trackObservation.Provenance.Owner != "semops.feed.mavlink" {
		t.Fatalf("track observation = %+v", trackObservation)
	}

	footprintObservation, ok := findObservation(catalog, footprintID+"/observations/current-footprint")
	if !ok {
		t.Fatalf("missing KLV footprint observation in %#v", catalog.Observations)
	}
	if footprintObservation.Geometry == nil ||
		footprintObservation.Geometry.Type != "Polygon" ||
		footprintObservation.ClaimPosture != "read-side KLV sensor footprint evidence" ||
		footprintObservation.SemOpsEntityType != "sensor_footprint" {
		t.Fatalf("footprint observation = %+v", footprintObservation)
	}

	weatherObservation, ok := findObservation(catalog, weatherID+"/observations/current")
	if !ok {
		t.Fatalf("missing weather observation in %#v", catalog.Observations)
	}
	if weatherObservation.Result["value"] != 29.4 ||
		weatherObservation.Result["unit"] != "degC" ||
		weatherObservation.SemOpsEntityType != "weather_observation" {
		t.Fatalf("weather observation = %+v", weatherObservation)
	}

	if _, ok := findDeployment(catalog, assetID+"/deployments/current"); !ok {
		t.Fatalf("missing asset deployment in %#v", catalog.Deployments)
	}
	event, ok := findSystemEvent(catalog, "alert.mavlink.track-freshness")
	if !ok {
		t.Fatalf("missing system event in %#v", catalog.SystemEvents)
	}
	if event.SystemID != assetID || event.EventType != "cop.alert" {
		t.Fatalf("event = %+v, want track alert mapped back to asset system %q", event, assetID)
	}
}

func TestProjectSnapshotDefersWriteSideAndCommandSurfaces(t *testing.T) {
	observed := time.Date(2026, 6, 26, 16, 30, 0, 0, time.UTC)
	snapshot := copapi.Snapshot{
		GeneratedAt: observed,
		Tasks: []copapi.Task{{
			ID:           "c360.edge.cop.command.task.route-42",
			Label:        "Route 42",
			Kind:         "route",
			Source:       "command",
			Status:       "requested",
			DesiredState: "route-to-sector-alpha",
			UpdatedAt:    observed,
			Provenance: copapi.Provenance{
				Owner:     "semops.command.intent",
				SourceRef: "csapi://commands/route-42",
				Observed:  observed,
			},
		}},
	}

	catalog := ProjectSnapshot(snapshot)
	if len(catalog.Datastreams) != 0 || len(catalog.Observations) != 0 || len(catalog.SystemEvents) != 0 {
		t.Fatalf("command-intent task should not create read-side CS API data: %+v", catalog)
	}
	if !hasDeferredSurface(catalog, "csapi.ingress.write") ||
		!hasDeferredSurface(catalog, "csapi.commands.controlstreams") {
		t.Fatalf("deferred write/command surfaces missing: %+v", catalog.DeferredSurfaces)
	}
}

func findSystem(catalog Catalog, id string) (System, bool) {
	for _, system := range catalog.Systems {
		if system.ID == id {
			return system, true
		}
	}
	return System{}, false
}

func findDatastream(catalog Catalog, id string) (Datastream, bool) {
	for _, stream := range catalog.Datastreams {
		if stream.ID == id {
			return stream, true
		}
	}
	return Datastream{}, false
}

func findObservation(catalog Catalog, id string) (Observation, bool) {
	for _, observation := range catalog.Observations {
		if observation.ID == id {
			return observation, true
		}
	}
	return Observation{}, false
}

func findDeployment(catalog Catalog, id string) (Deployment, bool) {
	for _, deployment := range catalog.Deployments {
		if deployment.ID == id {
			return deployment, true
		}
	}
	return Deployment{}, false
}

func findSystemEvent(catalog Catalog, id string) (SystemEvent, bool) {
	for _, event := range catalog.SystemEvents {
		if event.ID == id {
			return event, true
		}
	}
	return SystemEvent{}, false
}

func hasDeferredSurface(catalog Catalog, name string) bool {
	for _, surface := range catalog.DeferredSurfaces {
		if surface.Name == name {
			return true
		}
	}
	return false
}
