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
			return &BatteryPayload{}
		},
		Domain:      "robotics",
		Category:    "battery",
		Version:     "v1",
		Description: "MAVLink battery status message for power monitoring",
		Example: map[string]interface{}{
			"system_id":         1,
			"battery_id":        0,
			"voltages":          []uint16{4200, 4150, 4180, 4190},
			"current_battery":   1500,
			"battery_remaining": 75,
			"charge_state":      4,
			"temperature":       2500,
			"timestamp":         "2024-01-15T10:30:00Z",
		},
	})
}

// BatteryPayload represents comprehensive battery information from robotics systems.
// This payload implements both the Payload and Message interfaces, eliminating
// the need for a wrapper layer. It provides structured access to battery data
// for routing, entity extraction, and correlation analysis.
type BatteryPayload struct {

	// Battery identification
	SystemID  uint8     `json:"system_id"`  // ID of this system
	BatteryID uint8     `json:"battery_id"` // Battery ID (0-9)
	Ts        time.Time `json:"timestamp"`  // Message timestamp

	// Battery State Information
	Voltages         []uint16 `json:"voltages"`          // Battery voltage per cell (mV)
	CurrentBattery   int16    `json:"current_battery"`   // Battery current in 10*mA (-1 if not measured)
	CurrentConsumed  int32    `json:"current_consumed"`  // Consumed charge in mAh (-1 if not measured)
	EnergyConsumed   int32    `json:"energy_consumed"`   // Consumed energy in hJ (-1 if not measured)
	BatteryRemaining int8     `json:"battery_remaining"` // Battery remaining percentage (0-100, -1 if not measured)
	TimeRemaining    int32    `json:"time_remaining"`    // Remaining time in seconds (-1 if not measured)
	ChargeState      uint8    `json:"charge_state"`      // Battery charge state (MAV_BATTERY_CHARGE_STATE)

	// Temperature Information
	Temperature int16 `json:"temperature"` // Battery temperature in centigrade (-1 if not measured)

	// Additional Metadata
	BatteryFunction uint8 `json:"battery_function,omitempty"` // Battery function (main, backup, etc.)
	BatteryType     uint8 `json:"battery_type,omitempty"`     // Battery chemistry type

	// Derived and Extended Information
	TotalVoltage float64 `json:"total_voltage,omitempty"` // Total battery voltage (V)
	CurrentAmps  float64 `json:"current_amps,omitempty"`  // Current in Amps
	PowerWatts   float64 `json:"power_watts,omitempty"`   // Power consumption in Watts
	CellCount    uint8   `json:"cell_count,omitempty"`    // Number of cells detected

	Properties map[string]any `json:"properties,omitempty"` // Additional properties
}

// NewBatteryPayload creates a new battery status payload with default values.
func NewBatteryPayload(systemID, batteryID uint8, timestamp time.Time) *BatteryPayload {
	return &BatteryPayload{
		// Battery data
		SystemID:         systemID,
		BatteryID:        batteryID,
		Ts:               timestamp,
		Voltages:         make([]uint16, 0),
		CurrentBattery:   -1, // Not measured
		CurrentConsumed:  -1,
		EnergyConsumed:   -1,
		BatteryRemaining: -1,
		TimeRemaining:    -1,
		ChargeState:      constants.MavBatteryChargeStateUndefined,
		Temperature:      -1,
		Properties:       make(map[string]any),
	}
}

// Implement Payload interface

// Schema returns the MessageType that this payload conforms to.
func (b *BatteryPayload) Schema() message.Type {
	return message.Type{
		Domain:   "robotics",
		Category: "battery",
		Version:  "v1",
	}
}

// Validate performs validation of the battery payload data.
func (b *BatteryPayload) Validate() error {
	// Validate battery payload data
	if b.SystemID == 0 {
		return errors.New("system ID cannot be zero")
	}

	if b.Ts.IsZero() {
		return errors.New("timestamp is required")
	}

	// Validate battery percentage range
	if b.BatteryRemaining < -1 || b.BatteryRemaining > 100 {
		return fmt.Errorf("battery remaining out of range: %d (must be -1 or 0-100)", b.BatteryRemaining)
	}

	// Validate charge state
	if b.ChargeState > constants.MavBatteryChargeStateCharging {
		return fmt.Errorf("invalid charge state: %d", b.ChargeState)
	}

	// Validate cell voltages (typical Li-ion: 2.5V to 4.3V per cell)
	for i, voltage := range b.Voltages {
		volts := float64(voltage) / 1000.0
		if volts < 2.0 || volts > 5.0 { // Allow some margin for different chemistries
			return fmt.Errorf("cell %d voltage out of range: %.3fV (should be 2.0-5.0V)", i, volts)
		}
	}

	// Validate temperature range (-40°C to +85°C is typical for electronics)
	if b.Temperature != -1 {
		tempC := float64(b.Temperature) / 100.0
		if tempC < -50.0 || tempC > 100.0 {
			return fmt.Errorf("temperature out of range: %.1f°C (should be -50 to +100°C)", tempC)
		}
	}

	return nil
}

// MarshalJSON serializes the payload to JSON format.
func (b *BatteryPayload) MarshalJSON() ([]byte, error) {
	// Use alias to avoid infinite recursion
	type Alias BatteryPayload
	return json.Marshal((*Alias)(b))
}

// UnmarshalJSON deserializes JSON data back into the payload.
func (b *BatteryPayload) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type Alias BatteryPayload
	return json.Unmarshal(data, (*Alias)(b))
}

// Implement behavioral interfaces

// Identifiable - provides entity identity for graph storage
func (b *BatteryPayload) EntityID() string {
	// System field comes from MAVLink SystemID at runtime
	system := b.mapSystemIDToSystem()
	
	entityID := message.EntityID{
		Org:      "c360",        // TODO: Get from config when available
		Platform: "platform1",   // TODO: Get from config when available
		Domain:   "robotics",    // Domain moves up in hierarchy
		System:   system,        // RUNTIME value from message
		Type:     "battery",
		Instance: fmt.Sprintf("%d", b.BatteryID),  // Just battery ID, not compound
	}
	return entityID.Key()
}

// mapSystemIDToSystem converts MAVLink SystemID to meaningful system name
func (b *BatteryPayload) mapSystemIDToSystem() string {
	switch b.SystemID {
	case 255:
		return "gcs-main"
	case 254:
		return "gcs-backup"
	default:
		return fmt.Sprintf("mav%d", b.SystemID)
	}
}

func (b *BatteryPayload) EntityType() message.EntityType {
	return vocab.EntityTypeBattery
}

// Timestampable - provides temporal context
func (b *BatteryPayload) Timestamp() time.Time {
	return b.Ts
}

// Measurable - provides battery percentage as measured value
func (b *BatteryPayload) MeasuredValue() float64 {
	return float64(b.BatteryRemaining)
}

func (b *BatteryPayload) MeasuredUnit() string {
	return "percent"
}

// Observable - battery system observes power state of the vehicle
func (b *BatteryPayload) ObservedEntity() string {
	return b.EntityID()
}

func (b *BatteryPayload) ObservedProperty() string {
	return "power_status"
}

func (b *BatteryPayload) ObservedValue() any {
	return map[string]any{
		"charge_percentage": b.BatteryRemaining,
		"total_voltage":     b.GetTotalVoltage(),
		"current_amps":      b.GetCurrentAmps(),
		"power_watts":       b.GetPowerWatts(),
		"temperature_c":     b.GetTemperatureCelsius(),
		"charge_state":      constants.GetBatteryChargeStateName(b.ChargeState),
		"time_remaining":    b.TimeRemaining,
		"is_healthy":        b.IsHealthy(),
		"is_critical":       b.IsCritical(),
		"cell_count":        len(b.Voltages),
	}
}

// Deployable - if battery has deployment information
func (b *BatteryPayload) DeploymentID() string {
	if deploymentID, ok := b.Properties["deployment_id"].(string); ok {
		return deploymentID
	}
	return ""
}

func (b *BatteryPayload) DeploymentName() string {
	if deploymentName, ok := b.Properties["deployment_name"].(string); ok {
		return deploymentName
	}
	return ""
}

// MessageCorrelatable - provides correlation capabilities
func (b *BatteryPayload) CorrelationID() string {
	return fmt.Sprintf("battery_%d_%d", b.SystemID, b.BatteryID)
}

func (b *BatteryPayload) RelatedMessages() []string {
	// Battery messages can be correlated with position, attitude, and mission messages from the same vehicle
	return []string{
		fmt.Sprintf("robotics.position.%d", b.SystemID),
		fmt.Sprintf("robotics.attitude.%d", b.SystemID),
		fmt.Sprintf("robotics.mission.%d", b.SystemID),
		fmt.Sprintf("robotics.heartbeat.%d", b.SystemID),
	}
}





// Battery-specific helper methods

// GetTotalVoltage returns the total battery voltage in volts
func (b *BatteryPayload) GetTotalVoltage() float64 {
	if b.TotalVoltage > 0 {
		return b.TotalVoltage
	}

	// Calculate from individual cell voltages if available
	if len(b.Voltages) > 0 {
		total := uint32(0)
		for _, cellVoltage := range b.Voltages {
			total += uint32(cellVoltage)
		}
		return float64(total) / 1000.0
	}

	return 0.0
}

// GetCurrentAmps returns the current in Amperes
func (b *BatteryPayload) GetCurrentAmps() float64 {
	if b.CurrentAmps > 0 {
		return b.CurrentAmps
	}

	if b.CurrentBattery != -1 {
		return float64(b.CurrentBattery) / 100.0 // Convert from 10*mA to A
	}

	return 0.0
}

// GetPowerWatts returns the power consumption in Watts
func (b *BatteryPayload) GetPowerWatts() float64 {
	if b.PowerWatts > 0 {
		return b.PowerWatts
	}

	voltage := b.GetTotalVoltage()
	current := b.GetCurrentAmps()

	if voltage > 0 && current > 0 {
		return voltage * current
	}

	return 0.0
}

// GetTemperatureCelsius returns the temperature in Celsius
func (b *BatteryPayload) GetTemperatureCelsius() float64 {
	if b.Temperature == -1 {
		return 0.0
	}
	return float64(b.Temperature) / 100.0
}

// GetEnergyConsumedWh returns the consumed energy in Watt-hours
func (b *BatteryPayload) GetEnergyConsumedWh() float64 {
	if b.EnergyConsumed == -1 {
		return 0.0
	}
	return float64(b.EnergyConsumed) / 36000.0 // Convert from hJ to Wh
}

// GetChargeStateName returns the human-readable charge state
func (b *BatteryPayload) GetChargeStateName() string {
	return constants.GetBatteryChargeStateName(b.ChargeState)
}

// IsHealthy returns true if the battery is in good condition
func (b *BatteryPayload) IsHealthy() bool {
	return b.ChargeState == constants.MavBatteryChargeStateOk &&
		b.BatteryRemaining > int8(constants.DEFAULT_LOW_BATTERY_THRESHOLD)
}

// IsCritical returns true if the battery is in a critical state
func (b *BatteryPayload) IsCritical() bool {
	return b.ChargeState == constants.MavBatteryChargeStateCritical ||
		b.ChargeState == constants.MavBatteryChargeStateEmergency ||
		b.ChargeState == constants.MavBatteryChargeStateFailed ||
		b.BatteryRemaining <= int8(constants.DEFAULT_CRITICAL_BATTERY_THRESHOLD)
}

// IsLow returns true if the battery level is low
func (b *BatteryPayload) IsLow() bool {
	return b.BatteryRemaining > 0 &&
		b.BatteryRemaining <= int8(constants.DEFAULT_LOW_BATTERY_THRESHOLD)
}

// IsCharging returns true if the battery is currently charging
func (b *BatteryPayload) IsCharging() bool {
	return b.ChargeState == constants.MavBatteryChargeStateCharging
}

// IsCellBalanced checks if a specific cell is balanced (within 100mV of average)
func (b *BatteryPayload) IsCellBalanced(cellIndex int) bool {
	if cellIndex >= len(b.Voltages) || len(b.Voltages) < 2 {
		return true // Cannot determine balance with < 2 cells
	}

	// Calculate average cell voltage
	total := uint32(0)
	for _, voltage := range b.Voltages {
		total += uint32(voltage)
	}
	average := total / uint32(len(b.Voltages))

	// Check if this cell is within 100mV of average
	cellVoltage := b.Voltages[cellIndex]
	diff := int32(cellVoltage) - int32(average)
	if diff < 0 {
		diff = -diff
	}

	return diff <= 100 // 100mV tolerance
}

// GetCellImbalance returns the maximum voltage difference between cells in mV
func (b *BatteryPayload) GetCellImbalance() uint16 {
	if len(b.Voltages) < 2 {
		return 0
	}

	minVoltage := b.Voltages[0]
	maxVoltage := b.Voltages[0]

	for _, voltage := range b.Voltages[1:] {
		if voltage < minVoltage {
			minVoltage = voltage
		}
		if voltage > maxVoltage {
			maxVoltage = voltage
		}
	}

	return maxVoltage - minVoltage
}

// GetTimeRemainingMinutes returns the estimated time remaining in minutes
func (b *BatteryPayload) GetTimeRemainingMinutes() float64 {
	if b.TimeRemaining <= 0 {
		return 0.0
	}
	return float64(b.TimeRemaining) / 60.0
}

// GetAlertSeverity returns the alert severity level
func (b *BatteryPayload) GetAlertSeverity() string {
	switch b.ChargeState {
	case constants.MavBatteryChargeStateFailed:
		return "critical"
	case constants.MavBatteryChargeStateEmergency:
		return "emergency"
	case constants.MavBatteryChargeStateCritical:
		return "critical"
	case constants.MavBatteryChargeStateLow:
		return "warning"
	case constants.MavBatteryChargeStateUnhealthy:
		return "warning"
	default:
		if b.BatteryRemaining >= 0 {
			if b.BatteryRemaining <= int8(constants.DEFAULT_EMERGENCY_BATTERY_THRESHOLD) {
				return "emergency"
			} else if b.BatteryRemaining <= int8(constants.DEFAULT_CRITICAL_BATTERY_THRESHOLD) {
				return "critical"
			} else if b.BatteryRemaining <= int8(constants.DEFAULT_LOW_BATTERY_THRESHOLD) {
				return "warning"
			}
		}
		return "info"
	}
}

// SetProperty sets an additional property on the payload
func (b *BatteryPayload) SetProperty(key string, value any) {
	if b.Properties == nil {
		b.Properties = make(map[string]any)
	}
	b.Properties[key] = value
}

// GetProperty gets an additional property from the payload
func (b *BatteryPayload) GetProperty(key string) (any, bool) {
	if b.Properties == nil {
		return nil, false
	}
	value, exists := b.Properties[key]
	return value, exists
}

// Alpha Week 3 Implementation: TripleGenerator interface

// Triples generates semantic triples from the battery payload using three-level dotted predicates.
// This replaces the Properties map with structured semantic assertions enabling graph queries.
//
// Generated triples include:
//   - Battery power metrics (robotics.battery.*)
//   - Temperature readings (sensor.temperature.*)
//   - System identification (robotics.system.*)
//   - Component relationships (robotics.component.*)
//   - Quality assessments (quality.*)
func (b *BatteryPayload) Triples() []message.Triple {
	// Generate entity IDs using structured format
	system := b.mapSystemIDToSystem()
	droneEntityID := message.EntityID{
		Org:      "c360",
		Platform: "platform1",
		Domain:   "robotics",    // Domain-first hierarchy
		System:   system,
		Type:     "drone",
		Instance: "0",  // Single drone per system, no SystemID duplication
	}.Key()
	
	batteryEntityID := message.EntityID{
		Org:      "c360",
		Platform: "platform1",
		Domain:   "robotics",    // Domain-first hierarchy
		System:   system,
		Type:     "battery",
		Instance: fmt.Sprintf("%d", b.BatteryID),  // Just battery ID per new standard
	}.Key()
	
	timestamp := b.Ts
	var triples []message.Triple
	
	// Battery power and energy triples
	if b.BatteryRemaining >= 0 {
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_BATTERY_LEVEL,
			Object:     float64(b.BatteryRemaining),
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct measurement
		})
	}
	
	if voltage := b.GetTotalVoltage(); voltage > 0 {
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_BATTERY_VOLTAGE,
			Object:     voltage,
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 0.9, // Calculated from cells
		})
	}
	
	if current := b.GetCurrentAmps(); current > 0 {
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_BATTERY_CURRENT,
			Object:     current,
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 0.9,
		})
	}
	
	if b.TimeRemaining > 0 {
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_BATTERY_REMAINING,
			Object:     b.GetTimeRemainingMinutes(),
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 0.8, // Estimated value
		})
	}
	
	if b.CurrentConsumed > 0 {
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_BATTERY_CONSUMED,
			Object:     float64(b.CurrentConsumed),
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0, // Direct measurement
		})
	}
	
	// Temperature measurement
	if b.Temperature != -1 {
		tempCelsius := b.GetTemperatureCelsius()
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.SENSOR_TEMPERATURE_CELSIUS,
			Object:     tempCelsius,
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 0.9, // Temperature sensor
		})
	}
	
	// System identification
	triples = append(triples, []message.Triple{
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_ID,
			Object:     fmt.Sprintf("%d.%d", b.SystemID, b.BatteryID),
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_STATUS,
			Object:     b.GetChargeStateName(),
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.ROBOTICS_SYSTEM_TYPE,
			Object:     "battery",
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Battery powers drone (relationship from battery's perspective)
	triples = append(triples, message.Triple{
		Subject:    batteryEntityID,
		Predicate:  vocabulary.ROBOTICS_COMPONENT_POWERED,
		Object:     droneEntityID, // Entity reference
		Source:     "mavlink_battery",
		Timestamp:  timestamp,
		Confidence: 1.0,
	})
	
	// Lifecycle events
	triples = append(triples, []message.Triple{
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_SEEN,
			Object:     timestamp,
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.TIME_LIFECYCLE_UPDATED,
			Object:     timestamp,
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Quality and health assessments
	healthScore := 0.0
	if b.IsHealthy() {
		healthScore = 1.0
	} else if b.IsLow() {
		healthScore = 0.6
	} else if b.IsCritical() {
		healthScore = 0.2
	}
	
	triples = append(triples, []message.Triple{
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SCORE,
			Object:     healthScore,
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 0.9, // Assessment confidence
		},
		{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.QUALITY_CONFIDENCE_SOURCE,
			Object:     "battery_status_assessment",
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 1.0,
		},
	}...)
	
	// Cell voltage analysis if available
	if len(b.Voltages) > 1 {
		imbalance := b.GetCellImbalance()
		triples = append(triples, message.Triple{
			Subject:    batteryEntityID,
			Predicate:  vocabulary.QUALITY_ACCURACY_PRECISION,
			Object:     float64(imbalance),
			Source:     "mavlink_battery",
			Timestamp:  timestamp,
			Confidence: 0.8, // Cell balance analysis
		})
	}
	
	return triples
}
