package weather

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	weatherprojector "github.com/c360studio/semops/internal/projectors/weather"
	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/nats-io/nats.go"
)

func TestPayloadRegistryRoundTripsRawAndDecodedForecasts(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads should be idempotent: %v", err)
	}

	rawJSON := readFixture(t)
	now := time.Date(2026, 6, 23, 13, 0, 0, 0, time.UTC)
	raw := NewRawForecastPayload(
		"weather:fixture:test",
		weathercodec.ProviderOpenMeteo,
		weathercodec.QueryShapePosition,
		DefaultFixturePath,
		"file:///tmp/weather.json",
		now,
		rawJSON,
	)
	rawWire := mustBaseMessageJSON(t, RawForecastType, raw, "semops-input-weather-fixture", now)
	rawEnvelope, err := message.NewDecoder(registry).Decode(rawWire)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	gotRaw, ok := rawEnvelope.Payload().(*RawForecastPayload)
	if !ok {
		t.Fatalf("raw payload type = %T, want *RawForecastPayload", rawEnvelope.Payload())
	}
	forecast, err := weathercodec.ParseOpenMeteoPointForecast(gotRaw.RawJSON)
	if err != nil {
		t.Fatalf("parse raw forecast: %v", err)
	}

	decoded := NewDecodedForecastPayload(gotRaw, forecast)
	decodedWire := mustBaseMessageJSON(t, DecodedForecastType, decoded, "semops-processor-weather-decode", now)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decodedWire)
	if err != nil {
		t.Fatalf("decode decoded payload: %v", err)
	}
	gotDecoded, ok := decodedEnvelope.Payload().(*DecodedForecastPayload)
	if !ok {
		t.Fatalf("decoded payload type = %T, want *DecodedForecastPayload", decodedEnvelope.Payload())
	}
	if gotDecoded.Forecast.Provider != weathercodec.ProviderOpenMeteo ||
		gotDecoded.Forecast.QueryShape != weathercodec.QueryShapePosition ||
		len(gotDecoded.Forecast.Samples) != 2 {
		t.Fatalf("decoded forecast = %+v", gotDecoded.Forecast)
	}
}

func TestWeatherComponentsExposeFileAndStreamPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*FixtureInputComponent)(nil)
	var _ component.LifecycleComponent = (*DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*ProjectorComponent)(nil)

	bus := &recordingBus{}
	input, err := NewFixtureInputComponent(FixtureInputConfig{
		FixturePath: fixturePath(),
		Bus:         bus,
	})
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Writer:       &recordingWeatherPlanWriter{},
		WriteTimeout: 250 * time.Millisecond,
		WriteRetries: 2,
	}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	for name, lifecycle := range map[string]component.LifecycleComponent{
		"input":     input,
		"decoder":   decoder,
		"projector": projector,
	} {
		if err := lifecycle.Initialize(); err != nil {
			t.Fatalf("initialize %s: %v", name, err)
		}
		if err := lifecycle.Start(context.Background()); err != nil {
			t.Fatalf("start %s: %v", name, err)
		}
		if err := lifecycle.Stop(time.Second); err != nil {
			t.Fatalf("stop %s: %v", name, err)
		}
	}

	if input.Meta().Type != "input" {
		t.Fatalf("input component type = %q, want input", input.Meta().Type)
	}
	if decoder.Meta().Type != "processor" {
		t.Fatalf("decoder component type = %q, want processor", decoder.Meta().Type)
	}
	if projector.Meta().Type != "processor" {
		t.Fatalf("projector component type = %q, want processor", projector.Meta().Type)
	}
	filePort, ok := input.InputPorts()[0].Config.(component.FilePort)
	if !ok {
		t.Fatalf("input forecast_fixture config = %T, want FilePort", input.InputPorts()[0].Config)
	}
	if got, want := filePort.Type(), "file"; got != want {
		t.Fatalf("file port type = %q, want %q", got, want)
	}
	if got := input.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultRawSubject {
		t.Fatalf("input raw subject = %q, want %q", got, DefaultRawSubject)
	}
	if got := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultDecodedSubject {
		t.Fatalf("decoder decoded subject = %q, want %q", got, DefaultDecodedSubject)
	}
	if got := projector.InputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultDecodedSubject {
		t.Fatalf("projector decoded subject = %q, want %q", got, DefaultDecodedSubject)
	}
	if len(input.OutputPorts()) != 1 || len(decoder.OutputPorts()) != 1 {
		t.Fatalf("weather fixture components must not expose graph request ports")
	}
	graphPorts := projector.OutputPorts()
	if len(graphPorts) != 2 {
		t.Fatalf("projector graph ports = %d, want 2", len(graphPorts))
	}
	createPort, ok := graphPorts[0].Config.(component.NATSRequestPort)
	if !ok {
		t.Fatalf("graph_create config = %T, want NATSRequestPort", graphPorts[0].Config)
	}
	if createPort.Subject != weatherprojector.SubjectEntityCreateWithTriples ||
		createPort.Timeout != "250ms" ||
		createPort.Retries != 2 ||
		createPort.Interface.Type != "graph.CreateEntityWithTriplesRequest" {
		t.Fatalf("graph_create port = %+v", createPort)
	}
	updatePort, ok := graphPorts[1].Config.(component.NATSRequestPort)
	if !ok {
		t.Fatalf("graph_update config = %T, want NATSRequestPort", graphPorts[1].Config)
	}
	if updatePort.Subject != weatherprojector.SubjectEntityUpdateWithTriples ||
		updatePort.Interface.Type != "graph.UpdateEntityWithTriplesRequest" {
		t.Fatalf("graph_update port = %+v", updatePort)
	}
	requireProperty(t, input.ConfigSchema(), "fixture_path")
	requireProperty(t, input.ConfigSchema(), "provider")
	requireProperty(t, input.ConfigSchema(), "query_shape")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")
	requireProperty(t, projector.ConfigSchema(), "decoded_subject")
	requireProperty(t, projector.ConfigSchema(), "freshness")
	requireProperty(t, projector.ConfigSchema(), "max_observations")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add weather fixture input to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add weather decoder to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add weather projector to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect weather flow graph: %v", err)
	}
	requireEdge(t, fg.GetEdges(), input.Meta().Name, "raw_forecasts", decoder.Meta().Name, "raw_forecasts", DefaultRawSubject)
	requireEdge(t, fg.GetEdges(), decoder.Meta().Name, "decoded_forecasts", projector.Meta().Name, "decoded_forecasts", DefaultDecodedSubject)
}

func TestFixtureInputAndDecoderPublishBaseMessages(t *testing.T) {
	now := time.Date(2026, 6, 23, 13, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	input, err := NewFixtureInputComponent(FixtureInputConfig{
		Source:      "weather:fixture:test",
		FixturePath: fixturePath(),
		Registry:    registry,
		Bus:         bus,
		Clock:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Registry: registry,
		Clock:    func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := input.Initialize(); err != nil {
		t.Fatalf("initialize input: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}
	if err := input.PublishOnce(context.Background()); err != nil {
		t.Fatalf("publish fixture: %v", err)
	}
	raw := bus.singlePublished(t, DefaultRawSubject)
	rawEnvelope, err := message.NewDecoder(registry).Decode(raw.data)
	if err != nil {
		t.Fatalf("decode raw fixture payload: %v", err)
	}
	rawPayload, ok := rawEnvelope.Payload().(*RawForecastPayload)
	if !ok {
		t.Fatalf("raw payload = %T, want *RawForecastPayload", rawEnvelope.Payload())
	}
	if rawPayload.Source != "weather:fixture:test" ||
		rawPayload.Provider != weathercodec.ProviderOpenMeteo ||
		rawPayload.QueryShape != weathercodec.QueryShapePosition ||
		rawPayload.FixturePath != fixturePath() ||
		len(rawPayload.RawJSON) == 0 {
		t.Fatalf("raw payload = %+v", rawPayload)
	}

	if err := decoder.HandleRawMessage(context.Background(), raw.data); err != nil {
		t.Fatalf("decode raw fixture: %v", err)
	}
	decoded := bus.singlePublished(t, DefaultDecodedSubject)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decoded.data)
	if err != nil {
		t.Fatalf("decode decoded forecast payload: %v", err)
	}
	payload, ok := decodedEnvelope.Payload().(*DecodedForecastPayload)
	if !ok {
		t.Fatalf("decoded payload = %T, want *DecodedForecastPayload", decodedEnvelope.Payload())
	}
	if payload.Source != "weather:fixture:test" ||
		payload.Provider != weathercodec.ProviderOpenMeteo ||
		payload.QueryShape != weathercodec.QueryShapePosition ||
		payload.Forecast.Latitude != 38.9 ||
		payload.Forecast.Samples[0].WindSpeed10MKPH == nil ||
		*payload.Forecast.Samples[0].WindSpeed10MKPH != 12.5 {
		t.Fatalf("decoded payload = %+v", payload)
	}
	if got := decoder.DataFlow().MessagesPerSecond; got <= 0 {
		t.Fatalf("decoder messages per second = %v, want > 0", got)
	}
}

func TestDecoderPublishesOGCEDRPositionFixture(t *testing.T) {
	now := time.Date(2026, 6, 23, 14, 0, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	decoder, err := NewDecoderComponent(DecoderConfig{
		Registry: registry,
		Clock:    func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	raw := NewRawForecastPayload(
		"weather:fixture:ogc-edr",
		weathercodec.ProviderOGCEDR,
		weathercodec.QueryShapePosition,
		edrFixturePath(),
		"file:///tmp/ogc-edr-weather.json",
		now,
		readEDRFixture(t),
	)
	if err := decoder.HandleRawPayload(context.Background(), raw); err != nil {
		t.Fatalf("decode OGC EDR raw fixture: %v", err)
	}

	decoded := bus.singlePublished(t, DefaultDecodedSubject)
	envelope, err := message.NewDecoder(registry).Decode(decoded.data)
	if err != nil {
		t.Fatalf("decode decoded OGC EDR payload: %v", err)
	}
	payload, ok := envelope.Payload().(*DecodedForecastPayload)
	if !ok {
		t.Fatalf("decoded payload = %T, want *DecodedForecastPayload", envelope.Payload())
	}
	if payload.Source != "weather:fixture:ogc-edr" ||
		payload.Provider != weathercodec.ProviderOGCEDR ||
		payload.QueryShape != weathercodec.QueryShapePosition ||
		payload.Forecast.Longitude != -77.04 ||
		payload.Forecast.Latitude != 38.9 ||
		len(payload.Forecast.Samples) != 2 ||
		payload.Forecast.Samples[1].WindGusts10MKPH == nil ||
		*payload.Forecast.Samples[1].WindGusts10MKPH != 31.4 {
		t.Fatalf("decoded OGC EDR payload = %+v", payload)
	}
}

func TestDecoderRejectsUnsupportedWeatherShape(t *testing.T) {
	decoder, err := NewDecoderComponent(DecoderConfig{}, &recordingBus{})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	payload := NewRawForecastPayload(
		"weather:fixture:test",
		"ogc-edr",
		"area",
		fixturePath(),
		"file:///tmp/weather.json",
		time.Date(2026, 6, 23, 13, 45, 0, 0, time.UTC),
		readFixture(t),
	)
	err = decoder.HandleRawPayload(context.Background(), payload)
	if err == nil {
		t.Fatal("expected unsupported weather shape error")
	}
}

func TestProjectorConsumesDecodedForecastAndWritesGraphPlan(t *testing.T) {
	now := time.Date(2026, 6, 23, 14, 30, 0, 0, time.UTC)
	registry := payloadregistry.New()
	writer := &recordingWeatherPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Registry: registry,
		Projector: weatherprojector.NewProjector(weatherprojector.Config{
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerWeather: ownership.ExpectedOwnerToken(cop.OwnerWeather, "component-test"),
			},
			TraceID: "component-weather-001",
		}),
		Writer:          writer,
		Freshness:       45 * time.Minute,
		MaxObservations: 32,
		Clock:           func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	if err := projector.Initialize(); err != nil {
		t.Fatalf("initialize projector: %v", err)
	}

	payload := decodedOpenMeteoPayload(t, now)
	wire := mustBaseMessageJSON(t, DecodedForecastType, payload, "semops-processor-weather-decode", now)
	if err := projector.HandleDecodedMessage(context.Background(), wire); err != nil {
		t.Fatalf("handle decoded forecast: %v", err)
	}

	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	plan := writer.plans[0]
	if len(plan.Mutations) != 16 {
		t.Fatalf("mutations = %d, want 16 weather observations", len(plan.Mutations))
	}
	create := plan.Mutations[0].Create
	if plan.Mutations[0].Kind != weatherprojector.MutationCreate {
		t.Fatalf("first mutation = %+v, want create", plan.Mutations[0])
	}
	if create.OwnerToken != "semops.feed.weather#component-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "component-weather-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	requireWeatherTriple(t, create.Triples, cop.WeatherVariable, "temperature_2m")
	requireWeatherTriple(t, create.Triples, cop.WeatherValue, 29.4)
	requireWeatherTriple(t, create.Triples, cop.WeatherFreshUntil, now.Add(45*time.Minute))
	if got := projector.DataFlow().MessagesPerSecond; got <= 0 {
		t.Fatalf("projector messages per second = %f, want > 0", got)
	}
}

func TestProjectorRejectsObservationCapExceeded(t *testing.T) {
	now := time.Date(2026, 6, 23, 14, 45, 0, 0, time.UTC)
	writer := &recordingWeatherPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Writer:          writer,
		MaxObservations: 1,
		Clock:           func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	err = projector.HandleDecodedPayload(context.Background(), decodedOpenMeteoPayload(t, now))
	if err == nil {
		t.Fatal("expected max_observations error")
	}
	if !strings.Contains(err.Error(), "max_observations") {
		t.Fatalf("error = %v, want max_observations", err)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("plans = %d, want no graph writes", len(writer.plans))
	}
}

func TestProjectorReconcilesExistingWeatherObservationBirth(t *testing.T) {
	now := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)
	payload := decodedOpenMeteoPayload(t, now)
	observations, err := weatherprojector.ObservationsFromPointForecast(
		payload.Forecast,
		payload.RawRef,
		payload.ReceivedAt,
		DefaultFreshness,
	)
	if err != nil {
		t.Fatalf("build observations: %v", err)
	}
	firstEntityID := weatherprojector.EntityID("c360", "edge", observations[0].NativeID)
	writer := &recordingWeatherPlanWriter{
		failures: []error{
			&weatherprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      weatherprojector.MutationCreate,
				EntityID:  firstEntityID,
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
		},
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: weatherprojector.NewProjector(weatherprojector.Config{
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerWeather: ownership.ExpectedOwnerToken(cop.OwnerWeather, "component-test"),
			},
		}),
		Writer: writer,
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	if err := projector.HandleDecodedPayload(context.Background(), payload); err != nil {
		t.Fatalf("handle decoded with birth reconciliation: %v", err)
	}
	if len(writer.plans) != 2 {
		t.Fatalf("plans = %d, want create retry then update", len(writer.plans))
	}
	last := writer.plans[len(writer.plans)-1]
	if len(last.Mutations) != 16 || last.Mutations[0].Kind != weatherprojector.MutationUpdate {
		t.Fatalf("last plan = %+v, want first observation update after reconciling existing birth", last)
	}
}

func readEDRFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(edrFixturePath())
	if err != nil {
		t.Fatalf("read OGC EDR fixture: %v", err)
	}
	return data
}

func readFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath())
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func fixturePath() string {
	return filepath.Join("..", "..", "..", DefaultFixturePath)
}

func edrFixturePath() string {
	return filepath.Join("..", "..", "..", "fixtures", "weather", "ogc-edr-position.json")
}

func mustBaseMessageJSON(t *testing.T, msgType message.Type, payload message.Payload, source string, at time.Time) []byte {
	t.Helper()
	data, err := marshalBaseMessage(msgType, payload, source, at)
	if err != nil {
		t.Fatalf("marshal BaseMessage: %v", err)
	}
	return data
}

type publishedMessage struct {
	subject string
	data    []byte
}

type recordingBus struct {
	published []publishedMessage
	handlers  map[string]func(context.Context, *nats.Msg)
}

func (b *recordingBus) Publish(_ context.Context, subject string, data []byte) error {
	b.published = append(b.published, publishedMessage{subject: subject, data: append([]byte(nil), data...)})
	return nil
}

func (b *recordingBus) Subscribe(
	_ context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.handlers == nil {
		b.handlers = map[string]func(context.Context, *nats.Msg){}
	}
	b.handlers[subject] = handler
	return noopSubscription{}, nil
}

func (b *recordingBus) singlePublished(t *testing.T, subject string) publishedMessage {
	t.Helper()
	var matches []publishedMessage
	for _, published := range b.published {
		if published.subject == subject {
			matches = append(matches, published)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("published %q count = %d, want 1; all = %+v", subject, len(matches), b.published)
	}
	return matches[0]
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error {
	return nil
}

func requireEdge(
	t *testing.T,
	edges []flowgraph.FlowEdge,
	fromComponent string,
	fromPort string,
	toComponent string,
	toPort string,
	subject string,
) {
	t.Helper()
	want := flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: fromComponent, PortName: fromPort},
		To:           flowgraph.ComponentPortRef{ComponentName: toComponent, PortName: toPort},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: subject,
	}
	for _, edge := range edges {
		if edge.From == want.From &&
			edge.To == want.To &&
			edge.Pattern == want.Pattern &&
			edge.ConnectionID == want.ConnectionID {
			return
		}
	}
	t.Fatalf("missing flow edge %+v in %+v", want, edges)
}

func requireProperty(t *testing.T, schema component.ConfigSchema, name string) {
	t.Helper()
	if _, ok := schema.Properties[name]; !ok {
		t.Fatalf("missing config schema property %q in %+v", name, schema.Properties)
	}
}

type recordingWeatherPlanWriter struct {
	plans    []weatherprojector.Plan
	failures []error
}

func (w *recordingWeatherPlanWriter) Apply(_ context.Context, plan weatherprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.failures) == 0 {
		return nil
	}
	err := w.failures[0]
	w.failures = w.failures[1:]
	return err
}

func decodedOpenMeteoPayload(t *testing.T, receivedAt time.Time) *DecodedForecastPayload {
	t.Helper()
	raw := NewRawForecastPayload(
		"weather:fixture:test",
		weathercodec.ProviderOpenMeteo,
		weathercodec.QueryShapePosition,
		fixturePath(),
		"file:///tmp/weather.json",
		receivedAt,
		readFixture(t),
	)
	forecast, err := weathercodec.ParseOpenMeteoPointForecast(raw.RawJSON)
	if err != nil {
		t.Fatalf("parse Open-Meteo fixture: %v", err)
	}
	return NewDecodedForecastPayload(raw, forecast)
}

func requireWeatherTriple(t *testing.T, triples []message.Triple, predicate string, object any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate && triple.Object == object {
			return
		}
	}
	t.Fatalf("missing triple %s=%v in %#v", predicate, object, triples)
}
