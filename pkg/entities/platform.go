package entities

import (
	"fmt"
	"time"

	"github.com/c360/semstreams/pkg/interfaces/store"
)

// PlatformType represents different kinds of robotic platforms
type PlatformType string

const (
	PlatformTypeUAV      PlatformType = "uav"       // Unmanned Aerial Vehicle
	PlatformTypeUSV      PlatformType = "usv"       // Unmanned Surface Vehicle
	PlatformTypeUGV      PlatformType = "ugv"       // Unmanned Ground Vehicle
	PlatformTypeAUV      PlatformType = "auv"       // Autonomous Underwater Vehicle
	PlatformTypeStation  PlatformType = "station"   // Fixed ground station
)

// Platform represents a robotic platform (vehicle, station, etc.)
//
// Maps to semstreams Entity with type="platform" and domain-specific properties.
type Platform struct {
	// ID follows 6-part dotted notation: org.platform.domain.system.platform.instance
	// Example: c360.uav-alpha.ops.gcs1.platform.drone-1
	ID string

	// PlatformType classifies the platform (uav, usv, etc.)
	PlatformType PlatformType

	// Properties contains platform-specific metadata
	Properties PlatformProperties

	// Created and Updated timestamps
	Created time.Time
	Updated time.Time
}

// PlatformProperties contains platform-specific metadata
type PlatformProperties struct {
	// Make is the manufacturer (e.g., "DJI", "SeaRobotics")
	Make string `json:"make,omitempty"`

	// Model is the platform model (e.g., "Mavic 3", "ASV-C12")
	Model string `json:"model,omitempty"`

	// SerialNumber is the unique serial number
	SerialNumber string `json:"serial_number,omitempty"`

	// CallSign is the operational call sign/identifier
	CallSign string `json:"call_sign,omitempty"`

	// Status is the current operational status
	Status string `json:"status,omitempty"`

	// Custom properties (extensible for domain-specific needs)
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NewPlatform creates a new platform entity
func NewPlatform(id string, platformType PlatformType) *Platform {
	now := time.Now()
	return &Platform{
		ID:           id,
		PlatformType: platformType,
		Properties: PlatformProperties{
			Status: "unknown",
			Custom: make(map[string]interface{}),
		},
		Created: now,
		Updated: now,
	}
}

// ToEntity converts Platform to semstreams generic Entity
func (p *Platform) ToEntity() *store.Entity {
	props := map[string]interface{}{
		"platform_type": string(p.PlatformType),
		"make":          p.Properties.Make,
		"model":         p.Properties.Model,
		"serial_number": p.Properties.SerialNumber,
		"call_sign":     p.Properties.CallSign,
		"status":        p.Properties.Status,
	}

	// Add custom properties
	for k, v := range p.Properties.Custom {
		props[k] = v
	}

	return &store.Entity{
		ID:         p.ID,
		Type:       "platform",
		Properties: props,
		Created:    p.Created,
		Updated:    p.Updated,
	}
}

// PlatformFromEntity converts semstreams Entity to Platform
func PlatformFromEntity(entity *store.Entity) (*Platform, error) {
	if entity.Type != "platform" {
		return nil, fmt.Errorf("entity type is %s, expected platform", entity.Type)
	}

	platform := &Platform{
		ID:      entity.ID,
		Created: entity.Created,
		Updated: entity.Updated,
		Properties: PlatformProperties{
			Custom: make(map[string]interface{}),
		},
	}

	// Extract platform type
	if pt, ok := entity.Properties["platform_type"].(string); ok {
		platform.PlatformType = PlatformType(pt)
	}

	// Extract standard properties
	if make, ok := entity.Properties["make"].(string); ok {
		platform.Properties.Make = make
	}
	if model, ok := entity.Properties["model"].(string); ok {
		platform.Properties.Model = model
	}
	if sn, ok := entity.Properties["serial_number"].(string); ok {
		platform.Properties.SerialNumber = sn
	}
	if cs, ok := entity.Properties["call_sign"].(string); ok {
		platform.Properties.CallSign = cs
	}
	if status, ok := entity.Properties["status"].(string); ok {
		platform.Properties.Status = status
	}

	// Copy remaining properties to Custom
	standardProps := map[string]bool{
		"platform_type": true, "make": true, "model": true,
		"serial_number": true, "call_sign": true, "status": true,
	}
	for k, v := range entity.Properties {
		if !standardProps[k] {
			platform.Properties.Custom[k] = v
		}
	}

	return platform, nil
}
