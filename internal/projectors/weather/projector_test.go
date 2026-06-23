package weather

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectorCreatesPointWeatherObservationsWithoutHazardAuthority(t *testing.T) {
	forecast := readOpenMeteoForecast(t)
	modelTime := time.Date(2026, 6, 22, 14, 55, 0, 0, time.UTC)
	observations, err := ObservationsFromPointForecast(
		forecast,
		"weather://fixture/open-meteo-point",
		modelTime,
		45*time.Minute,
	)
	if err != nil {
		t.Fatalf("point observations: %v", err)
	}
	if len(observations) != 16 {
		t.Fatalf("observations = %d, want 16 variable samples", len(observations))
	}
	observation := observations[0]
	if observation.Variable != "temperature_2m" ||
		observation.Value != 29.4 ||
		observation.Unit != "degC" ||
		observation.QueryGeometryWKT != "POINT(-77.0400000 38.9000000)" ||
		observation.ValidTime != time.Date(2026, 6, 22, 15, 0, 0, 0, time.UTC) ||
		observation.FreshUntil == nil ||
		*observation.FreshUntil != modelTime.Add(45*time.Minute) {
		t.Fatalf("first observation = %+v", observation)
	}

	projector := NewProjector(Config{
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "weather-contract-001",
	})
	plan, err := projector.ProjectObservation(observation)
	if err != nil {
		t.Fatalf("project observation: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want observation birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	if create.Entity.ID != EntityID("c360", "edge", observation.NativeID) {
		t.Fatalf("observation id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.WeatherObservationContract().MessageType {
		t.Fatalf("message type = %q", create.Entity.MessageType.Key())
	}
	if create.Entity.UpdatedAt != observation.ValidTime {
		t.Fatalf("updated_at = %s, want %s", create.Entity.UpdatedAt, observation.ValidTime)
	}
	if create.IndexingProfile != cop.WeatherObservationContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.OwnerToken != "semops.feed.weather#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "weather-contract-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}

	requireTriple(t, create.Triples, cop.WeatherProvider, weathercodec.ProviderOpenMeteo)
	requireTriple(t, create.Triples, cop.WeatherQueryShape, weathercodec.QueryShapePosition)
	requireTriple(t, create.Triples, cop.WeatherQueryGeometry, "POINT(-77.0400000 38.9000000)")
	requireTriple(t, create.Triples, cop.WeatherVariable, "temperature_2m")
	requireTriple(t, create.Triples, cop.WeatherValue, 29.4)
	requireTriple(t, create.Triples, cop.WeatherUnit, "degC")
	requireTriple(t, create.Triples, cop.WeatherValidTime, observation.ValidTime)
	requireTriple(t, create.Triples, cop.WeatherModelTime, modelTime)
	requireTriple(t, create.Triples, cop.WeatherFreshUntil, modelTime.Add(45*time.Minute))
	requireTriple(t, create.Triples, cop.ProvenanceSource, "weather")
	requireTriple(t, create.Triples, cop.ProvenanceConfidence, 0.75)
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "weather://fixture/open-meteo-point")

	for _, predicate := range []string{
		cop.HazardGeometry,
		cop.HazardSeverity,
		cop.HazardStatus,
		cop.HazardAdvisoryText,
		cop.AlertSeverity,
		cop.AlertStatus,
		cop.TaskStatus,
		cop.TaskPosition,
	} {
		if hasPredicate(create.Triples, predicate) {
			t.Fatalf("weather observation must not emit authority predicate %q", predicate)
		}
	}
}

func TestProjectorUpdatesKnownWeatherObservationWithoutRebirth(t *testing.T) {
	forecast := readOpenMeteoForecast(t)
	observations, err := ObservationsFromPointForecast(
		forecast,
		"weather://fixture/open-meteo-point",
		time.Date(2026, 6, 22, 14, 55, 0, 0, time.UTC),
		time.Hour,
	)
	if err != nil {
		t.Fatalf("point observations: %v", err)
	}
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	birth, err := projector.ProjectObservation(observations[0])
	if err != nil {
		t.Fatalf("project birth: %v", err)
	}
	if marked := projector.MarkBornForPlan(birth); marked != 1 {
		t.Fatalf("marked births = %d, want 1", marked)
	}

	observations[0].Value = 29.8
	updatePlan, err := projector.ProjectObservation(observations[0])
	if err != nil {
		t.Fatalf("project update: %v", err)
	}
	if len(updatePlan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want update", len(updatePlan.Mutations))
	}
	update := requireUpdate(t, updatePlan.Mutations[0])
	if update.Entity.ID != EntityID("c360", "edge", observations[0].NativeID) {
		t.Fatalf("update id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.WeatherObservationContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", update.IndexingProfile)
	}
	requireTriple(t, update.AddTriples, cop.WeatherValue, 29.8)
}

func TestSpatialForecastProducesWeatherObservationsWithoutRuntimePromotion(t *testing.T) {
	forecast := readSpatialForecast(t, "ogc-edr-corridor.json")
	observations, err := ObservationsFromSpatialForecast(
		forecast,
		"weather://fixture/ogc-edr-corridor",
		time.Date(2026, 6, 22, 14, 50, 0, 0, time.UTC),
		30*time.Minute,
	)
	if err != nil {
		t.Fatalf("spatial observations: %v", err)
	}
	if len(observations) != 16 {
		t.Fatalf("observations = %d, want 16 variable samples", len(observations))
	}
	first := observations[0]
	if first.Provider != weathercodec.ProviderOGCEDR ||
		first.QueryShape != weathercodec.QueryShapeCorridor ||
		first.QueryGeometryWKT != `LINESTRING(-77.06 38.88,-77.04 38.90,-77.02 38.93)` ||
		first.Variable != "temperature_2m" ||
		first.Unit != "Cel" {
		t.Fatalf("first spatial observation = %+v", first)
	}

	plan, err := NewProjector(Config{OwnerTokens: testOwnerTokens("test")}).
		ProjectObservations(observations[:2])
	if err != nil {
		t.Fatalf("project spatial observations: %v", err)
	}
	if len(plan.Mutations) != 2 {
		t.Fatalf("mutations = %d, want two observation births", len(plan.Mutations))
	}
	for _, mutation := range plan.Mutations {
		create := requireCreate(t, mutation)
		requireTriple(t, create.Triples, cop.WeatherQueryShape, weathercodec.QueryShapeCorridor)
		if hasPredicate(create.Triples, cop.HazardGeometry) {
			t.Fatalf("spatial weather observation emitted hazard geometry: %+v", create.Triples)
		}
	}
}

func TestProjectorRejectsMalformedWeatherObservation(t *testing.T) {
	projector := NewProjector(Config{})
	tests := []struct {
		name        string
		observation Observation
		want        string
	}{
		{
			name:        "missing native id",
			observation: Observation{Provider: "open-meteo", QueryShape: "position", QueryGeometryWKT: "POINT(0 0)", ValidTime: sampleTime(), Variable: "temperature_2m"},
			want:        "native_id",
		},
		{
			name:        "missing geometry",
			observation: Observation{NativeID: "weather.test", Provider: "open-meteo", QueryShape: "position", ValidTime: sampleTime(), Variable: "temperature_2m"},
			want:        "query_geometry",
		},
		{
			name:        "missing valid time",
			observation: Observation{NativeID: "weather.test", Provider: "open-meteo", QueryShape: "position", QueryGeometryWKT: "POINT(0 0)", Variable: "temperature_2m"},
			want:        "valid_time",
		},
		{
			name:        "non-finite value",
			observation: Observation{NativeID: "weather.test", Provider: "open-meteo", QueryShape: "position", QueryGeometryWKT: "POINT(0 0)", ValidTime: sampleTime(), Variable: "temperature_2m", Value: math.NaN()},
			want:        "finite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := projector.ProjectObservation(tt.observation)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func readOpenMeteoForecast(t *testing.T) weathercodec.PointForecast {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "weather", "open-meteo-point.json"))
	if err != nil {
		t.Fatalf("read Open-Meteo fixture: %v", err)
	}
	forecast, err := weathercodec.ParseOpenMeteoPointForecast(data)
	if err != nil {
		t.Fatalf("parse Open-Meteo fixture: %v", err)
	}
	return forecast
}

func readSpatialForecast(t *testing.T, name string) weathercodec.SpatialForecast {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "weather", name))
	if err != nil {
		t.Fatalf("read spatial fixture: %v", err)
	}
	forecast, err := weathercodec.ParseOGCEDRSpatialForecast(data)
	if err != nil {
		t.Fatalf("parse spatial fixture: %v", err)
	}
	return forecast
}

func sampleTime() time.Time {
	return time.Date(2026, 6, 22, 15, 0, 0, 0, time.UTC)
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
		cop.OwnerWeather: ownership.ExpectedOwnerToken(cop.OwnerWeather, incarnation),
	}
}
