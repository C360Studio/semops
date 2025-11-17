package payloads

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/c360/streamkit/component"
	message "github.com/c360/semstreams/message"
	"github.com/c360/semops/pkg/processors/mavlink/constants"
	vocab "github.com/c360/semops/pkg/processors/mavlink/vocabulary"
	"github.com/c360/semstreams/vocabulary"
)

func init() {
	component.RegisterPayload(&component.PayloadRegistration{
		Factory: func() interface{} {
			return &HeartbeatPayload{}
		},
		Domain:      "robotics",
		Category:    "heartbeat",
		Version:     "v1",
		Description: "MAVLink heartbeat message for system status",
		Example: map[string]interface{}{
			"system_id":       1,
			"component_id":    1,
			"vehicle_type":    2,
			"system_status":   4,
			"base_mode":       81,
			"custom_mode":     0,
			"mavlink_version": 3,
			"timestamp":       "2024-01-15T10:30:00Z",
		},
	})
}

// HeartbeatPayload represents system status and heartbeat data from robotics systems.
// This payload implements both the Payload and Message interfaces, eliminating
// the need for a wrapper layer. It provides structured access to heartbeat data
// for routing, entity extraction, and correlation analysis.
type HeartbeatPayload struct {

	// MAVLink heartbeat data
	SystemID       uint8     `json:"system_id"`       // ID of this system (aircraft)
	ComponentID    uint8     `json:"component_id"`    // ID of this component (autopilot, camera, etc.)
	VehicleType    uint8     `json:"vehicle_type"`    // Vehicle type (MAV_TYPE)
	Autopilot      uint8     `json:"autopilot"`       // Autopilot type (MAV_AUTOPILOT)
	BaseMode       uint8     `json:"base_mode"`       // System mode flags (MAV_MODE_FLAG)
	CustomMode     uint32    `json:"custom_mode"`     // Custom mode, depends on autopilot
	SystemStatus   uint8     `json:"system_status"`   // System status flag (MAV_STATE)
	MavlinkVersion uint8     `json:"mavlink_version"` // MAVLink version (1 or 2)
	Ts             time.Time `json:"timestamp"`       // Message timestamp

	// Additional metadata for SemStreams
	SourceIP    string         `json:"source_ip,omitempty"`    // Source IP if received via network
	LinkQuality float64        `json:"link_quality,omitempty"` // Link quality 0-100%
	Properties  map[string]any `json:"properties,omitempty"`   // Additional properties
}

// NewHeartbeatPayload creates a new heartbeat payload with default values.
func NewHeartbeatPayload(systemID, componentID uint8, timestamp time.Time) *HeartbeatPayload {
	return &HeartbeatPayload{
		// MAVLink data
		SystemID:       systemID,
		ComponentID:    componentID,
		VehicleType:    constants.MavTypeGeneric,
		Autopilot:      constants.MavAutopilotGeneric,
		BaseMode:       0,
		CustomMode:     0,
		SystemStatus:   constants.MavStateStandby,
		MavlinkVersion: constants.MavlinkV2,
		Ts:             timestamp,
		Properties:     make(map[string]any),
	}
}

// Implement Payload interface

// Schema returns the MessageType that this payload conforms to.
func (h *HeartbeatPayload) Schema() message.Type {
	return message.Type{
		Domain:   "robotics",
		Category: "heartbeat",
		Version:  "v1",
	}
}

// Validate performs validation of the heartbeat payload data.
func (h *HeartbeatPayload) Validate() error {
	// Validate heartbeat payload data
	if h.SystemID == 0 {
		return errors.New("system ID cannot be zero")
	}

	if h.Ts.IsZero() {
		return errors.New("timestamp is required")
	}

	// Check if timestamp is reasonable (not too far in future/past)
	now := time.Now()
	if h.Ts.After(now.Add(1 * time.Minute)) {
		return fmt.Errorf("timestamp is too far in the future")
	}

	if h.Ts.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("timestamp is too far in the past")
	}

	// Validate MAVLink version
	if h.MavlinkVersion != constants.MavlinkV1 && h.MavlinkVersion != constants.MavlinkV2 {
		return fmt.Errorf("invalid MAVLink version: %d (must be 1 or 2)", h.MavlinkVersion)
	}

	// Validate link quality if present
	if h.LinkQuality < 0 || h.LinkQuality > 100 {
		return fmt.Errorf("link quality must be 0-100%%, got: %f", h.LinkQuality)
	}

	return nil
}

// MarshalJSON serializes the payload to JSON format.
func (h *HeartbeatPayload) MarshalJSON() ([]byte, error) {
	// Use alias to avoid infinite recursion
	type Alias HeartbeatPayload
	return json.Marshal((*Alias)(h))
}

// UnmarshalJSON deserializes JSON data back into the payload.
func (h *HeartbeatPayload) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type Alias HeartbeatPayload
	return json.Unmarshal(data, (*Alias)(h))
}

// Implement behavioral interfaces

// Identifiable - provides entity identity for graph storage
func (h *HeartbeatPayload) EntityID() string {
	// System field comes from MAVLink SystemID at runtime
	system := h.mapSystemIDToSystem()
	
	entityID := message.EntityID{
		Org:      "c360",        // TODO: Get from config when available
		Platform: "platform1",   // TODO: Get from config when available
		Domain:   "robotics",    // Domain-first hierarchy
		System:   system,        // RUNTIME value from message
		Type:     "drone",
		Instance: "0",  // Single drone per system, no SystemID duplication
	}
	return entityID.Key()
}

// mapSystemIDToSystem converts MAVLink SystemID to meaningful system name
func (h *HeartbeatPayload) mapSystemIDToSystem() string {
	switch h.SystemID {
	case 255:
		return "gcs-main"
	case 254:
		return "gcs-backup"
	default:
		return fmt.Sprintf("mav%d", h.SystemID)
	}
}


func (h *HeartbeatPayload) EntityType() message.EntityType {
	return vocab.EntityTypeDrone
}

// Timestampable - provides temporal context
func (h *HeartbeatPayload) Timestamp() time.Time {
	return h.Ts
}

// Observable - heartbeat observes system status
func (h *HeartbeatPayload) ObservedEntity() string {
	return h.EntityID()
}

func (h *HeartbeatPayload) ObservedProperty() string {
	return "system_status"
}

func (h *HeartbeatPayload) ObservedValue() any {
	return map[string]any{
		"vehicle_type":    constants.GetVehicleTypeName(h.VehicleType),
		"autopilot":       constants.GetAutopilotName(h.Autopilot),
		"system_status":   constants.GetSystemStatusName(h.SystemStatus),
		"base_mode":       h.BaseMode,
		"custom_mode":     h.CustomMode,
		"is_armed":        h.IsArmed(),
		"is_auto_mode":    h.IsInAutoMode(),
		"is_guided_mode":  h.IsInGuidedMode(),
		"is_healthy":      h.IsHealthy(),
		"is_critical":     h.IsCritical(),
		"mavlink_version": h.MavlinkVersion,
		"link_quality":    h.LinkQuality,
	}
}

// Deployable - if vehicle has deployment information
func (h *HeartbeatPayload) DeploymentID() string {
	if deploymentID, ok := h.Properties["deployment_id"].(string); ok {
		return deploymentID
	}
	return ""
}

func (h *HeartbeatPayload) DeploymentName() string {
	if deploymentName, ok := h.Properties["deployment_name"].(string); ok {
		return deploymentName
	}
	return ""
}

// MessageCorrelatable - provides correlation capabilities
func (h *HeartbeatPayload) CorrelationID() string {
	return fmt.Sprintf("heartbeat_%d", h.SystemID)
}

func (h *HeartbeatPayload) RelatedMessages() []string {
	// Heartbeat messages can be correlated with all other messages from the same vehicle
	return []string{
		fmt.Sprintf("robotics.position.%d", h.SystemID),
		fmt.Sprintf("robotics.attitude.%d", h.SystemID),
		fmt.Sprintf("robotics.battery.%d", h.SystemID),
		fmt.Sprintf("robotics.mission.%d", h.SystemID),
	}
}




// Heartbeat-specific helper methods

// GetSystemID returns the system ID for routing and identification
func (h *HeartbeatPayload) GetSystemID() uint8 {
	return h.SystemID
}

// GetVehicleTypeName returns the human-readable vehicle type name
func (h *HeartbeatPayload) GetVehicleTypeName() string {
	return constants.GetVehicleTypeName(h.VehicleType)
}

// GetAutopilotName returns the human-readable autopilot name
func (h *HeartbeatPayload) GetAutopilotName() string {
	return constants.GetAutopilotName(h.Autopilot)
}

// GetSystemStatusName returns the human-readable system status
func (h *HeartbeatPayload) GetSystemStatusName() string {
	return constants.GetSystemStatusName(h.SystemStatus)
}

// IsArmed returns true if the vehicle is armed
func (h *HeartbeatPayload) IsArmed() bool {
	return (h.BaseMode & constants.MavModeFlagSafetyArmed) != 0
}

// IsInAutoMode returns true if the vehicle is in autonomous mode
func (h *HeartbeatPayload) IsInAutoMode() bool {
	return (h.BaseMode & constants.MavModeFlagAutoEnabled) != 0
}

// IsInGuidedMode returns true if the vehicle is in guided mode
func (h *HeartbeatPayload) IsInGuidedMode() bool {
	return (h.BaseMode & constants.MavModeFlagGuidedEnabled) != 0
}

// IsStabilizeEnabled returns true if stabilize mode is enabled
func (h *HeartbeatPayload) IsStabilizeEnabled() bool {
	return (h.BaseMode & constants.MavModeFlagStabilizeEnabled) != 0
}

// IsManualInputEnabled returns true if manual control is enabled
func (h *HeartbeatPayload) IsManualInputEnabled() bool {
	return (h.BaseMode & constants.MavModeFlagManualInputEnabled) != 0
}

// IsInTestMode returns true if test mode is enabled
func (h *HeartbeatPayload) IsInTestMode() bool {
	return (h.BaseMode & constants.MavModeFlagTestEnabled) != 0
}

// IsHealthy returns true if the system status indicates healthy operation
func (h *HeartbeatPayload) IsHealthy() bool {
	return h.SystemStatus == constants.MavStateStandby ||
		h.SystemStatus == constants.MavStateActive
}

// IsCritical returns true if the system is in a critical state
func (h *HeartbeatPayload) IsCritical() bool {
	return h.SystemStatus == constants.MavStateCritical ||
		h.SystemStatus == constants.MavStateEmergency
}

// GetAge returns how long ago this heartbeat was generated
func (h *HeartbeatPayload) GetAge() time.Duration {
	return time.Since(h.Ts)
}

// GetControlModeName returns a human-readable description of the current control mode
func (h *HeartbeatPayload) GetControlModeName() string {
	modes := []string{}
	
	if h.IsArmed() {
		modes = append(modes, "armed")
	} else {
		modes = append(modes, "disarmed")
	}
	
	if h.IsInAutoMode() {
		modes = append(modes, "autonomous")
	}
	
	if h.IsInGuidedMode() {
		modes = append(modes, "guided")
	}
	
	if h.IsManualInputEnabled() {
		modes = append(modes, "manual")
	}
	
	if h.IsStabilizeEnabled() {
		modes = append(modes, "stabilized")
	}
	
	if len(modes) == 0 {
		return "unknown"
	}
	
	// Join modes with underscore for consistent naming
	result := ""
	for i, mode := range modes {
		if i > 0 {
			result += "_"
		}
		result += mode
	}
	
	return result
}

// SetProperty sets an additional property on the payload
func (h *HeartbeatPayload) SetProperty(key string, value any) {
	if h.Properties == nil {
		h.Properties = make(map[string]any)
	}
	h.Properties[key] = value
}

// GetProperty gets an additional property from the payload
func (h *HeartbeatPayload) GetProperty(key string) (any, bool) {
	if h.Properties == nil {
		return nil, false
	}
	value, exists := h.Properties[key]
	return value, exists
}

// Alpha Week 3 Implementation: TripleGenerator interface

// Triples generates semantic triples from the heartbeat payload using three-level dotted predicates.
// This replaces the Properties map with structured semantic assertions enabling graph queries.
//
// Generated triples include:
//   - System status and identification (robotics.system.*)
//   - Flight mode and armed status (robotics.flight.*)
//   - Component relationships (robotics.component.*)
//   - Network quality metrics (network.connection.*)
//   - Temporal lifecycle events (time.lifecycle.*)
func (h *HeartbeatPayload) Triples() []message.Triple {
	entityID := h.EntityID() // Uses structured EntityID.Key() format
	timestamp := h.Ts
	
	var triples []message.Triple
	
	// System identification triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_ID,
			Object:     fmt.Sprintf("%d", h.SystemID),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct telemetry
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_TYPE,
			Object:     h.GetVehicleTypeName(),
			Source:     "mavlink_heartbeat", 
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_STATUS,
			Object:     h.GetSystemStatusName(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Flight state triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_FLIGHT_ARMED,
			Object:     h.IsArmed(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_FLIGHT_MODE,
			Object:     h.GetControlModeName(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_FLIGHT_GUIDED,
			Object:     h.IsInGuidedMode(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_FLIGHT_AUTO,
			Object:     h.IsInAutoMode(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_FLIGHT_STABILIZE,
			Object:     h.IsStabilizeEnabled(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// System version and uptime if available
	if age := h.GetAge(); age > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_UPTIME,
			Object:     age.Seconds(),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 0.9, // Calculated value
		})
	}
	
	// Network connection quality if available
	if h.LinkQuality > 0 {
		triples = append(triples, []message.Triple{
			{
				Subject:    entityID,
				Predicate:  vocabulary.NETWORK_CONNECTION_STRENGTH,
				Object:     h.LinkQuality,
				Source:     "mavlink_heartbeat",
				Timestamp:  timestamp,
				Confidence: 0.8, // Measured value, some uncertainty
			},
			{
				Subject:    entityID,
				Predicate:  vocabulary.NETWORK_CONNECTION_STATUS,
				Object:     "active",
				Source:     "mavlink_heartbeat",
				Timestamp:  timestamp,
				Confidence: 0.9,
			},
		}...)
	}
	
	// Protocol information
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.NETWORK_PROTOCOL_TYPE,
			Object:     "mavlink",
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.NETWORK_PROTOCOL_VERSION,
			Object:     fmt.Sprintf("v%d", h.MavlinkVersion),
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Lifecycle events
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_SEEN,
			Object:     timestamp,
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_UPDATED,
			Object:     timestamp,
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Component relationship triples
	// Battery component relationship
	system := h.mapSystemIDToSystem()
	batteryEntityID := message.EntityID{
		Org:      "c360",
		Platform: "platform1",
		System:   system,
		Domain:   "robotics", 
		Type:     "battery",
		Instance: "0",  // Battery 0 for this system, no SystemID duplication
	}.Key()
	
	triples = append(triples, message.Triple{
		Subject:    entityID,
		Predicate:  vocabulary.ROBOTICS_COMPONENT_HAS,
		Object:     batteryEntityID, // Entity reference
		Source:     "mavlink_heartbeat",
		Timestamp:  timestamp,
		Confidence: 0.9, // Inferred relationship
	})
	
	// Autopilot component relationship
	autopilotEntityID := message.EntityID{
		Org:      "c360",
		Platform: "platform1",
		System:   system,
		Domain:   "robotics",
		Type:     "autopilot", 
		Instance: "0",  // Autopilot 0 for this system, no SystemID duplication
	}.Key()
	
	triples = append(triples, message.Triple{
		Subject:    entityID,
		Predicate:  vocabulary.ROBOTICS_COMPONENT_HAS,
		Object:     autopilotEntityID, // Entity reference
		Source:     "mavlink_heartbeat",
		Timestamp:  timestamp,
		Confidence: 0.9, // Inferred relationship
	})
	
	// Quality metadata for the heartbeat data itself
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SCORE,
			Object:     1.0, // High confidence in heartbeat data
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SOURCE,
			Object:     "direct_telemetry",
			Source:     "mavlink_heartbeat",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	return triples
}
