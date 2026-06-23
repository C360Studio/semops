package dji

import (
	"encoding/json"
	"errors"
	"time"

	djicodec "github.com/c360studio/semops/pkg/adapters/dji"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.dji.raw"
	DefaultDecodedSubject = "semops.feed.dji.decoded"
)

var (
	RawTelemetryType = message.Type{
		Domain:   "semops",
		Category: "dji_raw_telemetry",
		Version:  "v1",
	}
	DecodedTelemetryType = message.Type{
		Domain:   "semops",
		Category: "dji_telemetry",
		Version:  "v1",
	}
)

type RawTelemetryPayload struct {
	Source      string    `json:"source"`
	FixturePath string    `json:"fixture_path,omitempty"`
	FixtureURI  string    `json:"fixture_uri,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	RawJSON     []byte    `json:"raw_json"`
}

func NewRawTelemetryPayload(source, fixturePath, fixtureURI string, receivedAt time.Time, rawJSON []byte) *RawTelemetryPayload {
	return &RawTelemetryPayload{
		Source:      source,
		FixturePath: fixturePath,
		FixtureURI:  fixtureURI,
		ReceivedAt:  receivedAt.UTC(),
		RawJSON:     append([]byte(nil), rawJSON...),
	}
}

func (p *RawTelemetryPayload) Schema() message.Type {
	return RawTelemetryType
}

func (p *RawTelemetryPayload) Validate() error {
	if p == nil {
		return errors.New("raw DJI telemetry payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw DJI telemetry payload source is required")
	}
	if p.FixturePath == "" && p.FixtureURI == "" {
		return errors.New("raw DJI telemetry payload fixture_path or fixture_uri is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw DJI telemetry payload received_at is required")
	}
	if len(p.RawJSON) == 0 {
		return errors.New("raw DJI telemetry payload raw_json is required")
	}
	return nil
}

func (p *RawTelemetryPayload) MarshalJSON() ([]byte, error) {
	type alias RawTelemetryPayload
	return json.Marshal((*alias)(p))
}

func (p *RawTelemetryPayload) UnmarshalJSON(data []byte) error {
	type alias RawTelemetryPayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedTelemetryPayload struct {
	Source     string                   `json:"source"`
	RawRef     string                   `json:"raw_ref"`
	ReceivedAt time.Time                `json:"received_at"`
	ObservedAt time.Time                `json:"observed_at"`
	Record     djicodec.TelemetryRecord `json:"record"`
}

func NewDecodedTelemetryPayload(raw *RawTelemetryPayload, record djicodec.TelemetryRecord) *DecodedTelemetryPayload {
	rawRef := raw.FixtureURI
	if rawRef == "" {
		rawRef = raw.FixturePath
	}
	return &DecodedTelemetryPayload{
		Source:     raw.Source,
		RawRef:     rawRef,
		ReceivedAt: raw.ReceivedAt.UTC(),
		ObservedAt: record.ObservedAt.UTC(),
		Record:     cloneRecord(record),
	}
}

func (p *DecodedTelemetryPayload) Schema() message.Type {
	return DecodedTelemetryType
}

func (p *DecodedTelemetryPayload) Validate() error {
	if p == nil {
		return errors.New("decoded DJI telemetry payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded DJI telemetry payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded DJI telemetry payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded DJI telemetry payload received_at is required")
	}
	if p.ObservedAt.IsZero() {
		return errors.New("decoded DJI telemetry payload observed_at is required")
	}
	if err := p.Record.Validate(); err != nil {
		return err
	}
	return nil
}

func (p *DecodedTelemetryPayload) RecordCopy() (djicodec.TelemetryRecord, error) {
	if err := p.Validate(); err != nil {
		return djicodec.TelemetryRecord{}, err
	}
	return cloneRecord(p.Record), nil
}

func (p *DecodedTelemetryPayload) MarshalJSON() ([]byte, error) {
	type alias DecodedTelemetryPayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedTelemetryPayload) UnmarshalJSON(data []byte) error {
	type alias DecodedTelemetryPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(
		registry,
		RawTelemetryType,
		"Raw synthetic DJI-shaped telemetry fixture captured by a SemOps input component",
		func() any { return &RawTelemetryPayload{} },
	); err != nil {
		return err
	}
	return registerPayload(
		registry,
		DecodedTelemetryType,
		"Decoded synthetic DJI-shaped telemetry emitted by a SemOps processor component",
		func() any { return &DecodedTelemetryPayload{} },
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

func cloneRecord(record djicodec.TelemetryRecord) djicodec.TelemetryRecord {
	out := record
	out.ObservedAt = record.ObservedAt.UTC()
	out.Aircraft.AltitudeMSLM = cloneFloat64(record.Aircraft.AltitudeMSLM)
	out.Aircraft.AltitudeAGLM = cloneFloat64(record.Aircraft.AltitudeAGLM)
	out.Aircraft.HeadingDeg = cloneFloat64(record.Aircraft.HeadingDeg)
	out.Aircraft.GroundSpeedMPS = cloneFloat64(record.Aircraft.GroundSpeedMPS)
	out.Aircraft.VerticalSpeedMPS = cloneFloat64(record.Aircraft.VerticalSpeedMPS)
	out.Battery.Percent = cloneFloat64(record.Battery.Percent)
	out.Battery.VoltageV = cloneFloat64(record.Battery.VoltageV)
	out.Battery.TemperatureC = cloneFloat64(record.Battery.TemperatureC)
	out.Gimbal.YawDeg = cloneFloat64(record.Gimbal.YawDeg)
	out.Gimbal.PitchDeg = cloneFloat64(record.Gimbal.PitchDeg)
	out.Gimbal.RollDeg = cloneFloat64(record.Gimbal.RollDeg)
	out.Camera.ZoomRatio = cloneFloat64(record.Camera.ZoomRatio)
	out.MediaRefs = make([]djicodec.MediaRef, len(record.MediaRefs))
	for i, ref := range record.MediaRefs {
		out.MediaRefs[i] = ref
		out.MediaRefs[i].StartedAt = cloneTime(ref.StartedAt)
		out.MediaRefs[i].EndedAt = cloneTime(ref.EndedAt)
		out.MediaRefs[i].ByteLength = cloneInt64(ref.ByteLength)
	}
	return out
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	out := value.UTC()
	return &out
}
