package payloads

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/c360/streamkit/component"
	message "github.com/c360/semstreams/message"
	vocab "github.com/c360/semops/pkg/processors/mavlink/vocabulary"
	"github.com/c360/semstreams/vocabulary"
)

func init() {
	component.RegisterPayload(&component.PayloadRegistration{
		Factory: func() interface{} {
			return &PositionPayload{}
		},
		Domain:      "robotics",
		Category:    "position",
		Version:     "v1",
		Description: "MAVLink GPS/position data for location tracking",
		Example: map[string]interface{}{
			"system_id":      1,
			"lat":            123456789,
			"lon":            987654321,
			"alt":            1000,
			"relative_alt":   100,
			"vx":             100,
			"vy":             200,
			"vz":             -50,
			"hdg":            9000,
			"gps_fix":        3,
			"satellites":     12,
			"timestamp":      "2024-01-15T10:30:00Z",
		},
	})
}

// PositionPayload represents GPS/position data from robotics systems.
// This payload implements behavioral interfaces to provide structured access
// to position data for routing, entity extraction, and correlation analysis.
type PositionPayload struct {
	SystemID    uint8     `json:"system_id"`    // ID of this system
	Ts          time.Time `json:"timestamp"`    // Message timestamp
	Lat         int32     `json:"lat"`          // Latitude * 1e7 (degrees)
	Lon         int32     `json:"lon"`          // Longitude * 1e7 (degrees)
	Alt         int32     `json:"alt"`          // Altitude in mm (AMSL)
	RelativeAlt int32     `json:"relative_alt"` // Altitude above ground in mm
	Vx          int16     `json:"vx"`           // Ground speed North (cm/s)
	Vy          int16     `json:"vy"`           // Ground speed East (cm/s)
	Vz          int16     `json:"vz"`           // Ground speed Down (cm/s)
	Hdg         uint16    `json:"hdg"`          // Vehicle heading * 100 (degrees)

	// Additional metadata for enhanced functionality
	GPSFix           uint8   `json:"gps_fix,omitempty"`      // GPS fix type (0=none, 1=no fix, 2=2D, 3=3D)
	Satellites       uint8   `json:"satellites,omitempty"`   // Number of satellites visible
	HDOP             float64 `json:"hdop,omitempty"`         // Horizontal dilution of precision
	VDOP             float64 `json:"vdop,omitempty"`         // Vertical dilution of precision
	EPH              float64 `json:"eph,omitempty"`          // GPS horizontal accuracy (m)
	EPV              float64 `json:"epv,omitempty"`          // GPS vertical accuracy (m)
	GroundSpeed      float64 `json:"ground_speed,omitempty"` // Ground speed (m/s)
	CourseOverGround float64 `json:"cog,omitempty"`          // Course over ground (degrees)

	// Extended properties
	Properties map[string]any `json:"properties,omitempty"` // Additional properties
}

// NewPositionPayload creates a new position payload with default values.
func NewPositionPayload(systemID uint8, timestamp time.Time, lat, lon float64) *PositionPayload {
	return &PositionPayload{
		SystemID:    systemID,
		Ts:          timestamp,
		Lat:         int32(lat * 1e7),
		Lon:         int32(lon * 1e7),
		Alt:         0,
		RelativeAlt: 0,
		Vx:          0,
		Vy:          0,
		Vz:          0,
		Hdg:         0,
		GPSFix:      3, // Assume 3D fix by default
		Properties:  make(map[string]any),
	}
}

// Implement Payload interface

// Schema returns the MessageType that this payload conforms to.
func (p *PositionPayload) Schema() message.Type {
	return message.Type{
		Domain:   "robotics",
		Category: "position",
		Version:  "v1",
	}
}

// Validate performs validation of the position payload data.
func (p *PositionPayload) Validate() error {
	if p.SystemID == 0 {
		return errors.New("system ID cannot be zero")
	}

	if p.Ts.IsZero() {
		return errors.New("timestamp is required")
	}

	// Validate latitude range
	lat := p.GetLatitude()
	if lat < -90.0 || lat > 90.0 {
		return fmt.Errorf("latitude out of range: %f (must be -90 to 90)", lat)
	}

	// Validate longitude range
	lon := p.GetLongitude()
	if lon < -180.0 || lon > 180.0 {
		return fmt.Errorf("longitude out of range: %f (must be -180 to 180)", lon)
	}

	// Validate heading range
	heading := p.GetHeadingDegrees()
	if heading < 0.0 || heading >= 360.0 {
		return fmt.Errorf("heading out of range: %f (must be 0 to 359.99)", heading)
	}

	// Validate GPS fix type
	if p.GPSFix > 3 {
		return fmt.Errorf("invalid GPS fix type: %d (must be 0-3)", p.GPSFix)
	}

	// Validate accuracy values if present
	if p.EPH < 0 {
		return fmt.Errorf("horizontal accuracy cannot be negative: %f", p.EPH)
	}
	if p.EPV < 0 {
		return fmt.Errorf("vertical accuracy cannot be negative: %f", p.EPV)
	}

	return nil
}

// MarshalJSON serializes the payload to JSON format.
func (p *PositionPayload) MarshalJSON() ([]byte, error) {
	// Use alias to avoid infinite recursion
	type Alias PositionPayload
	return json.Marshal((*Alias)(p))
}

// UnmarshalJSON deserializes JSON data back into the payload.
func (p *PositionPayload) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type Alias PositionPayload
	return json.Unmarshal(data, (*Alias)(p))
}

// Implement behavioral interfaces

// Identifiable - provides entity identity for graph storage
func (p *PositionPayload) EntityID() string {
	// System field comes from MAVLink SystemID at runtime
	system := p.mapSystemIDToSystem()
	
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
func (p *PositionPayload) mapSystemIDToSystem() string {
	switch p.SystemID {
	case 255:
		return "gcs-main"
	case 254:
		return "gcs-backup"
	default:
		return fmt.Sprintf("mav%d", p.SystemID)
	}
}

func (p *PositionPayload) EntityType() message.EntityType {
	return vocab.EntityTypeDrone
}

// Locatable - provides spatial location information
func (p *PositionPayload) Location() (lat, lon float64) {
	return p.GetLatitude(), p.GetLongitude()
}

// Timestampable - provides temporal context
func (p *PositionPayload) Timestamp() time.Time {
	return p.Ts
}

// Measurable - provides GPS accuracy as measured value
func (p *PositionPayload) MeasuredValue() float64 {
	return p.EPH
}

func (p *PositionPayload) MeasuredUnit() string {
	return "meters"
}

// Observable - GPS observes position of the vehicle
func (p *PositionPayload) ObservedEntity() string {
	return p.EntityID()
}

func (p *PositionPayload) ObservedProperty() string {
	return "position"
}

func (p *PositionPayload) ObservedValue() any {
	return map[string]any{
		"latitude":            p.GetLatitude(),
		"longitude":           p.GetLongitude(),
		"altitude_amsl":       p.GetAltitudeMeters(),
		"altitude_agl":        p.GetRelativeAltitudeMeters(),
		"heading":             p.GetHeadingDegrees(),
		"ground_speed":        p.GetGroundSpeedMS(),
		"vertical_speed":      p.GetVerticalSpeed(),
		"horizontal_accuracy": p.EPH,
		"vertical_accuracy":   p.EPV,
	}
}

// Deployable - if vehicle has deployment information
func (p *PositionPayload) DeploymentID() string {
	if deploymentID, ok := p.Properties["deployment_id"].(string); ok {
		return deploymentID
	}
	return ""
}

func (p *PositionPayload) DeploymentName() string {
	if deploymentName, ok := p.Properties["deployment_name"].(string); ok {
		return deploymentName
	}
	return ""
}

// MessageCorrelatable - provides correlation capabilities
func (p *PositionPayload) CorrelationID() string {
	return fmt.Sprintf("vehicle_%d_position", p.SystemID)
}

func (p *PositionPayload) RelatedMessages() []string {
	// Position messages can be correlated with attitude, battery, and mission messages from the same vehicle
	return []string{
		fmt.Sprintf("robotics.attitude.%d", p.SystemID),
		fmt.Sprintf("robotics.battery.%d", p.SystemID),
		fmt.Sprintf("robotics.mission.%d", p.SystemID),
		fmt.Sprintf("robotics.heartbeat.%d", p.SystemID),
	}
}

// Position-specific helper methods

// GetLatitude returns latitude in degrees
func (p *PositionPayload) GetLatitude() float64 {
	return float64(p.Lat) / 1e7
}

// GetLongitude returns longitude in degrees
func (p *PositionPayload) GetLongitude() float64 {
	return float64(p.Lon) / 1e7
}

// GetAltitudeMeters returns altitude above mean sea level in meters
func (p *PositionPayload) GetAltitudeMeters() float64 {
	return float64(p.Alt) / 1000.0
}

// GetRelativeAltitudeMeters returns altitude above ground level in meters
func (p *PositionPayload) GetRelativeAltitudeMeters() float64 {
	return float64(p.RelativeAlt) / 1000.0
}

// GetHeadingDegrees returns vehicle heading in degrees (0-359.99)
func (p *PositionPayload) GetHeadingDegrees() float64 {
	return float64(p.Hdg) / 100.0
}

// GetVelocityNorth returns northward velocity in m/s
func (p *PositionPayload) GetVelocityNorth() float64 {
	return float64(p.Vx) / 100.0
}

// GetVelocityEast returns eastward velocity in m/s
func (p *PositionPayload) GetVelocityEast() float64 {
	return float64(p.Vy) / 100.0
}

// GetVelocityDown returns downward velocity in m/s
func (p *PositionPayload) GetVelocityDown() float64 {
	return float64(p.Vz) / 100.0
}

// GetGroundSpeedMS returns ground speed in m/s
func (p *PositionPayload) GetGroundSpeedMS() float64 {
	if p.GroundSpeed > 0 {
		return p.GroundSpeed
	}
	// Calculate from velocity components if not explicitly set
	vn := p.GetVelocityNorth()
	ve := p.GetVelocityEast()
	return math.Sqrt(vn*vn + ve*ve)
}

// GetGroundSpeedKnots returns ground speed in knots
func (p *PositionPayload) GetGroundSpeedKnots() float64 {
	return p.GetGroundSpeedMS() * 1.94384 // m/s to knots conversion
}

// GetVerticalSpeed returns vertical speed (climb rate) in m/s
func (p *PositionPayload) GetVerticalSpeed() float64 {
	return -p.GetVelocityDown() // Negative because MAVLink down is positive
}

// HasGPSFix returns true if there is a valid GPS fix
func (p *PositionPayload) HasGPSFix() bool {
	return p.GPSFix >= 2 // 2D or 3D fix
}

// Has3DFix returns true if there is a 3D GPS fix
func (p *PositionPayload) Has3DFix() bool {
	return p.GPSFix >= 3
}

// IsGPSHealthy returns true if GPS appears to be working well
func (p *PositionPayload) IsGPSHealthy() bool {
	return p.Has3DFix() && p.Satellites >= 6 && p.HDOP < 2.0
}

// GetDistanceTo calculates distance to another position in meters
func (p *PositionPayload) GetDistanceTo(otherLat, otherLon float64) float64 {
	lat1 := p.GetLatitude() * math.Pi / 180
	lon1 := p.GetLongitude() * math.Pi / 180
	lat2 := otherLat * math.Pi / 180
	lon2 := otherLon * math.Pi / 180

	deltaLat := lat2 - lat1
	deltaLon := lon2 - lon1

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Earth radius in meters
	return 6371000 * c
}

// GetBearingTo calculates bearing to another position in degrees
func (p *PositionPayload) GetBearingTo(otherLat, otherLon float64) float64 {
	lat1 := p.GetLatitude() * math.Pi / 180
	lon1 := p.GetLongitude() * math.Pi / 180
	lat2 := otherLat * math.Pi / 180
	lon2 := otherLon * math.Pi / 180

	deltaLon := lon2 - lon1

	y := math.Sin(deltaLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi

	// Normalize to 0-360 degrees
	if bearing < 0 {
		bearing += 360
	}

	return bearing
}

// SetProperty sets an additional property on the payload
func (p *PositionPayload) SetProperty(key string, value any) {
	if p.Properties == nil {
		p.Properties = make(map[string]any)
	}
	p.Properties[key] = value
}

// GetProperty gets an additional property from the payload
func (p *PositionPayload) GetProperty(key string) (any, bool) {
	if p.Properties == nil {
		return nil, false
	}
	value, exists := p.Properties[key]
	return value, exists
}

// Alpha Week 3 Implementation: TripleGenerator interface

// Triples generates semantic triples from the position payload using three-level dotted predicates.
// This replaces the Properties map with structured semantic assertions enabling graph queries.
//
// Generated triples include:
//   - Geospatial coordinates (geo.location.*)
//   - Velocity and movement (geo.velocity.*)
//   - GPS accuracy metrics (geo.accuracy.*)
//   - System identification (robotics.system.*)
//   - Lifecycle events (time.lifecycle.*)
func (p *PositionPayload) Triples() []message.Triple {
	system := p.mapSystemIDToSystem()
	entityID := message.EntityID{
		Org:      "c360",
		Platform: "platform1",
		System:   system,
		Domain:   "robotics",
		Type:     "drone",
		Instance: "0",  // Single drone per system, no SystemID duplication
	}.Key()
	
	timestamp := p.Ts
	var triples []message.Triple
	
	// Geospatial location triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_LOCATION_LATITUDE,
			Object:     p.GetLatitude(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct GPS measurement
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_LOCATION_LONGITUDE,
			Object:     p.GetLongitude(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_LOCATION_ALTITUDE,
			Object:     p.GetAltitudeMeters(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 0.9, // GPS altitude less accurate than lat/lon
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_LOCATION_ELEVATION,
			Object:     p.GetRelativeAltitudeMeters(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 0.8, // Relative altitude estimation
		},
	}...)
	
	// Velocity and movement triples
	if p.GetGroundSpeedMS() > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_VELOCITY_GROUND,
			Object:     p.GetGroundSpeedMS(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 0.9, // Calculated from GPS
		})
	}
	
	if p.GetVerticalSpeed() != 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_VELOCITY_VERTICAL,
			Object:     p.GetVerticalSpeed(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 0.8, // Vertical speed calculation
		})
	}
	
	if p.Hdg > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_VELOCITY_HEADING,
			Object:     p.GetHeadingDegrees(),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 0.9, // Direct compass measurement
		})
	}
	
	// GPS accuracy and quality triples
	if p.HDOP > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_ACCURACY_DILUTION,
			Object:     p.HDOP,
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct GPS metric
		})
	}
	
	if p.VDOP > 0 {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  vocabulary.GEO_ACCURACY_VERTICAL,
			Object:     p.VDOP,
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0,
		})
	}
	
	// GPS satellite and fix information
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_STATUS,
			Object:     fmt.Sprintf("gps_fix_%d", p.GPSFix),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_ACCURACY_PRECISION,
			Object:     float64(p.Satellites),
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0, // Satellite count is precise
		},
	}...)
	
	// Quality assessment based on GPS health
	confidenceScore := 0.0
	if p.IsGPSHealthy() {
		confidenceScore = 1.0
	} else if p.Has3DFix() {
		confidenceScore = 0.8
	} else if p.HasGPSFix() {
		confidenceScore = 0.6
	} else {
		confidenceScore = 0.2
	}
	
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SCORE,
			Object:     confidenceScore,
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 0.9,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SOURCE,
			Object:     "gps_status_assessment",
			Source:     "mavlink_position",
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
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_UPDATED,
			Object:     timestamp,
			Source:     "mavlink_position",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	return triples
}
