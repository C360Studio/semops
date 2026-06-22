package klv

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"time"
)

func TestDecodeMISB0601PacketDeterministicSubset(t *testing.T) {
	receivedAt := time.Date(2026, 6, 22, 15, 10, 0, 0, time.UTC)
	frameTime := time.Date(2026, 6, 22, 15, 9, 58, 123456000, time.UTC)

	sensorLatRaw := int32(0x10000000)
	sensorLonRaw := int32(-0x20000000)
	frameCenterLatRaw := int32(-0x10000000)
	frameCenterLonRaw := int32(0x20000000)
	azimuthRaw := uint32(0x40000000)
	elevationRaw := int32(-0x10000000)
	sensorAltRaw := uint16(0x8000)
	frameCenterAltRaw := uint16(0xffff)

	packet := NewPacketPayload(
		"klv:demux",
		"object://semops/klv/deterministic.ts",
		receivedAt,
		buildMISB0601Packet(
			misbField(misbTagPrecisionTimeStamp, beU64(uint64(frameTime.UnixMicro()))),
			misbField(misbTagPlatformDesignation, []byte("SYNTHETIC-UAS-1")),
			misbField(misbTagSensorLatitude, beI32(sensorLatRaw)),
			misbField(misbTagSensorLongitude, beI32(sensorLonRaw)),
			misbField(misbTagSensorTrueAltitude, beU16(sensorAltRaw)),
			misbField(misbTagSensorRelAzimuth, beU32(azimuthRaw)),
			misbField(misbTagSensorRelElevation, beI32(elevationRaw)),
			misbField(misbTagFrameCenterLatitude, beI32(frameCenterLatRaw)),
			misbField(misbTagFrameCenterLongitude, beI32(frameCenterLonRaw)),
			misbField(misbTagFrameCenterElevation, beU16(frameCenterAltRaw)),
		),
	)
	packet.PacketRef = "klv://packet/deterministic/00000001"

	frame, err := DecodeMISB0601Packet(packet)
	if err != nil {
		t.Fatalf("decode MISB ST 0601 deterministic packet: %v", err)
	}
	if frame.Source != DefaultDecodeSource {
		t.Fatalf("source = %q, want %q", frame.Source, DefaultDecodeSource)
	}
	if frame.MediaRef != packet.MediaRef || frame.PacketRef != packet.PacketRef {
		t.Fatalf("trace refs = %q/%q, want %q/%q", frame.MediaRef, frame.PacketRef, packet.MediaRef, packet.PacketRef)
	}
	if !frame.ReceivedAt.Equal(receivedAt) {
		t.Fatalf("received_at = %s, want %s", frame.ReceivedAt, receivedAt)
	}
	if !frame.FrameTime.Equal(frameTime) {
		t.Fatalf("frame_time = %s, want %s", frame.FrameTime, frameTime)
	}
	if frame.PlatformDesignation != "SYNTHETIC-UAS-1" {
		t.Fatalf("platform designation = %q", frame.PlatformDesignation)
	}

	requireClose(t, "sensor latitude", frame.SensorLatitude, scaleSigned32ForTest(sensorLatRaw, 90), 1e-9)
	requireClose(t, "sensor longitude", frame.SensorLongitude, scaleSigned32ForTest(sensorLonRaw, 180), 1e-9)
	requireClose(t, "sensor altitude", frame.SensorAltitudeMeters, scaleU16ForTest(sensorAltRaw, -900, 19000), 1e-9)
	requireClose(t, "sensor azimuth", frame.SensorAzimuthDegrees, scaleU32ForTest(azimuthRaw, 0, 360), 1e-9)
	requireClose(t, "sensor elevation", frame.SensorElevationDegrees, scaleSigned32ForTest(elevationRaw, 180), 1e-9)
	requireClose(t, "frame center latitude", frame.FrameCenterLatitude, scaleSigned32ForTest(frameCenterLatRaw, 90), 1e-9)
	requireClose(t, "frame center longitude", frame.FrameCenterLongitude, scaleSigned32ForTest(frameCenterLonRaw, 180), 1e-9)
	requireClose(t, "frame center elevation", frame.FrameCenterElevationMeters, scaleU16ForTest(frameCenterAltRaw, -900, 19000), 1e-9)

	for _, field := range []string{
		"PrecisionTimeStamp",
		"PlatformDesignation",
		"SensorLatitude",
		"SensorLongitude",
		"SensorTrueAltitude",
		"SensorRelativeAzimuthAngle",
		"SensorRelativeElevationAngle",
		"FrameCenterLatitude",
		"FrameCenterLongitude",
		"FrameCenterElevation",
	} {
		requireField(t, frame.Fields, field)
	}
}

func TestDecodeMISB0601PacketRejectsStorageOnlyPackets(t *testing.T) {
	packet := NewPacketPayload("klv:demux", "object://semops/klv/deterministic.ts", time.Now().UTC(), nil)
	packet.PacketRef = "klv://packet/deterministic/00000001"
	packet.StorageRef = "object://semops/klv/packets/00000001.bin"

	_, err := DecodeMISB0601Packet(packet)
	if err == nil {
		t.Fatal("expected storage-only packet decode to fail")
	}
	if !strings.Contains(err.Error(), "packet_bytes are required") {
		t.Fatalf("error = %q, want packet_bytes requirement", err)
	}
}

func TestDecodeMISB0601PacketRejectsMalformedPackets(t *testing.T) {
	receivedAt := time.Now().UTC()
	tests := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{
			name:  "unknown key",
			bytes: []byte{0x01, 0x02, 0x03, 0x04},
			want:  "packet is too short",
		},
		{
			name:  "truncated length",
			bytes: append(append([]byte{}, misb0601UASLocalSetKey...), 0x82, 0x01),
			want:  "truncated BER length",
		},
		{
			name:  "local set overrun",
			bytes: append(append([]byte{}, misb0601UASLocalSetKey...), 0x05, 0x02, 0x01),
			want:  "local set length exceeds packet bytes",
		},
		{
			name:  "supported tag wrong length",
			bytes: buildMISB0601Packet(misbField(misbTagSensorLatitude, []byte{0x01, 0x02})),
			want:  "decode SensorLatitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := NewPacketPayload("klv:demux", "object://semops/klv/deterministic.ts", receivedAt, tt.bytes)
			packet.PacketRef = "klv://packet/deterministic/00000001"

			_, err := DecodeMISB0601Packet(packet)
			if err == nil {
				t.Fatal("expected malformed packet to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestDecodeMISB0601PacketReportsUnsupportedTags(t *testing.T) {
	receivedAt := time.Now().UTC()
	packet := NewPacketPayload(
		"klv:demux",
		"object://semops/klv/deterministic.ts",
		receivedAt,
		buildMISB0601Packet(
			misbField(misbTagPlatformDesignation, []byte("SYNTHETIC-UAS-1")),
			misbField(42, []byte{0x01, 0x02, 0x03}),
		),
	)
	packet.PacketRef = "klv://packet/deterministic/00000001"

	frame, err := DecodeMISB0601Packet(packet)
	if err != nil {
		t.Fatalf("decode packet with unsupported tag: %v", err)
	}
	if len(frame.Warnings) != 1 || !strings.Contains(frame.Warnings[0], "42") {
		t.Fatalf("warnings = %#v, want unsupported tag 42", frame.Warnings)
	}
	requireField(t, frame.Fields, "PlatformDesignation")
}

type misbLocalField struct {
	tag   int
	value []byte
}

func misbField(tag int, value []byte) misbLocalField {
	return misbLocalField{tag: tag, value: append([]byte(nil), value...)}
}

func buildMISB0601Packet(fields ...misbLocalField) []byte {
	localSet := make([]byte, 0)
	for _, field := range fields {
		localSet = append(localSet, encodeBEROIDForTest(field.tag)...)
		localSet = append(localSet, encodeBERLengthForTest(len(field.value))...)
		localSet = append(localSet, field.value...)
	}
	packet := append([]byte{}, misb0601UASLocalSetKey...)
	packet = append(packet, encodeBERLengthForTest(len(localSet))...)
	packet = append(packet, localSet...)
	return packet
}

func encodeBEROIDForTest(value int) []byte {
	if value < 0 {
		panic("negative BER-OID")
	}
	if value == 0 {
		return []byte{0}
	}
	var reversed []byte
	for value > 0 {
		reversed = append(reversed, byte(value&0x7f))
		value >>= 7
	}
	encoded := make([]byte, len(reversed))
	for index := range reversed {
		b := reversed[len(reversed)-1-index]
		if index < len(reversed)-1 {
			b |= 0x80
		}
		encoded[index] = b
	}
	return encoded
}

func encodeBERLengthForTest(length int) []byte {
	if length < 0 {
		panic("negative BER length")
	}
	if length < 0x80 {
		return []byte{byte(length)}
	}
	if length <= 0xff {
		return []byte{0x81, byte(length)}
	}
	return []byte{0x82, byte(length >> 8), byte(length)}
}

func beU16(value uint16) []byte {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, value)
	return data
}

func beU32(value uint32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	return data
}

func beI32(value int32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(value))
	return data
}

func beU64(value uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, value)
	return data
}

func requireClose(t *testing.T, name string, got *float64, want float64, tolerance float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil, want %f", name, want)
	}
	if math.Abs(*got-want) > tolerance {
		t.Fatalf("%s = %.12f, want %.12f", name, *got, want)
	}
}

func requireField(t *testing.T, fields []string, want string) {
	t.Helper()
	for _, field := range fields {
		if field == want {
			return
		}
	}
	t.Fatalf("fields %#v missing %q", fields, want)
}

func scaleSigned32ForTest(raw int32, maxAbs float64) float64 {
	return float64(raw) * maxAbs / misbMaxInt32
}

func scaleU16ForTest(raw uint16, minValue, maxValue float64) float64 {
	return minValue + (float64(raw)*(maxValue-minValue))/misbMaxUint16
}

func scaleU32ForTest(raw uint32, minValue, maxValue float64) float64 {
	return minValue + (float64(raw)*(maxValue-minValue))/misbMaxUint32
}
