package weather

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
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
	for name, lifecycle := range map[string]component.LifecycleComponent{
		"input":   input,
		"decoder": decoder,
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
	if len(input.OutputPorts()) != 1 || len(decoder.OutputPorts()) != 1 {
		t.Fatalf("weather fixture components must not expose graph request ports")
	}
	requireProperty(t, input.ConfigSchema(), "fixture_path")
	requireProperty(t, input.ConfigSchema(), "provider")
	requireProperty(t, input.ConfigSchema(), "query_shape")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add weather fixture input to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add weather decoder to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect weather flow graph: %v", err)
	}
	requireEdge(t, fg.GetEdges(), input.Meta().Name, "raw_forecasts", decoder.Meta().Name, "raw_forecasts", DefaultRawSubject)
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
