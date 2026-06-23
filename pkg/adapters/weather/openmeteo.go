// Package weather implements first-slice weather fixture parsing for SemOps.
//
// The initial boundary is intentionally parser-only. It accepts deterministic
// provider-shaped JSON and does not claim live weather service reliability,
// OGC EDR conformance, radar tiles, or route-safety recommendations.
package weather

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	ProviderOpenMeteo    = "open-meteo"
	ProviderOGCEDR       = "ogc-edr"
	QueryShapePosition   = "position"
	QueryShapeArea       = "area"
	QueryShapeTrajectory = "trajectory"
	QueryShapeCorridor   = "corridor"
)

type PointForecast struct {
	Provider         string
	QueryShape       string
	Latitude         float64
	Longitude        float64
	ElevationM       *float64
	Timezone         string
	TimezoneAbbrev   string
	UTCOffsetSeconds int
	GenerationTimeMS *float64
	Units            map[string]string
	Samples          []WeatherSample
}

type WeatherSample struct {
	Time                time.Time
	TemperatureC        *float64
	PrecipitationMM     *float64
	VisibilityM         *float64
	SurfacePressureHPA  *float64
	WindSpeed10MKPH     *float64
	WindGusts10MKPH     *float64
	WindDirection10Deg  *float64
	WeatherCode         *int
	SupportedFieldNames []string
}

func ParseOpenMeteoPointForecast(data []byte) (PointForecast, error) {
	var raw openMeteoResponse
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return PointForecast{}, err
	}
	if err := raw.validate(); err != nil {
		return PointForecast{}, err
	}

	times, err := raw.timeValues()
	if err != nil {
		return PointForecast{}, err
	}
	out := PointForecast{
		Provider:         ProviderOpenMeteo,
		QueryShape:       QueryShapePosition,
		Latitude:         raw.Latitude,
		Longitude:        raw.Longitude,
		ElevationM:       raw.Elevation,
		Timezone:         strings.TrimSpace(raw.Timezone),
		TimezoneAbbrev:   strings.TrimSpace(raw.TimezoneAbbrev),
		UTCOffsetSeconds: raw.UTCOffsetSeconds,
		GenerationTimeMS: raw.GenerationTimeMS,
		Units:            copyUnits(raw.HourlyUnits),
		Samples:          make([]WeatherSample, 0, len(times)),
	}
	for i, sampleTime := range times {
		sample, err := raw.sampleAt(i, len(times))
		if err != nil {
			return PointForecast{}, fmt.Errorf("open-meteo hourly sample %d: %w", i+1, err)
		}
		sample.Time = sampleTime
		out.Samples = append(out.Samples, sample)
	}
	return out, nil
}

type openMeteoResponse struct {
	Latitude         float64                      `json:"latitude"`
	Longitude        float64                      `json:"longitude"`
	GenerationTimeMS *float64                     `json:"generationtime_ms"`
	UTCOffsetSeconds int                          `json:"utc_offset_seconds"`
	Timezone         string                       `json:"timezone"`
	TimezoneAbbrev   string                       `json:"timezone_abbreviation"`
	Elevation        *float64                     `json:"elevation"`
	HourlyUnits      map[string]string            `json:"hourly_units"`
	Hourly           map[string][]json.RawMessage `json:"hourly"`
}

func (r openMeteoResponse) validate() error {
	if r.Latitude < -90 || r.Latitude > 90 {
		return fmt.Errorf("latitude %v out of range", r.Latitude)
	}
	if r.Longitude < -180 || r.Longitude > 180 {
		return fmt.Errorf("longitude %v out of range", r.Longitude)
	}
	if len(r.Hourly) == 0 {
		return errors.New("hourly weather block is required")
	}
	if _, ok := r.Hourly["time"]; !ok {
		return errors.New("hourly time array is required")
	}
	return nil
}

func (r openMeteoResponse) timeValues() ([]time.Time, error) {
	rawTimes := r.Hourly["time"]
	if len(rawTimes) == 0 {
		return nil, errors.New("hourly time array must not be empty")
	}
	out := make([]time.Time, 0, len(rawTimes))
	for i, raw := range rawTimes {
		value, err := stringValue(raw, "time")
		if err != nil {
			return nil, fmt.Errorf("time %d: %w", i+1, err)
		}
		parsed, err := parseOpenMeteoTime(value, r.UTCOffsetSeconds)
		if err != nil {
			return nil, fmt.Errorf("time %d: %w", i+1, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func (r openMeteoResponse) sampleAt(index int, sampleCount int) (WeatherSample, error) {
	var sample WeatherSample
	var err error
	if sample.TemperatureC, err = r.floatAt("temperature_2m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.PrecipitationMM, err = r.floatAt("precipitation", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.VisibilityM, err = r.floatAt("visibility", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.SurfacePressureHPA, err = r.floatAt("surface_pressure", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WindSpeed10MKPH, err = r.floatAt("wind_speed_10m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WindGusts10MKPH, err = r.floatAt("wind_gusts_10m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WindDirection10Deg, err = r.floatAt("wind_direction_10m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WeatherCode, err = r.intAt("weather_code", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	sample.SupportedFieldNames = sample.supportedFields()
	return sample, nil
}

func (r openMeteoResponse) floatAt(field string, index int, sampleCount int) (*float64, error) {
	values, ok := r.Hourly[field]
	if !ok {
		return nil, nil
	}
	if len(values) != sampleCount {
		return nil, fmt.Errorf("%s length %d does not match time length %d", field, len(values), sampleCount)
	}
	return optionalFloat(values[index], field)
}

func (r openMeteoResponse) intAt(field string, index int, sampleCount int) (*int, error) {
	value, err := r.floatAt(field, index, sampleCount)
	if err != nil || value == nil {
		return nil, err
	}
	if math.Trunc(*value) != *value {
		return nil, fmt.Errorf("%s must be an integer code, got %v", field, *value)
	}
	code := int(*value)
	return &code, nil
}

func (s WeatherSample) supportedFields() []string {
	fields := make([]string, 0, 8)
	if s.TemperatureC != nil {
		fields = append(fields, "temperature_2m")
	}
	if s.PrecipitationMM != nil {
		fields = append(fields, "precipitation")
	}
	if s.VisibilityM != nil {
		fields = append(fields, "visibility")
	}
	if s.SurfacePressureHPA != nil {
		fields = append(fields, "surface_pressure")
	}
	if s.WindSpeed10MKPH != nil {
		fields = append(fields, "wind_speed_10m")
	}
	if s.WindGusts10MKPH != nil {
		fields = append(fields, "wind_gusts_10m")
	}
	if s.WindDirection10Deg != nil {
		fields = append(fields, "wind_direction_10m")
	}
	if s.WeatherCode != nil {
		fields = append(fields, "weather_code")
	}
	return fields
}

func stringValue(raw json.RawMessage, field string) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return value, nil
}

func optionalFloat(raw json.RawMessage, field string) (*float64, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return &value, nil
}

func parseOpenMeteoTime(value string, utcOffsetSeconds int) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse("2006-01-02T15:04", value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.Add(-time.Duration(utcOffsetSeconds) * time.Second).UTC(), nil
}

func copyUnits(units map[string]string) map[string]string {
	if len(units) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(units))
	for key, value := range units {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}
