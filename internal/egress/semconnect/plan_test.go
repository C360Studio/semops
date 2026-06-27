package semconnect

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	readmodel "github.com/c360studio/semops/internal/egress/csapi"
)

func TestBuildReadSidePlanTargetsSemConnectHTTPShapes(t *testing.T) {
	observed := time.Date(2026, 6, 26, 16, 30, 0, 0, time.UTC)
	systemID := "c360.edge.cop.mavlink.asset.system-42"
	streamID := "c360.edge.cop.mavlink.track.system-42/datastreams/position"
	deploymentID := systemID + "/deployments/current"
	observationID := "c360.edge.cop.mavlink.track.system-42/observations/current-position"
	eventID := "alert.mavlink.track-freshness"

	catalog := readmodel.Catalog{
		GeneratedAt: observed,
		ClaimScope:  readmodel.ClaimScopeReadSideEgress,
		Systems: []readmodel.System{{
			ID:          systemID,
			Name:        "MAVLink system 42",
			Description: "SemOps mavlink-system asset from mavlink",
			Source:      "mavlink",
			Location: &readmodel.Geometry{
				Type:        "Point",
				Coordinates: []float64{-77.0002, 38.9001},
			},
			Provenance: readmodel.Provenance{
				Owner:      "semops.feed.mavlink",
				SourceRef:  "mavlink://raw/asset",
				ObservedAt: observed,
			},
		}},
		Deployments: []readmodel.Deployment{{
			ID:         deploymentID,
			SystemID:   systemID,
			Name:       "MAVLink system 42 current deployment",
			Source:     "mavlink",
			ObservedAt: observed,
			Provenance: readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://raw/asset", ObservedAt: observed},
		}},
		Datastreams: []readmodel.Datastream{{
			ID:               streamID,
			SystemID:         systemID,
			Name:             "UAS 42 position",
			ObservedProperty: "position",
			ResultType:       "GeoJSON Point",
			Source:           "mavlink",
			Provenance:       readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://raw/track", ObservedAt: observed},
		}},
		Observations: []readmodel.Observation{{
			ID:             observationID,
			DatastreamID:   streamID,
			SystemID:       systemID,
			PhenomenonTime: observed,
			Result: map[string]any{
				"status":     "active.armed",
				"confidence": 0.99,
			},
			Geometry:         &readmodel.Geometry{Type: "Point", Coordinates: []float64{-77.0002, 38.9001}},
			Source:           "mavlink",
			Provenance:       readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://raw/track", ObservedAt: observed},
			SemOpsEntityID:   "c360.edge.cop.mavlink.track.system-42",
			SemOpsEntityType: "track",
		}},
		SystemEvents: []readmodel.SystemEvent{{
			ID:         eventID,
			SystemID:   systemID,
			EventType:  "cop.alert",
			Message:    "MAVLink position observed within freshness window",
			Severity:   "info",
			Status:     "active",
			ObservedAt: observed,
		}},
		DeferredSurfaces: []readmodel.DeferredSurface{{
			Name:   "csapi.commands.controlstreams",
			Reason: "stretch",
		}},
	}

	plan, err := BuildReadSidePlan(catalog)
	if err != nil {
		t.Fatalf("BuildReadSidePlan: %v", err)
	}
	if plan.ClaimScope != readmodel.ClaimScopeReadSideEgress {
		t.Fatalf("claim scope = %q", plan.ClaimScope)
	}
	if plan.IDs.Systems[systemID] != DefaultPrefixes().SystemIDPrefix+"."+stableToken(systemID) {
		t.Fatalf("system ID map = %#v", plan.IDs.Systems)
	}

	systemReq := requestByResource(t, plan, ResourceSystem)
	if systemReq.Method != "POST" || systemReq.Path != "/systems" || systemReq.ContentType != MediaJSON {
		t.Fatalf("system request = %+v", systemReq)
	}
	systemBody := systemReq.Body.(featureBody)
	if systemBody.Type != "Feature" ||
		systemBody.Properties.UID != stableToken(systemID) ||
		systemBody.Properties.Name != "MAVLink system 42" ||
		systemBody.Geometry == nil {
		t.Fatalf("system body = %+v", systemBody)
	}

	deploymentReq := requestByResource(t, plan, ResourceDeployment)
	deploymentBody := deploymentReq.Body.(featureBody)
	if deploymentReq.Path != "/deployments" ||
		deploymentBody.Properties.UID != stableToken(deploymentID) ||
		len(deploymentBody.Properties.DeployedSystemsLinks) != 1 ||
		deploymentBody.Properties.DeployedSystemsLinks[0].Href != "/systems/"+systemReq.SemConnectID {
		t.Fatalf("deployment request/body = %+v %+v", deploymentReq, deploymentBody)
	}

	streamReq := requestByResource(t, plan, ResourceDatastream)
	streamBody := streamReq.Body.(datastreamBody)
	if streamBody.ID != streamReq.SemConnectID ||
		streamBody.System != systemReq.SemConnectID ||
		streamBody.SystemID != systemReq.SemConnectID ||
		streamBody.ObservedProperty != "https://c360.studio/semops/cop/observed-property/position" ||
		strings.Count(streamBody.ID, ".") != 5 {
		t.Fatalf("datastream body = %+v", streamBody)
	}

	observationReq := requestByResource(t, plan, ResourceObservation)
	observationBody := observationReq.Body.(observationBody)
	if observationReq.Path != "/datastreams/"+streamReq.SemConnectID+"/observations" ||
		observationReq.ContentType != MediaOMS ||
		observationBody.ObservedProperty != streamBody.ObservedProperty ||
		observationBody.ResultTime != observed.Format(time.RFC3339Nano) {
		t.Fatalf("observation request/body = %+v %+v", observationReq, observationBody)
	}
	result := observationBody.Result.(map[string]any)
	if result["semops_entity_type"] != "track" || result["geometry"] == nil {
		t.Fatalf("observation result = %#v", result)
	}

	eventReq := requestByResource(t, plan, ResourceSystemEvent)
	eventBody := eventReq.Body.(systemEventBody)
	if eventReq.Path != "/systemEvents" ||
		eventBody.ID != eventReq.SemConnectID ||
		eventBody.SystemID != systemReq.SemConnectID ||
		eventBody.Source != "semops" ||
		eventBody.Payload["status"] != "active" {
		t.Fatalf("event request/body = %+v %+v", eventReq, eventBody)
	}

	for _, req := range plan.Requests {
		lowerPath := strings.ToLower(req.Path)
		if strings.Contains(lowerPath, "control") || strings.Contains(lowerPath, "command") {
			t.Fatalf("read-side plan emitted command/control request: %+v", req)
		}
	}
	if len(plan.DeferredSurfaces) != 1 || plan.DeferredSurfaces[0].Name != "csapi.commands.controlstreams" {
		t.Fatalf("deferred surfaces = %+v", plan.DeferredSurfaces)
	}
	if _, err := json.Marshal(plan); err != nil {
		t.Fatalf("plan must marshal as fixture JSON: %v", err)
	}
}

func TestBuildReadSidePlanIsDeterministic(t *testing.T) {
	catalog := readmodel.Catalog{
		GeneratedAt: time.Date(2026, 6, 26, 16, 30, 0, 0, time.UTC),
		ClaimScope:  readmodel.ClaimScopeReadSideEgress,
		Systems: []readmodel.System{{
			ID:     "c360.edge.cop.weather.system.open-meteo",
			Name:   "Open-Meteo weather source",
			Source: "weather",
		}},
		Datastreams: []readmodel.Datastream{{
			ID:               "c360.edge.cop.weather.weather_observation.temperature/datastreams/temperature_2m",
			SystemID:         "c360.edge.cop.weather.system.open-meteo",
			Name:             "Temperature",
			ObservedProperty: "temperature_2m",
			ResultType:       "Measure",
			Source:           "weather",
		}},
		Observations: []readmodel.Observation{{
			ID:             "c360.edge.cop.weather.weather_observation.temperature/observations/current",
			DatastreamID:   "c360.edge.cop.weather.weather_observation.temperature/datastreams/temperature_2m",
			SystemID:       "c360.edge.cop.weather.system.open-meteo",
			PhenomenonTime: time.Date(2026, 6, 26, 17, 0, 0, 0, time.UTC),
			Result:         map[string]any{"value": 29.4, "unit": "degC"},
			Source:         "weather",
		}},
	}

	left, err := BuildReadSidePlan(catalog)
	if err != nil {
		t.Fatalf("left plan: %v", err)
	}
	right, err := BuildReadSidePlan(catalog)
	if err != nil {
		t.Fatalf("right plan: %v", err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("plan should be deterministic\nleft=%+v\nright=%+v", left, right)
	}
}

func TestBuildReadSidePlanSkipsDanglingResources(t *testing.T) {
	catalog := readmodel.Catalog{
		GeneratedAt: time.Date(2026, 6, 26, 16, 30, 0, 0, time.UTC),
		ClaimScope:  readmodel.ClaimScopeReadSideEgress,
		Deployments: []readmodel.Deployment{{
			ID:       "dangling-deployment",
			SystemID: "missing-system",
		}},
		Datastreams: []readmodel.Datastream{{
			ID:               "dangling-datastream",
			SystemID:         "missing-system",
			ObservedProperty: "position",
		}},
		Observations: []readmodel.Observation{{
			ID:           "dangling-observation",
			DatastreamID: "missing-datastream",
			SystemID:     "missing-system",
			Result:       map[string]any{"status": "unknown"},
		}},
		SystemEvents: []readmodel.SystemEvent{{
			ID:       "dangling-event",
			SystemID: "missing-system",
		}},
	}

	plan, err := BuildReadSidePlan(catalog)
	if err != nil {
		t.Fatalf("BuildReadSidePlan: %v", err)
	}
	if len(plan.Requests) != 0 {
		t.Fatalf("dangling resources should not emit SemConnect requests: %+v", plan.Requests)
	}
	if len(plan.Warnings) != 4 {
		t.Fatalf("warnings = %+v, want one per dangling resource", plan.Warnings)
	}
}

func TestBuildReadSidePlanValidatesSemConnectPrefixes(t *testing.T) {
	_, err := BuildReadSidePlan(readmodel.Catalog{}, WithPrefixes(Prefixes{
		SystemIDPrefix:      "too.short",
		DatastreamIDPrefix:  defaultDatastreamIDPrefix,
		DeploymentIDPrefix:  defaultDeploymentIDPrefix,
		SystemEventIDPrefix: defaultSystemEventPrefix,
	}))
	if err == nil {
		t.Fatal("expected invalid prefix error")
	}
}

func requestByResource(t *testing.T, plan ReadSidePlan, resource string) Request {
	t.Helper()
	for _, req := range plan.Requests {
		if req.Resource == resource {
			return req
		}
	}
	t.Fatalf("missing request for resource %q in %+v", resource, plan.Requests)
	return Request{}
}
