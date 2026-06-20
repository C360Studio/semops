package mavlink

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.mavlink.raw"
	DefaultDecodedSubject = "semops.feed.mavlink.decoded"
)

var (
	RawFrameType = message.Type{
		Domain:   "semops",
		Category: "mavlink_raw_frame",
		Version:  "v1",
	}
	DecodedPacketType = message.Type{
		Domain:   "semops",
		Category: "mavlink_packet",
		Version:  "v1",
	}
)

type RawFramePayload struct {
	Source     string    `json:"source"`
	RemoteAddr string    `json:"remote_addr,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
	Frame      []byte    `json:"frame"`
}

func NewRawFramePayload(source, remoteAddr string, receivedAt time.Time, frame []byte) *RawFramePayload {
	return &RawFramePayload{
		Source:     source,
		RemoteAddr: remoteAddr,
		ReceivedAt: receivedAt.UTC(),
		Frame:      append([]byte(nil), frame...),
	}
}

func (p *RawFramePayload) Schema() message.Type {
	return RawFrameType
}

func (p *RawFramePayload) Validate() error {
	if p == nil {
		return errors.New("raw MAVLink payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw MAVLink payload source is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw MAVLink payload received_at is required")
	}
	if len(p.Frame) == 0 {
		return errors.New("raw MAVLink payload frame is required")
	}
	return nil
}

func (p *RawFramePayload) MarshalJSON() ([]byte, error) {
	type alias RawFramePayload
	return json.Marshal((*alias)(p))
}

func (p *RawFramePayload) UnmarshalJSON(data []byte) error {
	type alias RawFramePayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedPacketPayload struct {
	Source      string    `json:"source"`
	RawRef      string    `json:"raw_ref"`
	ReceivedAt  time.Time `json:"received_at"`
	Version     uint8     `json:"version"`
	Sequence    uint8     `json:"sequence"`
	SystemID    uint8     `json:"system_id"`
	ComponentID uint8     `json:"component_id"`
	MessageID   uint32    `json:"message_id"`
	Checksum    uint16    `json:"checksum"`
	Frame       []byte    `json:"frame"`
}

func NewDecodedPacketPayload(source string, record mavcodec.RawFrameRecord) *DecodedPacketPayload {
	return &DecodedPacketPayload{
		Source:      source,
		RawRef:      record.Ref,
		ReceivedAt:  record.ReceivedAt.UTC(),
		Version:     record.Version,
		Sequence:    record.Sequence,
		SystemID:    record.SystemID,
		ComponentID: record.ComponentID,
		MessageID:   record.MessageID,
		Checksum:    record.Checksum,
		Frame:       append([]byte(nil), record.Frame...),
	}
}

func (p *DecodedPacketPayload) Schema() message.Type {
	return DecodedPacketType
}

func (p *DecodedPacketPayload) Validate() error {
	if p == nil {
		return errors.New("decoded MAVLink payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded MAVLink payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded MAVLink payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded MAVLink payload received_at is required")
	}
	if len(p.Frame) == 0 {
		return errors.New("decoded MAVLink payload frame is required")
	}
	return nil
}

func (p *DecodedPacketPayload) Packet(parser *mavcodec.Parser) (*mavcodec.Packet, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if parser == nil {
		parser = mavcodec.NewParser()
	}
	packets, err := parser.Parse(p.Frame)
	if err != nil {
		return nil, fmt.Errorf("parse decoded MAVLink frame: %w", err)
	}
	if len(packets) != 1 {
		return nil, fmt.Errorf("decoded MAVLink frame produced %d packets, want 1", len(packets))
	}
	packet := packets[0]
	packet.SourceRef = p.RawRef
	packet.Timestamp = p.ReceivedAt.UTC()
	return packet, nil
}

func (p *DecodedPacketPayload) MarshalJSON() ([]byte, error) {
	type alias DecodedPacketPayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedPacketPayload) UnmarshalJSON(data []byte) error {
	type alias DecodedPacketPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(registry, RawFrameType, "Raw MAVLink frame captured by a SemOps input component", func() any {
		return &RawFramePayload{}
	}); err != nil {
		return err
	}
	return registerPayload(registry, DecodedPacketType, "Decoded MAVLink packet emitted by a SemOps processor component", func() any {
		return &DecodedPacketPayload{}
	})
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
