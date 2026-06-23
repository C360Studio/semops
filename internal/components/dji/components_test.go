package dji

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	djicodec "github.com/c360studio/semops/pkg/adapters/dji"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

func TestPayloadRegistryRoundTripsRawAndDecodedTelemetry(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads should be idempotent: %v", err)
	}

	rawJSON := readFixture(t)
	now := time.Date(2026, 6, 22, 19, 0, 0, 0, time.UTC)
	raw := NewRawTelemetryPayload("dji:fixture:test", DefaultFixturePath, "file:///tmp/dji.json", now, rawJSON)
	rawWire := mustBaseMessageJSON(t, RawTelemetryType, raw, "semops-input-dji-fixture", now)
	rawEnvelope, err := message.NewDecoder(registry).Decode(rawWire)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	gotRaw, ok := rawEnvelope.Payload().(*RawTelemetryPayload)
	if !ok {
		t.Fatalf("raw payload type = %T, want *RawTelemetryPayload", rawEnvelope.Payload())
	}
	record, err := djicodec.ParseTelemetryRecord(gotRaw.RawJSON)
	if err != nil {
		t.Fatalf("parse raw telemetry: %v", err)
	}

	decoded := NewDecodedTelemetryPayload(gotRaw, record)
	decodedWire := mustBaseMessageJSON(t, DecodedTelemetryType, decoded, "semops-processor-dji-decode", record.ObservedAt)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decodedWire)
	if err != nil {
		t.Fatalf("decode decoded payload: %v", err)
	}
	gotDecoded, ok := decodedEnvelope.Payload().(*DecodedTelemetryPayload)
	if !ok {
		t.Fatalf("decoded payload type = %T, want *DecodedTelemetryPayload", decodedEnvelope.Payload())
	}
	if gotDecoded.Record.Source.SourceID != "dji://fixture/matrice-350/alpha" {
		t.Fatalf("decoded record = %+v", gotDecoded.Record)
	}
}

func TestDJIComponentsExposeFileAndStreamPorts(t *testing.T) {
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
		t.Fatalf("input telemetry_fixture config = %T, want FilePort", input.InputPorts()[0].Config)
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
		t.Fatalf("DJI fixture components must not expose graph request ports")
	}
	requireProperty(t, input.ConfigSchema(), "fixture_path")
	requireProperty(t, input.ConfigSchema(), "raw_subject")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add DJI fixture input to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add DJI decoder to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect DJI flow graph: %v", err)
	}
	requireEdge(t, fg.GetEdges(), input.Meta().Name, "raw_telemetry", decoder.Meta().Name, "raw_telemetry", DefaultRawSubject)
}

func TestFixtureInputAndDecoderPublishBaseMessages(t *testing.T) {
	now := time.Date(2026, 6, 22, 19, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	input, err := NewFixtureInputComponent(FixtureInputConfig{
		Source:      "dji:fixture:test",
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
	rawPayload, ok := rawEnvelope.Payload().(*RawTelemetryPayload)
	if !ok {
		t.Fatalf("raw payload = %T, want *RawTelemetryPayload", rawEnvelope.Payload())
	}
	if rawPayload.Source != "dji:fixture:test" ||
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
		t.Fatalf("decode decoded telemetry payload: %v", err)
	}
	payload, ok := decodedEnvelope.Payload().(*DecodedTelemetryPayload)
	if !ok {
		t.Fatalf("decoded payload = %T, want *DecodedTelemetryPayload", decodedEnvelope.Payload())
	}
	if payload.Source != "dji:fixture:test" ||
		payload.Record.Camera.Payload != "Zenmuse H20T" ||
		payload.Record.CommandAuthority.RemoteCommandsEnabled {
		t.Fatalf("decoded payload = %+v", payload)
	}
	if got := decoder.DataFlow().MessagesPerSecond; got <= 0 {
		t.Fatalf("decoder messages per second = %v, want > 0", got)
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
