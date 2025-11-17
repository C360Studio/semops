package rules

import (
	"fmt"
	"sync"
	"time"

	message "github.com/c360/semstreams/message"
	gtypes "github.com/c360/semstreams/types/graph"
	"github.com/c360/semops/pkg/processors/mavlink/constants"
	roboticspayloads "github.com/c360/semops/pkg/processors/mavlink/payloads"
)

// BatteryAlert represents a battery-related alert or warning
type BatteryAlert struct {
	SystemID       uint8          `json:"system_id"`
	BatteryID      uint8          `json:"battery_id"`
	AlertType      string         `json:"alert_type"` // "low_voltage", "critical_level", "temperature", "imbalance"
	Severity       string         `json:"severity"`   // "info", "warning", "critical", "emergency"
	Message        string         `json:"message"`
	CurrentValue   float64        `json:"current_value"`
	ThresholdValue float64        `json:"threshold_value"`
	Action         string         `json:"action"` // "warn", "rtl", "land", "emergency_land"
	Timestamp      time.Time      `json:"timestamp"`
	Properties     map[string]any `json:"properties,omitempty"`
}

// BatteryStatus tracks battery state over time for trend analysis
type BatteryStatus struct {
	SystemID          uint8     `json:"system_id"`
	BatteryID         uint8     `json:"battery_id"`
	LastLevel         int8      `json:"last_level"`
	LastVoltage       float64   `json:"last_voltage"`
	LastTemperature   float64   `json:"last_temperature"`
	LastUpdate        time.Time `json:"last_update"`
	TrendLevel        float64   `json:"trend_level"`     // Battery level change per minute
	TrendVoltage      float64   `json:"trend_voltage"`   // Voltage change per minute
	LowLevelCount     int       `json:"low_level_count"` // Number of consecutive low readings
	CriticalCount     int       `json:"critical_count"`  // Number of consecutive critical readings
	LastAlertTime     time.Time `json:"last_alert_time"`
	ConsecutiveAlerts int       `json:"consecutive_alerts"`
}

// BatteryMonitorRule monitors battery status and generates alerts
type BatteryMonitorRule struct {
	mu                           sync.RWMutex              // Protects concurrent access to maps
	name                         string
	enabled                      bool
	lowBatteryThreshold          float64                   // Battery level threshold for low warning (%)
	criticalBatteryThreshold     float64                   // Battery level threshold for critical warning (%)
	emergencyBatteryThreshold    float64                   // Battery level threshold for emergency action (%)
	lowVoltageThreshold          float64                   // Voltage threshold per cell (V)
	criticalVoltageThreshold     float64                   // Critical voltage threshold per cell (V)
	highTemperatureThreshold     float64                   // High temperature threshold (°C)
	criticalTemperatureThreshold float64                   // Critical temperature threshold (°C)
	cellImbalanceThreshold       float64                   // Maximum allowed cell imbalance (mV)
	alertCooldownPeriod          time.Duration             // Minimum time between repeated alerts
	batteryStatus                map[string]*BatteryStatus // Map[systemID_batteryID] -> status
	batteryHistory               map[string][]BatteryAlert // Recent alerts per battery
	maxHistorySize               int                       // Maximum number of alerts to keep in history
}

// NewBatteryMonitorRule creates a new battery monitoring rule
func NewBatteryMonitorRule(name string) *BatteryMonitorRule {
	return &BatteryMonitorRule{
		name:                         name,
		enabled:                      true,
		lowBatteryThreshold:          constants.DEFAULT_LOW_BATTERY_THRESHOLD,
		criticalBatteryThreshold:     constants.DEFAULT_CRITICAL_BATTERY_THRESHOLD,
		emergencyBatteryThreshold:    constants.DEFAULT_EMERGENCY_BATTERY_THRESHOLD,
		lowVoltageThreshold:          3.3,   // 3.3V per cell for Li-ion
		criticalVoltageThreshold:     3.0,   // 3.0V per cell (critical)
		highTemperatureThreshold:     50.0,  // 50°C
		criticalTemperatureThreshold: 65.0,  // 65°C
		cellImbalanceThreshold:       200.0, // 200mV imbalance
		alertCooldownPeriod:          2 * time.Minute,
		batteryStatus:                make(map[string]*BatteryStatus),
		batteryHistory:               make(map[string][]BatteryAlert),
		maxHistorySize:               50,
	}
}

// Name returns the rule name
func (b *BatteryMonitorRule) Name() string {
	return b.name
}

// Subscribe returns the NATS subject pattern this rule monitors
func (b *BatteryMonitorRule) Subscribe() []string {
	// Subscribe to process.robotics.battery where RoboticsProcessor publishes
	return []string{"process.robotics.battery", "process.robotics.battery.>"}
}

// Evaluate checks if battery alerts are needed
func (b *BatteryMonitorRule) Evaluate(messages []message.Message) bool {
	if !b.enabled {
		return false
	}

	for _, msg := range messages {
		if battPayload, ok := msg.Payload().(*roboticspayloads.BatteryPayload); ok {
			if b.wouldGenerateAlert(battPayload) {
				return true
			}
		}
	}
	return false
}

// wouldGenerateAlert checks if a battery message would generate an alert without side effects
func (b *BatteryMonitorRule) wouldGenerateAlert(battery *roboticspayloads.BatteryPayload) bool {
	// Use payload directly (no wrapper layer)
	statusKey := fmt.Sprintf("%d_%d", battery.SystemID, battery.BatteryID)

	// Get or create battery status tracking (without updating state)
	status := b.getBatteryStatus(statusKey, battery.SystemID, battery.BatteryID)

	// Check cooldown period for this battery
	now := battery.Ts
	if !status.LastAlertTime.IsZero() && now.Sub(status.LastAlertTime) < b.alertCooldownPeriod {
		return false
	}

	// Create a temporary copy of status to simulate updates for trend calculation
	tempStatus := *status
	b.updateBatteryStatusTemp(&tempStatus, battery)

	// Check if any alert conditions would be met with temporary status
	return b.checkBatteryLevel(battery, &tempStatus, now) != nil ||
		b.checkBatteryVoltage(battery, &tempStatus, now) != nil ||
		b.checkBatteryTemperature(battery, &tempStatus, now) != nil ||
		b.checkCellImbalance(battery, &tempStatus, now) != nil ||
		b.checkRapidDischarge(battery, &tempStatus, now) != nil ||
		b.checkChargeState(battery, &tempStatus, now) != nil
}

// updateBatteryStatusTemp updates battery status temporarily for evaluation without side effects
func (b *BatteryMonitorRule) updateBatteryStatusTemp(status *BatteryStatus, battery *roboticspayloads.BatteryPayload) {
	// Use payload directly (no wrapper layer)
	now := battery.Ts
	timeDelta := now.Sub(status.LastUpdate).Minutes()

	if timeDelta > 0 && status.LastLevel >= 0 {
		// Calculate trends (change per minute)
		if battery.BatteryRemaining >= 0 {
			levelDelta := float64(battery.BatteryRemaining - status.LastLevel)
			status.TrendLevel = levelDelta / timeDelta
		}

		voltage := battery.GetTotalVoltage()
		if voltage > 0 && status.LastVoltage > 0 {
			voltageDelta := voltage - status.LastVoltage
			status.TrendVoltage = voltageDelta / timeDelta
		}
	}

	// Update consecutive counters for temporary evaluation
	if battery.BatteryRemaining >= 0 && battery.BatteryRemaining <= int8(b.lowBatteryThreshold) {
		status.LowLevelCount++
	} else {
		status.LowLevelCount = 0
	}

	if battery.BatteryRemaining >= 0 && battery.BatteryRemaining <= int8(b.criticalBatteryThreshold) {
		status.CriticalCount++
	} else {
		status.CriticalCount = 0
	}
}

// ExecuteEvents processes battery messages and returns graph events (new pattern)
func (b *BatteryMonitorRule) ExecuteEvents(messages []message.Message) ([]gtypes.Event, error) {
	if !b.enabled {
		return nil, nil
	}

	var events []gtypes.Event

	for _, msg := range messages {
		if battPayload, ok := msg.Payload().(*roboticspayloads.BatteryPayload); ok {
			// Create entity creation/update events
			entityEvents := b.createEntityEvents(battPayload)
			events = append(events, entityEvents...)

			// Create alert events if needed
			alert := b.processBatteryMessage(battPayload)
			if alert != nil {
				alertEvent := b.createAlertEvent(alert, battPayload)
				if alertEvent != nil {
					events = append(events, *alertEvent)
				}
			}
		}
	}

	return events, nil
}






// processBatteryMessage analyzes a battery payload for issues
func (b *BatteryMonitorRule) processBatteryMessage(battery *roboticspayloads.BatteryPayload) *BatteryAlert {
	// Use payload directly (no wrapper layer)
	statusKey := fmt.Sprintf("%d_%d", battery.SystemID, battery.BatteryID)

	// Get or create battery status tracking
	status := b.getBatteryStatus(statusKey, battery.SystemID, battery.BatteryID)

	// Update status with current reading
	b.updateBatteryStatus(status, battery)

	// Check for various battery issues
	alert := b.checkBatteryIssues(battery, status)

	// Update alert history if we generated an alert
	if alert != nil {
		b.addToHistory(statusKey, *alert)
		status.LastAlertTime = alert.Timestamp
		status.ConsecutiveAlerts++
	} else {
		// Reset consecutive alerts on healthy reading
		status.ConsecutiveAlerts = 0
	}

	return alert
}

// getBatteryStatus gets or creates battery status tracking
func (b *BatteryMonitorRule) getBatteryStatus(key string, systemID, batteryID uint8) *BatteryStatus {
	b.mu.RLock()
	if status, exists := b.batteryStatus[key]; exists {
		b.mu.RUnlock()
		return status
	}
	b.mu.RUnlock()

	// Need write lock to create new status
	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check pattern - another goroutine might have created it
	if status, exists := b.batteryStatus[key]; exists {
		return status
	}

	status := &BatteryStatus{
		SystemID:        systemID,
		BatteryID:       batteryID,
		LastLevel:       -1,
		LastVoltage:     0.0,
		LastTemperature: 0.0,
		LastUpdate:      time.Now(),
	}

	b.batteryStatus[key] = status
	return status
}

// updateBatteryStatus updates battery status with current reading
func (b *BatteryMonitorRule) updateBatteryStatus(status *BatteryStatus, battery *roboticspayloads.BatteryPayload) {
	// Use payload directly (no wrapper layer)
	now := battery.Ts
	timeDelta := now.Sub(status.LastUpdate).Minutes()

	if timeDelta > 0 && status.LastLevel >= 0 {
		// Calculate trends (change per minute)
		if battery.BatteryRemaining >= 0 {
			levelDelta := float64(battery.BatteryRemaining - status.LastLevel)
			status.TrendLevel = levelDelta / timeDelta
		}

		voltage := battery.GetTotalVoltage()
		if voltage > 0 && status.LastVoltage > 0 {
			voltageDelta := voltage - status.LastVoltage
			status.TrendVoltage = voltageDelta / timeDelta
		}
	}

	// Update current values
	if battery.BatteryRemaining >= 0 {
		status.LastLevel = battery.BatteryRemaining
	}
	status.LastVoltage = battery.GetTotalVoltage()
	status.LastTemperature = battery.GetTemperatureCelsius()
	status.LastUpdate = now

	// Update consecutive counters
	if battery.BatteryRemaining >= 0 && battery.BatteryRemaining <= int8(b.lowBatteryThreshold) {
		status.LowLevelCount++
	} else {
		status.LowLevelCount = 0
	}

	if battery.BatteryRemaining >= 0 && battery.BatteryRemaining <= int8(b.criticalBatteryThreshold) {
		status.CriticalCount++
	} else {
		status.CriticalCount = 0
	}
}

// checkBatteryIssues analyzes battery for various issues
func (b *BatteryMonitorRule) checkBatteryIssues(battery *roboticspayloads.BatteryPayload, status *BatteryStatus) *BatteryAlert {
	now := battery.Timestamp()

	// Check cooldown period for this battery
	if !status.LastAlertTime.IsZero() && now.Sub(status.LastAlertTime) < b.alertCooldownPeriod {
		return nil
	}

	// Check battery level alerts
	if alert := b.checkBatteryLevel(battery, status, now); alert != nil {
		return alert
	}

	// Check voltage alerts
	if alert := b.checkBatteryVoltage(battery, status, now); alert != nil {
		return alert
	}

	// Check temperature alerts
	if alert := b.checkBatteryTemperature(battery, status, now); alert != nil {
		return alert
	}

	// Check cell imbalance
	if alert := b.checkCellImbalance(battery, status, now); alert != nil {
		return alert
	}

	// Check rapid discharge
	if alert := b.checkRapidDischarge(battery, status, now); alert != nil {
		return alert
	}

	// Check charge state issues
	if alert := b.checkChargeState(battery, status, now); alert != nil {
		return alert
	}

	return nil
}

// checkBatteryLevel checks battery percentage level
func (b *BatteryMonitorRule) checkBatteryLevel(battery *roboticspayloads.BatteryPayload, status *BatteryStatus, now time.Time) *BatteryAlert {
	// Use payload directly (no wrapper layer)
	if battery.BatteryRemaining < 0 {
		return nil // No valid reading
	}

	level := float64(battery.BatteryRemaining)

	if level <= b.emergencyBatteryThreshold {
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "emergency_level",
			Severity:       "emergency",
			Message:        fmt.Sprintf("Battery %d emergency low: %.0f%% remaining", battery.BatteryID, level),
			CurrentValue:   level,
			ThresholdValue: b.emergencyBatteryThreshold,
			Action:         "emergency_land",
			Timestamp:      now,
			Properties: map[string]any{
				"voltage":           battery.GetTotalVoltage(),
				"time_remaining":    battery.TimeRemaining,
				"consecutive_count": status.CriticalCount,
			},
		}
	}

	if level <= b.criticalBatteryThreshold && status.CriticalCount >= 3 {
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "critical_level",
			Severity:       "critical",
			Message:        fmt.Sprintf("Battery %d critically low: %.0f%% remaining", battery.BatteryID, level),
			CurrentValue:   level,
			ThresholdValue: b.criticalBatteryThreshold,
			Action:         "rtl",
			Timestamp:      now,
			Properties: map[string]any{
				"voltage":           battery.GetTotalVoltage(),
				"time_remaining":    battery.TimeRemaining,
				"consecutive_count": status.CriticalCount,
			},
		}
	}

	if level <= b.lowBatteryThreshold && status.LowLevelCount >= 5 {
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "low_level",
			Severity:       "warning",
			Message:        fmt.Sprintf("Battery %d low: %.0f%% remaining", battery.BatteryID, level),
			CurrentValue:   level,
			ThresholdValue: b.lowBatteryThreshold,
			Action:         "warn",
			Timestamp:      now,
			Properties: map[string]any{
				"voltage":           battery.GetTotalVoltage(),
				"time_remaining":    battery.TimeRemaining,
				"consecutive_count": status.LowLevelCount,
			},
		}
	}

	return nil
}

// checkBatteryVoltage checks individual cell voltages
func (b *BatteryMonitorRule) checkBatteryVoltage(battery *roboticspayloads.BatteryPayload, _ *BatteryStatus, now time.Time) *BatteryAlert {
	if len(battery.Voltages) == 0 {
		return nil
	}

	for i, cellVoltage := range battery.Voltages {
		if cellVoltage == 0 {
			continue
		}

		volts := float64(cellVoltage) / 1000.0

		if volts <= b.criticalVoltageThreshold {
			return &BatteryAlert{
				SystemID:       battery.SystemID,
				BatteryID:      battery.BatteryID,
				AlertType:      "critical_voltage",
				Severity:       "critical",
				Message:        fmt.Sprintf("Battery %d cell %d critical voltage: %.2fV", battery.BatteryID, i, volts),
				CurrentValue:   volts,
				ThresholdValue: b.criticalVoltageThreshold,
				Action:         "land",
				Timestamp:      now,
				Properties: map[string]any{
					"cell_index":    i,
					"total_voltage": battery.GetTotalVoltage(),
					"cell_count":    len(battery.Voltages),
				},
			}
		}

		if volts <= b.lowVoltageThreshold {
			return &BatteryAlert{
				SystemID:       battery.SystemID,
				BatteryID:      battery.BatteryID,
				AlertType:      "low_voltage",
				Severity:       "warning",
				Message:        fmt.Sprintf("Battery %d cell %d low voltage: %.2fV", battery.BatteryID, i, volts),
				CurrentValue:   volts,
				ThresholdValue: b.lowVoltageThreshold,
				Action:         "warn",
				Timestamp:      now,
				Properties: map[string]any{
					"cell_index":    i,
					"total_voltage": battery.GetTotalVoltage(),
					"cell_count":    len(battery.Voltages),
				},
			}
		}
	}

	return nil
}

// checkBatteryTemperature checks battery temperature
func (b *BatteryMonitorRule) checkBatteryTemperature(battery *roboticspayloads.BatteryPayload, _ *BatteryStatus, now time.Time) *BatteryAlert {
	temp := battery.GetTemperatureCelsius()
	if temp <= 0 {
		return nil // No valid reading
	}

	if temp >= b.criticalTemperatureThreshold {
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "critical_temperature",
			Severity:       "critical",
			Message:        fmt.Sprintf("Battery %d critical temperature: %.1f°C", battery.BatteryID, temp),
			CurrentValue:   temp,
			ThresholdValue: b.criticalTemperatureThreshold,
			Action:         "land",
			Timestamp:      now,
			Properties: map[string]any{
				"battery_level": battery.BatteryRemaining,
				"voltage":       battery.GetTotalVoltage(),
			},
		}
	}

	if temp >= b.highTemperatureThreshold {
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "high_temperature",
			Severity:       "warning",
			Message:        fmt.Sprintf("Battery %d high temperature: %.1f°C", battery.BatteryID, temp),
			CurrentValue:   temp,
			ThresholdValue: b.highTemperatureThreshold,
			Action:         "warn",
			Timestamp:      now,
			Properties: map[string]any{
				"battery_level": battery.BatteryRemaining,
				"voltage":       battery.GetTotalVoltage(),
			},
		}
	}

	return nil
}

// checkCellImbalance checks for voltage imbalance between cells
func (b *BatteryMonitorRule) checkCellImbalance(battery *roboticspayloads.BatteryPayload, _ *BatteryStatus, now time.Time) *BatteryAlert {
	imbalance := battery.GetCellImbalance()
	if imbalance == 0 || len(battery.Voltages) < 2 {
		return nil
	}

	imbalanceVolts := float64(imbalance) / 1000.0
	thresholdVolts := b.cellImbalanceThreshold / 1000.0

	if imbalanceVolts >= thresholdVolts {
		severity := "warning"
		action := "warn"

		if imbalanceVolts >= thresholdVolts*2 {
			severity = "critical"
			action = "rtl"
		}

		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "cell_imbalance",
			Severity:       severity,
			Message:        fmt.Sprintf("Battery %d cell imbalance: %.0fmV", battery.BatteryID, float64(imbalance)),
			CurrentValue:   imbalanceVolts,
			ThresholdValue: thresholdVolts,
			Action:         action,
			Timestamp:      now,
			Properties: map[string]any{
				"imbalance_mv": imbalance,
				"cell_count":   len(battery.Voltages),
				"voltages":     battery.Voltages,
			},
		}
	}

	return nil
}

// checkRapidDischarge checks for unusually rapid battery discharge
func (b *BatteryMonitorRule) checkRapidDischarge(battery *roboticspayloads.BatteryPayload, status *BatteryStatus, now time.Time) *BatteryAlert {
	if status.TrendLevel == 0 || status.TrendLevel >= 0 {
		return nil // No discharge trend data or charging
	}

	// Alert if losing more than 5% per minute (very rapid discharge)
	if status.TrendLevel <= -5.0 {
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "rapid_discharge",
			Severity:       "critical",
			Message:        fmt.Sprintf("Battery %d rapid discharge: %.1f%%/min", battery.BatteryID, -status.TrendLevel),
			CurrentValue:   -status.TrendLevel,
			ThresholdValue: 5.0,
			Action:         "rtl",
			Timestamp:      now,
			Properties: map[string]any{
				"discharge_rate": status.TrendLevel,
				"voltage_trend":  status.TrendVoltage,
				"current_level":  battery.BatteryRemaining,
				"current_amps":   battery.GetCurrentAmps(),
				"power_watts":    battery.GetPowerWatts(),
			},
		}
	}

	return nil
}

// checkChargeState checks MAVLink charge state for issues
func (b *BatteryMonitorRule) checkChargeState(battery *roboticspayloads.BatteryPayload, _ *BatteryStatus, now time.Time) *BatteryAlert {
	switch battery.ChargeState {
	case constants.MavBatteryChargeStateFailed:
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "battery_failed",
			Severity:       "emergency",
			Message:        fmt.Sprintf("Battery %d failure detected", battery.BatteryID),
			CurrentValue:   float64(battery.ChargeState),
			ThresholdValue: float64(constants.MavBatteryChargeStateFailed),
			Action:         "emergency_land",
			Timestamp:      now,
			Properties: map[string]any{
				"charge_state":  constants.GetBatteryChargeStateName(battery.ChargeState),
				"battery_level": battery.BatteryRemaining,
				"voltage":       battery.GetTotalVoltage(),
			},
		}

	case constants.MavBatteryChargeStateUnhealthy:
		return &BatteryAlert{
			SystemID:       battery.SystemID,
			BatteryID:      battery.BatteryID,
			AlertType:      "battery_unhealthy",
			Severity:       "warning",
			Message:        fmt.Sprintf("Battery %d unhealthy", battery.BatteryID),
			CurrentValue:   float64(battery.ChargeState),
			ThresholdValue: float64(constants.MavBatteryChargeStateUnhealthy),
			Action:         "warn",
			Timestamp:      now,
			Properties: map[string]any{
				"charge_state":  constants.GetBatteryChargeStateName(battery.ChargeState),
				"battery_level": battery.BatteryRemaining,
				"voltage":       battery.GetTotalVoltage(),
			},
		}
	}

	return nil
}

// checkStaleData checks for stale battery data
func (b *BatteryMonitorRule) checkStaleData(systemID uint8, heartbeatTime time.Time) {
	// Remove stale battery status entries (no data for > 5 minutes)
	staleThreshold := heartbeatTime.Add(-5 * time.Minute)

	b.mu.Lock()
	defer b.mu.Unlock()

	for key, status := range b.batteryStatus {
		if status.SystemID == systemID && status.LastUpdate.Before(staleThreshold) {
			delete(b.batteryStatus, key)
		}
	}
}

// addToHistory adds alert to battery history
func (b *BatteryMonitorRule) addToHistory(key string, alert BatteryAlert) {
	b.mu.Lock()
	defer b.mu.Unlock()

	history := b.batteryHistory[key]
	history = append(history, alert)

	// Trim history to max size
	if len(history) > b.maxHistorySize {
		history = history[len(history)-b.maxHistorySize:]
	}

	b.batteryHistory[key] = history
}

// SetLowBatteryThreshold sets the low battery warning threshold percentage
func (b *BatteryMonitorRule) SetLowBatteryThreshold(threshold float64) {
	b.lowBatteryThreshold = threshold
}

// SetCriticalBatteryThreshold sets the critical battery warning threshold percentage
func (b *BatteryMonitorRule) SetCriticalBatteryThreshold(threshold float64) {
	b.criticalBatteryThreshold = threshold
}

// SetEmergencyBatteryThreshold sets the emergency battery action threshold percentage
func (b *BatteryMonitorRule) SetEmergencyBatteryThreshold(threshold float64) {
	b.emergencyBatteryThreshold = threshold
}

// SetVoltageThresholds sets the low and critical voltage thresholds per cell in volts
func (b *BatteryMonitorRule) SetVoltageThresholds(low, critical float64) {
	b.lowVoltageThreshold = low
	b.criticalVoltageThreshold = critical
}

// SetTemperatureThresholds sets the high and critical temperature thresholds in Celsius
func (b *BatteryMonitorRule) SetTemperatureThresholds(high, critical float64) {
	b.highTemperatureThreshold = high
	b.criticalTemperatureThreshold = critical
}

// SetCellImbalanceThreshold sets the maximum allowed cell voltage imbalance in millivolts
func (b *BatteryMonitorRule) SetCellImbalanceThreshold(threshold float64) {
	b.cellImbalanceThreshold = threshold
}

// SetAlertCooldown sets the minimum time between repeated alerts for the same battery
func (b *BatteryMonitorRule) SetAlertCooldown(duration time.Duration) {
	b.alertCooldownPeriod = duration
}

// Enable enables the battery monitoring rule
func (b *BatteryMonitorRule) Enable() {
	b.enabled = true
}

// Disable disables the battery monitoring rule
func (b *BatteryMonitorRule) Disable() {
	b.enabled = false
}

// IsEnabled returns whether the battery monitoring rule is enabled
func (b *BatteryMonitorRule) IsEnabled() bool {
	return b.enabled
}

// GetBatteryStatus returns current status for a battery
func (b *BatteryMonitorRule) GetBatteryStatus(systemID, batteryID uint8) *BatteryStatus {
	key := fmt.Sprintf("%d_%d", systemID, batteryID)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.batteryStatus[key]
}

// GetBatteryHistory returns alert history for a battery
func (b *BatteryMonitorRule) GetBatteryHistory(systemID, batteryID uint8) []BatteryAlert {
	key := fmt.Sprintf("%d_%d", systemID, batteryID)
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Return a copy to prevent external modification
	history := b.batteryHistory[key]
	if history == nil {
		return nil
	}
	result := make([]BatteryAlert, len(history))
	copy(result, history)
	return result
}

// AllBatteryStatus returns status for all tracked batteries
func (b *BatteryMonitorRule) AllBatteryStatus() map[string]*BatteryStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[string]*BatteryStatus)
	for k, v := range b.batteryStatus {
		result[k] = v
	}
	return result
}

// CreateEntityStates creates graph entity states for battery messages
func (b *BatteryMonitorRule) CreateEntityStates(messages []message.Message) ([]*gtypes.EntityState, error) {
	if !b.enabled {
		return nil, nil
	}

	var entityStates []*gtypes.EntityState

	for _, msg := range messages {
		if battPayload, ok := msg.Payload().(*roboticspayloads.BatteryPayload); ok {
			// Create drone entity state
			droneState := b.createDroneEntityState(battPayload)
			if droneState != nil {
				entityStates = append(entityStates, droneState)
			}

			// Create battery entity state
			batteryState := b.createBatteryEntityState(battPayload)
			if batteryState != nil {
				entityStates = append(entityStates, batteryState)
			}
		}
	}

	return entityStates, nil
}

// createDroneEntityState creates a drone entity state with battery information
func (b *BatteryMonitorRule) createDroneEntityState(battery *roboticspayloads.BatteryPayload) *gtypes.EntityState {
	droneID := fmt.Sprintf("drone_%03d", battery.SystemID)
	batteryID := fmt.Sprintf("battery_%03d_%d", battery.SystemID, battery.BatteryID)

	// Create drone entity state with minimal properties
	droneState := &gtypes.EntityState{
		Node: gtypes.NodeProperties{
			ID:   droneID,
			Type: "robotics.drone",
			Properties: map[string]any{
				"battery_percent": battery.BatteryRemaining,
				"battery_voltage": battery.GetTotalVoltage(),
				"battery_current": battery.GetCurrentAmps(),
				"battery_temp_c":  battery.GetTemperatureCelsius(),
			},
			Status: b.getBatteryEntityStatus(battery.BatteryRemaining),
		},
		Edges: []gtypes.Edge{
			{
				ToEntityID: batteryID,
				EdgeType:   "POWERED_BY",
				Weight:     1.0,
				Confidence: 1.0,
				Properties: map[string]any{
					"voltage_v":          battery.GetTotalVoltage(),
					"current_a":          battery.GetCurrentAmps(),
					"power_w":            battery.GetPowerWatts(),
					"battery_function":   battery.BatteryFunction,
					"primary_power":      battery.BatteryFunction == 0, // 0 typically means main battery
					"charge_percent":     battery.BatteryRemaining,
					"time_remaining_min": battery.GetTimeRemainingMinutes(),
				},
				CreatedAt: battery.Ts,
			},
		},
		Version:   1,
		UpdatedAt: battery.Ts,
	}

	return droneState
}

// createBatteryEntityState creates a battery entity state
func (b *BatteryMonitorRule) createBatteryEntityState(battery *roboticspayloads.BatteryPayload) *gtypes.EntityState {
	batteryID := fmt.Sprintf("battery_%03d_%d", battery.SystemID, battery.BatteryID)

	batteryState := &gtypes.EntityState{
		Node: gtypes.NodeProperties{
			ID:   batteryID,
			Type: "robotics.battery",
			Properties: map[string]any{
				"system_id":           battery.SystemID,
				"battery_id":          battery.BatteryID,
				"charge_percent":      battery.BatteryRemaining,
				"voltage_v":           battery.GetTotalVoltage(),
				"current_a":           battery.GetCurrentAmps(),
				"power_w":             battery.GetPowerWatts(),
				"temperature_c":       battery.GetTemperatureCelsius(),
				"cell_count":          len(battery.Voltages),
				"cell_imbalance_mv":   battery.GetCellImbalance(),
				"charge_state":        battery.GetChargeStateName(),
				"time_remaining_min":  battery.GetTimeRemainingMinutes(),
				"energy_consumed_wh":  battery.GetEnergyConsumedWh(),
				"is_healthy":          battery.IsHealthy(),
				"is_critical":         battery.IsCritical(),
				"is_charging":         battery.IsCharging(),
				"alert_severity":      battery.GetAlertSeverity(),
			},
			Status: b.getBatteryEntityStatus(battery.BatteryRemaining),
		},
		Edges:     []gtypes.Edge{},
		Version:   1,
		UpdatedAt: battery.Ts,
	}

	return batteryState
}

// getBatteryEntityStatus determines entity status from battery level
func (b *BatteryMonitorRule) getBatteryEntityStatus(percent int8) gtypes.EntityStatus {
	if percent < 0 {
		return gtypes.StatusUnknown
	}
	
	switch {
	case percent <= int8(b.emergencyBatteryThreshold):
		return gtypes.StatusEmergency
	case percent <= int8(b.criticalBatteryThreshold):
		return gtypes.StatusCritical
	case percent <= int8(b.lowBatteryThreshold):
		return gtypes.StatusWarning
	default:
		return gtypes.StatusActive
	}
}

// createEntityEvents creates graph events for entity creation/updates from battery data
func (b *BatteryMonitorRule) createEntityEvents(battery *roboticspayloads.BatteryPayload) []gtypes.Event {
	var events []gtypes.Event
	now := battery.Ts
	
	// Create metadata for events
	metadata := gtypes.EventMetadata{
		RuleName:  b.name,
		Timestamp: now,
		Source:    "battery_monitor_rule",
		Reason:    "Battery telemetry update",
		Version:   "1.0.0",
	}
	
	// Create/update drone entity
	droneID := fmt.Sprintf("drone_%03d", battery.SystemID)
	droneProperties := map[string]any{
		"type":            "robotics.drone",
		"battery_percent": battery.BatteryRemaining,
		"battery_voltage": battery.GetTotalVoltage(),
		"battery_current": battery.GetCurrentAmps(),
		"battery_temp_c":  battery.GetTemperatureCelsius(),
		"status":          b.getBatteryEntityStatus(battery.BatteryRemaining),
	}
	
	droneEvent := gtypes.NewEntityUpdateEvent(droneID, droneProperties, metadata)
	droneEvent.Confidence = 1.0
	events = append(events, *droneEvent)
	
	// Create/update battery entity
	batteryID := fmt.Sprintf("battery_%03d_%d", battery.SystemID, battery.BatteryID)
	batteryProperties := map[string]any{
		"type":               "robotics.battery",
		"system_id":          battery.SystemID,
		"battery_id":         battery.BatteryID,
		"charge_percent":     battery.BatteryRemaining,
		"voltage_v":          battery.GetTotalVoltage(),
		"current_a":          battery.GetCurrentAmps(),
		"power_w":            battery.GetPowerWatts(),
		"temperature_c":      battery.GetTemperatureCelsius(),
		"cell_count":         len(battery.Voltages),
		"cell_imbalance_mv":  battery.GetCellImbalance(),
		"charge_state":       battery.GetChargeStateName(),
		"time_remaining_min": battery.GetTimeRemainingMinutes(),
		"energy_consumed_wh": battery.GetEnergyConsumedWh(),
		"is_healthy":         battery.IsHealthy(),
		"is_critical":        battery.IsCritical(),
		"is_charging":        battery.IsCharging(),
		"alert_severity":     battery.GetAlertSeverity(),
		"status":             b.getBatteryEntityStatus(battery.BatteryRemaining),
	}
	
	batteryEvent := gtypes.NewEntityUpdateEvent(batteryID, batteryProperties, metadata)
	batteryEvent.Confidence = 1.0
	events = append(events, *batteryEvent)
	
	// Create relationship between drone and battery
	relationshipEvent := gtypes.NewRelationshipCreateEvent(droneID, batteryID, "POWERED_BY", metadata)
	relationshipEvent.Properties["voltage_v"] = battery.GetTotalVoltage()
	relationshipEvent.Properties["current_a"] = battery.GetCurrentAmps()
	relationshipEvent.Properties["power_w"] = battery.GetPowerWatts()
	relationshipEvent.Properties["battery_function"] = battery.BatteryFunction
	relationshipEvent.Properties["primary_power"] = battery.BatteryFunction == 0
	relationshipEvent.Properties["charge_percent"] = battery.BatteryRemaining
	relationshipEvent.Properties["time_remaining_min"] = battery.GetTimeRemainingMinutes()
	events = append(events, *relationshipEvent)
	
	return events
}

// createAlertEvent creates a graph event for battery alerts
func (b *BatteryMonitorRule) createAlertEvent(alert *BatteryAlert, _ *roboticspayloads.BatteryPayload) *gtypes.Event {
	metadata := gtypes.EventMetadata{
		RuleName:  b.name,
		Timestamp: alert.Timestamp,
		Source:    "battery_monitor_rule",
		Reason:    fmt.Sprintf("Battery alert: %s", alert.Message),
		Version:   "1.0.0",
	}
	
	// Create alert properties
	alertProperties := map[string]any{
		"alert_type":       alert.AlertType,
		"severity":         alert.Severity,
		"message":          alert.Message,
		"current_value":    alert.CurrentValue,
		"threshold_value":  alert.ThresholdValue,
		"action":           alert.Action,
		"system_id":        alert.SystemID,
		"battery_id":       alert.BatteryID,
		"source_entity":    fmt.Sprintf("drone_%03d", alert.SystemID),
		"target_entity":    fmt.Sprintf("battery_%03d_%d", alert.SystemID, alert.BatteryID),
	}
	
	// Add custom properties from alert
	for k, v := range alert.Properties {
		alertProperties[k] = v
	}
	
	// Create alert entity event
	return gtypes.NewAlertEvent(alert.AlertType, fmt.Sprintf("drone_%03d", alert.SystemID), alertProperties, metadata)
}

// String returns string representation of the rule
func (b *BatteryMonitorRule) String() string {
	return fmt.Sprintf("BatteryMonitorRule[%s enabled=%t low=%.1f%% critical=%.1f%% emergency=%.1f%% tracked=%d]",
		b.name, b.enabled, b.lowBatteryThreshold, b.criticalBatteryThreshold,
		b.emergencyBatteryThreshold, len(b.batteryStatus))
}
