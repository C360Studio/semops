package klv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

func TestKLVComponentsExposeFlowgraphPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*MediaRefInputComponent)(nil)
	var _ component.LifecycleComponent = (*DemuxComponent)(nil)
	var _ component.LifecycleComponent = (*DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*ProjectorComponent)(nil)

	input, err := NewMediaRefInputComponent(MediaRefInputConfig{
		MediaPath:    "/tmp/semops-klv-fixtures",
		MediaPattern: "*.ts",
	})
	if err != nil {
		t.Fatalf("new media-ref input: %v", err)
	}
	demux, err := NewDemuxComponent(DemuxConfig{})
	if err != nil {
		t.Fatalf("new demux: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	projector, err := NewProjectorComponent(ProjectorConfig{WriteTimeout: 250 * time.Millisecond, WriteRetries: 2})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	for name, lifecycle := range map[string]component.LifecycleComponent{
		"input":     input,
		"demux":     demux,
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
		t.Fatalf("input type = %q, want input", input.Meta().Type)
	}
	if demux.Meta().Type != "processor" || decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf(
			"processor types = %q/%q/%q, want processor/processor/processor",
			demux.Meta().Type,
			decoder.Meta().Type,
			projector.Meta().Type,
		)
	}
	if got := input.InputPorts()[0].Config.Type(); got != "file" {
		t.Fatalf("input media_files port type = %q, want file", got)
	}
	if got := input.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultMediaRefSubject {
		t.Fatalf("media-ref subject = %q, want %q", got, DefaultMediaRefSubject)
	}
	if got := demux.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultPacketSubject {
		t.Fatalf("packet subject = %q, want %q", got, DefaultPacketSubject)
	}
	if got := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject; got != DefaultFrameSubject {
		t.Fatalf("frame subject = %q, want %q", got, DefaultFrameSubject)
	}
	graphPorts := projector.OutputPorts()
	if got := graphPorts[0].Config.Type(); got != "nats-request" {
		t.Fatalf("projector graph_create port type = %q, want nats-request", got)
	}
	if got := graphPorts[0].Config.(component.NATSRequestPort).Timeout; got != "250ms" {
		t.Fatalf("projector write timeout = %q, want 250ms", got)
	}
	if got := graphPorts[0].Config.(component.NATSRequestPort).Retries; got != 2 {
		t.Fatalf("projector write retries = %d, want 2", got)
	}

	requireProperty(t, input.ConfigSchema(), "media_ref_subject")
	requireProperty(t, demux.ConfigSchema(), "max_packet_bytes")
	requireProperty(t, decoder.ConfigSchema(), "supported_subset")
	requireProperty(t, decoder.ConfigSchema(), "max_packet_bytes")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	for _, comp := range []component.Discoverable{input, demux, decoder, projector} {
		if err := fg.AddComponentNode(comp.Meta().Name, comp); err != nil {
			t.Fatalf("add %s: %v", comp.Meta().Name, err)
		}
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect KLV flow graph: %v", err)
	}
	requireEdge(t, fg.GetEdges(), input.Meta().Name, "media_refs", demux.Meta().Name, "media_refs", DefaultMediaRefSubject)
	requireEdge(t, fg.GetEdges(), demux.Meta().Name, "klv_packets", decoder.Meta().Name, "klv_packets", DefaultPacketSubject)
	requireEdge(t, fg.GetEdges(), decoder.Meta().Name, "misb0601_frames", projector.Meta().Name, "misb0601_frames", DefaultFrameSubject)
}

func TestKLVComponentConstructorsValidateNegativeBounds(t *testing.T) {
	if _, err := NewDemuxComponent(DemuxConfig{MaxPacketBytes: -1}); err == nil {
		t.Fatal("expected negative max_packet_bytes to fail")
	}
	if _, err := NewDecoderComponent(DecoderConfig{MaxPacketBytes: -1}); err == nil {
		t.Fatal("expected negative decoder max_packet_bytes to fail")
	}
	if _, err := NewProjectorComponent(ProjectorConfig{WriteTimeout: -time.Second}); err == nil {
		t.Fatal("expected negative write_timeout to fail")
	}
}

func TestKLVMediaRefInputPublishesDiscoveredLocalFiles(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "fixture.ts")
	if err := os.WriteFile(mediaPath, []byte("mpeg-ts placeholder"), 0o644); err != nil {
		t.Fatalf("write media fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.bin"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write ignored fixture: %v", err)
	}
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	bus := &klvRecordingBus{}
	observedAt := time.Date(2026, 6, 22, 17, 30, 0, 0, time.UTC)
	input, err := NewMediaRefInputComponent(MediaRefInputConfig{
		Source:       "klv:media-ref:test",
		MediaPath:    dir,
		MediaPattern: "*.ts",
		Bus:          bus,
		Clock:        func() time.Time { return observedAt },
	})
	if err != nil {
		t.Fatalf("new media-ref input: %v", err)
	}
	if err := input.Initialize(); err != nil {
		t.Fatalf("initialize media-ref input: %v", err)
	}
	if err := input.Start(context.Background()); err != nil {
		t.Fatalf("start media-ref input: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("published %d media refs, want 1", len(bus.published))
	}
	published := bus.published[0]
	if published.subject != DefaultMediaRefSubject {
		t.Fatalf("published subject = %q, want %q", published.subject, DefaultMediaRefSubject)
	}
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode media-ref BaseMessage: %v", err)
	}
	payload, ok := envelope.Payload().(*MediaRefPayload)
	if !ok {
		t.Fatalf("published payload = %T, want *MediaRefPayload", envelope.Payload())
	}
	if payload.Source != "klv:media-ref:test" {
		t.Fatalf("media-ref source = %q", payload.Source)
	}
	if !strings.HasPrefix(payload.URI, "file://") || !strings.Contains(payload.URI, "fixture.ts") {
		t.Fatalf("media-ref URI = %q, want file URI for fixture.ts", payload.URI)
	}
	if payload.FixtureKind != "local-file" {
		t.Fatalf("fixture kind = %q, want local-file", payload.FixtureKind)
	}
	if !payload.ReceivedAt.Equal(observedAt) {
		t.Fatalf("received_at = %s, want %s", payload.ReceivedAt, observedAt)
	}
	if got := input.DataFlow().LastActivity; !got.Equal(observedAt) {
		t.Fatalf("media-ref input last activity = %s, want %s", got, observedAt)
	}
}

func TestKLVDecoderComponentPublishesDecodedFrameFromPacketMessage(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	bus := &klvRecordingBus{}
	decoder, err := NewDecoderComponent(DecoderConfig{Registry: registry, Bus: bus})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	receivedAt := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	frameTime := time.Date(2026, 6, 22, 15, 59, 58, 0, time.UTC)
	packet := NewPacketPayload(
		"klv:demux",
		"object://semops/klv/deterministic.ts",
		receivedAt,
		buildMISB0601Packet(
			misbField(misbTagPrecisionTimeStamp, beU64(uint64(frameTime.UnixMicro()))),
			misbField(misbTagPlatformDesignation, []byte("SYNTHETIC-UAS-1")),
			misbField(misbTagSensorLatitude, beI32(0x10000000)),
			misbField(misbTagSensorLongitude, beI32(-0x20000000)),
			misbField(misbTagFrameCenterLatitude, beI32(-0x10000000)),
			misbField(misbTagFrameCenterLongitude, beI32(0x20000000)),
		),
	)
	packet.PacketRef = "klv://packet/deterministic/00000001"
	packetWire, err := marshalBaseMessage(PacketType, packet, "semops-processor-klv-demux", receivedAt)
	if err != nil {
		t.Fatalf("marshal packet: %v", err)
	}

	if err := decoder.HandlePacketMessage(context.Background(), packetWire); err != nil {
		t.Fatalf("handle packet message: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("published %d messages, want 1", len(bus.published))
	}
	published := bus.published[0]
	if published.subject != DefaultFrameSubject {
		t.Fatalf("published subject = %q, want %q", published.subject, DefaultFrameSubject)
	}
	if published.subject == SubjectEntityCreateWithTriples || published.subject == SubjectEntityUpdateWithTriples {
		t.Fatalf("decoder published graph mutation subject %q", published.subject)
	}

	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published frame: %v", err)
	}
	frame, ok := envelope.Payload().(*MISB0601FramePayload)
	if !ok {
		t.Fatalf("published payload = %T, want *MISB0601FramePayload", envelope.Payload())
	}
	if frame.Source != DefaultDecodeSource {
		t.Fatalf("frame source = %q, want %q", frame.Source, DefaultDecodeSource)
	}
	if !frame.FrameTime.Equal(frameTime) {
		t.Fatalf("frame time = %s, want %s", frame.FrameTime, frameTime)
	}
	requireField(t, frame.Fields, "SensorLatitude")
	requireField(t, frame.Fields, "FrameCenterLongitude")
	if got := decoder.DataFlow().MessagesPerSecond; got <= 0 {
		t.Fatalf("decoder messages per second = %f, want > 0", got)
	}
}

func TestKLVDecoderComponentMaterializesStorageRefPacketMessage(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	bus := &klvRecordingBus{}
	receivedAt := time.Date(2026, 6, 22, 16, 30, 0, 0, time.UTC)
	frameTime := time.Date(2026, 6, 22, 16, 29, 58, 0, time.UTC)
	packetBytes := buildMISB0601Packet(
		misbField(misbTagPrecisionTimeStamp, beU64(uint64(frameTime.UnixMicro()))),
		misbField(misbTagPlatformDesignation, []byte("SYNTHETIC-UAS-STORAGE")),
		misbField(misbTagFrameCenterLatitude, beI32(-0x10000000)),
		misbField(misbTagFrameCenterLongitude, beI32(0x20000000)),
	)
	materializer := &recordingPacketMaterializer{
		result: MaterializedPacket{Bytes: packetBytes},
	}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Registry:           registry,
		Bus:                bus,
		PacketMaterializer: materializer,
		MaxPacketBytes:     len(packetBytes) + 1,
	})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}
	packet := NewPacketPayload("klv:demux", "object://semops/klv/deterministic.ts", receivedAt, nil)
	packet.PacketRef = "klv://packet/deterministic/00000002"
	packet.StorageRef = "object://semops/klv/packets/00000002.bin"
	packet.ByteLength = len(packetBytes)
	packetWire, err := marshalBaseMessage(PacketType, packet, "semops-processor-klv-demux", receivedAt)
	if err != nil {
		t.Fatalf("marshal packet: %v", err)
	}

	if err := decoder.HandlePacketMessage(context.Background(), packetWire); err != nil {
		t.Fatalf("handle storage-ref packet message: %v", err)
	}
	if materializer.maxBytes != len(packetBytes)+1 {
		t.Fatalf("materializer max bytes = %d, want %d", materializer.maxBytes, len(packetBytes)+1)
	}
	if materializer.packet.StorageRef != packet.StorageRef {
		t.Fatalf("materializer storage ref = %q, want %q", materializer.packet.StorageRef, packet.StorageRef)
	}
	if !materializer.cleaned {
		t.Fatal("expected materialized packet cleanup to run")
	}
	if len(bus.published) != 1 {
		t.Fatalf("published %d messages, want 1", len(bus.published))
	}
	envelope, err := message.NewDecoder(registry).Decode(bus.published[0].data)
	if err != nil {
		t.Fatalf("decode published frame: %v", err)
	}
	frame, ok := envelope.Payload().(*MISB0601FramePayload)
	if !ok {
		t.Fatalf("published payload = %T, want *MISB0601FramePayload", envelope.Payload())
	}
	if frame.PlatformDesignation != "SYNTHETIC-UAS-STORAGE" {
		t.Fatalf("platform designation = %q", frame.PlatformDesignation)
	}
	if !frame.FrameTime.Equal(frameTime) {
		t.Fatalf("frame time = %s, want %s", frame.FrameTime, frameTime)
	}
}

func TestKLVDecoderComponentRejectsUnboundedStorageRefPackets(t *testing.T) {
	now := time.Now().UTC()
	packetBytes := buildMISB0601Packet(misbField(misbTagPlatformDesignation, []byte("SYNTHETIC-UAS-1")))
	tests := []struct {
		name string
		cfg  DecoderConfig
		pkt  *PacketPayload
		want string
	}{
		{
			name: "missing materializer",
			cfg:  DecoderConfig{Bus: &klvRecordingBus{}},
			pkt: func() *PacketPayload {
				packet := NewPacketPayload("klv:demux", "object://semops/klv/deterministic.ts", now, nil)
				packet.PacketRef = "klv://packet/deterministic/00000003"
				packet.StorageRef = "object://semops/klv/packets/00000003.bin"
				return packet
			}(),
			want: "storage_ref-only decode requires a bounded packet materializer",
		},
		{
			name: "metadata byte length",
			cfg:  DecoderConfig{Bus: &klvRecordingBus{}, MaxPacketBytes: len(packetBytes) - 1},
			pkt: func() *PacketPayload {
				packet := NewPacketPayload("klv:demux", "object://semops/klv/deterministic.ts", now, nil)
				packet.PacketRef = "klv://packet/deterministic/00000004"
				packet.StorageRef = "object://semops/klv/packets/00000004.bin"
				packet.ByteLength = len(packetBytes)
				return packet
			}(),
			want: "byte_length",
		},
		{
			name: "materialized bytes",
			cfg: DecoderConfig{
				Bus: &klvRecordingBus{},
				PacketMaterializer: &recordingPacketMaterializer{
					result: MaterializedPacket{Bytes: packetBytes},
				},
				MaxPacketBytes: len(packetBytes) - 1,
			},
			pkt: func() *PacketPayload {
				packet := NewPacketPayload("klv:demux", "object://semops/klv/deterministic.ts", now, nil)
				packet.PacketRef = "klv://packet/deterministic/00000005"
				packet.StorageRef = "object://semops/klv/packets/00000005.bin"
				return packet
			}(),
			want: "exceeds max_packet_bytes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder, err := NewDecoderComponent(tt.cfg)
			if err != nil {
				t.Fatalf("new decoder: %v", err)
			}
			if err := decoder.HandlePacketPayload(context.Background(), tt.pkt); err == nil {
				t.Fatal("expected storage-ref packet decode to fail")
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func requireProperty(t *testing.T, schema component.ConfigSchema, name string) {
	t.Helper()
	if _, ok := schema.Properties[name]; !ok {
		t.Fatalf("schema missing property %q", name)
	}
}

type recordingPacketMaterializer struct {
	packet   *PacketPayload
	maxBytes int
	result   MaterializedPacket
	err      error
	cleaned  bool
}

func (m *recordingPacketMaterializer) MaterializePacket(_ context.Context, packet *PacketPayload, maxBytes int) (MaterializedPacket, error) {
	m.packet = packet
	m.maxBytes = maxBytes
	if m.err != nil {
		return MaterializedPacket{}, m.err
	}
	result := m.result
	if result.Cleanup == nil {
		result.Cleanup = func() error {
			m.cleaned = true
			return nil
		}
	}
	return result, nil
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
			edge.ConnectionID == connectionID &&
			edge.Pattern == flowgraph.PatternStream {
			return
		}
	}
	t.Fatalf(
		"missing flow edge %s/%s -> %s/%s on %s; got %#v",
		fromComponent,
		fromPort,
		toComponent,
		toPort,
		connectionID,
		edges,
	)
}

type klvRecordingBus struct {
	published []klvPublishedMessage
	handlers  map[string]func(context.Context, *nats.Msg)
}

type klvPublishedMessage struct {
	subject string
	data    []byte
}

func (b *klvRecordingBus) Publish(_ context.Context, subject string, data []byte) error {
	b.published = append(b.published, klvPublishedMessage{
		subject: subject,
		data:    append([]byte(nil), data...),
	})
	return nil
}

func (b *klvRecordingBus) Subscribe(
	_ context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.handlers == nil {
		b.handlers = make(map[string]func(context.Context, *nats.Msg))
	}
	b.handlers[subject] = handler
	return klvNoopSubscription{}, nil
}

type klvNoopSubscription struct{}

func (klvNoopSubscription) Unsubscribe() error {
	return nil
}
