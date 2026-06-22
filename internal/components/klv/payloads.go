package klv

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultMediaRefSubject = "semops.feed.klv.media_ref"
	DefaultPacketSubject   = "semops.feed.klv.packet"
	DefaultFrameSubject    = "semops.feed.klv.misb0601"
)

var (
	MediaRefType = message.Type{
		Domain:   "semops",
		Category: "klv_media_ref",
		Version:  "v1",
	}
	PacketType = message.Type{
		Domain:   "semops",
		Category: "klv_packet",
		Version:  "v1",
	}
	MISB0601FrameType = message.Type{
		Domain:   "semops",
		Category: "klv_misb0601_frame",
		Version:  "v1",
	}
)

type ByteRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type MediaRefPayload struct {
	Source      string     `json:"source"`
	URI         string     `json:"uri,omitempty"`
	StorageRef  string     `json:"storage_ref,omitempty"`
	ContentHash string     `json:"content_hash,omitempty"`
	MediaType   string     `json:"media_type,omitempty"`
	FixtureKind string     `json:"fixture_kind,omitempty"`
	Provenance  string     `json:"provenance,omitempty"`
	ReceivedAt  time.Time  `json:"received_at"`
	ByteRange   *ByteRange `json:"byte_range,omitempty"`
}

func NewMediaRefPayload(source, uri, storageRef string, receivedAt time.Time) *MediaRefPayload {
	return &MediaRefPayload{
		Source:     source,
		URI:        uri,
		StorageRef: storageRef,
		ReceivedAt: receivedAt.UTC(),
	}
}

func (p *MediaRefPayload) Schema() message.Type {
	return MediaRefType
}

func (p *MediaRefPayload) Validate() error {
	if p == nil {
		return errors.New("KLV media-ref payload is nil")
	}
	if p.Source == "" {
		return errors.New("KLV media-ref payload source is required")
	}
	if p.URI == "" && p.StorageRef == "" {
		return errors.New("KLV media-ref payload uri or storage_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("KLV media-ref payload received_at is required")
	}
	if p.ByteRange != nil && p.ByteRange.End < p.ByteRange.Start {
		return errors.New("KLV media-ref payload byte_range end is before start")
	}
	return nil
}

func (p *MediaRefPayload) MarshalJSON() ([]byte, error) {
	type alias MediaRefPayload
	return json.Marshal((*alias)(p))
}

func (p *MediaRefPayload) UnmarshalJSON(data []byte) error {
	type alias MediaRefPayload
	return json.Unmarshal(data, (*alias)(p))
}

type PacketPayload struct {
	Source      string    `json:"source"`
	MediaRef    string    `json:"media_ref"`
	PacketRef   string    `json:"packet_ref,omitempty"`
	StorageRef  string    `json:"storage_ref,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	PacketTime  time.Time `json:"packet_time,omitempty"`
	StreamIndex int       `json:"stream_index,omitempty"`
	PID         int       `json:"pid,omitempty"`
	PTS         int64     `json:"pts,omitempty"`
	ByteOffset  int64     `json:"byte_offset,omitempty"`
	ByteLength  int       `json:"byte_length,omitempty"`
	PacketBytes []byte    `json:"packet_bytes,omitempty"`
	Checksum    string    `json:"checksum,omitempty"`
}

func NewPacketPayload(source, mediaRef string, receivedAt time.Time, packetBytes []byte) *PacketPayload {
	return &PacketPayload{
		Source:      source,
		MediaRef:    mediaRef,
		ReceivedAt:  receivedAt.UTC(),
		PacketBytes: append([]byte(nil), packetBytes...),
		ByteLength:  len(packetBytes),
	}
}

func (p *PacketPayload) Schema() message.Type {
	return PacketType
}

func (p *PacketPayload) Validate() error {
	if p == nil {
		return errors.New("KLV packet payload is nil")
	}
	if p.Source == "" {
		return errors.New("KLV packet payload source is required")
	}
	if p.MediaRef == "" {
		return errors.New("KLV packet payload media_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("KLV packet payload received_at is required")
	}
	if p.ByteOffset < 0 {
		return errors.New("KLV packet payload byte_offset is invalid")
	}
	if p.ByteLength < 0 {
		return errors.New("KLV packet payload byte_length is invalid")
	}
	if len(p.PacketBytes) == 0 && p.StorageRef == "" {
		return errors.New("KLV packet payload packet_bytes or storage_ref is required")
	}
	return nil
}

func (p *PacketPayload) MarshalJSON() ([]byte, error) {
	type alias PacketPayload
	return json.Marshal((*alias)(p))
}

func (p *PacketPayload) UnmarshalJSON(data []byte) error {
	type alias PacketPayload
	return json.Unmarshal(data, (*alias)(p))
}

type MISB0601FramePayload struct {
	Source                     string    `json:"source"`
	MediaRef                   string    `json:"media_ref"`
	PacketRef                  string    `json:"packet_ref"`
	ReceivedAt                 time.Time `json:"received_at"`
	FrameTime                  time.Time `json:"frame_time,omitempty"`
	PlatformDesignation        string    `json:"platform_designation,omitempty"`
	PlatformLatitude           *float64  `json:"platform_latitude,omitempty"`
	PlatformLongitude          *float64  `json:"platform_longitude,omitempty"`
	PlatformAltitudeMeters     *float64  `json:"platform_altitude_meters,omitempty"`
	SensorLatitude             *float64  `json:"sensor_latitude,omitempty"`
	SensorLongitude            *float64  `json:"sensor_longitude,omitempty"`
	SensorAltitudeMeters       *float64  `json:"sensor_altitude_meters,omitempty"`
	SensorAzimuthDegrees       *float64  `json:"sensor_azimuth_degrees,omitempty"`
	SensorElevationDegrees     *float64  `json:"sensor_elevation_degrees,omitempty"`
	FrameCenterLatitude        *float64  `json:"frame_center_latitude,omitempty"`
	FrameCenterLongitude       *float64  `json:"frame_center_longitude,omitempty"`
	FrameCenterElevationMeters *float64  `json:"frame_center_elevation_meters,omitempty"`
	FootprintWKT               string    `json:"footprint_wkt,omitempty"`
	Fields                     []string  `json:"fields,omitempty"`
	Warnings                   []string  `json:"warnings,omitempty"`
}

func NewMISB0601FramePayload(source, mediaRef, packetRef string, receivedAt time.Time) *MISB0601FramePayload {
	return &MISB0601FramePayload{
		Source:     source,
		MediaRef:   mediaRef,
		PacketRef:  packetRef,
		ReceivedAt: receivedAt.UTC(),
	}
}

func (p *MISB0601FramePayload) Schema() message.Type {
	return MISB0601FrameType
}

func (p *MISB0601FramePayload) Validate() error {
	if p == nil {
		return errors.New("MISB ST 0601 frame payload is nil")
	}
	if p.Source == "" {
		return errors.New("MISB ST 0601 frame payload source is required")
	}
	if p.MediaRef == "" {
		return errors.New("MISB ST 0601 frame payload media_ref is required")
	}
	if p.PacketRef == "" {
		return errors.New("MISB ST 0601 frame payload packet_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("MISB ST 0601 frame payload received_at is required")
	}
	if len(p.Fields) == 0 {
		return errors.New("MISB ST 0601 frame payload fields are required")
	}
	return nil
}

func (p *MISB0601FramePayload) MarshalJSON() ([]byte, error) {
	type alias MISB0601FramePayload
	return json.Marshal((*alias)(p))
}

func (p *MISB0601FramePayload) UnmarshalJSON(data []byte) error {
	type alias MISB0601FramePayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(
		registry,
		MediaRefType,
		"KLV media reference emitted by a SemOps input component",
		func() any { return &MediaRefPayload{} },
	); err != nil {
		return err
	}
	if err := registerPayload(
		registry,
		PacketType,
		"KLV packet trace emitted by a SemOps demux processor component",
		func() any { return &PacketPayload{} },
	); err != nil {
		return err
	}
	return registerPayload(
		registry,
		MISB0601FrameType,
		"MISB ST 0601 decoded frame emitted by a SemOps processor component",
		func() any { return &MISB0601FramePayload{} },
	)
}

func registerPayload(
	registry *payloadregistry.Registry,
	msgType message.Type,
	description string,
	factory payloadregistry.Factory,
) error {
	if _, ok := registry.GetRegistration(msgType.Key()); ok {
		return nil
	}
	return registry.Register(&payloadregistry.Registration{
		Factory:     factory,
		Domain:      msgType.Domain,
		Category:    msgType.Category,
		Version:     msgType.Version,
		Description: description,
	})
}
