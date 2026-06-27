package csapi

import (
	"fmt"
	"sort"
	"strings"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
)

const ClaimScopeReadSideEgress = "read-side-egress-only"

type Catalog struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	ClaimScope       string            `json:"claim_scope"`
	Systems          []System          `json:"systems"`
	Datastreams      []Datastream      `json:"datastreams"`
	Observations     []Observation     `json:"observations"`
	Deployments      []Deployment      `json:"deployments"`
	SystemEvents     []SystemEvent     `json:"system_events"`
	DeferredSurfaces []DeferredSurface `json:"deferred_surfaces"`
}

type System struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Source      string     `json:"source"`
	Location    *Geometry  `json:"location,omitempty"`
	Provenance  Provenance `json:"provenance"`
}

type Datastream struct {
	ID               string     `json:"id"`
	SystemID         string     `json:"system_id"`
	Name             string     `json:"name"`
	ObservedProperty string     `json:"observed_property"`
	ResultType       string     `json:"result_type"`
	Unit             string     `json:"unit,omitempty"`
	Source           string     `json:"source"`
	Provenance       Provenance `json:"provenance"`
}

type Observation struct {
	ID               string         `json:"id"`
	DatastreamID     string         `json:"datastream_id"`
	SystemID         string         `json:"system_id"`
	PhenomenonTime   time.Time      `json:"phenomenon_time"`
	Result           map[string]any `json:"result"`
	Geometry         *Geometry      `json:"geometry,omitempty"`
	Source           string         `json:"source"`
	Provenance       Provenance     `json:"provenance"`
	ClaimPosture     string         `json:"claim_posture,omitempty"`
	SemOpsEntityID   string         `json:"semops_entity_id"`
	SemOpsEntityType string         `json:"semops_entity_type"`
}

type Deployment struct {
	ID         string     `json:"id"`
	SystemID   string     `json:"system_id"`
	Name       string     `json:"name"`
	Source     string     `json:"source"`
	ObservedAt time.Time  `json:"observed_at"`
	Provenance Provenance `json:"provenance"`
}

type SystemEvent struct {
	ID         string     `json:"id"`
	SystemID   string     `json:"system_id,omitempty"`
	EventType  string     `json:"event_type"`
	Message    string     `json:"message"`
	Severity   string     `json:"severity,omitempty"`
	Status     string     `json:"status,omitempty"`
	ObservedAt time.Time  `json:"observed_at"`
	Provenance Provenance `json:"provenance"`
}

type DeferredSurface struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Geometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

type Provenance struct {
	Owner      string    `json:"owner"`
	SourceRef  string    `json:"source_ref,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

func ProjectSnapshot(snapshot copapi.Snapshot) Catalog {
	systems := make(map[string]System)
	assetIDs := make(map[string]struct{}, len(snapshot.Assets))
	trackSystemIDs := make(map[string]string, len(snapshot.Tracks))
	var datastreams []Datastream
	var observations []Observation
	var events []SystemEvent

	for _, asset := range snapshot.Assets {
		assetIDs[asset.ID] = struct{}{}
		system := System{
			ID:          asset.ID,
			Name:        asset.Label,
			Description: fmt.Sprintf("SemOps %s asset from %s", asset.Kind, asset.Source),
			Source:      asset.Source,
			Location:    optionalPointGeometry(asset.Position),
			Provenance:  provenance(asset.Provenance),
		}
		addSystem(systems, system)
	}

	for _, track := range snapshot.Tracks {
		systemID := resolveTrackSystemID(track.ID, assetIDs)
		trackSystemIDs[track.ID] = systemID
		addSystem(systems, System{
			ID:          systemID,
			Name:        track.Label,
			Description: fmt.Sprintf("SemOps track system from %s current state", track.Source),
			Source:      track.Source,
			Location:    pointGeometry(track.Position),
			Provenance:  provenance(track.Provenance),
		})

		streamID := track.ID + "/datastreams/position"
		datastreams = append(datastreams, Datastream{
			ID:               streamID,
			SystemID:         systemID,
			Name:             track.Label + " position",
			ObservedProperty: "position",
			ResultType:       "GeoJSON Point",
			Source:           track.Source,
			Provenance:       provenance(track.Provenance),
		})
		observations = append(observations, Observation{
			ID:             track.ID + "/observations/current-position",
			DatastreamID:   streamID,
			SystemID:       systemID,
			PhenomenonTime: observedAt(track.UpdatedAt, track.Provenance.Observed),
			Result: map[string]any{
				"status":     track.Status,
				"velocity":   track.Velocity,
				"confidence": track.Confidence,
			},
			Geometry:         pointGeometry(track.Position),
			Source:           track.Source,
			Provenance:       provenance(track.Provenance),
			SemOpsEntityID:   track.ID,
			SemOpsEntityType: "track",
		})
	}

	for _, footprint := range snapshot.SensorFootprints {
		systemID := footprint.ID + "/systems/sensor"
		addSystem(systems, System{
			ID:          systemID,
			Name:        firstNonEmpty(footprint.PlatformDesignation, footprint.Label),
			Description: "SemOps sensor footprint source",
			Source:      footprint.Source,
			Location:    pointGeometry(footprint.SensorPosition),
			Provenance:  provenance(footprint.Provenance),
		})

		streamID := footprint.ID + "/datastreams/sensor-footprint"
		datastreams = append(datastreams, Datastream{
			ID:               streamID,
			SystemID:         systemID,
			Name:             footprint.Label + " footprint",
			ObservedProperty: "sensor-footprint",
			ResultType:       "GeoJSON Geometry",
			Source:           footprint.Source,
			Provenance:       provenance(footprint.Provenance),
		})
		observations = append(observations, Observation{
			ID:             footprint.ID + "/observations/current-footprint",
			DatastreamID:   streamID,
			SystemID:       systemID,
			PhenomenonTime: observedAt(footprint.FrameTime, footprint.UpdatedAt, footprint.Provenance.Observed),
			Result: map[string]any{
				"sensor_position":        lonLat(footprint.SensorPosition),
				"frame_center":           lonLat(footprint.FrameCenter),
				"media_ref":              footprint.MediaRef,
				"packet_ref":             footprint.PacketRef,
				"platform_designation":   footprint.PlatformDesignation,
				"decoded_fields":         append([]string(nil), footprint.DecodedFields...),
				"warnings":               append([]string(nil), footprint.Warnings...),
				"confidence":             footprint.Confidence,
				"claim_posture":          footprint.ClaimPosture,
				"sensor_altitude_meters": optionalFloatResult(footprint.SensorAltitudeMeters),
			},
			Geometry:         footprintGeometry(footprint),
			Source:           footprint.Source,
			Provenance:       provenance(footprint.Provenance),
			ClaimPosture:     footprint.ClaimPosture,
			SemOpsEntityID:   footprint.ID,
			SemOpsEntityType: "sensor_footprint",
		})
	}

	for _, weather := range snapshot.Weather {
		systemID := "csapi.weather.system." + safeID(firstNonEmpty(weather.Provider, weather.Source, "provider"))
		addSystem(systems, System{
			ID:          systemID,
			Name:        firstNonEmpty(weather.Provider, weather.Source) + " weather source",
			Description: "SemOps tactical weather source",
			Source:      weather.Source,
			Location:    optionalPointGeometry(weather.Position),
			Provenance:  provenance(weather.Provenance),
		})

		streamID := weather.ID + "/datastreams/" + safeID(weather.Variable)
		datastreams = append(datastreams, Datastream{
			ID:               streamID,
			SystemID:         systemID,
			Name:             weather.Label,
			ObservedProperty: weather.Variable,
			ResultType:       "Measure",
			Unit:             weather.Unit,
			Source:           weather.Source,
			Provenance:       provenance(weather.Provenance),
		})
		observations = append(observations, Observation{
			ID:             weather.ID + "/observations/current",
			DatastreamID:   streamID,
			SystemID:       systemID,
			PhenomenonTime: observedAt(weather.ValidTime, weather.UpdatedAt, weather.Provenance.Observed),
			Result: map[string]any{
				"value":              weather.Value,
				"unit":               weather.Unit,
				"provider":           weather.Provider,
				"query_shape":        weather.QueryShape,
				"query_geometry_wkt": weather.QueryGeometryWKT,
				"model_time":         weather.ModelTime,
				"fresh_until":        weather.FreshUntil,
				"confidence":         weather.Confidence,
			},
			Geometry:         optionalPointGeometry(weather.Position),
			Source:           weather.Source,
			Provenance:       provenance(weather.Provenance),
			ClaimPosture:     weather.ClaimPosture,
			SemOpsEntityID:   weather.ID,
			SemOpsEntityType: "weather_observation",
		})
	}

	for _, alert := range snapshot.Alerts {
		systemID := alert.EntityID
		if mapped, ok := trackSystemIDs[alert.EntityID]; ok {
			systemID = mapped
		}
		events = append(events, SystemEvent{
			ID:         alert.ID,
			SystemID:   systemID,
			EventType:  "cop.alert",
			Message:    alert.Reason,
			Severity:   alert.Severity,
			Status:     alert.Status,
			ObservedAt: alert.UpdatedAt,
			Provenance: Provenance{ObservedAt: alert.UpdatedAt},
		})
	}

	systemList := sortedSystems(systems)
	deployments := deploymentsForSystems(systemList)
	sortDatastreams(datastreams)
	sortObservations(observations)
	sortEvents(events)

	return Catalog{
		GeneratedAt:      snapshot.GeneratedAt,
		ClaimScope:       ClaimScopeReadSideEgress,
		Systems:          systemList,
		Datastreams:      datastreams,
		Observations:     observations,
		Deployments:      deployments,
		SystemEvents:     events,
		DeferredSurfaces: deferredSurfaces(),
	}
}

func addSystem(systems map[string]System, system System) {
	if system.ID == "" {
		return
	}
	if _, exists := systems[system.ID]; exists {
		return
	}
	systems[system.ID] = system
}

func resolveTrackSystemID(trackID string, assetIDs map[string]struct{}) string {
	candidate := strings.Replace(trackID, ".track.", ".asset.", 1)
	if _, ok := assetIDs[candidate]; ok {
		return candidate
	}
	return trackID
}

func sortedSystems(systems map[string]System) []System {
	out := make([]System, 0, len(systems))
	for _, system := range systems {
		out = append(out, system)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func deploymentsForSystems(systems []System) []Deployment {
	out := make([]Deployment, 0, len(systems))
	for _, system := range systems {
		out = append(out, Deployment{
			ID:         system.ID + "/deployments/current",
			SystemID:   system.ID,
			Name:       system.Name + " current deployment",
			Source:     system.Source,
			ObservedAt: system.Provenance.ObservedAt,
			Provenance: system.Provenance,
		})
	}
	return out
}

func sortDatastreams(datastreams []Datastream) {
	sort.Slice(datastreams, func(i, j int) bool {
		return datastreams[i].ID < datastreams[j].ID
	})
}

func sortObservations(observations []Observation) {
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].ID < observations[j].ID
	})
}

func sortEvents(events []SystemEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].ID < events[j].ID
	})
}

func deferredSurfaces() []DeferredSurface {
	return []DeferredSurface{
		{
			Name:   "csapi.ingress.write",
			Reason: "Stretch goal: CS API source ingestion must use born-first SemOps projection contracts before graph writes.",
		},
		{
			Name:   "csapi.commands.controlstreams",
			Reason: "Stretch goal: command/control input must remain behind command authority, TTL, priority, local override, and native safety gates.",
		},
	}
}

func provenance(in copapi.Provenance) Provenance {
	return Provenance{
		Owner:      in.Owner,
		SourceRef:  in.SourceRef,
		ObservedAt: in.Observed,
	}
}

func observedAt(candidates ...time.Time) time.Time {
	for _, candidate := range candidates {
		if !candidate.IsZero() {
			return candidate
		}
	}
	return time.Time{}
}

func optionalPointGeometry(point *copapi.GeoPoint) *Geometry {
	if point == nil {
		return nil
	}
	return pointGeometry(*point)
}

func pointGeometry(point copapi.GeoPoint) *Geometry {
	return &Geometry{
		Type:        "Point",
		Coordinates: lonLat(point),
	}
}

func footprintGeometry(footprint copapi.SensorFootprint) *Geometry {
	if len(footprint.Footprint) > 0 {
		ring := make([][]float64, 0, len(footprint.Footprint)+1)
		for _, point := range footprint.Footprint {
			ring = append(ring, lonLat(point))
		}
		if !samePoint(footprint.Footprint[0], footprint.Footprint[len(footprint.Footprint)-1]) {
			ring = append(ring, lonLat(footprint.Footprint[0]))
		}
		return &Geometry{Type: "Polygon", Coordinates: [][][]float64{ring}}
	}
	if len(footprint.Ray) > 0 {
		line := make([][]float64, 0, len(footprint.Ray))
		for _, point := range footprint.Ray {
			line = append(line, lonLat(point))
		}
		return &Geometry{Type: "LineString", Coordinates: line}
	}
	return pointGeometry(footprint.FrameCenter)
}

func lonLat(point copapi.GeoPoint) []float64 {
	return []float64{point.Lon, point.Lat}
}

func samePoint(left, right copapi.GeoPoint) bool {
	return left.Lat == right.Lat && left.Lon == right.Lon
}

func optionalFloatResult(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func safeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
