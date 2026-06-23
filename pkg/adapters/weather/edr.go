package weather

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const SyntheticOGCEDRPositionFixtureClass = "semops.synthetic.ogc-edr.position.v1"

func ParseOGCEDRPositionForecast(data []byte) (PointForecast, error) {
	var raw edrPositionResponse
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return PointForecast{}, err
	}
	if err := raw.validate(); err != nil {
		return PointForecast{}, err
	}

	lon, lat, err := parseWKTPoint(raw.EDR.Query.Coords)
	if err != nil {
		return PointForecast{}, err
	}
	if err := raw.Coverage.validatePointAxes(lon, lat); err != nil {
		return PointForecast{}, err
	}
	times, err := raw.Coverage.timeValues()
	if err != nil {
		return PointForecast{}, err
	}
	elevation, err := raw.Coverage.optionalAxisFloatValue("z")
	if err != nil {
		return PointForecast{}, err
	}

	out := PointForecast{
		Provider:   ProviderOGCEDR,
		QueryShape: QueryShapePosition,
		Latitude:   lat,
		Longitude:  lon,
		ElevationM: elevation,
		Units:      raw.Coverage.unitMap(),
		Samples:    make([]WeatherSample, 0, len(times)),
	}
	for i, sampleTime := range times {
		sample, err := raw.Coverage.sampleAt(i, len(times))
		if err != nil {
			return PointForecast{}, fmt.Errorf("ogc edr coverage sample %d: %w", i+1, err)
		}
		sample.Time = sampleTime
		out.Samples = append(out.Samples, sample)
	}
	return out, nil
}

type edrPositionResponse struct {
	FixtureClass string      `json:"fixture_class"`
	EDR          edrMetadata `json:"edr"`
	Coverage     edrCoverage `json:"coverage"`
}

type edrMetadata struct {
	API          string   `json:"api"`
	Version      string   `json:"version"`
	CollectionID string   `json:"collection_id"`
	QueryType    string   `json:"query_type"`
	Query        edrQuery `json:"query"`
}

type edrQuery struct {
	Coords         string   `json:"coords"`
	DateTime       string   `json:"datetime"`
	ParameterNames []string `json:"parameter_names"`
	CRS            string   `json:"crs"`
	Format         string   `json:"f"`
}

type edrCoverage struct {
	Type       string                  `json:"type"`
	Domain     edrDomain               `json:"domain"`
	Parameters map[string]edrParameter `json:"parameters"`
	Ranges     map[string]edrRange     `json:"ranges"`
}

type edrDomain struct {
	Type       string             `json:"type"`
	DomainType string             `json:"domainType"`
	Axes       map[string]edrAxis `json:"axes"`
}

type edrAxis struct {
	Values []json.RawMessage `json:"values"`
}

type edrParameter struct {
	Unit edrUnit `json:"unit"`
}

type edrUnit struct {
	Label  string          `json:"label"`
	Symbol json.RawMessage `json:"symbol"`
}

type edrRange struct {
	Values []json.RawMessage `json:"values"`
	Shape  []int             `json:"shape"`
}

func (r edrPositionResponse) validate() error {
	if strings.TrimSpace(r.FixtureClass) != SyntheticOGCEDRPositionFixtureClass {
		return fmt.Errorf("ogc edr fixture_class must be %q", SyntheticOGCEDRPositionFixtureClass)
	}
	if strings.TrimSpace(r.EDR.QueryType) != QueryShapePosition {
		return fmt.Errorf("ogc edr query_type must be %q", QueryShapePosition)
	}
	if strings.TrimSpace(r.EDR.Query.Coords) == "" {
		return errors.New("ogc edr query coords are required")
	}
	if strings.TrimSpace(r.Coverage.Type) != "Coverage" {
		return errors.New("ogc edr coverage type must be Coverage")
	}
	if len(r.Coverage.Domain.Axes) == 0 {
		return errors.New("ogc edr coverage domain axes are required")
	}
	if len(r.Coverage.Ranges) == 0 {
		return errors.New("ogc edr coverage ranges are required")
	}
	return nil
}

func (c edrCoverage) validatePointAxes(lon, lat float64) error {
	axisLon, err := c.requiredAxisFloatValue("x")
	if err != nil {
		return err
	}
	axisLat, err := c.requiredAxisFloatValue("y")
	if err != nil {
		return err
	}
	if math.Abs(axisLon-lon) > 0.0000001 {
		return fmt.Errorf("ogc edr x axis longitude %v does not match coords longitude %v", axisLon, lon)
	}
	if math.Abs(axisLat-lat) > 0.0000001 {
		return fmt.Errorf("ogc edr y axis latitude %v does not match coords latitude %v", axisLat, lat)
	}
	return nil
}

func (c edrCoverage) timeValues() ([]time.Time, error) {
	axis, ok := c.Domain.Axes["t"]
	if !ok || len(axis.Values) == 0 {
		return nil, errors.New("ogc edr t axis values are required")
	}
	out := make([]time.Time, 0, len(axis.Values))
	for i, raw := range axis.Values {
		value, err := stringValue(raw, "t")
		if err != nil {
			return nil, fmt.Errorf("ogc edr t axis value %d: %w", i+1, err)
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("ogc edr t axis value %d: %w", i+1, err)
		}
		out = append(out, parsed.UTC())
	}
	return out, nil
}

func (c edrCoverage) sampleAt(index int, sampleCount int) (WeatherSample, error) {
	var sample WeatherSample
	var err error
	if sample.TemperatureC, err = c.floatAt("temperature_2m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.PrecipitationMM, err = c.floatAt("precipitation", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.VisibilityM, err = c.floatAt("visibility", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.SurfacePressureHPA, err = c.floatAt("surface_pressure", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WindSpeed10MKPH, err = c.floatAt("wind_speed_10m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WindGusts10MKPH, err = c.floatAt("wind_gusts_10m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WindDirection10Deg, err = c.floatAt("wind_direction_10m", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	if sample.WeatherCode, err = c.intAt("weather_code", index, sampleCount); err != nil {
		return WeatherSample{}, err
	}
	sample.SupportedFieldNames = sample.supportedFields()
	return sample, nil
}

func (c edrCoverage) floatAt(field string, index int, sampleCount int) (*float64, error) {
	values, ok := c.Ranges[field]
	if !ok {
		return nil, nil
	}
	if len(values.Shape) > 0 && values.Shape[0] != sampleCount {
		return nil, fmt.Errorf("%s shape %d does not match time length %d", field, values.Shape[0], sampleCount)
	}
	if len(values.Values) != sampleCount {
		return nil, fmt.Errorf("%s length %d does not match time length %d", field, len(values.Values), sampleCount)
	}
	return optionalFloat(values.Values[index], field)
}

func (c edrCoverage) intAt(field string, index int, sampleCount int) (*int, error) {
	value, err := c.floatAt(field, index, sampleCount)
	if err != nil || value == nil {
		return nil, err
	}
	if math.Trunc(*value) != *value {
		return nil, fmt.Errorf("%s must be an integer code, got %v", field, *value)
	}
	code := int(*value)
	return &code, nil
}

func (c edrCoverage) requiredAxisFloatValue(axisName string) (float64, error) {
	value, err := c.optionalAxisFloatValue(axisName)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, fmt.Errorf("ogc edr %s axis numeric value is required", axisName)
	}
	return *value, nil
}

func (c edrCoverage) optionalAxisFloatValue(axisName string) (*float64, error) {
	axis, ok := c.Domain.Axes[axisName]
	if !ok || len(axis.Values) == 0 {
		return nil, nil
	}
	value, err := optionalFloat(axis.Values[0], axisName)
	if err != nil {
		return nil, fmt.Errorf("ogc edr %s axis: %w", axisName, err)
	}
	return value, nil
}

func (c edrCoverage) unitMap() map[string]string {
	if len(c.Parameters) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(c.Parameters))
	for field, parameter := range c.Parameters {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if symbol := parameter.Unit.symbolValue(); symbol != "" {
			out[field] = symbol
		}
	}
	return out
}

func (u edrUnit) symbolValue() string {
	if len(bytes.TrimSpace(u.Symbol)) > 0 {
		var object struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(u.Symbol, &object); err == nil {
			if value := strings.TrimSpace(object.Value); value != "" {
				return value
			}
		}
		var text string
		if err := json.Unmarshal(u.Symbol, &text); err == nil {
			if value := strings.TrimSpace(text); value != "" {
				return value
			}
		}
	}
	return strings.TrimSpace(u.Label)
}

func parseWKTPoint(value string) (float64, float64, error) {
	value = strings.TrimSpace(value)
	upper := strings.ToUpper(value)
	if !strings.HasPrefix(upper, "POINT") {
		return 0, 0, fmt.Errorf("ogc edr coords must be WKT POINT, got %q", value)
	}
	open := strings.Index(value, "(")
	close := strings.LastIndex(value, ")")
	if open < 0 || close <= open {
		return 0, 0, fmt.Errorf("ogc edr coords must be WKT POINT, got %q", value)
	}
	parts := strings.Fields(value[open+1 : close])
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("ogc edr coords POINT must contain longitude latitude, got %q", value)
	}
	lon, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("ogc edr coords longitude: %w", err)
	}
	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("ogc edr coords latitude: %w", err)
	}
	if lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("ogc edr coords longitude %v out of range", lon)
	}
	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("ogc edr coords latitude %v out of range", lat)
	}
	return lon, lat, nil
}
