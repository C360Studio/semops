package mavlink

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
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

	now := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	raw := NewRawFramePayload("mavlink:udp", "127.0.0.1:14551", now, []byte{0xfd, 0x00, 0x00})
	rawWire := mustBaseMessageJSON(t, RawFrameType, raw, "semops-input-mavlink-udp", now)
	rawEnvelope, err := message.NewDecoder(registry).Decode(rawWire)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	if _, ok := rawEnvelope.Payload().(*RawFramePayload); !ok {
		t.Fatalf("raw payload type = %T, want *RawFramePayload", rawEnvelope.Payload())
	}

	decoded := &DecodedPacketPayload{
		Source:      "mavlink:decoder",
		RawRef:      "mavlink://raw/mavlink-udp/00000001",
		ReceivedAt:  now,
		MessageID:   mavcodec.MessageIDHeartbeat,
		SystemID:    42,
		ComponentID: 7,
		Frame:       []byte{0xfd, 0x00, 0x00},
	}
	decodedWire := mustBaseMessageJSON(t, DecodedPacketType, decoded, "semops-processor-mavlink-decode", now)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decodedWire)
	if err != nil {
		t.Fatalf("decode decoded-packet payload: %v", err)
	}
	if _, ok := decodedEnvelope.Payload().(*DecodedPacketPayload); !ok {
		t.Fatalf("decoded payload type = %T, want *DecodedPacketPayload", decodedEnvelope.Payload())
	}
}

func TestMAVLinkComponentsExposeFlowgraphPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*UDPInputComponent)(nil)
	var _ component.LifecycleComponent = (*DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*ProjectorComponent)(nil)

	bus := &recordingBus{}
	input, err := NewUDPInputComponent(UDPInputConfig{ListenAddr: "127.0.0.1:14550"}, bus)
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	projector, err := NewProjectorComponent(ProjectorConfig{Writer: &recordingPlanWriter{}}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	if input.Meta().Type != "input" {
		t.Fatalf("input type = %q", input.Meta().Type)
	}
	if decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf("processor types = %q/%q", decoder.Meta().Type, projector.Meta().Type)
	}
	if got := input.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultRawSubject {
		t.Fatalf("input raw subject = %q", got)
	}
	if got := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultDecodedSubject {
		t.Fatalf("decoder decoded subject = %q", got)
	}
	if got := projector.OutputPorts()[0].Config.Type(); got != "nats-request" {
		t.Fatalf("projector graph output port type = %q", got)
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add input: %v", err)
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
	requireEdge(t, fg.GetEdges(), input.Meta().Name, "raw_frames", decoder.Meta().Name, "raw_frames", DefaultRawSubject)
	requireEdge(t, fg.GetEdges(), decoder.Meta().Name, "decoded_packets", projector.Meta().Name, "decoded_packets", DefaultDecodedSubject)
}

func TestUDPInputPublishesRawBaseMessage(t *testing.T) {
	now := time.Date(2026, 6, 20, 9, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	input, err := NewUDPInputComponent(UDPInputConfig{
		Source:     "udp:14550",
		ListenAddr: "127.0.0.1:14550",
		Clock:      func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	frame := mustHeartbeatFrame(t)
	if err := input.PublishFrame(context.Background(), frame, "127.0.0.1:50000"); err != nil {
		t.Fatalf("publish frame: %v", err)
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
	payload, ok := envelope.Payload().(*RawFramePayload)
	if !ok {
		t.Fatalf("payload = %T, want *RawFramePayload", envelope.Payload())
	}
	if payload.Source != "udp:14550" || payload.RemoteAddr != "127.0.0.1:50000" {
		t.Fatalf("payload source/remote = %+v", payload)
	}
	if string(payload.Frame) != string(frame) {
		t.Fatalf("frame changed across raw publish")
	}
}

func TestDecoderConsumesRawAndPublishesDecodedPacket(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
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

	frame := mustHeartbeatFrame(t)
	raw := NewRawFramePayload("udp:test", "127.0.0.1:50000", now, frame)
	rawWire := mustBaseMessageJSON(t, RawFrameType, raw, "semops-input-mavlink-udp", now)
	if err := decoder.HandleRawMessage(context.Background(), rawWire); err != nil {
		t.Fatalf("handle raw: %v", err)
	}

	published := bus.singlePublished(t, DefaultDecodedSubject)
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published decoded message: %v", err)
	}
	payload, ok := envelope.Payload().(*DecodedPacketPayload)
	if !ok {
		t.Fatalf("payload = %T, want *DecodedPacketPayload", envelope.Payload())
	}
	if payload.MessageID != mavcodec.MessageIDHeartbeat || payload.SystemID != 42 || payload.ComponentID != 7 {
		t.Fatalf("decoded packet metadata = %+v", payload)
	}
	if payload.RawRef == "" {
		t.Fatalf("decoded payload missing raw ref: %+v", payload)
	}
}

func TestProjectorConsumesDecodedPacketAndWritesGraphPlan(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Registry: registry,
		Projector: mavprojector.NewProjector(mavprojector.Config{
			Org:      "c360",
			Platform: "edge",
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerAsset:   ownership.ExpectedOwnerToken(cop.OwnerAsset, "component-test"),
				cop.OwnerMAVLink: ownership.ExpectedOwnerToken(cop.OwnerMAVLink, "component-test"),
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

	payload := decodedPayloadFromFrame(t, "decoder:test", "mavlink://raw/test/00000001", now, mustHeartbeatFrame(t))
	wire := mustBaseMessageJSON(t, DecodedPacketType, payload, "semops-processor-mavlink-decode", now)
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
	if trackCreate.OwnerToken != "semops.feed.mavlink#component-test" {
		t.Fatalf("track owner token = %q", trackCreate.OwnerToken)
	}
	if trackCreate.TraceID != "component-test" {
		t.Fatalf("trace id = %q", trackCreate.TraceID)
	}
}

func TestProjectorConsumesDecodedCommandAckAndWritesControlTask(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 45, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Registry: registry,
		Projector: mavprojector.NewProjector(mavprojector.Config{
			Org:      "c360",
			Platform: "edge",
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerAsset:   ownership.ExpectedOwnerToken(cop.OwnerAsset, "component-test"),
				cop.OwnerMAVLink: ownership.ExpectedOwnerToken(cop.OwnerMAVLink, "component-test"),
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

	frame, err := mavcodec.NewGenerator(42, 7).GenerateCommandAck(mavcodec.CommandAckMessage{
		Command:           mavcodec.CommandComponentArmDisarm,
		Result:            mavcodec.MAVResultAccepted,
		Progress:          100,
		TargetSystemID:    255,
		TargetComponentID: 1,
	})
	if err != nil {
		t.Fatalf("generate command ack: %v", err)
	}
	payload := decodedPayloadFromFrame(t, "decoder:test", "mavlink://raw/test/00000002", now, frame)
	wire := mustBaseMessageJSON(t, DecodedPacketType, payload, "semops-processor-mavlink-decode", now)
	if err := projector.HandleDecodedMessage(context.Background(), wire); err != nil {
		t.Fatalf("handle decoded command ack: %v", err)
	}

	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	if len(writer.plans[0].Mutations) != 2 {
		t.Fatalf("mutations = %d, want asset + task create", len(writer.plans[0].Mutations))
	}
	taskCreate := writer.plans[0].Mutations[1].Create
	if taskCreate.Entity.ID != "c360.edge.cop.mavlink.task.system-42-command-400-target-255-1" {
		t.Fatalf("task id = %q", taskCreate.Entity.ID)
	}
	if taskCreate.IndexingProfile != cop.MAVLinkCommandTaskContract().IndexingProfile {
		t.Fatalf("task indexing profile = %q", taskCreate.IndexingProfile)
	}
	requireMessageTriple(t, taskCreate.Triples, cop.TaskTarget, "c360.edge.cop.mavlink.asset.system-42")
	requireMessageTriple(t, taskCreate.Triples, cop.ProvenanceSourceRef, "mavlink://raw/test/00000002")
}

func TestProjectorReconcilesExistingBirths(t *testing.T) {
	now := time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)
	bus := &recordingBus{}
	writer := &recordingPlanWriter{
		failures: []error{
			&mavprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      mavprojector.MutationCreate,
				EntityID:  "c360.edge.cop.mavlink.asset.system-42",
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
			&mavprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      mavprojector.MutationCreate,
				EntityID:  "c360.edge.cop.mavlink.track.system-42",
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
		},
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: mavprojector.NewProjector(mavprojector.Config{Org: "c360", Platform: "edge"}),
		Writer:    writer,
		Clock:     func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	payload := decodedPayloadFromFrame(t, "decoder:test", "mavlink://raw/test/00000001", now, mustHeartbeatFrame(t))
	if err := projector.HandleDecodedPayload(context.Background(), payload); err != nil {
		t.Fatalf("handle decoded with birth reconciliation: %v", err)
	}
	if len(writer.plans) != 3 {
		t.Fatalf("plans = %d, want create retry sequence", len(writer.plans))
	}
	last := writer.plans[len(writer.plans)-1]
	if len(last.Mutations) != 1 || last.Mutations[0].Kind != mavprojector.MutationUpdate {
		t.Fatalf("last plan = %+v, want update after reconciling existing births", last)
	}
}

func TestProjectorReconcilesTextEntityExistsFailures(t *testing.T) {
	now := time.Date(2026, 6, 20, 11, 15, 0, 0, time.UTC)
	writer := &recordingPlanWriter{
		failures: []error{
			errors.New("request create_with_triples: entity_already_exists: entity already exists: c360.edge.cop.mavlink.asset.system-42"),
			errors.New("request create_with_triples: entity_already_exists: entity already exists: c360.edge.cop.mavlink.track.system-42"),
		},
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: mavprojector.NewProjector(mavprojector.Config{Org: "c360", Platform: "edge"}),
		Writer:    writer,
		Clock:     func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	payload := decodedPayloadFromFrame(t, "decoder:test", "mavlink://raw/test/00000001", now, mustHeartbeatFrame(t))
	if err := projector.HandleDecodedPayload(context.Background(), payload); err != nil {
		t.Fatalf("handle decoded with text birth reconciliation: %v", err)
	}
	if len(writer.plans) != 3 {
		t.Fatalf("plans = %d, want create retry sequence", len(writer.plans))
	}
	last := writer.plans[len(writer.plans)-1]
	if len(last.Mutations) != 1 || last.Mutations[0].Kind != mavprojector.MutationUpdate {
		t.Fatalf("last plan = %+v, want update after reconciling existing births", last)
	}
}

func TestProjectorSerializesConcurrentGraphWrites(t *testing.T) {
	now := time.Date(2026, 6, 20, 11, 30, 0, 0, time.UTC)
	writer := newBlockingPlanWriter()
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: mavprojector.NewProjector(mavprojector.Config{Org: "c360", Platform: "edge"}),
		Writer:    writer,
		Clock:     func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	payload := decodedPayloadFromFrame(t, "decoder:test", "mavlink://raw/test/00000001", now, mustHeartbeatFrame(t))

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- projector.HandleDecodedPayload(ctx, payload)
	}()

	select {
	case <-writer.firstEntered:
	case <-ctx.Done():
		t.Fatalf("first graph write did not start: %v", ctx.Err())
	}

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- projector.HandleDecodedPayload(ctx, payload)
	}()

	select {
	case err := <-secondDone:
		t.Fatalf("second graph write completed while first write was active: %v", err)
	case err := <-writer.concurrentApply:
		t.Fatalf("projector allowed concurrent graph apply: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(writer.releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("first graph write: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second graph write: %v", err)
	}
}

func requireMessageTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
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
	plans    []mavprojector.Plan
	failures []error
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan mavprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.failures) == 0 {
		return nil
	}
	err := w.failures[0]
	w.failures = w.failures[1:]
	return err
}

type blockingPlanWriter struct {
	mu              sync.Mutex
	active          bool
	applyCount      int
	firstEntered    chan struct{}
	releaseFirst    chan struct{}
	concurrentApply chan error
}

func newBlockingPlanWriter() *blockingPlanWriter {
	return &blockingPlanWriter{
		firstEntered:    make(chan struct{}),
		releaseFirst:    make(chan struct{}),
		concurrentApply: make(chan error, 1),
	}
}

func (w *blockingPlanWriter) Apply(ctx context.Context, _ mavprojector.Plan) error {
	w.mu.Lock()
	if w.active {
		err := errors.New("concurrent graph apply")
		w.mu.Unlock()
		select {
		case w.concurrentApply <- err:
		default:
		}
		return err
	}
	w.active = true
	w.applyCount++
	applyCount := w.applyCount
	if applyCount == 1 {
		close(w.firstEntered)
	}
	w.mu.Unlock()

	if applyCount == 1 {
		select {
		case <-w.releaseFirst:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	w.mu.Lock()
	w.active = false
	w.mu.Unlock()
	return nil
}

func mustHeartbeatFrame(t *testing.T) []byte {
	t.Helper()
	frame, err := mavcodec.NewGenerator(42, 7).GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	return frame
}

func decodedPayloadFromFrame(
	t *testing.T,
	source string,
	rawRef string,
	receivedAt time.Time,
	frame []byte,
) *DecodedPacketPayload {
	t.Helper()
	parser := mavcodec.NewParser()
	packets, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("parse frame: %v", err)
	}
	if len(packets) != 1 {
		t.Fatalf("packets = %d, want 1", len(packets))
	}
	packet := packets[0]
	return &DecodedPacketPayload{
		Source:      source,
		RawRef:      rawRef,
		ReceivedAt:  receivedAt.UTC(),
		Version:     packet.Version,
		Sequence:    packet.Sequence,
		SystemID:    packet.SystemID,
		ComponentID: packet.ComponentID,
		MessageID:   packet.MessageID,
		Checksum:    packet.Checksum,
		Frame:       append([]byte(nil), frame...),
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
	err := writer.Apply(context.Background(), mavprojector.Plan{})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
}

func TestPayloadJSONRejectsInvalidRawFrame(t *testing.T) {
	_, err := json.Marshal(message.NewBaseMessage(RawFrameType, &RawFramePayload{}, "test"))
	if err == nil {
		t.Fatal("expected invalid raw payload to fail BaseMessage marshal")
	}
}
