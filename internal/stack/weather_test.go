package stack

import (
	"context"
	"strings"
	"testing"
	"time"

	weatherprojector "github.com/c360studio/semops/internal/projectors/weather"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
)

func TestNewWeatherPlanWriterWiresNATSGraphWriter(t *testing.T) {
	now := time.Date(2026, 6, 23, 16, 0, 0, 0, time.UTC)
	retry := natsclient.RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
	client := &recordingRetryRequester{}
	writer, err := NewWeatherPlanWriter(WeatherAdapterConfig{
		Source:       "weather:fixture",
		Org:          "c360",
		Platform:     "edge",
		OwnerTokens:  testOwnerTokens("stack-test"),
		TraceID:      "weather-stack-test",
		WriteTimeout: 25 * time.Millisecond,
		Retry:        retry,
		Clock:        func() time.Time { return now },
	}, WeatherAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new weather plan writer: %v", err)
	}

	projector := weatherprojector.NewProjector(weatherprojector.Config{
		OwnerTokens: testOwnerTokens("stack-test"),
		TraceID:     "weather-stack-test",
	})
	plan, err := projector.ProjectObservation(weatherprojector.Observation{
		NativeID:         "weather.open-meteo.position.fixture.temperature",
		Provider:         "open-meteo",
		QueryShape:       "position",
		QueryGeometryWKT: "POINT(-77.0400000 38.9000000)",
		ValidTime:        now,
		ModelTime:        now.Add(-5 * time.Minute),
		Variable:         "temperature_2m",
		Value:            29.4,
		Unit:             "degC",
		SourceRef:        "file:///fixtures/weather/open-meteo-point.json",
	})
	if err != nil {
		t.Fatalf("project weather observation: %v", err)
	}
	if err := writer.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply weather plan: %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("requests = %d, want one weather observation create", len(client.calls))
	}
	call := client.calls[0]
	if call.subject != weatherprojector.SubjectEntityCreateWithTriples {
		t.Fatalf("request subject = %q", call.subject)
	}
	if call.timeout != 25*time.Millisecond {
		t.Fatalf("request timeout = %s, want 25ms", call.timeout)
	}
	if call.retry != retry {
		t.Fatalf("retry = %+v, want %+v", call.retry, retry)
	}

	var create graph.CreateEntityWithTriplesRequest
	decodePayload(t, call.payload, &create)
	if create.OwnerToken != "semops.feed.weather#stack-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "weather-stack-test" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	if create.IndexingProfile != cop.WeatherObservationContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	requireTriple(t, create.Triples, cop.WeatherVariable, "temperature_2m")
	requireTriple(t, create.Triples, cop.WeatherValue, 29.4)
}

func TestNewWeatherPlanWriterRequiresGraphDependency(t *testing.T) {
	_, err := NewWeatherPlanWriter(WeatherAdapterConfig{}, WeatherAdapterDeps{})
	if err == nil {
		t.Fatal("expected missing graph dependency error")
	}
	if !strings.Contains(err.Error(), "NATS requester or injected plan writer") {
		t.Fatalf("error = %v", err)
	}
}
