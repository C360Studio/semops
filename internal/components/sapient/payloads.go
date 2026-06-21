package sapient

import (
	"encoding/json"
	"errors"
	"time"

	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.sapient.raw"
	DefaultDecodedSubject = "semops.feed.sapient.decoded"
)

var (
	RawMessageType = message.Type{
		Domain:   "semops",
		Category: "sapient_raw_message",
		Version:  "v1",
	}
	DecodedMessageType = message.Type{
		Domain:   "semops",
		Category: "sapient_message",
		Version:  "v1",
	}
)

type RawMessagePayload struct {
	Source      string                `json:"source"`
	Endpoint    string                `json:"endpoint,omitempty"`
	ReceivedAt  time.Time             `json:"received_at"`
	StatusCode  int                   `json:"status_code,omitempty"`
	ContentType string                `json:"content_type,omitempty"`
	Encoding    sapientcodec.Encoding `json:"encoding"`
	RawPayload  []byte                `json:"raw_payload"`
}

func NewRawMessagePayload(
	source string,
	endpoint string,
	receivedAt time.Time,
	statusCode int,
	encoding sapientcodec.Encoding,
	rawPayload []byte,
) *RawMessagePayload {
	return &RawMessagePayload{
		Source:     source,
		Endpoint:   endpoint,
		ReceivedAt: receivedAt.UTC(),
		StatusCode: statusCode,
		Encoding:   encoding,
		RawPayload: append([]byte(nil), rawPayload...),
	}
}

func (p *RawMessagePayload) Schema() message.Type {
	return RawMessageType
}

func (p *RawMessagePayload) Validate() error {
	if p == nil {
		return errors.New("raw SAPIENT payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw SAPIENT payload source is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw SAPIENT payload received_at is required")
	}
	if p.StatusCode != 0 && (p.StatusCode < 100 || p.StatusCode > 599) {
		return errors.New("raw SAPIENT payload status_code is invalid")
	}
	if !p.Encoding.Valid() {
		return errors.New("raw SAPIENT payload encoding is invalid")
	}
	if len(p.RawPayload) == 0 {
		return errors.New("raw SAPIENT payload raw_payload is required")
	}
	return nil
}

func (p *RawMessagePayload) MarshalJSON() ([]byte, error) {
	type alias RawMessagePayload
	return json.Marshal((*alias)(p))
}

func (p *RawMessagePayload) UnmarshalJSON(data []byte) error {
	type alias RawMessagePayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedMessagePayload struct {
	Source     string                   `json:"source"`
	RawRef     string                   `json:"raw_ref"`
	ReceivedAt time.Time                `json:"received_at"`
	Encoding   sapientcodec.Encoding    `json:"encoding"`
	Content    sapientcodec.ContentKind `json:"content"`
	NodeID     string                   `json:"node_id"`
	MessageAt  time.Time                `json:"message_at"`
	Message    sapientcodec.Message     `json:"message"`
}

func NewDecodedMessagePayload(record sapientcodec.RawMessageRecord, msg sapientcodec.Message) *DecodedMessagePayload {
	return &DecodedMessagePayload{
		Source:     record.Source,
		RawRef:     record.Ref,
		ReceivedAt: record.ReceivedAt.UTC(),
		Encoding:   record.Encoding,
		Content:    msg.Content,
		NodeID:     msg.NodeID,
		MessageAt:  msg.Timestamp.UTC(),
		Message:    cloneMessage(msg),
	}
}

func (p *DecodedMessagePayload) Schema() message.Type {
	return DecodedMessageType
}

func (p *DecodedMessagePayload) Validate() error {
	if p == nil {
		return errors.New("decoded SAPIENT payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded SAPIENT payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded SAPIENT payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded SAPIENT payload received_at is required")
	}
	if !p.Encoding.Valid() {
		return errors.New("decoded SAPIENT payload encoding is invalid")
	}
	if p.Content == "" {
		return errors.New("decoded SAPIENT payload content is required")
	}
	if p.NodeID == "" {
		return errors.New("decoded SAPIENT payload node_id is required")
	}
	if p.MessageAt.IsZero() {
		return errors.New("decoded SAPIENT payload message_at is required")
	}
	return nil
}

func (p *DecodedMessagePayload) MessageCopy() (sapientcodec.Message, error) {
	if err := p.Validate(); err != nil {
		return sapientcodec.Message{}, err
	}
	return cloneMessage(p.Message), nil
}

func (p *DecodedMessagePayload) MarshalJSON() ([]byte, error) {
	type alias DecodedMessagePayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedMessagePayload) UnmarshalJSON(data []byte) error {
	type alias DecodedMessagePayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(
		registry,
		RawMessageType,
		"Raw SAPIENT JSON or protobuf payload captured by a SemOps input component",
		func() any { return &RawMessagePayload{} },
	); err != nil {
		return err
	}
	return registerPayload(
		registry,
		DecodedMessageType,
		"Decoded SAPIENT preflight message emitted by a SemOps processor component",
		func() any { return &DecodedMessagePayload{} },
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

func cloneMessage(msg sapientcodec.Message) sapientcodec.Message {
	out := msg
	if msg.DestinationID != nil {
		value := *msg.DestinationID
		out.DestinationID = &value
	}
	out.Timestamp = msg.Timestamp.UTC()
	out.DetectionReport = cloneDetectionReport(msg.DetectionReport)
	out.Registration = cloneRegistration(msg.Registration)
	out.StatusReport = cloneStatusReport(msg.StatusReport)
	out.TaskAck = cloneTaskAck(msg.TaskAck)
	return out
}

func cloneDetectionReport(report *sapientcodec.DetectionReport) *sapientcodec.DetectionReport {
	if report == nil {
		return nil
	}
	out := *report
	if report.TaskID != nil {
		value := *report.TaskID
		out.TaskID = &value
	}
	out.Location = cloneLocation(report.Location)
	out.RangeBearing = cloneRangeBearing(report.RangeBearing)
	if report.DetectionConfidence != nil {
		value := *report.DetectionConfidence
		out.DetectionConfidence = &value
	}
	out.Classifications = cloneClassifications(report.Classifications)
	out.AssociatedDetection = cloneDetectionRefs(report.AssociatedDetection)
	out.DerivedDetection = cloneDetectionRefs(report.DerivedDetection)
	return &out
}

func cloneRegistration(registration *sapientcodec.Registration) *sapientcodec.Registration {
	if registration == nil {
		return nil
	}
	out := *registration
	out.NodeTypes = append([]string(nil), registration.NodeTypes...)
	out.Capabilities = append([]sapientcodec.Capability(nil), registration.Capabilities...)
	return &out
}

func cloneStatusReport(status *sapientcodec.StatusReport) *sapientcodec.StatusReport {
	if status == nil {
		return nil
	}
	out := *status
	if status.ActiveTaskID != nil {
		value := *status.ActiveTaskID
		out.ActiveTaskID = &value
	}
	out.NodeLocation = cloneLocation(status.NodeLocation)
	return &out
}

func cloneTaskAck(taskAck *sapientcodec.TaskAck) *sapientcodec.TaskAck {
	if taskAck == nil {
		return nil
	}
	out := *taskAck
	out.Reason = append([]string(nil), taskAck.Reason...)
	return &out
}

func cloneLocation(location *sapientcodec.Location) *sapientcodec.Location {
	if location == nil {
		return nil
	}
	out := *location
	if location.Z != nil {
		value := *location.Z
		out.Z = &value
	}
	return &out
}

func cloneRangeBearing(rangeBearing *sapientcodec.RangeBearing) *sapientcodec.RangeBearing {
	if rangeBearing == nil {
		return nil
	}
	out := *rangeBearing
	if rangeBearing.Elevation != nil {
		value := *rangeBearing.Elevation
		out.Elevation = &value
	}
	if rangeBearing.Azimuth != nil {
		value := *rangeBearing.Azimuth
		out.Azimuth = &value
	}
	if rangeBearing.Range != nil {
		value := *rangeBearing.Range
		out.Range = &value
	}
	return &out
}

func cloneClassifications(items []sapientcodec.Classification) []sapientcodec.Classification {
	out := append([]sapientcodec.Classification(nil), items...)
	for i := range out {
		if out[i].Confidence != nil {
			value := *out[i].Confidence
			out[i].Confidence = &value
		}
		out[i].SubClass = cloneSubClasses(out[i].SubClass)
	}
	return out
}

func cloneSubClasses(items []sapientcodec.SubClass) []sapientcodec.SubClass {
	out := append([]sapientcodec.SubClass(nil), items...)
	for i := range out {
		if out[i].Confidence != nil {
			value := *out[i].Confidence
			out[i].Confidence = &value
		}
		out[i].SubClass = cloneSubClasses(out[i].SubClass)
	}
	return out
}

func cloneDetectionRefs(items []sapientcodec.DetectionRef) []sapientcodec.DetectionRef {
	out := append([]sapientcodec.DetectionRef(nil), items...)
	for i := range out {
		if out[i].Timestamp != nil {
			value := out[i].Timestamp.UTC()
			out[i].Timestamp = &value
		}
	}
	return out
}
