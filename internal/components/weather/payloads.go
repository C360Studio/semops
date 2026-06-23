package weather

import (
	"encoding/json"
	"errors"
	"time"

	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	DefaultRawSubject     = "semops.feed.weather.raw"
	DefaultDecodedSubject = "semops.feed.weather.decoded"
)

var (
	RawForecastType = message.Type{
		Domain:   "semops",
		Category: "weather_raw_forecast",
		Version:  "v1",
	}
	DecodedForecastType = message.Type{
		Domain:   "semops",
		Category: "weather_point_forecast",
		Version:  "v1",
	}
)

type RawForecastPayload struct {
	Source      string    `json:"source"`
	Provider    string    `json:"provider"`
	QueryShape  string    `json:"query_shape"`
	Endpoint    string    `json:"endpoint,omitempty"`
	FixturePath string    `json:"fixture_path,omitempty"`
	FixtureURI  string    `json:"fixture_uri,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	RawJSON     []byte    `json:"raw_json"`
}

func NewRawForecastPayload(
	source string,
	provider string,
	queryShape string,
	fixturePath string,
	fixtureURI string,
	receivedAt time.Time,
	rawJSON []byte,
) *RawForecastPayload {
	return &RawForecastPayload{
		Source:      source,
		Provider:    provider,
		QueryShape:  queryShape,
		FixturePath: fixturePath,
		FixtureURI:  fixtureURI,
		ReceivedAt:  receivedAt.UTC(),
		RawJSON:     append([]byte(nil), rawJSON...),
	}
}

func (p *RawForecastPayload) Schema() message.Type {
	return RawForecastType
}

func (p *RawForecastPayload) Validate() error {
	if p == nil {
		return errors.New("raw weather forecast payload is nil")
	}
	if p.Source == "" {
		return errors.New("raw weather forecast payload source is required")
	}
	if p.Provider == "" {
		return errors.New("raw weather forecast payload provider is required")
	}
	if p.QueryShape == "" {
		return errors.New("raw weather forecast payload query_shape is required")
	}
	if p.Endpoint == "" && p.FixturePath == "" && p.FixtureURI == "" {
		return errors.New("raw weather forecast payload endpoint, fixture_path, or fixture_uri is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("raw weather forecast payload received_at is required")
	}
	if len(p.RawJSON) == 0 {
		return errors.New("raw weather forecast payload raw_json is required")
	}
	return nil
}

func (p *RawForecastPayload) MarshalJSON() ([]byte, error) {
	type alias RawForecastPayload
	return json.Marshal((*alias)(p))
}

func (p *RawForecastPayload) UnmarshalJSON(data []byte) error {
	type alias RawForecastPayload
	return json.Unmarshal(data, (*alias)(p))
}

type DecodedForecastPayload struct {
	Source     string                     `json:"source"`
	RawRef     string                     `json:"raw_ref"`
	ReceivedAt time.Time                  `json:"received_at"`
	Provider   string                     `json:"provider"`
	QueryShape string                     `json:"query_shape"`
	Forecast   weathercodec.PointForecast `json:"forecast"`
}

func NewDecodedForecastPayload(raw *RawForecastPayload, forecast weathercodec.PointForecast) *DecodedForecastPayload {
	rawRef := raw.FixtureURI
	if rawRef == "" {
		rawRef = raw.FixturePath
	}
	if rawRef == "" {
		rawRef = raw.Endpoint
	}
	return &DecodedForecastPayload{
		Source:     raw.Source,
		RawRef:     rawRef,
		ReceivedAt: raw.ReceivedAt.UTC(),
		Provider:   forecast.Provider,
		QueryShape: forecast.QueryShape,
		Forecast:   cloneForecast(forecast),
	}
}

func (p *DecodedForecastPayload) Schema() message.Type {
	return DecodedForecastType
}

func (p *DecodedForecastPayload) Validate() error {
	if p == nil {
		return errors.New("decoded weather forecast payload is nil")
	}
	if p.Source == "" {
		return errors.New("decoded weather forecast payload source is required")
	}
	if p.RawRef == "" {
		return errors.New("decoded weather forecast payload raw_ref is required")
	}
	if p.ReceivedAt.IsZero() {
		return errors.New("decoded weather forecast payload received_at is required")
	}
	if p.Provider == "" {
		return errors.New("decoded weather forecast payload provider is required")
	}
	if p.QueryShape == "" {
		return errors.New("decoded weather forecast payload query_shape is required")
	}
	if len(p.Forecast.Samples) == 0 {
		return errors.New("decoded weather forecast payload samples are required")
	}
	return nil
}

func (p *DecodedForecastPayload) ForecastCopy() (weathercodec.PointForecast, error) {
	if err := p.Validate(); err != nil {
		return weathercodec.PointForecast{}, err
	}
	return cloneForecast(p.Forecast), nil
}

func (p *DecodedForecastPayload) MarshalJSON() ([]byte, error) {
	type alias DecodedForecastPayload
	return json.Marshal((*alias)(p))
}

func (p *DecodedForecastPayload) UnmarshalJSON(data []byte) error {
	type alias DecodedForecastPayload
	return json.Unmarshal(data, (*alias)(p))
}

func RegisterPayloads(registry *payloadregistry.Registry) error {
	if registry == nil {
		return errors.New("payload registry is nil")
	}
	if err := registerPayload(
		registry,
		RawForecastType,
		"Raw provider-shaped weather forecast captured by a SemOps input component",
		func() any { return &RawForecastPayload{} },
	); err != nil {
		return err
	}
	return registerPayload(
		registry,
		DecodedForecastType,
		"Decoded weather point forecast emitted by a SemOps processor component",
		func() any { return &DecodedForecastPayload{} },
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

func cloneForecast(forecast weathercodec.PointForecast) weathercodec.PointForecast {
	out := forecast
	out.ElevationM = cloneFloat64(forecast.ElevationM)
	out.GenerationTimeMS = cloneFloat64(forecast.GenerationTimeMS)
	out.Units = make(map[string]string, len(forecast.Units))
	for key, value := range forecast.Units {
		out.Units[key] = value
	}
	out.Samples = make([]weathercodec.WeatherSample, len(forecast.Samples))
	for i, sample := range forecast.Samples {
		out.Samples[i] = sample
		out.Samples[i].Time = sample.Time.UTC()
		out.Samples[i].TemperatureC = cloneFloat64(sample.TemperatureC)
		out.Samples[i].PrecipitationMM = cloneFloat64(sample.PrecipitationMM)
		out.Samples[i].VisibilityM = cloneFloat64(sample.VisibilityM)
		out.Samples[i].SurfacePressureHPA = cloneFloat64(sample.SurfacePressureHPA)
		out.Samples[i].WindSpeed10MKPH = cloneFloat64(sample.WindSpeed10MKPH)
		out.Samples[i].WindGusts10MKPH = cloneFloat64(sample.WindGusts10MKPH)
		out.Samples[i].WindDirection10Deg = cloneFloat64(sample.WindDirection10Deg)
		out.Samples[i].WeatherCode = cloneInt(sample.WeatherCode)
		out.Samples[i].SupportedFieldNames = append([]string(nil), sample.SupportedFieldNames...)
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

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}
