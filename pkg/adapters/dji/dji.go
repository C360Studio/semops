// Package dji implements first-slice DJI-shaped fixture parsing for SemOps.
//
// The initial boundary is intentionally synthetic and parser-only. It proves
// the SemOps telemetry/media-reference contract shape without claiming DJI SDK,
// Cloud API, flight-log, or media compatibility.
package dji

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const SyntheticTelemetryFixtureClass = "semops.synthetic.dji.telemetry.v1"

type TelemetryRecord struct {
	FixtureClass     string
	Source           Source
	ObservedAt       time.Time
	Aircraft         AircraftState
	Battery          BatteryState
	Gimbal           GimbalState
	Camera           CameraState
	MediaRefs        []MediaRef
	CommandAuthority CommandAuthority
}

type Source struct {
	Provider          string
	SourceID          string
	AircraftModel     string
	ControllerID      string
	AircraftSerialRef string
}

type AircraftState struct {
	Latitude         float64
	Longitude        float64
	AltitudeMSLM     *float64
	AltitudeAGLM     *float64
	HeadingDeg       *float64
	GroundSpeedMPS   *float64
	VerticalSpeedMPS *float64
}

type BatteryState struct {
	Percent      *float64
	VoltageV     *float64
	TemperatureC *float64
}

type GimbalState struct {
	YawDeg   *float64
	PitchDeg *float64
	RollDeg  *float64
	Mode     string
}

type CameraState struct {
	Payload        string
	Mode           string
	Recording      bool
	ZoomRatio      *float64
	ThermalEnabled bool
}

type MediaRef struct {
	URI        string
	Kind       string
	Role       string
	StartedAt  *time.Time
	EndedAt    *time.Time
	SHA256     string
	ByteLength *int64
}

type CommandAuthority struct {
	Mode                  string
	Holder                string
	RemoteCommandsEnabled bool
	LocalOverrideRequired bool
	Notes                 string
}

func ParseTelemetryRecord(data []byte) (TelemetryRecord, error) {
	var raw telemetryJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return TelemetryRecord{}, err
	}
	record, err := raw.toRecord()
	if err != nil {
		return TelemetryRecord{}, err
	}
	if err := record.Validate(); err != nil {
		return TelemetryRecord{}, err
	}
	return record, nil
}

func (r TelemetryRecord) Validate() error {
	if strings.TrimSpace(r.FixtureClass) == "" {
		return errors.New("dji fixture_class is required")
	}
	if r.FixtureClass != SyntheticTelemetryFixtureClass {
		return fmt.Errorf("unsupported dji fixture_class %q", r.FixtureClass)
	}
	if strings.TrimSpace(r.Source.Provider) == "" {
		return errors.New("dji source provider is required")
	}
	if strings.TrimSpace(r.Source.SourceID) == "" {
		return errors.New("dji source_id is required")
	}
	if r.ObservedAt.IsZero() {
		return errors.New("dji observed_at is required")
	}
	if r.Aircraft.Latitude < -90 || r.Aircraft.Latitude > 90 {
		return fmt.Errorf("dji aircraft latitude %v out of range", r.Aircraft.Latitude)
	}
	if r.Aircraft.Longitude < -180 || r.Aircraft.Longitude > 180 {
		return fmt.Errorf("dji aircraft longitude %v out of range", r.Aircraft.Longitude)
	}
	if r.Battery.Percent != nil && (*r.Battery.Percent < 0 || *r.Battery.Percent > 100) {
		return fmt.Errorf("dji battery percent %v out of range", *r.Battery.Percent)
	}
	if r.CommandAuthority.Mode == "" {
		return errors.New("dji command authority mode is required")
	}
	for i, mediaRef := range r.MediaRefs {
		if err := mediaRef.Validate(); err != nil {
			return fmt.Errorf("dji media_ref %d: %w", i+1, err)
		}
	}
	return nil
}

func (m MediaRef) Validate() error {
	if strings.TrimSpace(m.URI) == "" {
		return errors.New("uri is required")
	}
	if parsed, err := url.Parse(m.URI); err != nil || parsed.Scheme == "" {
		return fmt.Errorf("uri must be absolute: %s", m.URI)
	}
	if strings.TrimSpace(m.Kind) == "" {
		return errors.New("kind is required")
	}
	if m.ByteLength != nil && *m.ByteLength < 0 {
		return fmt.Errorf("byte_length must not be negative: %d", *m.ByteLength)
	}
	return nil
}

type telemetryJSON struct {
	FixtureClass     string               `json:"fixture_class"`
	Source           sourceJSON           `json:"source"`
	ObservedAt       string               `json:"observed_at"`
	Aircraft         aircraftJSON         `json:"aircraft"`
	Battery          batteryJSON          `json:"battery"`
	Gimbal           gimbalJSON           `json:"gimbal"`
	Camera           cameraJSON           `json:"camera"`
	MediaRefs        []mediaRefJSON       `json:"media_refs"`
	CommandAuthority commandAuthorityJSON `json:"command_authority"`
}

type sourceJSON struct {
	Provider          string `json:"provider"`
	SourceID          string `json:"source_id"`
	AircraftModel     string `json:"aircraft_model"`
	ControllerID      string `json:"controller_id"`
	AircraftSerialRef string `json:"aircraft_serial_ref"`
}

type aircraftJSON struct {
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	AltitudeMSLM     *float64 `json:"altitude_msl_m"`
	AltitudeAGLM     *float64 `json:"altitude_agl_m"`
	HeadingDeg       *float64 `json:"heading_deg"`
	GroundSpeedMPS   *float64 `json:"ground_speed_mps"`
	VerticalSpeedMPS *float64 `json:"vertical_speed_mps"`
}

type batteryJSON struct {
	Percent      *float64 `json:"percent"`
	VoltageV     *float64 `json:"voltage_v"`
	TemperatureC *float64 `json:"temperature_c"`
}

type gimbalJSON struct {
	YawDeg   *float64 `json:"yaw_deg"`
	PitchDeg *float64 `json:"pitch_deg"`
	RollDeg  *float64 `json:"roll_deg"`
	Mode     string   `json:"mode"`
}

type cameraJSON struct {
	Payload        string   `json:"payload"`
	Mode           string   `json:"mode"`
	Recording      bool     `json:"recording"`
	ZoomRatio      *float64 `json:"zoom_ratio"`
	ThermalEnabled bool     `json:"thermal_enabled"`
}

type mediaRefJSON struct {
	URI        string `json:"uri"`
	Kind       string `json:"kind"`
	Role       string `json:"role"`
	StartedAt  string `json:"started_at"`
	EndedAt    string `json:"ended_at"`
	SHA256     string `json:"sha256"`
	ByteLength *int64 `json:"byte_length"`
}

type commandAuthorityJSON struct {
	Mode                  string `json:"mode"`
	Holder                string `json:"holder"`
	RemoteCommandsEnabled bool   `json:"remote_commands_enabled"`
	LocalOverrideRequired bool   `json:"local_override_required"`
	Notes                 string `json:"notes"`
}

func (x telemetryJSON) toRecord() (TelemetryRecord, error) {
	observedAt, err := requiredTime(x.ObservedAt, "observed_at")
	if err != nil {
		return TelemetryRecord{}, err
	}
	mediaRefs := make([]MediaRef, 0, len(x.MediaRefs))
	for i, raw := range x.MediaRefs {
		mediaRef, err := raw.toMediaRef()
		if err != nil {
			return TelemetryRecord{}, fmt.Errorf("media_refs %d: %w", i+1, err)
		}
		mediaRefs = append(mediaRefs, mediaRef)
	}
	return TelemetryRecord{
		FixtureClass: strings.TrimSpace(x.FixtureClass),
		Source: Source{
			Provider:          strings.TrimSpace(x.Source.Provider),
			SourceID:          strings.TrimSpace(x.Source.SourceID),
			AircraftModel:     strings.TrimSpace(x.Source.AircraftModel),
			ControllerID:      strings.TrimSpace(x.Source.ControllerID),
			AircraftSerialRef: strings.TrimSpace(x.Source.AircraftSerialRef),
		},
		ObservedAt: observedAt,
		Aircraft: AircraftState{
			Latitude:         x.Aircraft.Latitude,
			Longitude:        x.Aircraft.Longitude,
			AltitudeMSLM:     x.Aircraft.AltitudeMSLM,
			AltitudeAGLM:     x.Aircraft.AltitudeAGLM,
			HeadingDeg:       x.Aircraft.HeadingDeg,
			GroundSpeedMPS:   x.Aircraft.GroundSpeedMPS,
			VerticalSpeedMPS: x.Aircraft.VerticalSpeedMPS,
		},
		Battery: BatteryState{
			Percent:      x.Battery.Percent,
			VoltageV:     x.Battery.VoltageV,
			TemperatureC: x.Battery.TemperatureC,
		},
		Gimbal: GimbalState{
			YawDeg:   x.Gimbal.YawDeg,
			PitchDeg: x.Gimbal.PitchDeg,
			RollDeg:  x.Gimbal.RollDeg,
			Mode:     strings.TrimSpace(x.Gimbal.Mode),
		},
		Camera: CameraState{
			Payload:        strings.TrimSpace(x.Camera.Payload),
			Mode:           strings.TrimSpace(x.Camera.Mode),
			Recording:      x.Camera.Recording,
			ZoomRatio:      x.Camera.ZoomRatio,
			ThermalEnabled: x.Camera.ThermalEnabled,
		},
		MediaRefs: mediaRefs,
		CommandAuthority: CommandAuthority{
			Mode:                  strings.TrimSpace(x.CommandAuthority.Mode),
			Holder:                strings.TrimSpace(x.CommandAuthority.Holder),
			RemoteCommandsEnabled: x.CommandAuthority.RemoteCommandsEnabled,
			LocalOverrideRequired: x.CommandAuthority.LocalOverrideRequired,
			Notes:                 strings.TrimSpace(x.CommandAuthority.Notes),
		},
	}, nil
}

func (x mediaRefJSON) toMediaRef() (MediaRef, error) {
	startedAt, err := optionalTime(x.StartedAt, "started_at")
	if err != nil {
		return MediaRef{}, err
	}
	endedAt, err := optionalTime(x.EndedAt, "ended_at")
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{
		URI:        strings.TrimSpace(x.URI),
		Kind:       strings.TrimSpace(x.Kind),
		Role:       strings.TrimSpace(x.Role),
		StartedAt:  startedAt,
		EndedAt:    endedAt,
		SHA256:     strings.TrimSpace(x.SHA256),
		ByteLength: x.ByteLength,
	}, nil
}

func requiredTime(value string, field string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", field, err)
	}
	return parsed.UTC(), nil
}

func optionalTime(value string, field string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}
