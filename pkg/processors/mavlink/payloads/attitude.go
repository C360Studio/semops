//go:build ignore
// +build ignore

package payloads

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	vocab "github.com/c360/semops/pkg/processors/mavlink/vocabulary"
	message "github.com/c360/semstreams/message"
	"github.com/c360/semstreams/vocabulary"
	"github.com/c360/streamkit/component"
)

func init() {
	component.RegisterPayload(&component.PayloadRegistration{
		Factory: func() interface{} {
			return &AttitudePayload{}
		},
		Domain:      "robotics",
		Category:    "attitude",
		Version:     "v1",
		Description: "MAVLink attitude message for orientation tracking",
		Example: map[string]interface{}{
			"system_id":        1,
			"roll":             0.1,
			"pitch":            -0.05,
			"yaw":              1.57,
			"rollspeed":        0.02,
			"pitchspeed":       -0.01,
			"yawspeed":         0.03,
			"gyro_calibrated":  true,
			"accel_calibrated": true,
			"mag_calibrated":   false,
			"timestamp":        "2024-01-15T10:30:00Z",
		},
	})
}

// AttitudePayload represents orientation and angular rate data from robotics systems.
// This payload implements behavioral interfaces to provide structured access
// to attitude data for routing, entity extraction, and correlation analysis.
type AttitudePayload struct {
	SystemID   uint8     `json:"system_id"`  // ID of this system
	Ts         time.Time `json:"timestamp"`  // Message timestamp
	Roll       float32   `json:"roll"`       // Roll angle in radians
	Pitch      float32   `json:"pitch"`      // Pitch angle in radians
	Yaw        float32   `json:"yaw"`        // Yaw angle in radians
	RollSpeed  float32   `json:"rollspeed"`  // Roll angular velocity in rad/s
	PitchSpeed float32   `json:"pitchspeed"` // Pitch angular velocity in rad/s
	YawSpeed   float32   `json:"yawspeed"`   // Yaw angular velocity in rad/s

	// Additional derived values for convenience
	RollDegrees  float64 `json:"roll_degrees,omitempty"`  // Roll in degrees
	PitchDegrees float64 `json:"pitch_degrees,omitempty"` // Pitch in degrees
	YawDegrees   float64 `json:"yaw_degrees,omitempty"`   // Yaw in degrees

	// Quality indicators
	GyroCalibrated  bool `json:"gyro_calibrated,omitempty"`  // True if gyroscopes are calibrated
	AccelCalibrated bool `json:"accel_calibrated,omitempty"` // True if accelerometers are calibrated
	MagCalibrated   bool `json:"mag_calibrated,omitempty"`   // True if magnetometer is calibrated

	Properties map[string]any `json:"properties,omitempty"` // Additional properties
}

// NewAttitudePayload creates a new attitude payload with default values.
func NewAttitudePayload(systemID uint8, timestamp time.Time) *AttitudePayload {
	return &AttitudePayload{
		SystemID:   systemID,
		Ts:         timestamp,
		Roll:       0.0,
		Pitch:      0.0,
		Yaw:        0.0,
		RollSpeed:  0.0,
		PitchSpeed: 0.0,
		YawSpeed:   0.0,
		Properties: make(map[string]any),
	}
}

// Implement Payload interface

// Schema returns the MessageType that this payload conforms to.
func (a *AttitudePayload) Schema() message.Type {
	return message.Type{
		Domain:   "robotics",
		Category: "attitude",
		Version:  "v1",
	}
}

// Validate performs validation of the attitude payload data.
func (a *AttitudePayload) Validate() error {
	if a.SystemID == 0 {
		return errors.New("system ID cannot be zero")
	}

	if a.Ts.IsZero() {
		return errors.New("timestamp is required")
	}

	// Check for reasonable angle values (should be within -π to π)
	if math.Abs(float64(a.Roll)) > math.Pi {
		return fmt.Errorf("roll angle out of range: %f rad (should be -π to π)", a.Roll)
	}

	if math.Abs(float64(a.Pitch)) > math.Pi {
		return fmt.Errorf("pitch angle out of range: %f rad (should be -π to π)", a.Pitch)
	}

	if math.Abs(float64(a.Yaw)) > math.Pi {
		return fmt.Errorf("yaw angle out of range: %f rad (should be -π to π)", a.Yaw)
	}

	// Check for reasonable angular velocity values (should be < 10 rad/s typically)
	maxAngularRate := float32(20.0) // 20 rad/s is very high but possible

	if math.Abs(float64(a.RollSpeed)) > float64(maxAngularRate) {
		return fmt.Errorf("roll rate out of range: %f rad/s (should be < %f)", a.RollSpeed, maxAngularRate)
	}

	if math.Abs(float64(a.PitchSpeed)) > float64(maxAngularRate) {
		return fmt.Errorf("pitch rate out of range: %f rad/s (should be < %f)", a.PitchSpeed, maxAngularRate)
	}

	if math.Abs(float64(a.YawSpeed)) > float64(maxAngularRate) {
		return fmt.Errorf("yaw rate out of range: %f rad/s (should be < %f)", a.YawSpeed, maxAngularRate)
	}

	return nil
}

// MarshalJSON serializes the payload to JSON format.
func (a *AttitudePayload) MarshalJSON() ([]byte, error) {
	// Use alias to avoid infinite recursion
	type Alias AttitudePayload
	return json.Marshal((*Alias)(a))
}

// UnmarshalJSON deserializes JSON data back into the payload.
func (a *AttitudePayload) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type Alias AttitudePayload
	return json.Unmarshal(data, (*Alias)(a))
}

// Implement behavioral interfaces

// Identifiable - provides entity identity for graph storage
func (a *AttitudePayload) EntityID() string {
	// System field comes from MAVLink SystemID at runtime
	system := a.mapSystemIDToSystem()

	entityID := message.EntityID{
		Org:      "c360",      // TODO: Get from config when available
		Platform: "platform1", // TODO: Get from config when available
		Domain:   "robotics",  // Domain-first hierarchy
		System:   system,      // RUNTIME value from message
		Type:     "drone",
		Instance: "0", // Single drone per system, no SystemID duplication
	}
	return entityID.Key()
}

// mapSystemIDToSystem converts MAVLink SystemID to meaningful system name
func (a *AttitudePayload) mapSystemIDToSystem() string {
	switch a.SystemID {
	case 255:
		return "gcs-main"
	case 254:
		return "gcs-backup"
	default:
		return fmt.Sprintf("mav%d", a.SystemID)
	}
}

func (a *AttitudePayload) EntityType() message.EntityType {
	return vocab.EntityTypeDrone
}

// Timestampable - provides temporal context
func (a *AttitudePayload) Timestamp() time.Time {
	return a.Ts
}

// Measurable - provides angular rate magnitude as measured value
func (a *AttitudePayload) MeasuredValue() float64 {
	return a.GetTotalAngularRate()
}

func (a *AttitudePayload) MeasuredUnit() string {
	return "rad/s"
}

// Observable - attitude system observes orientation of the vehicle
func (a *AttitudePayload) ObservedEntity() string {
	return a.EntityID()
}

func (a *AttitudePayload) ObservedProperty() string {
	return "attitude"
}

func (a *AttitudePayload) ObservedValue() any {
	return map[string]any{
		"roll_radians":       a.Roll,
		"pitch_radians":      a.Pitch,
		"yaw_radians":        a.Yaw,
		"roll_degrees":       a.GetRollDegrees(),
		"pitch_degrees":      a.GetPitchDegrees(),
		"yaw_degrees":        a.GetYawDegrees(),
		"roll_rate":          a.GetRollSpeedDegrees(),
		"pitch_rate":         a.GetPitchSpeedDegrees(),
		"yaw_rate":           a.GetYawSpeedDegrees(),
		"total_angular_rate": a.GetTotalAngularRate(),
		"is_stable":          a.IsStable(),
		"calibration_score":  a.GetCalibrationScore(),
	}
}

// Deployable - if vehicle has deployment information
func (a *AttitudePayload) DeploymentID() string {
	if deploymentID, ok := a.Properties["deployment_id"].(string); ok {
		return deploymentID
	}
	return ""
}

func (a *AttitudePayload) DeploymentName() string {
	if deploymentName, ok := a.Properties["deployment_name"].(string); ok {
		return deploymentName
	}
	return ""
}

// MessageCorrelatable - provides correlation capabilities
func (a *AttitudePayload) CorrelationID() string {
	return fmt.Sprintf("vehicle_%d_attitude", a.SystemID)
}

func (a *AttitudePayload) RelatedMessages() []string {
	// Attitude messages can be correlated with position, battery, and mission messages from the same vehicle
	return []string{
		fmt.Sprintf("robotics.position.%d", a.SystemID),
		fmt.Sprintf("robotics.battery.%d", a.SystemID),
		fmt.Sprintf("robotics.mission.%d", a.SystemID),
		fmt.Sprintf("robotics.heartbeat.%d", a.SystemID),
	}
}

// Attitude-specific helper methods

// GetRollDegrees returns roll angle in degrees
func (a *AttitudePayload) GetRollDegrees() float64 {
	return float64(a.Roll) * 180.0 / math.Pi
}

// GetPitchDegrees returns pitch angle in degrees
func (a *AttitudePayload) GetPitchDegrees() float64 {
	return float64(a.Pitch) * 180.0 / math.Pi
}

// GetYawDegrees returns yaw angle in degrees (0-360)
func (a *AttitudePayload) GetYawDegrees() float64 {
	yawDeg := float64(a.Yaw) * 180.0 / math.Pi
	if yawDeg < 0 {
		yawDeg += 360
	}
	return yawDeg
}

// GetRollSpeedDegrees returns roll angular velocity in degrees per second
func (a *AttitudePayload) GetRollSpeedDegrees() float64 {
	return float64(a.RollSpeed) * 180.0 / math.Pi
}

// GetPitchSpeedDegrees returns pitch angular velocity in degrees per second
func (a *AttitudePayload) GetPitchSpeedDegrees() float64 {
	return float64(a.PitchSpeed) * 180.0 / math.Pi
}

// GetYawSpeedDegrees returns yaw angular velocity in degrees per second
func (a *AttitudePayload) GetYawSpeedDegrees() float64 {
	return float64(a.YawSpeed) * 180.0 / math.Pi
}

// IsLevel returns true if the vehicle is approximately level (roll and pitch < 5 degrees)
func (a *AttitudePayload) IsLevel() bool {
	return math.Abs(a.GetRollDegrees()) < 5.0 && math.Abs(a.GetPitchDegrees()) < 5.0
}

// IsInverted returns true if the vehicle is inverted (roll > 90 degrees)
func (a *AttitudePayload) IsInverted() bool {
	return math.Abs(a.GetRollDegrees()) > 90.0
}

// IsSpinning returns true if the vehicle has high angular rates (> 90 deg/s in any axis)
func (a *AttitudePayload) IsSpinning() bool {
	return math.Abs(a.GetRollSpeedDegrees()) > 90.0 ||
		math.Abs(a.GetPitchSpeedDegrees()) > 90.0 ||
		math.Abs(a.GetYawSpeedDegrees()) > 90.0
}

// IsVibrating returns true if there are small, rapid oscillations
func (a *AttitudePayload) IsVibrating() bool {
	// Vibration is indicated by moderate angular rates (10-90 deg/s) in multiple axes
	rollRate := math.Abs(a.GetRollSpeedDegrees())
	pitchRate := math.Abs(a.GetPitchSpeedDegrees())
	yawRate := math.Abs(a.GetYawSpeedDegrees())

	activeAxes := 0
	if rollRate > 10.0 && rollRate < 90.0 {
		activeAxes++
	}
	if pitchRate > 10.0 && pitchRate < 90.0 {
		activeAxes++
	}
	if yawRate > 10.0 && yawRate < 90.0 {
		activeAxes++
	}

	return activeAxes >= 2
}

// GetCalibrationScore returns a score 0-1 indicating calibration quality
func (a *AttitudePayload) GetCalibrationScore() float64 {
	score := 0.0
	if a.GyroCalibrated {
		score += 0.4 // Gyro is most important
	}
	if a.AccelCalibrated {
		score += 0.3 // Accelerometer is important for level
	}
	if a.MagCalibrated {
		score += 0.3 // Magnetometer is important for heading
	}
	return score
}

// IsWellCalibrated returns true if all sensors are calibrated
func (a *AttitudePayload) IsWellCalibrated() bool {
	return a.GyroCalibrated && a.AccelCalibrated && a.MagCalibrated
}

// GetTotalAngularRate returns the magnitude of angular velocity vector
func (a *AttitudePayload) GetTotalAngularRate() float64 {
	return math.Sqrt(float64(a.RollSpeed*a.RollSpeed + a.PitchSpeed*a.PitchSpeed + a.YawSpeed*a.YawSpeed))
}

// GetBankAngle returns the bank angle (roll for aircraft, lean for ground vehicles)
func (a *AttitudePayload) GetBankAngle() float64 {
	return a.GetRollDegrees()
}

// GetElevationAngle returns the elevation angle (pitch)
func (a *AttitudePayload) GetElevationAngle() float64 {
	return a.GetPitchDegrees()
}

// GetCompassHeading returns the compass heading (yaw) in degrees 0-360
func (a *AttitudePayload) GetCompassHeading() float64 {
	return a.GetYawDegrees()
}

// IsStable returns true if angular rates are low (< 5 deg/s)
func (a *AttitudePayload) IsStable() bool {
	return math.Abs(a.GetRollSpeedDegrees()) < 5.0 &&
		math.Abs(a.GetPitchSpeedDegrees()) < 5.0 &&
		math.Abs(a.GetYawSpeedDegrees()) < 5.0
}

// GetOrientationVector returns a unit vector representing the vehicle's forward direction
func (a *AttitudePayload) GetOrientationVector() (x, y, z float64) {
	// Convert to direction cosines
	// X points forward, Y points right, Z points down (MAVLink convention)

	pitch := float64(a.Pitch)
	yaw := float64(a.Yaw)

	// Forward vector components
	x = math.Cos(pitch) * math.Cos(yaw)
	y = math.Cos(pitch) * math.Sin(yaw)
	z = -math.Sin(pitch)

	return x, y, z
}

// Alpha Week 3 Implementation: TripleGenerator interface

// Triples generates semantic triples from the attitude payload using three-level dotted predicates.
// This replaces the Properties map with structured semantic assertions enabling graph queries.
//
// Generated triples include:
//   - Orientation angles (robotics.attitude.*)
//   - Angular velocity rates (robotics.velocity.*)
//   - Calibration status (quality.calibration.*)
//   - System identification (robotics.system.*)
//   - Lifecycle events (time.lifecycle.*)
func (a *AttitudePayload) Triples() []message.Triple {
	entityID := a.EntityID() // Uses structured EntityID.Key() format
	timestamp := a.Ts

	var triples []message.Triple

	// Attitude orientation triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_ROLL,
			Object:     a.GetRollDegrees(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct IMU measurement
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_PITCH,
			Object:     a.GetPitchDegrees(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_YAW,
			Object:     a.GetYawDegrees(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_YAW, // Use yaw instead of heading
			Object:     a.GetCompassHeading(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 0.9, // Compass can be affected by magnetic interference
		},
	}...)

	// Angular velocity triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_ROLLSPEED,
			Object:     a.GetRollSpeedDegrees(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct gyro measurement
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_PITCHSPEED,
			Object:     a.GetPitchSpeedDegrees(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_ATTITUDE_YAWSPEED,
			Object:     a.GetYawSpeedDegrees(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)

	// Add total angular rate as calculated value
	triples = append(triples, message.Triple{
		Subject:    entityID,
		Predicate:  "robotics.attitude.angular.total", // Custom predicate for total angular rate
		Object:     a.GetTotalAngularRate(),
		Source:     "mavlink_attitude",
		Timestamp:  timestamp,
		Confidence: 0.9, // Calculated from components
	})

	// System status triples based on attitude analysis
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_STATUS,
			Object:     fmt.Sprintf("attitude_%s", map[bool]string{true: "stable", false: "dynamic"}[a.IsStable()]),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 0.9,
		},
		{
			Subject:    entityID,
			Predicate:  "robotics.attitude.level", // Custom predicate for level status
			Object:     a.IsLevel(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 0.9,
		},
	}...)

	// Add critical status indicators if present
	if a.IsInverted() {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "robotics.system.alert", // Custom predicate for alerts
			Object:     "inverted",
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		})
	}

	if a.IsSpinning() {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "robotics.system.alert", // Custom predicate for alerts
			Object:     "spinning",
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		})
	}

	if a.IsVibrating() {
		triples = append(triples, message.Triple{
			Subject:    entityID,
			Predicate:  "robotics.system.alert", // Custom predicate for alerts
			Object:     "vibrating",
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 0.8, // Vibration detection has some uncertainty
		})
	}

	// Calibration quality triples
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  "quality.calibration.gyro", // Custom predicates for calibration
			Object:     a.GyroCalibrated,
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  "quality.calibration.accelerometer",
			Object:     a.AccelCalibrated,
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  "quality.calibration.magnetometer",
			Object:     a.MagCalibrated,
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SCORE,
			Object:     a.GetCalibrationScore(),
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 0.9,
		},
	}...)

	// Lifecycle events
	triples = append(triples, []message.Triple{
		{
			Subject:    entityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_SEEN,
			Object:     timestamp,
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    entityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_UPDATED,
			Object:     timestamp,
			Source:     "mavlink_attitude",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)

	return triples
}

// SetProperty sets an additional property on the payload
func (a *AttitudePayload) SetProperty(key string, value any) {
	if a.Properties == nil {
		a.Properties = make(map[string]any)
	}
	a.Properties[key] = value
}

// GetProperty gets an additional property from the payload
func (a *AttitudePayload) GetProperty(key string) (any, bool) {
	if a.Properties == nil {
		return nil, false
	}
	value, exists := a.Properties[key]
	return value, exists
}
