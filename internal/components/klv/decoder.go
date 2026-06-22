package klv

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultDecodeSource                  = "klv:decode"
	DefaultMaxMaterializedPacketBytes    = DefaultMaxPacketBytes
	storageRefPacketRequiresMaterializer = "decode MISB ST 0601 packet: packet_bytes are required; storage_ref-only decode requires a bounded packet materializer"
)

type PacketMaterializer interface {
	MaterializePacket(ctx context.Context, packet *PacketPayload, maxBytes int) (MaterializedPacket, error)
}

type MaterializedPacket struct {
	Bytes   []byte
	Cleanup func() error
}

var misb0601UASLocalSetKey = []byte{
	0x06, 0x0e, 0x2b, 0x34,
	0x02, 0x0b, 0x01, 0x01,
	0x0e, 0x01, 0x03, 0x01,
	0x01, 0x00, 0x00, 0x00,
}

const (
	misbTagChecksum             = 1
	misbTagPrecisionTimeStamp   = 2
	misbTagPlatformDesignation  = 10
	misbTagSensorLatitude       = 13
	misbTagSensorLongitude      = 14
	misbTagSensorTrueAltitude   = 15
	misbTagSensorRelAzimuth     = 18
	misbTagSensorRelElevation   = 19
	misbTagFrameCenterLatitude  = 23
	misbTagFrameCenterLongitude = 24
	misbTagFrameCenterElevation = 25

	misbMaxInt32  = float64(1<<31 - 1)
	misbMaxUint16 = float64(1<<16 - 1)
	misbMaxUint32 = float64(1<<32 - 1)
)

func DecodeMISB0601Packet(packet *PacketPayload) (*MISB0601FramePayload, error) {
	if packet == nil {
		return nil, errors.New("decode MISB ST 0601 packet: packet is nil")
	}
	if err := packet.Validate(); err != nil {
		return nil, fmt.Errorf("decode MISB ST 0601 packet: %w", err)
	}
	if packet.PacketRef == "" {
		return nil, errors.New("decode MISB ST 0601 packet: packet_ref is required")
	}
	if len(packet.PacketBytes) == 0 {
		return nil, errors.New("decode MISB ST 0601 packet: packet_bytes are required; storage_ref-only decode is not implemented")
	}

	localSet, err := extractMISB0601LocalSet(packet.PacketBytes)
	if err != nil {
		return nil, err
	}

	frame := NewMISB0601FramePayload(DefaultDecodeSource, packet.MediaRef, packet.PacketRef, packet.ReceivedAt)
	if !packet.PacketTime.IsZero() {
		frame.FrameTime = packet.PacketTime.UTC()
	}
	if err := applyMISB0601LocalSet(frame, localSet); err != nil {
		return nil, err
	}
	if err := frame.Validate(); err != nil {
		return nil, fmt.Errorf("decode MISB ST 0601 packet: %w", err)
	}
	return frame, nil
}

func materializedPacketPayload(
	ctx context.Context,
	packet *PacketPayload,
	materializer PacketMaterializer,
	maxBytes int,
) (*PacketPayload, int, func() error, error) {
	if packet == nil {
		return nil, 0, nil, errors.New("decode MISB ST 0601 packet: packet is nil")
	}
	if err := packet.Validate(); err != nil {
		return nil, 0, nil, fmt.Errorf("decode MISB ST 0601 packet: %w", err)
	}
	if packet.PacketRef == "" {
		return nil, 0, nil, errors.New("decode MISB ST 0601 packet: packet_ref is required")
	}
	if maxBytes <= 0 {
		return nil, 0, nil, errors.New("decode MISB ST 0601 packet: max_packet_bytes must be greater than zero")
	}
	if len(packet.PacketBytes) > 0 {
		if len(packet.PacketBytes) > maxBytes {
			return nil, 0, nil, fmt.Errorf("decode MISB ST 0601 packet: packet_bytes length %d exceeds max_packet_bytes=%d", len(packet.PacketBytes), maxBytes)
		}
		return packet, len(packet.PacketBytes), nil, nil
	}
	if packet.ByteLength > maxBytes {
		return nil, 0, nil, fmt.Errorf("decode MISB ST 0601 packet: byte_length %d exceeds max_packet_bytes=%d", packet.ByteLength, maxBytes)
	}
	if materializer == nil {
		return nil, 0, nil, errors.New(storageRefPacketRequiresMaterializer)
	}
	materialized, err := materializer.MaterializePacket(ctx, packet, maxBytes)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("materialize KLV packet storage_ref: %w", err)
	}
	if len(materialized.Bytes) == 0 {
		if materialized.Cleanup != nil {
			_ = materialized.Cleanup()
		}
		return nil, 0, nil, errors.New("materialize KLV packet storage_ref: packet bytes are required")
	}
	if len(materialized.Bytes) > maxBytes {
		if materialized.Cleanup != nil {
			_ = materialized.Cleanup()
		}
		return nil, 0, nil, fmt.Errorf("materialize KLV packet storage_ref: size %d exceeds max_packet_bytes=%d", len(materialized.Bytes), maxBytes)
	}
	decodePacket := *packet
	decodePacket.PacketBytes = append([]byte(nil), materialized.Bytes...)
	decodePacket.ByteLength = len(decodePacket.PacketBytes)
	return &decodePacket, len(decodePacket.PacketBytes), materialized.Cleanup, nil
}

func extractMISB0601LocalSet(packetBytes []byte) ([]byte, error) {
	if len(packetBytes) < len(misb0601UASLocalSetKey)+1 {
		return nil, errors.New("decode MISB ST 0601 packet: packet is too short")
	}
	if !bytes.Equal(packetBytes[:len(misb0601UASLocalSetKey)], misb0601UASLocalSetKey) {
		return nil, errors.New("decode MISB ST 0601 packet: missing MISB ST 0601 universal key")
	}
	offset := len(misb0601UASLocalSetKey)
	length, next, err := readBERLength(packetBytes, offset)
	if err != nil {
		return nil, fmt.Errorf("decode MISB ST 0601 packet length: %w", err)
	}
	if length < 0 || next+length > len(packetBytes) {
		return nil, errors.New("decode MISB ST 0601 packet: local set length exceeds packet bytes")
	}
	return packetBytes[next : next+length], nil
}

func applyMISB0601LocalSet(frame *MISB0601FramePayload, localSet []byte) error {
	var unsupported []int
	for offset := 0; offset < len(localSet); {
		tag, next, err := readBEROID(localSet, offset)
		if err != nil {
			return fmt.Errorf("decode MISB ST 0601 local set tag at byte %d: %w", offset, err)
		}
		length, valueOffset, err := readBERLength(localSet, next)
		if err != nil {
			return fmt.Errorf("decode MISB ST 0601 local set tag %d length: %w", tag, err)
		}
		valueEnd := valueOffset + length
		if length < 0 || valueEnd > len(localSet) {
			return fmt.Errorf("decode MISB ST 0601 local set tag %d: length exceeds local set", tag)
		}
		value := localSet[valueOffset:valueEnd]
		offset = valueEnd

		switch tag {
		case misbTagChecksum:
			addFrameWarning(frame, "MISB ST 0601 checksum tag present; checksum validation is not implemented in the first spike")
		case misbTagPrecisionTimeStamp:
			if len(value) != 8 {
				return fmt.Errorf("decode PrecisionTimeStamp: got %d bytes, want 8", len(value))
			}
			frame.FrameTime = unixMicrosToTime(binary.BigEndian.Uint64(value))
			addFrameField(frame, "PrecisionTimeStamp")
		case misbTagPlatformDesignation:
			frame.PlatformDesignation = string(value)
			addFrameField(frame, "PlatformDesignation")
		case misbTagSensorLatitude:
			v, err := decodeSigned32(value, 90)
			if err != nil {
				return fmt.Errorf("decode SensorLatitude: %w", err)
			}
			frame.SensorLatitude = &v
			addFrameField(frame, "SensorLatitude")
		case misbTagSensorLongitude:
			v, err := decodeSigned32(value, 180)
			if err != nil {
				return fmt.Errorf("decode SensorLongitude: %w", err)
			}
			frame.SensorLongitude = &v
			addFrameField(frame, "SensorLongitude")
		case misbTagSensorTrueAltitude:
			v, err := decodeUnsigned16Range(value, -900, 19000)
			if err != nil {
				return fmt.Errorf("decode SensorTrueAltitude: %w", err)
			}
			frame.SensorAltitudeMeters = &v
			addFrameField(frame, "SensorTrueAltitude")
		case misbTagSensorRelAzimuth:
			v, err := decodeUnsigned32Range(value, 0, 360)
			if err != nil {
				return fmt.Errorf("decode SensorRelativeAzimuthAngle: %w", err)
			}
			frame.SensorAzimuthDegrees = &v
			addFrameField(frame, "SensorRelativeAzimuthAngle")
		case misbTagSensorRelElevation:
			v, err := decodeSigned32(value, 180)
			if err != nil {
				return fmt.Errorf("decode SensorRelativeElevationAngle: %w", err)
			}
			frame.SensorElevationDegrees = &v
			addFrameField(frame, "SensorRelativeElevationAngle")
		case misbTagFrameCenterLatitude:
			v, err := decodeSigned32(value, 90)
			if err != nil {
				return fmt.Errorf("decode FrameCenterLatitude: %w", err)
			}
			frame.FrameCenterLatitude = &v
			addFrameField(frame, "FrameCenterLatitude")
		case misbTagFrameCenterLongitude:
			v, err := decodeSigned32(value, 180)
			if err != nil {
				return fmt.Errorf("decode FrameCenterLongitude: %w", err)
			}
			frame.FrameCenterLongitude = &v
			addFrameField(frame, "FrameCenterLongitude")
		case misbTagFrameCenterElevation:
			v, err := decodeUnsigned16Range(value, -900, 19000)
			if err != nil {
				return fmt.Errorf("decode FrameCenterElevation: %w", err)
			}
			frame.FrameCenterElevationMeters = &v
			addFrameField(frame, "FrameCenterElevation")
		default:
			unsupported = append(unsupported, tag)
		}
	}
	if len(unsupported) > 0 {
		addFrameWarning(frame, fmt.Sprintf("unsupported MISB ST 0601 tags ignored: %v", unsupported))
	}
	return nil
}

func readBEROID(data []byte, offset int) (int, int, error) {
	if offset >= len(data) {
		return 0, offset, errors.New("missing BER-OID tag")
	}
	value := 0
	for index := offset; index < len(data); index++ {
		value = value<<7 | int(data[index]&0x7f)
		if data[index]&0x80 == 0 {
			return value, index + 1, nil
		}
		if index-offset >= 4 {
			return 0, offset, errors.New("BER-OID tag is too long")
		}
	}
	return 0, offset, errors.New("unterminated BER-OID tag")
}

func readBERLength(data []byte, offset int) (int, int, error) {
	if offset >= len(data) {
		return 0, offset, errors.New("missing BER length")
	}
	first := data[offset]
	if first&0x80 == 0 {
		return int(first), offset + 1, nil
	}
	count := int(first & 0x7f)
	if count == 0 {
		return 0, offset, errors.New("indefinite BER length is not supported")
	}
	if count > 4 {
		return 0, offset, errors.New("BER length uses more than four bytes")
	}
	if offset+1+count > len(data) {
		return 0, offset, errors.New("truncated BER length")
	}
	length := 0
	for _, b := range data[offset+1 : offset+1+count] {
		length = length<<8 | int(b)
	}
	return length, offset + 1 + count, nil
}

func decodeSigned32(value []byte, maxAbs float64) (float64, error) {
	if len(value) != 4 {
		return 0, fmt.Errorf("got %d bytes, want 4", len(value))
	}
	raw := int32(binary.BigEndian.Uint32(value))
	if raw == -1<<31 {
		return 0, errors.New("reserved out-of-range value")
	}
	return float64(raw) * maxAbs / misbMaxInt32, nil
}

func decodeUnsigned16Range(value []byte, minValue, maxValue float64) (float64, error) {
	if len(value) != 2 {
		return 0, fmt.Errorf("got %d bytes, want 2", len(value))
	}
	raw := binary.BigEndian.Uint16(value)
	return minValue + (float64(raw)*(maxValue-minValue))/misbMaxUint16, nil
}

func decodeUnsigned32Range(value []byte, minValue, maxValue float64) (float64, error) {
	if len(value) != 4 {
		return 0, fmt.Errorf("got %d bytes, want 4", len(value))
	}
	raw := binary.BigEndian.Uint32(value)
	return minValue + (float64(raw)*(maxValue-minValue))/misbMaxUint32, nil
}

func unixMicrosToTime(micros uint64) time.Time {
	seconds := int64(micros / 1_000_000)
	nanos := int64(micros%1_000_000) * int64(time.Microsecond)
	return time.Unix(seconds, nanos).UTC()
}

func addFrameField(frame *MISB0601FramePayload, field string) {
	for _, existing := range frame.Fields {
		if existing == field {
			return
		}
	}
	frame.Fields = append(frame.Fields, field)
}

func addFrameWarning(frame *MISB0601FramePayload, warning string) {
	frame.Warnings = append(frame.Warnings, warning)
}
