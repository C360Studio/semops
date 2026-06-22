package klv

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

func TestPayloadRegistryRoundTripsKLVPayloads(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads should be idempotent: %v", err)
	}

	now := time.Date(2026, 6, 22, 14, 30, 0, 0, time.UTC)
	media := NewMediaRefPayload("klv:fixture", "file:///fixtures/klv/demo.ts", "object://semops/klv/demo.ts", now)
	media.ContentHash = "sha256:abc123"
	media.FixtureKind = "deterministic"
	mediaWire := mustBaseMessageJSON(t, MediaRefType, media, "semops-input-klv-media-ref", now)
	mediaEnvelope, err := message.NewDecoder(registry).Decode(mediaWire)
	if err != nil {
		t.Fatalf("decode media-ref payload: %v", err)
	}
	if _, ok := mediaEnvelope.Payload().(*MediaRefPayload); !ok {
		t.Fatalf("media-ref payload type = %T, want *MediaRefPayload", mediaEnvelope.Payload())
	}

	packet := NewPacketPayload("klv:demux", media.StorageRef, now, []byte{0x06, 0x0e, 0x2b, 0x34})
	packet.PacketRef = "klv://packet/demo/00000001"
	packet.ByteOffset = 188
	packetWire := mustBaseMessageJSON(t, PacketType, packet, "semops-processor-klv-demux", now)
	packetEnvelope, err := message.NewDecoder(registry).Decode(packetWire)
	if err != nil {
		t.Fatalf("decode packet payload: %v", err)
	}
	gotPacket, ok := packetEnvelope.Payload().(*PacketPayload)
	if !ok {
		t.Fatalf("packet payload type = %T, want *PacketPayload", packetEnvelope.Payload())
	}
	if !bytes.Equal(gotPacket.PacketBytes, packet.PacketBytes) {
		t.Fatalf("packet bytes = %x, want %x", gotPacket.PacketBytes, packet.PacketBytes)
	}

	lat, lon, alt := 34.12345, -117.12345, 1420.0
	frame := NewMISB0601FramePayload("klv:decode", media.StorageRef, packet.PacketRef, now)
	frame.FrameTime = now
	frame.PlatformDesignation = "SYNTHETIC-UAS-1"
	frame.SensorLatitude = &lat
	frame.SensorLongitude = &lon
	frame.SensorAltitudeMeters = &alt
	frame.Fields = []string{"PrecisionTimeStamp", "SensorLatitude", "SensorLongitude", "SensorTrueAltitude"}
	frameWire := mustBaseMessageJSON(t, MISB0601FrameType, frame, "semops-processor-klv-decode", now)
	frameEnvelope, err := message.NewDecoder(registry).Decode(frameWire)
	if err != nil {
		t.Fatalf("decode MISB frame payload: %v", err)
	}
	if _, ok := frameEnvelope.Payload().(*MISB0601FramePayload); !ok {
		t.Fatalf("frame payload type = %T, want *MISB0601FramePayload", frameEnvelope.Payload())
	}
}

func TestKLVPacketPayloadRequiresBytesOrStorageReference(t *testing.T) {
	payload := NewPacketPayload("klv:demux", "object://semops/klv/demo.ts", time.Now().UTC(), nil)
	if err := payload.Validate(); err == nil {
		t.Fatal("expected packet without bytes or storage reference to fail validation")
	}
	payload.StorageRef = "object://semops/klv/packet/00000001"
	if err := payload.Validate(); err != nil {
		t.Fatalf("packet with storage reference should validate: %v", err)
	}
}

func TestKLVFramePayloadRequiresDecodedFields(t *testing.T) {
	payload := NewMISB0601FramePayload("klv:decode", "object://semops/klv/demo.ts", "klv://packet/demo/00000001", time.Now().UTC())
	if err := payload.Validate(); err == nil {
		t.Fatal("expected frame without fields to fail validation")
	}
	payload.Fields = []string{"PrecisionTimeStamp"}
	if err := payload.Validate(); err != nil {
		t.Fatalf("frame with fields should validate: %v", err)
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
	envelope := message.NewBaseMessage(msgType, payload, source, message.WithTime(observedAt.UTC()))
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal BaseMessage: %v", err)
	}
	return data
}
