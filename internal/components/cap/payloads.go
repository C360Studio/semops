package cap

import (
	"encoding/json"
	"errors"
	"time"

	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.cap.raw"
	DefaultDecodedSubject = "semops.feed.cap.decoded"
)

var (
	RawAlertType = message.Type{
		Domain:   "semops",
		Category: "cap_raw_alert",
		Version:  "v1",
	}
	DecodedAlertType = message.Type{
		Domain:   "semops",
		Category: "cap_alert",
		Version:  "v1",
	}
)

type RawAlertPayload struct {
	Source       string    `json:"source"`
	Endpoint     string    `json:"endpoint,omitempty"`
	ReceivedAt   time.Time `json:"received_at"`
	StatusCode   int       `json:"status_code,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	RawXML       []byte    `json:"raw_xml"`
}

func NewRawAlertPayload(
	source string,
	endpoint string,
	receivedAt time.Time,
	statusCode int,
	rawXML []byte,
) *RawAlertPayload {
	return &RawAlertPayload{
		Source:     source,
		Endpoint:   endpoint,
		ReceivedAt: receivedAt.UTC(),
		StatusCode: statusCode,
		RawXML:     append([]byte(nil), rawXML...),
	}
}

func (p *RawAlertPayload) Schema() message.Type {
	return RawAlertType
}

func (p *RawAlertPayload) Validate() error {
	if p == nil {
		return errors.New("raw CAP payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw CAP payload source is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw CAP payload received_at is required")
	}
	if p.StatusCode != 0 && (p.StatusCode < 100 || p.StatusCode > 599) {
		return errors.New("raw CAP payload status_code is invalid")
	}
	if len(p.RawXML) == 0 {
		return errors.New("raw CAP payload raw_xml is required")
	}
	return nil
}

func (p *RawAlertPayload) MarshalJSON() ([]byte, error) {
	type alias RawAlertPayload
	return json.Marshal((*alias)(p))
}

func (p *RawAlertPayload) UnmarshalJSON(data []byte) error {
	type alias RawAlertPayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedAlertPayload struct {
	Source     string         `json:"source"`
	RawRef     string         `json:"raw_ref"`
	ReceivedAt time.Time      `json:"received_at"`
	Alert      capcodec.Alert `json:"alert"`
}

func NewDecodedAlertPayload(source string, record capcodec.RawAlertRecord, alert capcodec.Alert) *DecodedAlertPayload {
	return &DecodedAlertPayload{
		Source:     source,
		RawRef:     record.Ref,
		ReceivedAt: record.ReceivedAt.UTC(),
		Alert:      cloneAlert(alert),
	}
}

func (p *DecodedAlertPayload) Schema() message.Type {
	return DecodedAlertType
}

func (p *DecodedAlertPayload) Validate() error {
	if p == nil {
		return errors.New("decoded CAP payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded CAP payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded CAP payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded CAP payload received_at is required")
	}
	return p.Alert.Validate()
}

func (p *DecodedAlertPayload) CAPAlert() (capcodec.Alert, error) {
	if err := p.Validate(); err != nil {
		return capcodec.Alert{}, err
	}
	return cloneAlert(p.Alert), nil
}

func (p *DecodedAlertPayload) MarshalJSON() ([]byte, error) {
	type alias DecodedAlertPayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedAlertPayload) UnmarshalJSON(data []byte) error {
	type alias DecodedAlertPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(registry, RawAlertType, "Raw CAP XML captured by a SemOps input component", func() any {
		return &RawAlertPayload{}
	}); err != nil {
		return err
	}
	return registerPayload(
		registry,
		DecodedAlertType,
		"Decoded CAP alert emitted by a SemOps processor component",
		func() any {
			return &DecodedAlertPayload{}
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

func cloneAlert(alert capcodec.Alert) capcodec.Alert {
	alert.Infos = append([]capcodec.Info(nil), alert.Infos...)
	for i := range alert.Infos {
		alert.Infos[i] = cloneInfo(alert.Infos[i])
	}
	return alert
}

func cloneInfo(info capcodec.Info) capcodec.Info {
	info.Categories = append([]string(nil), info.Categories...)
	info.Parameters = append([]capcodec.NameValue(nil), info.Parameters...)
	info.Resources = append([]capcodec.Resource(nil), info.Resources...)
	info.Areas = append([]capcodec.Area(nil), info.Areas...)
	for i := range info.Areas {
		info.Areas[i] = cloneArea(info.Areas[i])
	}
	return info
}

func cloneArea(area capcodec.Area) capcodec.Area {
	area.Polygons = append([][]capcodec.Point(nil), area.Polygons...)
	for i := range area.Polygons {
		area.Polygons[i] = append([]capcodec.Point(nil), area.Polygons[i]...)
	}
	area.Circles = append([]capcodec.Circle(nil), area.Circles...)
	area.Geocodes = append([]capcodec.NameValue(nil), area.Geocodes...)
	return area
}
