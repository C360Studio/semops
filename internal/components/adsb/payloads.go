package adsb

import (
	"encoding/json"
	"errors"
	"time"

	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.adsb.raw"
	DefaultDecodedSubject = "semops.feed.adsb.decoded"
)

var (
	RawSnapshotType = message.Type{
		Domain:   "semops",
		Category: "adsb_raw_snapshot",
		Version:  "v1",
	}
	DecodedSnapshotType = message.Type{
		Domain:   "semops",
		Category: "adsb_snapshot",
		Version:  "v1",
	}
)

type RawSnapshotPayload struct {
	Source       string    `json:"source"`
	Endpoint     string    `json:"endpoint,omitempty"`
	ReceivedAt   time.Time `json:"received_at"`
	StatusCode   int       `json:"status_code,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	RawJSON      []byte    `json:"raw_json"`
}

func NewRawSnapshotPayload(
	source string,
	endpoint string,
	receivedAt time.Time,
	statusCode int,
	rawJSON []byte,
) *RawSnapshotPayload {
	return &RawSnapshotPayload{
		Source:     source,
		Endpoint:   endpoint,
		ReceivedAt: receivedAt.UTC(),
		StatusCode: statusCode,
		RawJSON:    append([]byte(nil), rawJSON...),
	}
}

func (p *RawSnapshotPayload) Schema() message.Type {
	return RawSnapshotType
}

func (p *RawSnapshotPayload) Validate() error {
	if p == nil {
		return errors.New("raw ADS-B payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw ADS-B payload source is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw ADS-B payload received_at is required")
	}
	if p.StatusCode != 0 && (p.StatusCode < 100 || p.StatusCode > 599) {
		return errors.New("raw ADS-B payload status_code is invalid")
	}
	if len(p.RawJSON) == 0 {
		return errors.New("raw ADS-B payload raw_json is required")
	}
	return nil
}

func (p *RawSnapshotPayload) MarshalJSON() ([]byte, error) {
	type alias RawSnapshotPayload
	return json.Marshal((*alias)(p))
}

func (p *RawSnapshotPayload) UnmarshalJSON(data []byte) error {
	type alias RawSnapshotPayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedSnapshotPayload struct {
	Source     string                    `json:"source"`
	RawRef     string                    `json:"raw_ref"`
	ReceivedAt time.Time                 `json:"received_at"`
	Snapshot   adsbcodec.OpenSkySnapshot `json:"snapshot"`
}

func NewDecodedSnapshotPayload(
	record adsbcodec.RawSnapshotRecord,
	snapshot adsbcodec.OpenSkySnapshot,
) *DecodedSnapshotPayload {
	return &DecodedSnapshotPayload{
		Source:     record.Source,
		RawRef:     record.Ref,
		ReceivedAt: record.ReceivedAt.UTC(),
		Snapshot:   cloneSnapshot(snapshot),
	}
}

func (p *DecodedSnapshotPayload) Schema() message.Type {
	return DecodedSnapshotType
}

func (p *DecodedSnapshotPayload) Validate() error {
	if p == nil {
		return errors.New("decoded ADS-B payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded ADS-B payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded ADS-B payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded ADS-B payload received_at is required")
	}
	if p.Snapshot.Time.IsZero() {
		return errors.New("decoded ADS-B payload snapshot time is required")
	}
	if p.Snapshot.States == nil {
		return errors.New("decoded ADS-B payload snapshot states are required")
	}
	return nil
}

func (p *DecodedSnapshotPayload) SnapshotCopy() (adsbcodec.OpenSkySnapshot, error) {
	if err := p.Validate(); err != nil {
		return adsbcodec.OpenSkySnapshot{}, err
	}
	return cloneSnapshot(p.Snapshot), nil
}

func (p *DecodedSnapshotPayload) MarshalJSON() ([]byte, error) {
	type alias DecodedSnapshotPayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedSnapshotPayload) UnmarshalJSON(data []byte) error {
	type alias DecodedSnapshotPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(
		registry,
		RawSnapshotType,
		"Raw ADS-B OpenSky snapshot captured by a SemOps input component",
		func() any {
			return &RawSnapshotPayload{}
		},
	); err != nil {
		return err
	}
	return registerPayload(
		registry,
		DecodedSnapshotType,
		"Decoded ADS-B OpenSky snapshot emitted by a SemOps processor component",
		func() any {
			return &DecodedSnapshotPayload{}
		},
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

func cloneSnapshot(snapshot adsbcodec.OpenSkySnapshot) adsbcodec.OpenSkySnapshot {
	out := adsbcodec.OpenSkySnapshot{
		Time:   snapshot.Time.UTC(),
		States: append([]adsbcodec.StateVector(nil), snapshot.States...),
	}
	for i := range out.States {
		out.States[i] = cloneState(out.States[i])
	}
	return out
}

func cloneState(state adsbcodec.StateVector) adsbcodec.StateVector {
	if state.Callsign != nil {
		value := *state.Callsign
		state.Callsign = &value
	}
	if state.TimePosition != nil {
		value := state.TimePosition.UTC()
		state.TimePosition = &value
	}
	if state.Longitude != nil {
		value := *state.Longitude
		state.Longitude = &value
	}
	if state.Latitude != nil {
		value := *state.Latitude
		state.Latitude = &value
	}
	if state.BaroAltitudeM != nil {
		value := *state.BaroAltitudeM
		state.BaroAltitudeM = &value
	}
	if state.VelocityMPS != nil {
		value := *state.VelocityMPS
		state.VelocityMPS = &value
	}
	if state.TrueTrackDeg != nil {
		value := *state.TrueTrackDeg
		state.TrueTrackDeg = &value
	}
	if state.VerticalRateMPS != nil {
		value := *state.VerticalRateMPS
		state.VerticalRateMPS = &value
	}
	state.SensorIDs = append([]int(nil), state.SensorIDs...)
	if state.GeoAltitudeM != nil {
		value := *state.GeoAltitudeM
		state.GeoAltitudeM = &value
	}
	if state.Squawk != nil {
		value := *state.Squawk
		state.Squawk = &value
	}
	if state.Category != nil {
		value := *state.Category
		state.Category = &value
	}
	state.LastContact = state.LastContact.UTC()
	return state
}
