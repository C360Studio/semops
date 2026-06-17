//go:build ignore
// +build ignore

package payloads

import (
	"testing"
	"time"

	"github.com/c360/semops/pkg/processors/mavlink/constants"
	"github.com/c360/semstreams/message"
	"github.com/c360/semstreams/vocabulary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatteryPayload_BasicFunctionality(t *testing.T) {
	timestamp := time.Now()
	systemID := uint8(1)
	batteryID := uint8(0)

	payload := NewBatteryPayload(systemID, batteryID, timestamp)

	// Set battery state
	payload.BatteryRemaining = 75
	payload.CurrentConsumed = 1500
	payload.EnergyConsumed = 10000
	payload.Temperature = 3500 // 35 degrees C
	payload.Voltages = []uint16{3700, 3750, 3725, 3740}
	payload.CurrentBattery = 2500 // 25 amps
	payload.BatteryFunction = 0   // Main function
	payload.BatteryType = 0       // Default type
	payload.ChargeState = constants.MavBatteryChargeStateOk

	t.Run("Graphable interface - new simplified version", func(t *testing.T) {
		graphable, ok := any(payload).(message.Graphable)
		require.True(t, ok, "BatteryPayload should implement Graphable")

		// Test EntityID
		entityID := graphable.EntityID()
		assert.Equal(t, "c360.platform1.robotics.mav1.battery.0", entityID)

		// Test Triples
		triples := graphable.Triples()
		require.NotEmpty(t, triples, "BatteryPayload should provide triples")

		// Check for battery level triple
		var foundBatteryLevel bool
		for _, triple := range triples {
			if triple.Predicate == vocabulary.ROBOTICS_BATTERY_LEVEL {
				foundBatteryLevel = true
				assert.Equal(t, float64(75), triple.Object) // BatteryRemaining is 75
				break
			}
		}
		assert.True(t, foundBatteryLevel, "Should have battery level triple")
	})

	t.Run("Timeable", func(t *testing.T) {
		timeable, ok := any(payload).(message.Timeable)
		require.True(t, ok, "BatteryPayload should implement Timeable")

		assert.Equal(t, timestamp, timeable.Timestamp())

		// SetTimestamp not part of Timeable interface - just check we can read it
		assert.Equal(t, timestamp, payload.Ts)
	})

	t.Run("JSONMarshaling", func(t *testing.T) {
		data, err := payload.MarshalJSON()
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal into new payload
		newPayload := &BatteryPayload{}
		err = newPayload.UnmarshalJSON(data)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, payload.SystemID, newPayload.SystemID)
		assert.Equal(t, payload.BatteryID, newPayload.BatteryID)
		assert.Equal(t, payload.BatteryRemaining, newPayload.BatteryRemaining)
		assert.Equal(t, payload.CurrentConsumed, newPayload.CurrentConsumed)
		assert.Equal(t, payload.Temperature, newPayload.Temperature)
		assert.ElementsMatch(t, payload.Voltages, newPayload.Voltages)
		assert.Equal(t, payload.CurrentBattery, newPayload.CurrentBattery)
		assert.Equal(t, payload.ChargeState, newPayload.ChargeState)
	})
}

func TestBatteryPayload_HelperMethods(t *testing.T) {
	timestamp := time.Now()
	payload := NewBatteryPayload(1, 0, timestamp)

	// Set test values
	payload.Voltages = []uint16{3700, 3750, 3725, 3740} // 4 cells
	payload.CurrentBattery = 2500                       // 25 amps
	payload.Temperature = 3500                          // 35 degrees C
	payload.BatteryRemaining = 75

	t.Run("Voltage calculations", func(t *testing.T) {
		assert.InDelta(t, 14.915, payload.GetTotalVoltage(), 0.001)
	})

	t.Run("Current calculations", func(t *testing.T) {
		assert.InDelta(t, 25.0, payload.GetCurrentAmps(), 0.1)
	})

	t.Run("Power calculations", func(t *testing.T) {
		assert.InDelta(t, 372.875, payload.GetPowerWatts(), 0.001)
	})

	t.Run("Temperature calculations", func(t *testing.T) {
		assert.InDelta(t, 35.0, payload.GetTemperatureCelsius(), 0.1)
	})

	t.Run("Health status", func(t *testing.T) {
		// Healthy state
		payload.ChargeState = constants.MavBatteryChargeStateOk
		assert.True(t, payload.IsHealthy())
		assert.False(t, payload.IsCritical())

		// Critical state
		payload.ChargeState = constants.MavBatteryChargeStateCritical
		assert.False(t, payload.IsHealthy())
		assert.True(t, payload.IsCritical())

		// Emergency state
		payload.ChargeState = constants.MavBatteryChargeStateEmergency
		assert.False(t, payload.IsHealthy())
		assert.True(t, payload.IsCritical())
	})

	t.Run("Charging status", func(t *testing.T) {
		// Not charging
		payload.ChargeState = constants.MavBatteryChargeStateOk
		assert.False(t, payload.IsCharging())

		// Charging
		payload.ChargeState = constants.MavBatteryChargeStateCharging
		assert.True(t, payload.IsCharging())
	})
}

func TestBatteryPayload_EdgeCases(t *testing.T) {
	timestamp := time.Now()

	t.Run("Empty voltages array", func(t *testing.T) {
		payload := NewBatteryPayload(1, 0, timestamp)
		payload.Voltages = []uint16{}

		assert.Equal(t, 0.0, payload.GetTotalVoltage())
		assert.Equal(t, 0.0, payload.GetPowerWatts())
	})

	t.Run("Invalid temperature", func(t *testing.T) {
		payload := NewBatteryPayload(1, 0, timestamp)
		payload.Temperature = 0

		assert.Equal(t, 0.0, payload.GetTemperatureCelsius()) // Temperature 0 means not measured, returns 0.0
	})

	t.Run("Invalid current", func(t *testing.T) {
		payload := NewBatteryPayload(1, 0, timestamp)
		payload.CurrentBattery = -1

		assert.Equal(t, 0.0, payload.GetCurrentAmps())
	})

	t.Run("Unknown charge state", func(t *testing.T) {
		payload := NewBatteryPayload(1, 0, timestamp)
		payload.ChargeState = constants.MavBatteryChargeStateUndefined

		assert.False(t, payload.IsHealthy()) // Undefined state is not healthy
		assert.True(t, payload.IsCritical()) // Undefined state is critical (BatteryRemaining is 0 by default)
		assert.False(t, payload.IsCharging())
	})
}

func TestBatteryPayload_GraphableDetails(t *testing.T) {
	timestamp := time.Now()
	payload := NewBatteryPayload(1, 0, timestamp)

	// Set comprehensive battery state
	payload.BatteryRemaining = 75
	payload.CurrentConsumed = 1500
	payload.EnergyConsumed = 10000
	payload.Temperature = 3500 // 35 degrees C
	payload.Voltages = []uint16{3700, 3750, 3725, 3740}
	payload.CurrentBattery = 2500 // 25 amps
	payload.BatteryFunction = 0   // Main function
	payload.BatteryType = 0       // Default type
	payload.ChargeState = constants.MavBatteryChargeStateOk

	t.Run("EntityID and Triples provide battery data", func(t *testing.T) {
		graphable, ok := any(payload).(message.Graphable)
		require.True(t, ok, "BatteryPayload should implement Graphable")

		// Check EntityID
		entityID := graphable.EntityID()
		assert.Equal(t, "c360.platform1.robotics.mav1.battery.0", entityID)

		// Check Triples contain battery data
		triples := graphable.Triples()
		require.NotEmpty(t, triples, "Should return triples")

		// Verify key battery triples exist
		predicateChecks := map[string]bool{
			vocabulary.ROBOTICS_BATTERY_LEVEL:   false,
			vocabulary.ROBOTICS_BATTERY_VOLTAGE: false,
			vocabulary.ROBOTICS_BATTERY_CURRENT: false,
			vocabulary.ROBOTICS_SYSTEM_STATUS:   false,
		}

		for _, triple := range triples {
			if _, exists := predicateChecks[triple.Predicate]; exists {
				predicateChecks[triple.Predicate] = true
			}
		}

		// Verify all expected predicates were found
		for predicate, found := range predicateChecks {
			assert.True(t, found, "Missing expected predicate: %s", predicate)
		}
	})

	t.Run("Triples include relationship data", func(t *testing.T) {
		graphable, ok := any(payload).(message.Graphable)
		require.True(t, ok, "BatteryPayload should implement Graphable")

		triples := graphable.Triples()
		// Check that relationship triple exists - battery powers drone
		var foundPoweredBy bool
		for _, triple := range triples {
			if triple.Predicate == vocabulary.ROBOTICS_COMPONENT_POWERED {
				foundPoweredBy = true
				assert.Equal(t, "c360.platform1.robotics.mav1.drone.0", triple.Object) // Battery powers the drone
				break
			}
		}
		assert.True(t, foundPoweredBy, "Should have powers relationship triple")
	})

	t.Run("ResolutionHints for battery component", func(t *testing.T) {
		// Tests removed - old interface no longer used
	})
}

func TestBatteryPayload_TripleGeneration(t *testing.T) {
	timestamp := time.Now()
	payload := NewBatteryPayload(1, 0, timestamp)

	// Set battery values
	payload.BatteryRemaining = 75
	payload.Voltages = []uint16{3700, 3750, 3725, 3740}
	payload.CurrentBattery = 2500
	payload.Temperature = 3500
	payload.ChargeState = constants.MavBatteryChargeStateOk

	t.Run("Generates semantic triples", func(t *testing.T) {
		triples := payload.Triples()
		require.NotEmpty(t, triples, "Should generate triples")

		// Check that all triples have the correct subject
		entityID := payload.EntityID()
		for _, triple := range triples {
			assert.Equal(t, entityID, triple.Subject, "All triples should have entity ID as subject")
		}

		// Verify specific predicates exist
		predicates := make(map[string]bool)
		for _, triple := range triples {
			predicates[triple.Predicate] = true
		}

		// Check key predicates
		assert.True(t, predicates[vocabulary.ROBOTICS_BATTERY_LEVEL])
		assert.True(t, predicates[vocabulary.ROBOTICS_BATTERY_VOLTAGE])
		assert.True(t, predicates[vocabulary.ROBOTICS_BATTERY_CURRENT])
	})

	t.Run("Triple values match payload data", func(t *testing.T) {
		triples := payload.Triples()

		// Create a map for easier lookup
		tripleMap := make(map[string]interface{})
		for _, triple := range triples {
			tripleMap[triple.Predicate] = triple.Object
		}

		// Verify values
		if level, ok := tripleMap[vocabulary.ROBOTICS_BATTERY_LEVEL]; ok {
			assert.Equal(t, float64(75), level) // Triples store float64 not int8
		}

		if voltage, ok := tripleMap[vocabulary.ROBOTICS_BATTERY_VOLTAGE]; ok {
			assert.InDelta(t, 14.915, voltage, 0.001)
		}

		if current, ok := tripleMap[vocabulary.ROBOTICS_BATTERY_CURRENT]; ok {
			assert.InDelta(t, 25.0, current, 0.1)
		}
	})
}
