package payloads

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
			return &MissionPayload{}
		},
		Domain:      "robotics",
		Category:    "mission",
		Version:     "v1",
		Description: "MAVLink mission planning and execution data",
		Example: map[string]interface{}{
			"system_id":          1,
			"component_id":       1,
			"mission_state":      "active",
			"current_item":       3,
			"total_items":        10,
			"mission_name":       "Patrol Route Alpha",
			"mission_type":       0,
			"distance_total":     1500.0,
			"distance_remaining": 900.0,
			"time_elapsed":       120.0,
			"timestamp":          "2024-01-15T10:30:00Z",
		},
	})
}

// MissionItem represents a single waypoint or command in a mission
type MissionItem struct {
	Seq          uint16  `json:"seq"`                    // Waypoint sequence number
	Frame        uint8   `json:"frame"`                  // Coordinate frame (MAV_FRAME)
	Command      uint16  `json:"command"`                // Command ID (MAV_CMD)
	Current      uint8   `json:"current"`                // 1 if current waypoint
	Autocontinue uint8   `json:"autocontinue"`           // Autocontinue to next waypoint
	Param1       float32 `json:"param1"`                 // Parameter 1
	Param2       float32 `json:"param2"`                 // Parameter 2
	Param3       float32 `json:"param3"`                 // Parameter 3
	Param4       float32 `json:"param4"`                 // Parameter 4
	X            float32 `json:"x"`                      // X coordinate (latitude or local X)
	Y            float32 `json:"y"`                      // Y coordinate (longitude or local Y)
	Z            float32 `json:"z"`                      // Z coordinate (altitude or local Z)
	MissionType  uint8   `json:"mission_type,omitempty"` // Mission type (0=mission, 1=fence, 2=rally)
}

// MissionPayload represents mission planning and execution data from robotics systems.
// This payload implements behavioral interfaces to provide structured access
// to mission data for routing, entity extraction, and correlation analysis.
type MissionPayload struct {
	SystemID    uint8     `json:"system_id"`    // ID of this system
	ComponentID uint8     `json:"component_id"` // ID of this component
	Ts          time.Time `json:"timestamp"`    // Message timestamp

	// Mission Status
	MissionState string `json:"mission_state"` // "planning", "uploading", "active", "paused", "completed", "failed"
	CurrentItem  uint16 `json:"current_item"`  // Current mission item sequence
	TotalItems   uint16 `json:"total_items"`   // Total number of mission items

	// Mission Items
	Items []MissionItem `json:"items,omitempty"` // Mission waypoints and commands

	// Mission Metadata
	MissionName       string    `json:"mission_name,omitempty"`       // Human-readable mission name
	MissionType       uint8     `json:"mission_type,omitempty"`       // 0=mission, 1=fence, 2=rally
	UploadTime        time.Time `json:"upload_time,omitempty"`        // When mission was uploaded
	StartTime         time.Time `json:"start_time,omitempty"`         // Mission start time
	EstimatedDuration float64   `json:"estimated_duration,omitempty"` // Estimated duration in seconds

	// Progress Information
	DistanceTotal     float64 `json:"distance_total,omitempty"`     // Total mission distance (m)
	DistanceRemaining float64 `json:"distance_remaining,omitempty"` // Remaining distance (m)
	TimeElapsed       float64 `json:"time_elapsed,omitempty"`       // Mission elapsed time (s)
	TimeRemaining     float64 `json:"time_remaining,omitempty"`     // Estimated time remaining (s)

	Properties map[string]any `json:"properties,omitempty"` // Additional properties
}

// NewMissionPayload creates a new mission payload with default values.
func NewMissionPayload(systemID, componentID uint8, timestamp time.Time) *MissionPayload {
	return &MissionPayload{
		SystemID:     systemID,
		ComponentID:  componentID,
		Ts:           timestamp,
		MissionState: "planning",
		CurrentItem:  0,
		TotalItems:   0,
		Items:        make([]MissionItem, 0),
		Properties:   make(map[string]any),
	}
}

// Implement Payload interface

// Schema returns the MessageType that this payload conforms to.
func (m *MissionPayload) Schema() message.Type {
	return message.Type{
		Domain:   "robotics",
		Category: "mission",
		Version:  "v1",
	}
}

// Validate performs validation of the mission payload data.
func (m *MissionPayload) Validate() error {
	if m.SystemID == 0 {
		return errors.New("system ID cannot be zero")
	}

	if m.Ts.IsZero() {
		return errors.New("timestamp is required")
	}

	// Validate mission state
	validStates := []string{"planning", "uploading", "active", "paused", "completed", "failed"}
	validState := false
	for _, state := range validStates {
		if m.MissionState == state {
			validState = true
			break
		}
	}
	if !validState {
		return fmt.Errorf("invalid mission state: %s", m.MissionState)
	}

	// Validate current item is within bounds
	if m.TotalItems > 0 && m.CurrentItem >= m.TotalItems {
		return fmt.Errorf("current item %d exceeds total items %d", m.CurrentItem, m.TotalItems)
	}

	// Validate waypoint sequence numbers are unique and sequential
	seqMap := make(map[uint16]bool)
	for i, item := range m.Items {
		if seqMap[item.Seq] {
			return fmt.Errorf("duplicate waypoint sequence number: %d", item.Seq)
		}
		seqMap[item.Seq] = true

		// Validate coordinate ranges for geographic coordinates
		if m.IsGeographicFrame(item.Frame) {
			lat := m.GetWaypointLatitude(item)
			lon := m.GetWaypointLongitude(item)

			if lat < -90.0 || lat > 90.0 {
				return fmt.Errorf("waypoint %d latitude out of range: %f", i, lat)
			}
			if lon < -180.0 || lon > 180.0 {
				return fmt.Errorf("waypoint %d longitude out of range: %f", i, lon)
			}
		}
	}

	// Validate distances and times are non-negative
	if m.DistanceTotal < 0 {
		return fmt.Errorf("total distance cannot be negative: %f", m.DistanceTotal)
	}
	if m.DistanceRemaining < 0 {
		return fmt.Errorf("remaining distance cannot be negative: %f", m.DistanceRemaining)
	}
	if m.TimeElapsed < 0 {
		return fmt.Errorf("elapsed time cannot be negative: %f", m.TimeElapsed)
	}

	return nil
}

// MarshalJSON serializes the payload to JSON format.
func (m *MissionPayload) MarshalJSON() ([]byte, error) {
	// Use alias to avoid infinite recursion
	type Alias MissionPayload
	return json.Marshal((*Alias)(m))
}

// UnmarshalJSON deserializes JSON data back into the payload.
func (m *MissionPayload) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type Alias MissionPayload
	return json.Unmarshal(data, (*Alias)(m))
}

// Implement behavioral interfaces

// Identifiable - provides entity identity for graph storage
func (m *MissionPayload) EntityID() string {
	// System field comes from MAVLink SystemID at runtime
	system := m.mapSystemIDToSystem()
	
	entityID := message.EntityID{
		Org:      "c360",        // TODO: Get from config when available
		Platform: "platform1",   // TODO: Get from config when available
		Domain:   "robotics",    // Domain-first hierarchy
		System:   system,        // RUNTIME value from message
		Type:     "mission",
		Instance: fmt.Sprintf("%d", m.ComponentID),  // Just ComponentID, no SystemID duplication
	}
	return entityID.Key()
}

// mapSystemIDToSystem converts MAVLink SystemID to meaningful system name
func (m *MissionPayload) mapSystemIDToSystem() string {
	switch m.SystemID {
	case 255:
		return "gcs-main"
	case 254:
		return "gcs-backup"
	default:
		return fmt.Sprintf("mav%d", m.SystemID)
	}
}

func (m *MissionPayload) EntityType() message.EntityType {
	return vocab.EntityTypeMission
}

// DroneEntityID returns the federated entity ID for the drone executing this mission
func (m *MissionPayload) DroneEntityID() string {
	// Use structured EntityID format for drone
	system := m.mapSystemIDToSystem()
	entityID := message.EntityID{
		Org:      "c360",
		Platform: "platform1",
		Domain:   "robotics",    // Domain-first hierarchy
		System:   system,
		Type:     "drone",
		Instance: "0",  // Single drone per system, no SystemID duplication
	}
	return entityID.Key()
}

// Timestampable - provides temporal context
func (m *MissionPayload) Timestamp() time.Time {
	return m.Ts
}

// Measurable - provides mission progress as measured value
func (m *MissionPayload) MeasuredValue() float64 {
	return m.GetProgressPercentage()
}

func (m *MissionPayload) MeasuredUnit() string {
	return "percent"
}

// Observable - mission system observes execution progress
func (m *MissionPayload) ObservedEntity() string {
	return m.DroneEntityID()
}

func (m *MissionPayload) ObservedProperty() string {
	return "mission_progress"
}

func (m *MissionPayload) ObservedValue() any {
	return map[string]any{
		"mission_name":       m.MissionName,
		"mission_state":      m.MissionState,
		"mission_type":       m.GetMissionTypeName(),
		"current_waypoint":   m.CurrentItem,
		"total_waypoints":    m.TotalItems,
		"progress_percent":   m.GetProgressPercentage(),
		"total_distance":     m.DistanceTotal,
		"remaining_distance": m.DistanceRemaining,
		"elapsed_time":       m.TimeElapsed,
		"remaining_time":     m.TimeRemaining,
		"is_active":          m.IsActive(),
		"is_completed":       m.IsCompleted(),
		"is_behind_schedule": m.IsBehindSchedule(),
	}
}

// Deployable - if mission has deployment information
func (m *MissionPayload) DeploymentID() string {
	if deploymentID, ok := m.Properties["deployment_id"].(string); ok {
		return deploymentID
	}
	return ""
}

func (m *MissionPayload) DeploymentName() string {
	if deploymentName, ok := m.Properties["deployment_name"].(string); ok {
		return deploymentName
	}
	return ""
}

// MessageCorrelatable - provides correlation capabilities
func (m *MissionPayload) CorrelationID() string {
	return fmt.Sprintf("vehicle_%d_mission", m.SystemID)
}

func (m *MissionPayload) RelatedMessages() []string {
	// Mission messages can be correlated with position, attitude, and battery messages from the same vehicle
	return []string{
		fmt.Sprintf("robotics.position.%d", m.SystemID),
		fmt.Sprintf("robotics.attitude.%d", m.SystemID),
		fmt.Sprintf("robotics.battery.%d", m.SystemID),
		fmt.Sprintf("robotics.heartbeat.%d", m.SystemID),
	}
}

// Mission-specific helper methods

// AddWaypoint adds a navigation waypoint to the mission
func (m *MissionPayload) AddWaypoint(lat, lon, alt float64) {
	item := MissionItem{
		Seq:          uint16(len(m.Items)),
		Frame:        3, // MAV_FRAME_GLOBAL_RELATIVE_ALT
		Command:      constants.MAV_CMD_NAV_WAYPOINT,
		Current:      0,
		Autocontinue: 1,
		Param1:       0, // Hold time
		Param2:       0, // Acceptance radius
		Param3:       0, // Pass radius
		Param4:       0, // Yaw
		X:            float32(lat),
		Y:            float32(lon),
		Z:            float32(alt),
		MissionType:  m.MissionType,
	}

	m.Items = append(m.Items, item)
	m.TotalItems = uint16(len(m.Items))
}

// AddLoiterPoint adds a loiter command to the mission
func (m *MissionPayload) AddLoiterPoint(lat, lon, alt, radius, loiterTime float64) {
	item := MissionItem{
		Seq:          uint16(len(m.Items)),
		Frame:        3, // MAV_FRAME_GLOBAL_RELATIVE_ALT
		Command:      constants.MAV_CMD_NAV_LOITER_TIME,
		Current:      0,
		Autocontinue: 1,
		Param1:       float32(loiterTime),   // Loiter time in seconds
		Param2:       0,               // Empty
		Param3:       float32(radius), // Loiter radius
		Param4:       0,               // Yaw
		X:            float32(lat),
		Y:            float32(lon),
		Z:            float32(alt),
		MissionType:  m.MissionType,
	}

	m.Items = append(m.Items, item)
	m.TotalItems = uint16(len(m.Items))
}

// AddLandingPoint adds a landing command to the mission
func (m *MissionPayload) AddLandingPoint(lat, lon, alt float64) {
	item := MissionItem{
		Seq:          uint16(len(m.Items)),
		Frame:        3, // MAV_FRAME_GLOBAL_RELATIVE_ALT
		Command:      constants.MAV_CMD_NAV_LAND,
		Current:      0,
		Autocontinue: 1,
		Param1:       0, // Abort altitude
		Param2:       0, // Precision land mode
		Param3:       0, // Empty
		Param4:       0, // Yaw angle
		X:            float32(lat),
		Y:            float32(lon),
		Z:            float32(alt),
		MissionType:  m.MissionType,
	}

	m.Items = append(m.Items, item)
	m.TotalItems = uint16(len(m.Items))
}

// GetCurrentWaypoint returns the current active waypoint
func (m *MissionPayload) GetCurrentWaypoint() *MissionItem {
	for i := range m.Items {
		if m.Items[i].Seq == m.CurrentItem {
			return &m.Items[i]
		}
	}
	return nil
}

// GetNextWaypoint returns the next waypoint in the mission
func (m *MissionPayload) GetNextWaypoint() *MissionItem {
	currentIndex := -1
	for i := range m.Items {
		if m.Items[i].Seq == m.CurrentItem {
			currentIndex = i
			break
		}
	}

	if currentIndex >= 0 && currentIndex < len(m.Items)-1 {
		return &m.Items[currentIndex+1]
	}
	return nil
}

// GetProgressPercentage returns mission completion percentage (0-100)
func (m *MissionPayload) GetProgressPercentage() float64 {
	if m.TotalItems == 0 {
		return 0.0
	}
	return float64(m.CurrentItem) / float64(m.TotalItems) * 100.0
}

// GetMissionTypeName returns the human-readable mission type name
func (m *MissionPayload) GetMissionTypeName() string {
	switch m.MissionType {
	case 0:
		return "mission"
	case 1:
		return "fence"
	case 2:
		return "rally"
	default:
		return "unknown"
	}
}

// GetWaypointType returns the human-readable waypoint type
func (m *MissionPayload) GetWaypointType(command uint16) string {
	switch command {
	case constants.MAV_CMD_NAV_WAYPOINT:
		return "waypoint"
	case constants.MAV_CMD_NAV_LOITER_UNLIM:
		return "loiter_unlimited"
	case constants.MAV_CMD_NAV_LOITER_TIME:
		return "loiter_time"
	case constants.MAV_CMD_NAV_RETURN_TO_LAUNCH:
		return "return_to_launch"
	case constants.MAV_CMD_NAV_LAND:
		return "land"
	case constants.MAV_CMD_NAV_TAKEOFF:
		return "takeoff"
	case constants.MAV_CMD_NAV_ROI:
		return "region_of_interest"
	case constants.MAV_CMD_DO_CHANGE_SPEED:
		return "change_speed"
	case constants.MAV_CMD_DO_SET_HOME:
		return "set_home"
	default:
		return fmt.Sprintf("command_%d", command)
	}
}

// IsGeographicFrame returns true if the coordinate frame uses lat/lon coordinates
func (m *MissionPayload) IsGeographicFrame(frame uint8) bool {
	// MAV_FRAME values: 0=global, 3=global_relative_alt, 10=global_int, 11=global_relative_alt_int
	return frame == 0 || frame == 3 || frame == 10 || frame == 11
}

// GetWaypointLatitude extracts latitude from waypoint coordinates
func (m *MissionPayload) GetWaypointLatitude(item MissionItem) float64 {
	if m.IsGeographicFrame(item.Frame) {
		if item.Frame == 10 || item.Frame == 11 { // INT frames use degrees * 1e7
			return float64(item.X) / 1e7
		}
		return float64(item.X)
	}
	return 0.0 // Local coordinates don't have latitude
}

// GetWaypointLongitude extracts longitude from waypoint coordinates
func (m *MissionPayload) GetWaypointLongitude(item MissionItem) float64 {
	if m.IsGeographicFrame(item.Frame) {
		if item.Frame == 10 || item.Frame == 11 { // INT frames use degrees * 1e7
			return float64(item.Y) / 1e7
		}
		return float64(item.Y)
	}
	return 0.0 // Local coordinates don't have longitude
}

// GetBoundingBox returns the lat/lon bounding box of all waypoints
func (m *MissionPayload) GetBoundingBox() (minLat, maxLat, minLon, maxLon float64) {
	if len(m.Items) == 0 {
		return 0, 0, 0, 0
	}

	initialized := false
	for _, item := range m.Items {
		if m.IsGeographicFrame(item.Frame) {
			lat := m.GetWaypointLatitude(item)
			lon := m.GetWaypointLongitude(item)

			if !initialized {
				minLat, maxLat = lat, lat
				minLon, maxLon = lon, lon
				initialized = true
			} else {
				if lat < minLat {
					minLat = lat
				}
				if lat > maxLat {
					maxLat = lat
				}
				if lon < minLon {
					minLon = lon
				}
				if lon > maxLon {
					maxLon = lon
				}
			}
		}
	}

	return minLat, maxLat, minLon, maxLon
}

// CalculateTotalDistance estimates the total mission distance
func (m *MissionPayload) CalculateTotalDistance() float64 {
	totalDistance := 0.0

	for i := 0; i < len(m.Items)-1; i++ {
		current := m.Items[i]
		next := m.Items[i+1]

		// Only calculate distance for navigation commands with geographic coordinates
		if m.IsGeographicFrame(current.Frame) && m.IsGeographicFrame(next.Frame) {
			lat1 := m.GetWaypointLatitude(current)
			lon1 := m.GetWaypointLongitude(current)
			lat2 := m.GetWaypointLatitude(next)
			lon2 := m.GetWaypointLongitude(next)

			distance := m.calculateDistance(lat1, lon1, lat2, lon2)
			totalDistance += distance
		}
	}

	return totalDistance
}

// calculateDistance calculates distance between two lat/lon points using Haversine formula
func (m *MissionPayload) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth radius in meters

	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// IsBehindSchedule returns true if mission is taking longer than estimated
func (m *MissionPayload) IsBehindSchedule() bool {
	if m.EstimatedDuration <= 0 {
		return false
	}

	expectedProgress := m.TimeElapsed / m.EstimatedDuration
	actualProgress := float64(m.CurrentItem) / float64(m.TotalItems)

	return actualProgress < expectedProgress*0.9 // 10% tolerance
}

// IsCompleted returns true if the mission is completed
func (m *MissionPayload) IsCompleted() bool {
	return m.MissionState == "completed"
}

// IsActive returns true if the mission is currently executing
func (m *MissionPayload) IsActive() bool {
	return m.MissionState == "active"
}

// Alpha Week 3 Implementation: TripleGenerator interface

// Triples generates semantic triples from the mission payload using three-level dotted predicates.
// This replaces the Properties map with structured semantic assertions enabling graph queries.
//
// Generated triples include:
//   - Mission metadata (robotics.mission.*)
//   - Progress tracking (robotics.mission.progress, robotics.mission.status)
//   - Waypoint navigation (geo.location.* for waypoints)
//   - System identification (robotics.system.*)
//   - Lifecycle events (time.lifecycle.*)
func (m *MissionPayload) Triples() []message.Triple {
	entityID := m.EntityID() // mission_systemid_componentid format
	timestamp := m.Ts
	
	var triples []message.Triple
	
	// Mission metadata triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_MISSION_STATUS, // Use existing constant
			Object:     m.MissionState,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct mission status
		},
		{
			Subject:    entityID,
			Predicate:  "robotics.mission.type", // Custom predicate
			Object:     m.GetMissionTypeName(),
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Mission progress triples
	if m.TotalItems > 0 {
		triples = append(triples, []message.Triple{
			{
				Subject:    entityID,
				Predicate:  vocabulary.ROBOTICS_MISSION_PROGRESS,
				Object:     m.GetProgressPercentage(),
				Source:     "mavlink_mission",
				Timestamp:  timestamp,
				Confidence: 1.0,
			},
			{
				Subject:    entityID,
				Predicate:  vocabulary.ROBOTICS_MISSION_WAYPOINT, // Use existing constant
				Object:     fmt.Sprintf("%d", m.CurrentItem),
				Source:     "mavlink_mission",
				Timestamp:  timestamp,
				Confidence: 1.0,
			},
			{
				Subject:    entityID,
				Predicate:  "robotics.mission.total", // Custom predicate
				Object:     fmt.Sprintf("%d", m.TotalItems),
				Source:     "mavlink_mission",
				Timestamp:  timestamp,
				Confidence: 1.0,
			},
		}...)
	}
	
	// Mission name if available
	if m.MissionName != "" {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "robotics.mission.name", // Custom predicate
			Object:     m.MissionName,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		})
	}
	
	// Distance and time metrics if available
	if m.DistanceTotal > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "geo.distance.total", // Custom predicate
			Object:     m.DistanceTotal,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 0.9, // Calculated distance
		})
	}
	
	if m.DistanceRemaining > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "geo.distance.remaining", // Custom predicate
			Object:     m.DistanceRemaining,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 0.8, // Estimated remaining
		})
	}
	
	if m.TimeElapsed > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "time.mission.elapsed", // Custom predicate
			Object:     m.TimeElapsed,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		})
	}
	
	if m.EstimatedDuration > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "time.mission.estimated", // Custom predicate
			Object:     m.EstimatedDuration,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 0.7, // Estimation uncertainty
		})
	}
	
	// Mission status assessments
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  "robotics.mission.active", // Custom predicate
			Object:     m.IsActive(),
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  "robotics.mission.completed", // Custom predicate
			Object:     m.IsCompleted(),
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Behind schedule assessment if we have timing data
	if m.EstimatedDuration > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "robotics.mission.delayed", // Custom predicate
			Object:     m.IsBehindSchedule(),
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 0.8, // Schedule assessment has uncertainty
		})
	}
	
	// Current waypoint location if available
	if currentWaypoint := m.GetCurrentWaypoint(); currentWaypoint != nil && m.IsGeographicFrame(currentWaypoint.Frame) {
		droneEntityID := m.DroneEntityID()
		triples = append(triples, []message.Triple{
			{
				Subject:    droneEntityID,
				Predicate:  "geo.location.target.latitude", // Custom predicate
				Object:     m.GetWaypointLatitude(*currentWaypoint),
				Source:     "mavlink_mission",
				Timestamp:  timestamp,
				Confidence: 0.9, // Target location
			},
			{
				Subject:    droneEntityID,
				Predicate:  "geo.location.target.longitude", // Custom predicate
				Object:     m.GetWaypointLongitude(*currentWaypoint),
				Source:     "mavlink_mission",
				Timestamp:  timestamp,
				Confidence: 0.9,
			},
			{
				Subject:    droneEntityID,
				Predicate:  "geo.location.target.altitude", // Custom predicate
				Object:     float64(currentWaypoint.Z),
				Source:     "mavlink_mission",
				Timestamp:  timestamp,
				Confidence: 0.9,
			},
		}...)
	}
	
	// Relationship triple - Drone executes Mission
	droneEntityID := m.DroneEntityID()
	triples = append(triples, message.Triple{
		Subject:    droneEntityID,
		Predicate:  "robotics.component.executes", // Custom predicate
		Object:     entityID, // Entity reference
		Source:     "mavlink_mission",
		Timestamp:  timestamp,
		Confidence: 1.0,
	})
	
	// System identification for the drone
	triples = append(triples, message.Triple{
		Subject:    droneEntityID,
		Predicate:  vocabulary.ROBOTICS_SYSTEM_ID,
		Object:     fmt.Sprintf("%d", m.SystemID),
		Source:     "mavlink_mission",
		Timestamp:  timestamp,
		Confidence: 1.0,
	})
	
	// Quality metadata
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SCORE,
			Object:     1.0, // High confidence in mission data
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SOURCE,
			Object:     "mission_planning",
			Source:     "mavlink_mission",
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
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_UPDATED,
			Object:     timestamp,
			Source:     "mavlink_mission",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	return triples
}

// SetProperty sets an additional property on the payload
func (m *MissionPayload) SetProperty(key string, value any) {
	if m.Properties == nil {
		m.Properties = make(map[string]any)
	}
	m.Properties[key] = value
}

// GetProperty gets an additional property from the payload
func (m *MissionPayload) GetProperty(key string) (any, bool) {
	if m.Properties == nil {
		return nil, false
	}
	value, exists := m.Properties[key]
	return value, exists
}
