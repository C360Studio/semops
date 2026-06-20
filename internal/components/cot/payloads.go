package cot

import (
	"encoding/json"
	"errors"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.cot.raw"
	DefaultDecodedSubject = "semops.feed.cot.decoded"
)

var (
	RawEventType = message.Type{
		Domain:   "semops",
		Category: "cot_raw_event",
		Version:  "v1",
	}
	DecodedEventType = message.Type{
		Domain:   "semops",
		Category: "cot_event",
		Version:  "v1",
	}
)

type RawEventPayload struct {
	Source     string    `json:"source"`
	RemoteAddr string    `json:"remote_addr,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
	RawXML     []byte    `json:"raw_xml"`
}

func NewRawEventPayload(source, remoteAddr string, receivedAt time.Time, rawXML []byte) *RawEventPayload {
	return &RawEventPayload{
		Source:     source,
		RemoteAddr: remoteAddr,
		ReceivedAt: receivedAt.UTC(),
		RawXML:     append([]byte(nil), rawXML...),
	}
}

func (p *RawEventPayload) Schema() message.Type {
	return RawEventType
}

func (p *RawEventPayload) Validate() error {
	if p == nil {
		return errors.New("raw CoT payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw CoT payload source is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw CoT payload received_at is required")
	}
	if len(p.RawXML) == 0 {
		return errors.New("raw CoT payload raw_xml is required")
	}
	return nil
}

func (p *RawEventPayload) MarshalJSON() ([]byte, error) {
	type alias RawEventPayload
	return json.Marshal((*alias)(p))
}

func (p *RawEventPayload) UnmarshalJSON(data []byte) error {
	type alias RawEventPayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedEventPayload struct {
	Source     string         `json:"source"`
	RawRef     string         `json:"raw_ref"`
	ReceivedAt time.Time      `json:"received_at"`
	Event      cotcodec.Event `json:"event"`
}

func NewDecodedEventPayload(source string, record cotcodec.RawEventRecord, event cotcodec.Event) *DecodedEventPayload {
	return &DecodedEventPayload{
		Source:     source,
		RawRef:     record.Ref,
		ReceivedAt: record.ReceivedAt.UTC(),
		Event:      cloneEvent(event),
	}
}

func (p *DecodedEventPayload) Schema() message.Type {
	return DecodedEventType
}

func (p *DecodedEventPayload) Validate() error {
	if p == nil {
		return errors.New("decoded CoT payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded CoT payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded CoT payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded CoT payload received_at is required")
	}
	if err := p.Event.Validate(); err != nil {
		return err
	}
	return nil
}

func (p *DecodedEventPayload) CoTEvent() (cotcodec.Event, error) {
	if err := p.Validate(); err != nil {
		return cotcodec.Event{}, err
	}
	return cloneEvent(p.Event), nil
}

func (p *DecodedEventPayload) MarshalJSON() ([]byte, error) {
	type alias DecodedEventPayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedEventPayload) UnmarshalJSON(data []byte) error {
	type alias DecodedEventPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(registry, RawEventType, "Raw CoT XML captured by a SemOps input component", func() any {
		return &RawEventPayload{}
	}); err != nil {
		return err
	}
	return registerPayload(registry, DecodedEventType, "Decoded CoT event emitted by a SemOps processor component", func() any {
		return &DecodedEventPayload{}
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

func cloneEvent(event cotcodec.Event) cotcodec.Event {
	if event.Point != nil {
		point := *event.Point
		event.Point = &point
	}
	return event
}
