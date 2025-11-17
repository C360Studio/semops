package entities

import (
	"fmt"
	"time"

	"github.com/c360/semstreams/pkg/interfaces/store"
)

// Observation represents a sensor observation following W3C SOSA ontology.
//
// Maps to semstreams Entity with type="observation" and SOSA-compliant properties.
//
// SOSA Relationships (stored as semstreams Triples):
//   - observation --made-by--> sensor
//   - observation --observed-property--> property
//   - observation --has-feature-of-interest--> feature
//   - observation --has-result--> value (stored in Properties)
type Observation struct {
	// ID follows 6-part dotted notation: org.platform.domain.system.observation.instance
	// Example: c360.uav-alpha.ops.gcs1.observation.temp-reading-12345
	ID string

	// SensorID is the sensor that made this observation
	SensorID string

	// ObservedProperty is what was measured (e.g., "temperature", "altitude")
	ObservedProperty string

	// FeatureOfInterest is what was observed (entity ID or description)
	FeatureOfInterest string

	// Result contains the observation result
	Result ObservationResult

	// Timestamp when the observation was made
	Timestamp time.Time

	// Created and Updated timestamps (different from observation timestamp)
	Created time.Time
	Updated time.Time
}

// ObservationResult contains the observation value and metadata
type ObservationResult struct {
	// Value is the numeric or string observation value
	Value interface{} `json:"value"`

	// Units is the measurement unit (e.g., "celsius", "meters")
	Units string `json:"units,omitempty"`

	// Quality is a quality indicator (0.0-1.0, where 1.0 is highest quality)
	Quality float64 `json:"quality,omitempty"`

	// Custom result properties
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NewObservation creates a new observation entity
func NewObservation(id, sensorID, observedProperty string, result interface{}) *Observation {
	now := time.Now()
	return &Observation{
		ID:               id,
		SensorID:         sensorID,
		ObservedProperty: observedProperty,
		Result: ObservationResult{
			Value:   result,
			Quality: 1.0, // Assume good quality by default
			Custom:  make(map[string]interface{}),
		},
		Timestamp: now,
		Created:   now,
		Updated:   now,
	}
}

// ToEntity converts Observation to semstreams generic Entity
func (o *Observation) ToEntity() *store.Entity {
	props := map[string]interface{}{
		"sensor_id":           o.SensorID,
		"observed_property":   o.ObservedProperty,
		"feature_of_interest": o.FeatureOfInterest,
		"result_value":        o.Result.Value,
		"result_units":        o.Result.Units,
		"result_quality":      o.Result.Quality,
		"observation_time":    o.Timestamp.Format(time.RFC3339),
	}

	// Add custom result properties
	for k, v := range o.Result.Custom {
		props["result_"+k] = v
	}

	return &store.Entity{
		ID:         o.ID,
		Type:       "observation",
		Properties: props,
		Created:    o.Created,
		Updated:    o.Updated,
	}
}

// ObservationFromEntity converts semstreams Entity to Observation
func ObservationFromEntity(entity *store.Entity) (*Observation, error) {
	if entity.Type != "observation" {
		return nil, fmt.Errorf("entity type is %s, expected observation", entity.Type)
	}

	obs := &Observation{
		ID:      entity.ID,
		Created: entity.Created,
		Updated: entity.Updated,
		Result: ObservationResult{
			Custom: make(map[string]interface{}),
		},
	}

	// Extract standard properties
	if sensorID, ok := entity.Properties["sensor_id"].(string); ok {
		obs.SensorID = sensorID
	}
	if prop, ok := entity.Properties["observed_property"].(string); ok {
		obs.ObservedProperty = prop
	}
	if foi, ok := entity.Properties["feature_of_interest"].(string); ok {
		obs.FeatureOfInterest = foi
	}

	// Extract result value and metadata
	if value, ok := entity.Properties["result_value"]; ok {
		obs.Result.Value = value
	}
	if units, ok := entity.Properties["result_units"].(string); ok {
		obs.Result.Units = units
	}
	if quality, ok := entity.Properties["result_quality"].(float64); ok {
		obs.Result.Quality = quality
	}

	// Parse observation timestamp
	if obsTime, ok := entity.Properties["observation_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, obsTime); err == nil {
			obs.Timestamp = t
		}
	}

	// Extract custom result properties (those starting with "result_")
	for k, v := range entity.Properties {
		if len(k) > 7 && k[:7] == "result_" {
			customKey := k[7:]
			if customKey != "value" && customKey != "units" && customKey != "quality" {
				obs.Result.Custom[customKey] = v
			}
		}
	}

	return obs, nil
}
