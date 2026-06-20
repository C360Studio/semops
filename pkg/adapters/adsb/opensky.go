// Package adsb implements the first ADS-B feed boundary for SemOps.
//
// The initial boundary is intentionally OpenSky-shaped fixture parsing. It does
// not claim live feed reliability, ASTERIX support, or raw receiver protocol
// support.
package adsb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PositionSource int

const (
	PositionSourceADSB    PositionSource = 0
	PositionSourceASTERIX PositionSource = 1
	PositionSourceMLAT    PositionSource = 2
	PositionSourceFLARM   PositionSource = 3
)

type OpenSkySnapshot struct {
	Time   time.Time
	States []StateVector
}

type StateVector struct {
	ICAO24            string
	Callsign          *string
	OriginCountry     string
	TimePosition      *time.Time
	LastContact       time.Time
	Longitude         *float64
	Latitude          *float64
	BaroAltitudeM     *float64
	OnGround          bool
	VelocityMPS       *float64
	TrueTrackDeg      *float64
	VerticalRateMPS   *float64
	SensorIDs         []int
	GeoAltitudeM      *float64
	Squawk            *string
	SPI               bool
	PositionSource    PositionSource
	HasPositionSource bool
	Category          *int
}

func ParseOpenSkySnapshot(data []byte) (OpenSkySnapshot, error) {
	var raw struct {
		Time   int64             `json:"time"`
		States []json.RawMessage `json:"states"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return OpenSkySnapshot{}, err
	}
	if raw.Time <= 0 {
		return OpenSkySnapshot{}, errors.New("opensky snapshot time is required")
	}
	if raw.States == nil {
		return OpenSkySnapshot{}, errors.New("opensky states array is required")
	}
	out := OpenSkySnapshot{
		Time:   time.Unix(raw.Time, 0).UTC(),
		States: make([]StateVector, 0, len(raw.States)),
	}
	for i, row := range raw.States {
		state, err := parseStateVector(row)
		if err != nil {
			return OpenSkySnapshot{}, fmt.Errorf("opensky state %d: %w", i+1, err)
		}
		out.States = append(out.States, state)
	}
	return out, nil
}

func (s StateVector) HasPosition() bool {
	return s.Latitude != nil && s.Longitude != nil
}

func (s StateVector) PositionSourceLabel() string {
	if !s.HasPositionSource {
		return "unknown"
	}
	switch s.PositionSource {
	case PositionSourceADSB:
		return "ads-b"
	case PositionSourceASTERIX:
		return "asterix"
	case PositionSourceMLAT:
		return "mlat"
	case PositionSourceFLARM:
		return "flarm"
	default:
		return "unknown"
	}
}

func parseStateVector(raw json.RawMessage) (StateVector, error) {
	var row []json.RawMessage
	if err := json.Unmarshal(raw, &row); err != nil {
		return StateVector{}, err
	}
	if len(row) < 17 {
		return StateVector{}, fmt.Errorf("expected at least 17 fields, got %d", len(row))
	}

	icao24, err := requiredString(row[0], "icao24")
	if err != nil {
		return StateVector{}, err
	}
	originCountry, err := requiredString(row[2], "origin_country")
	if err != nil {
		return StateVector{}, err
	}
	lastContact, err := requiredUnixTime(row[4], "last_contact")
	if err != nil {
		return StateVector{}, err
	}
	onGround, err := requiredBool(row[8], "on_ground")
	if err != nil {
		return StateVector{}, err
	}
	spi, err := requiredBool(row[15], "spi")
	if err != nil {
		return StateVector{}, err
	}
	callsign, err := optionalString(row[1], "callsign")
	if err != nil {
		return StateVector{}, err
	}
	timePosition, err := optionalUnixTime(row[3], "time_position")
	if err != nil {
		return StateVector{}, err
	}
	longitude, err := optionalFloat64(row[5], "longitude")
	if err != nil {
		return StateVector{}, err
	}
	latitude, err := optionalFloat64(row[6], "latitude")
	if err != nil {
		return StateVector{}, err
	}
	baroAltitude, err := optionalFloat64(row[7], "baro_altitude")
	if err != nil {
		return StateVector{}, err
	}
	velocity, err := optionalFloat64(row[9], "velocity")
	if err != nil {
		return StateVector{}, err
	}
	trueTrack, err := optionalFloat64(row[10], "true_track")
	if err != nil {
		return StateVector{}, err
	}
	verticalRate, err := optionalFloat64(row[11], "vertical_rate")
	if err != nil {
		return StateVector{}, err
	}
	sensorIDs, err := optionalIntSlice(row[12], "sensors")
	if err != nil {
		return StateVector{}, err
	}
	geoAltitude, err := optionalFloat64(row[13], "geo_altitude")
	if err != nil {
		return StateVector{}, err
	}
	squawk, err := optionalString(row[14], "squawk")
	if err != nil {
		return StateVector{}, err
	}
	positionSource, err := optionalInt(row[16], "position_source")
	if err != nil {
		return StateVector{}, err
	}

	out := StateVector{
		ICAO24:          icao24,
		OriginCountry:   originCountry,
		LastContact:     lastContact,
		OnGround:        onGround,
		SPI:             spi,
		Callsign:        callsign,
		TimePosition:    timePosition,
		Longitude:       longitude,
		Latitude:        latitude,
		BaroAltitudeM:   baroAltitude,
		VelocityMPS:     velocity,
		TrueTrackDeg:    trueTrack,
		VerticalRateMPS: verticalRate,
		SensorIDs:       sensorIDs,
		GeoAltitudeM:    geoAltitude,
		Squawk:          squawk,
	}
	if positionSource != nil {
		out.PositionSource = PositionSource(*positionSource)
		out.HasPositionSource = true
	}
	if len(row) > 17 {
		category, err := optionalInt(row[17], "category")
		if err != nil {
			return StateVector{}, err
		}
		out.Category = category
	}
	return out, nil
}

func requiredString(raw json.RawMessage, field string) (string, error) {
	value, err := optionalString(raw, field)
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", fmt.Errorf("%s is required", field)
	}
	return *value, nil
}

func requiredUnixTime(raw json.RawMessage, field string) (time.Time, error) {
	var seconds int64
	if err := json.Unmarshal(raw, &seconds); err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", field, err)
	}
	if seconds <= 0 {
		return time.Time{}, fmt.Errorf("%s must be positive", field)
	}
	return time.Unix(seconds, 0).UTC(), nil
}

func requiredBool(raw json.RawMessage, field string) (bool, error) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("%s: %w", field, err)
	}
	return value, nil
}

func optionalString(raw json.RawMessage, field string) (*string, error) {
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	return &trimmed, nil
}

func optionalUnixTime(raw json.RawMessage, field string) (*time.Time, error) {
	var seconds *int64
	if err := json.Unmarshal(raw, &seconds); err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	if seconds == nil || *seconds <= 0 {
		return nil, nil
	}
	value := time.Unix(*seconds, 0).UTC()
	return &value, nil
}

func optionalFloat64(raw json.RawMessage, field string) (*float64, error) {
	var value *float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return value, nil
}

func optionalInt(raw json.RawMessage, field string) (*int, error) {
	var value *int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return value, nil
}

func optionalIntSlice(raw json.RawMessage, field string) ([]int, error) {
	var value []int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return value, nil
}
