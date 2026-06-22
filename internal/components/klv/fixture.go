package klv

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

type MISB0601Truth struct {
	ID                         string    `json:"id"`
	Source                     string    `json:"source,omitempty"`
	MediaRef                   string    `json:"media_ref"`
	PacketRef                  string    `json:"packet_ref"`
	ReceivedAt                 time.Time `json:"received_at"`
	FrameTime                  time.Time `json:"frame_time"`
	PlatformDesignation        string    `json:"platform_designation,omitempty"`
	SensorLatitude             *float64  `json:"sensor_latitude,omitempty"`
	SensorLongitude            *float64  `json:"sensor_longitude,omitempty"`
	SensorAltitudeMeters       *float64  `json:"sensor_altitude_meters,omitempty"`
	SensorAzimuthDegrees       *float64  `json:"sensor_azimuth_degrees,omitempty"`
	SensorElevationDegrees     *float64  `json:"sensor_elevation_degrees,omitempty"`
	FrameCenterLatitude        *float64  `json:"frame_center_latitude,omitempty"`
	FrameCenterLongitude       *float64  `json:"frame_center_longitude,omitempty"`
	FrameCenterElevationMeters *float64  `json:"frame_center_elevation_meters,omitempty"`
}

func (truth MISB0601Truth) Validate() error {
	if truth.ID == "" {
		return errors.New("MISB ST 0601 truth id is required")
	}
	if truth.MediaRef == "" {
		return errors.New("MISB ST 0601 truth media_ref is required")
	}
	if truth.PacketRef == "" {
		return errors.New("MISB ST 0601 truth packet_ref is required")
	}
	if truth.ReceivedAt.IsZero() {
		return errors.New("MISB ST 0601 truth received_at is required")
	}
	if truth.FrameTime.IsZero() {
		return errors.New("MISB ST 0601 truth frame_time is required")
	}
	return nil
}

func (truth MISB0601Truth) PacketPayload() (*PacketPayload, error) {
	if err := truth.Validate(); err != nil {
		return nil, err
	}
	packetBytes, err := EncodeMISB0601Truth(truth)
	if err != nil {
		return nil, err
	}
	source := truth.Source
	if source == "" {
		source = DefaultDemuxSource
	}
	packet := NewPacketPayload(source, truth.MediaRef, truth.ReceivedAt.UTC(), packetBytes)
	packet.PacketRef = truth.PacketRef
	packet.PacketTime = truth.FrameTime.UTC()
	return packet, nil
}

func EncodeMISB0601Truth(truth MISB0601Truth) ([]byte, error) {
	if err := truth.Validate(); err != nil {
		return nil, err
	}
	fields := []misbEncodedField{
		newMISBEncodedField(misbTagPrecisionTimeStamp, encodeUint64BE(uint64(truth.FrameTime.UTC().UnixMicro()))),
	}
	if truth.PlatformDesignation != "" {
		fields = append(fields, newMISBEncodedField(misbTagPlatformDesignation, []byte(truth.PlatformDesignation)))
	}
	if err := addEncodedFloatField(&fields, misbTagSensorLatitude, "sensor_latitude", truth.SensorLatitude, func(value float64) ([]byte, error) {
		return encodeSigned32Range(value, 90)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagSensorLongitude, "sensor_longitude", truth.SensorLongitude, func(value float64) ([]byte, error) {
		return encodeSigned32Range(value, 180)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagSensorTrueAltitude, "sensor_altitude_meters", truth.SensorAltitudeMeters, func(value float64) ([]byte, error) {
		return encodeUnsigned16Range(value, -900, 19000)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagSensorRelAzimuth, "sensor_azimuth_degrees", truth.SensorAzimuthDegrees, func(value float64) ([]byte, error) {
		return encodeUnsigned32Range(value, 0, 360)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagSensorRelElevation, "sensor_elevation_degrees", truth.SensorElevationDegrees, func(value float64) ([]byte, error) {
		return encodeSigned32Range(value, 180)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagFrameCenterLatitude, "frame_center_latitude", truth.FrameCenterLatitude, func(value float64) ([]byte, error) {
		return encodeSigned32Range(value, 90)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagFrameCenterLongitude, "frame_center_longitude", truth.FrameCenterLongitude, func(value float64) ([]byte, error) {
		return encodeSigned32Range(value, 180)
	}); err != nil {
		return nil, err
	}
	if err := addEncodedFloatField(&fields, misbTagFrameCenterElevation, "frame_center_elevation_meters", truth.FrameCenterElevationMeters, func(value float64) ([]byte, error) {
		return encodeUnsigned16Range(value, -900, 19000)
	}); err != nil {
		return nil, err
	}
	return encodeMISB0601Packet(fields...)
}

type misbEncodedField struct {
	tag   int
	value []byte
}

func newMISBEncodedField(tag int, value []byte) misbEncodedField {
	return misbEncodedField{tag: tag, value: append([]byte(nil), value...)}
}

func addEncodedFloatField(
	fields *[]misbEncodedField,
	tag int,
	name string,
	value *float64,
	encode func(float64) ([]byte, error),
) error {
	if value == nil {
		return nil
	}
	data, err := encode(*value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	*fields = append(*fields, newMISBEncodedField(tag, data))
	return nil
}

func encodeMISB0601Packet(fields ...misbEncodedField) ([]byte, error) {
	localSet := make([]byte, 0)
	for _, field := range fields {
		tag, err := encodeBEROID(field.tag)
		if err != nil {
			return nil, fmt.Errorf("encode MISB ST 0601 tag %d: %w", field.tag, err)
		}
		length, err := encodeBERLength(len(field.value))
		if err != nil {
			return nil, fmt.Errorf("encode MISB ST 0601 tag %d length: %w", field.tag, err)
		}
		localSet = append(localSet, tag...)
		localSet = append(localSet, length...)
		localSet = append(localSet, field.value...)
	}
	packetLength, err := encodeBERLength(len(localSet))
	if err != nil {
		return nil, fmt.Errorf("encode MISB ST 0601 packet length: %w", err)
	}
	packet := append([]byte{}, misb0601UASLocalSetKey...)
	packet = append(packet, packetLength...)
	packet = append(packet, localSet...)
	return packet, nil
}

func encodeBEROID(value int) ([]byte, error) {
	if value < 0 {
		return nil, errors.New("BER-OID value is negative")
	}
	if value == 0 {
		return []byte{0}, nil
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
	return encoded, nil
}

func encodeBERLength(length int) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("BER length is negative")
	}
	if uint64(length) > uint64(^uint32(0)) {
		return nil, errors.New("BER length exceeds four-byte encoding limit")
	}
	if length < 0x80 {
		return []byte{byte(length)}, nil
	}
	var scratch [4]byte
	binary.BigEndian.PutUint32(scratch[:], uint32(length))
	start := 0
	for start < len(scratch)-1 && scratch[start] == 0 {
		start++
	}
	return append([]byte{0x80 | byte(len(scratch)-start)}, scratch[start:]...), nil
}

func encodeSigned32Range(value float64, maxAbs float64) ([]byte, error) {
	if invalidFloat(value) {
		return nil, errors.New("value must be finite")
	}
	if value < -maxAbs || value > maxAbs {
		return nil, fmt.Errorf("value %.12f outside range [-%.12f, %.12f]", value, maxAbs, maxAbs)
	}
	raw := math.Round(value * misbMaxInt32 / maxAbs)
	if raw < -misbMaxInt32 || raw > misbMaxInt32 {
		return nil, fmt.Errorf("scaled value %.0f outside MISB signed32 range", raw)
	}
	return encodeInt32BE(int32(raw)), nil
}

func encodeUnsigned16Range(value float64, minValue, maxValue float64) ([]byte, error) {
	raw, err := encodeUnsignedRange(value, minValue, maxValue, misbMaxUint16)
	if err != nil {
		return nil, err
	}
	return encodeUint16BE(uint16(raw)), nil
}

func encodeUnsigned32Range(value float64, minValue, maxValue float64) ([]byte, error) {
	raw, err := encodeUnsignedRange(value, minValue, maxValue, misbMaxUint32)
	if err != nil {
		return nil, err
	}
	return encodeUint32BE(uint32(raw)), nil
}

func encodeUnsignedRange(value float64, minValue, maxValue, maxRaw float64) (uint64, error) {
	if invalidFloat(value) {
		return 0, errors.New("value must be finite")
	}
	if maxValue <= minValue {
		return 0, errors.New("range maximum must be greater than minimum")
	}
	if value < minValue || value > maxValue {
		return 0, fmt.Errorf("value %.12f outside range [%.12f, %.12f]", value, minValue, maxValue)
	}
	raw := math.Round(((value - minValue) / (maxValue - minValue)) * maxRaw)
	if raw < 0 || raw > maxRaw {
		return 0, fmt.Errorf("scaled value %.0f outside MISB unsigned range", raw)
	}
	return uint64(raw), nil
}

func invalidFloat(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0)
}

func encodeUint16BE(value uint16) []byte {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, value)
	return data
}

func encodeUint32BE(value uint32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	return data
}

func encodeInt32BE(value int32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(value))
	return data
}

func encodeUint64BE(value uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, value)
	return data
}
