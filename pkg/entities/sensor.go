// Package entities provides domain-specific entity types for robotics operations.
//
// These types map to semstreams generic Entity model while providing
// type-safe, domain-specific APIs for robotics applications.
package entities

import (
	"fmt"
	"time"

	"github.com/c360/semstreams/pkg/interfaces/store"
)

// SensorType represents different kinds of sensors
type SensorType string

const (
	SensorTypeTemperature SensorType = "temperature"
	SensorTypePressure    SensorType = "pressure"
	SensorTypeGPS         SensorType = "gps"
	SensorTypeCamera      SensorType = "camera"
	SensorTypeIMU         SensorType = "imu"
	SensorTypeLidar       SensorType = "lidar"
)

// Sensor represents a physical or virtual sensor in the robotics domain.
//
// Maps to semstreams Entity with type="sensor" and domain-specific properties.
// Complies with W3C SOSA/SSN Sensor ontology.
type Sensor struct {
	// ID follows 6-part dotted notation: org.platform.domain.system.sensor.instance
	// Example: c360.uav-alpha.ops.gcs1.sensor.temp-1
	ID string

	// SensorType classifies the sensor (temperature, gps, etc.)
	SensorType SensorType

	// Properties contains sensor-specific metadata
	Properties SensorProperties

	// Created and Updated timestamps
	Created time.Time
	Updated time.Time
}

// SensorProperties contains sensor-specific metadata
type SensorProperties struct {
	// Make is the manufacturer (e.g., "Bosch", "Garmin")
	Make string `json:"make,omitempty"`

	// Model is the sensor model (e.g., "BME280", "GPS-18x")
	Model string `json:"model,omitempty"`

	// Units is the measurement unit (e.g., "celsius", "meters/second")
	Units string `json:"units,omitempty"`

	// Accuracy is the sensor accuracy specification
	Accuracy string `json:"accuracy,omitempty"`

	// SamplingRate is the sampling frequency (e.g., "10Hz")
	SamplingRate string `json:"sampling_rate,omitempty"`

	// Custom properties (extensible for domain-specific needs)
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NewSensor creates a new sensor entity
func NewSensor(id string, sensorType SensorType) *Sensor {
	now := time.Now()
	return &Sensor{
		ID:         id,
		SensorType: sensorType,
		Properties: SensorProperties{
			Custom: make(map[string]interface{}),
		},
		Created: now,
		Updated: now,
	}
}

// ToEntity converts Sensor to semstreams generic Entity
func (s *Sensor) ToEntity() *store.Entity {
	props := map[string]interface{}{
		"sensor_type":   string(s.SensorType),
		"make":          s.Properties.Make,
		"model":         s.Properties.Model,
		"units":         s.Properties.Units,
		"accuracy":      s.Properties.Accuracy,
		"sampling_rate": s.Properties.SamplingRate,
	}

	// Add custom properties
	for k, v := range s.Properties.Custom {
		props[k] = v
	}

	return &store.Entity{
		ID:         s.ID,
		Type:       "sensor",
		Properties: props,
		Created:    s.Created,
		Updated:    s.Updated,
	}
}

// FromEntity converts semstreams Entity to Sensor
func FromEntity(entity *store.Entity) (*Sensor, error) {
	if entity.Type != "sensor" {
		return nil, fmt.Errorf("entity type is %s, expected sensor", entity.Type)
	}

	sensor := &Sensor{
		ID:      entity.ID,
		Created: entity.Created,
		Updated: entity.Updated,
		Properties: SensorProperties{
			Custom: make(map[string]interface{}),
		},
	}

	// Extract sensor type
	if st, ok := entity.Properties["sensor_type"].(string); ok {
		sensor.SensorType = SensorType(st)
	}

	// Extract standard properties
	if make, ok := entity.Properties["make"].(string); ok {
		sensor.Properties.Make = make
	}
	if model, ok := entity.Properties["model"].(string); ok {
		sensor.Properties.Model = model
	}
	if units, ok := entity.Properties["units"].(string); ok {
		sensor.Properties.Units = units
	}
	if accuracy, ok := entity.Properties["accuracy"].(string); ok {
		sensor.Properties.Accuracy = accuracy
	}
	if rate, ok := entity.Properties["sampling_rate"].(string); ok {
		sensor.Properties.SamplingRate = rate
	}

	// Copy remaining properties to Custom
	standardProps := map[string]bool{
		"sensor_type": true, "make": true, "model": true,
		"units": true, "accuracy": true, "sampling_rate": true,
	}
	for k, v := range entity.Properties {
		if !standardProps[k] {
			sensor.Properties.Custom[k] = v
		}
	}

	return sensor, nil
}
