package cot

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/nats-io/nats.go"
)

func TestPayloadRegistryRoundTripsRawAndDecodedPayloads(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads should be idempotent: %v", err)
	}

	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	rawXML := mustCoTRawEvent(t, now)
	raw := NewRawEventPayload("cot:udp", "127.0.0.1:8087", now, rawXML)
	rawWire := mustBaseMessageJSON(t, RawEventType, raw, "semops-input-cot-udp", now)
	rawEnvelope, err := message.NewDecoder(registry).Decode(rawWire)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	if _, ok := rawEnvelope.Payload().(*RawEventPayload); !ok {
		t.Fatalf("raw payload type = %T, want *RawEventPayload", rawEnvelope.Payload())
	}

	event, err := cotcodec.Unmarshal(rawXML)
	if err != nil {
		t.Fatalf("unmarshal raw event: %v", err)
	}
	decoded := &DecodedEventPayload{
		Source:     "cot:decoder",
		RawRef:     "cot://raw/cot-udp/00000001",
		ReceivedAt: now,
		Event:      event,
	}
	decodedWire := mustBaseMessageJSON(t, DecodedEventType, decoded, "semops-processor-cot-decode", now)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decodedWire)
	if err != nil {
		t.Fatalf("decode decoded-event payload: %v", err)
	}
	if _, ok := decodedEnvelope.Payload().(*DecodedEventPayload); !ok {
		t.Fatalf("decoded payload type = %T, want *DecodedEventPayload", decodedEnvelope.Payload())
	}
}

func TestCoTComponentsExposeFlowgraphPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*UDPInputComponent)(nil)
	var _ component.LifecycleComponent = (*TCPInputComponent)(nil)
	var _ component.LifecycleComponent = (*DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*ProjectorComponent)(nil)

	bus := &recordingBus{}
	udpInput, err := NewUDPInputComponent(UDPInputConfig{ListenAddr: "127.0.0.1:8087"}, bus)
	if err != nil {
		t.Fatalf("new UDP input: %v", err)
	}
	tcpInput, err := NewTCPInputComponent(TCPInputConfig{ListenAddr: "127.0.0.1:8088"}, bus)
	if err != nil {
		t.Fatalf("new TCP input: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	projector, err := NewProjectorComponent(ProjectorConfig{Writer: &recordingPlanWriter{}}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	if udpInput.Meta().Type != "input" || tcpInput.Meta().Type != "input" {
		t.Fatalf("input types = %q/%q", udpInput.Meta().Type, tcpInput.Meta().Type)
	}
	if decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf("processor types = %q/%q", decoder.Meta().Type, projector.Meta().Type)
	}
	if got := udpInput.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultRawSubject {
		t.Fatalf("UDP raw subject = %q", got)
	}
	if got := tcpInput.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultRawSubject {
		t.Fatalf("TCP raw subject = %q", got)
	}
	if got := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultDecodedSubject {
		t.Fatalf("decoder decoded subject = %q", got)
	}
	if got := projector.OutputPorts()[0].Config.Type(); got != "nats-request" {
		t.Fatalf("projector graph output port type = %q", got)
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(udpInput.Meta().Name, udpInput); err != nil {
		t.Fatalf("add UDP input: %v", err)
	}
	if err := fg.AddComponentNode(tcpInput.Meta().Name, tcpInput); err != nil {
		t.Fatalf("add TCP input: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add decoder: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add projector: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect flowgraph: %v", err)
	}
	requireEdge(t, fg.GetEdges(), udpInput.Meta().Name, "raw_events", decoder.Meta().Name, "raw_events", DefaultRawSubject)
	requireEdge(t, fg.GetEdges(), tcpInput.Meta().Name, "raw_events", decoder.Meta().Name, "raw_events", DefaultRawSubject)
	requireEdge(t, fg.GetEdges(), decoder.Meta().Name, "decoded_events", projector.Meta().Name, "decoded_events", DefaultDecodedSubject)
}

func TestUDPInputPublishesRawBaseMessage(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	input, err := NewUDPInputComponent(UDPInputConfig{
		Source:     "udp:8087",
		ListenAddr: "127.0.0.1:8087",
		Clock:      func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	rawXML := mustCoTRawEvent(t, now)
	if err := input.PublishEvent(context.Background(), rawXML, "127.0.0.1:50000"); err != nil {
		t.Fatalf("publish event: %v", err)
	}

	published := bus.singlePublished(t, DefaultRawSubject)
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published raw message: %v", err)
	}
	payload, ok := envelope.Payload().(*RawEventPayload)
	if !ok {
		t.Fatalf("payload = %T, want *RawEventPayload", envelope.Payload())
	}
	if payload.Source != "udp:8087" || payload.RemoteAddr != "127.0.0.1:50000" {
		t.Fatalf("payload source/remote = %+v", payload)
	}
	if string(payload.RawXML) != string(rawXML) {
		t.Fatalf("raw XML changed across raw publish")
	}
}

func TestDecoderConsumesRawAndPublishesDecodedEvent(t *testing.T) {
	now := time.Date(2026, 6, 20, 13, 0, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source:   "decoder:test",
		Clock:    func() time.Time { return now },
		Registry: registry,
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	raw := NewRawEventPayload("udp:test", "127.0.0.1:50000", now, mustCoTRawEvent(t, now))
	rawWire := mustBaseMessageJSON(t, RawEventType, raw, "semops-input-cot-udp", now)
	if err := decoder.HandleRawMessage(context.Background(), rawWire); err != nil {
		t.Fatalf("handle raw: %v", err)
	}

	published := bus.singlePublished(t, DefaultDecodedSubject)
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published decoded message: %v", err)
	}
	payload, ok := envelope.Payload().(*DecodedEventPayload)
	if !ok {
		t.Fatalf("payload = %T, want *DecodedEventPayload", envelope.Payload())
	}
	if payload.Event.UID != "ANDROID-ALPHA" || payload.Event.Callsign != "Alpha Team" {
		t.Fatalf("decoded event = %+v", payload.Event)
	}
	if payload.RawRef == "" {
		t.Fatalf("decoded payload missing raw ref: %+v", payload)
	}
}

func TestProjectorConsumesDecodedEventAndWritesGraphPlan(t *testing.T) {
	now := time.Date(2026, 6, 20, 13, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Registry: registry,
		Projector: cotprojector.NewProjector(cotprojector.Config{
			Org:      "c360",
			Platform: "edge",
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerAsset: ownership.ExpectedOwnerToken(cop.OwnerAsset, "component-test"),
				cop.OwnerTAK:   ownership.ExpectedOwnerToken(cop.OwnerTAK, "component-test"),
			},
			TraceID: "component-test",
		}),
		Writer: writer,
		Clock:  func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	if err := projector.Initialize(); err != nil {
		t.Fatalf("initialize projector: %v", err)
	}

	payload := decodedPayloadFromRaw(t, "decoder:test", "cot://raw/test/00000001", now, mustCoTRawEvent(t, now))
	wire := mustBaseMessageJSON(t, DecodedEventType, payload, "semops-processor-cot-decode", now)
	if err := projector.HandleDecodedMessage(context.Background(), wire); err != nil {
		t.Fatalf("handle decoded: %v", err)
	}

	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	if len(writer.plans[0].Mutations) != 2 {
		t.Fatalf("mutations = %d, want asset + track create", len(writer.plans[0].Mutations))
	}
	trackCreate := writer.plans[0].Mutations[1].Create
	if trackCreate.OwnerToken != "semops.feed.tak#component-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	if trackCreate.TraceID != "component-test" {
		t.Fatalf("trace id = %q", trackCreate.TraceID)
	}
}

func TestProjectorReconcilesExistingBirths(t *testing.T) {
	now := time.Date(2026, 6, 20, 14, 0, 0, 0, time.UTC)
	bus := &recordingBus{}
	writer := &recordingPlanWriter{
		failures: []error{
			&cotprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      cotprojector.MutationCreate,
				EntityID:  cotprojector.EntityID("c360", "edge", cop.EntityAsset, "ANDROID-ALPHA"),
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
			&cotprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      cotprojector.MutationCreate,
				EntityID:  cotprojector.EntityID("c360", "edge", cop.EntityTrack, "ANDROID-ALPHA"),
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
		},
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: cotprojector.NewProjector(cotprojector.Config{Org: "c360", Platform: "edge"}),
		Writer:    writer,
		Clock:     func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	payload := decodedPayloadFromRaw(t, "decoder:test", "cot://raw/test/00000001", now, mustCoTRawEvent(t, now))
	if err := projector.HandleDecodedPayload(context.Background(), payload); err != nil {
		t.Fatalf("handle decoded with birth reconciliation: %v", err)
	}
	if len(writer.plans) != 3 {
		t.Fatalf("plans = %d, want create retry sequence", len(writer.plans))
	}
	last := writer.plans[len(writer.plans)-1]
	if len(last.Mutations) != 1 || last.Mutations[0].Kind != cotprojector.MutationUpdate {
		t.Fatalf("last plan = %+v, want update after reconciling existing births", last)
	}
}

type publishedMessage struct {
	subject string
	data    []byte
}

type recordingBus struct {
	published     []publishedMessage
	subscriptions []string
}

func (b *recordingBus) Publish(_ context.Context, subject string, data []byte) error {
	b.published = append(b.published, publishedMessage{
		subject: subject,
		data:    append([]byte(nil), data...),
	})
	return nil
}

func (b *recordingBus) Subscribe(
	_ context.Context,
	subject string,
	_ func(context.Context, *nats.Msg),
) (Subscription, error) {
	b.subscriptions = append(b.subscriptions, subject)
	return fakeSubscription{}, nil
}

func (b *recordingBus) singlePublished(t *testing.T, subject string) publishedMessage {
	t.Helper()
	var matches []publishedMessage
	for _, msg := range b.published {
		if msg.subject == subject {
			matches = append(matches, msg)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("published messages for %s = %d, want 1; all=%+v", subject, len(matches), b.published)
	}
	return matches[0]
}

type fakeSubscription struct{}

func (fakeSubscription) Unsubscribe() error { return nil }

type recordingPlanWriter struct {
	plans    []cotprojector.Plan
	failures []error
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan cotprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.failures) == 0 {
		return nil
	}
	err := w.failures[0]
	w.failures = w.failures[1:]
	return err
}

func mustCoTRawEvent(t *testing.T, now time.Time) []byte {
	t.Helper()
	raw, err := cotcodec.Marshal(cotcodec.Event{
		UID:      "ANDROID-ALPHA",
		Type:     cotcodec.TypeOperatorPosition,
		How:      cotcodec.DefaultHow,
		Time:     now.UTC(),
		Start:    now.UTC(),
		Stale:    now.Add(2 * time.Minute).UTC(),
		Callsign: "Alpha Team",
		Point: &cotcodec.Point{
			Lat: 30.2672,
			Lon: -97.7431,
			HAE: 188,
			CE:  5,
			LE:  8,
		},
	})
	if err != nil {
		t.Fatalf("marshal CoT event: %v", err)
	}
	return raw
}

func decodedPayloadFromRaw(
	t *testing.T,
	source string,
	rawRef string,
	receivedAt time.Time,
	rawXML []byte,
) *DecodedEventPayload {
	t.Helper()
	event, err := cotcodec.Unmarshal(rawXML)
	if err != nil {
		t.Fatalf("unmarshal CoT event: %v", err)
	}
	return &DecodedEventPayload{
		Source:     source,
		RawRef:     rawRef,
		ReceivedAt: receivedAt.UTC(),
		Event:      event,
	}
}

func mustBaseMessageJSON(
	t *testing.T,
	msgType message.Type,
	payload message.Payload,
	source string,
	observedAt time.Time,
) []byte {
	t.Helper()
	data, err := marshalBaseMessage(msgType, payload, source, observedAt)
	if err != nil {
		t.Fatalf("marshal base message: %v", err)
	}
	return data
}

func requireEdge(
	t *testing.T,
	edges []flowgraph.FlowEdge,
	fromComponent string,
	fromPort string,
	toComponent string,
	toPort string,
	connectionID string,
) {
	t.Helper()
	for _, edge := range edges {
		if edge.From.ComponentName == fromComponent &&
			edge.From.PortName == fromPort &&
			edge.To.ComponentName == toComponent &&
			edge.To.PortName == toPort &&
			edge.Pattern == flowgraph.PatternStream &&
			edge.ConnectionID == connectionID {
			return
		}
	}
	t.Fatalf("missing edge %s.%s -> %s.%s (%s) in %+v", fromComponent, fromPort, toComponent, toPort, connectionID, edges)
}

func TestRecordingPlanWriterReturnsFailures(t *testing.T) {
	want := errors.New("boom")
	writer := &recordingPlanWriter{failures: []error{want}}
	err := writer.Apply(context.Background(), cotprojector.Plan{})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
}

func TestPayloadJSONRejectsInvalidRawEvent(t *testing.T) {
	_, err := json.Marshal(message.NewBaseMessage(RawEventType, &RawEventPayload{}, "test"))
	if err == nil {
		t.Fatal("expected invalid raw payload to fail BaseMessage marshal")
	}
}
