package mavlink

import (
	"testing"

	"github.com/c360/semops/pkg/processors/mavlink/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatteryAlertScenario tests the complete pipeline from generator to parsed fields
// This verifies that battery rules will receive correct battery_remaining values
func TestBatteryAlertScenario(t *testing.T) {
	// Test scenarios with different battery levels
	testCases := []struct {
		name                string
		batteryPercent      int8
		expectLowAlert      bool
		expectCriticalAlert bool
	}{
		{"Normal Battery", 75, false, false},
		{"Low Battery", 15, true, false},
		{"Critical Battery", 5, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create parser
			parser := parser.NewMAVLinkParser()

			// Generate realistic battery status
			seq := NewMessageSequence(1, 1)
			frame, err := seq.GenerateRealisticBatteryStatus(tc.batteryPercent)
			require.NoError(t, err)
			require.NotNil(t, frame)

			// Parse with MAVLink parser
			packets, err := parser.Parse(frame)
			require.NoError(t, err)
			require.Len(t, packets, 1)

			// Extract parsed battery_remaining field
			packet := packets[0]
			require.NotNil(t, packet.ParsedFields, "Should have parsed fields")
			
			batteryRemaining, ok := packet.ParsedFields["battery_remaining"].(int8)
			require.True(t, ok, "battery_remaining should be parsed as int8")

			// Verify the battery_remaining field is correctly parsed
			assert.Equal(t, tc.batteryPercent, batteryRemaining, 
				"Battery remaining should match generated value")
			
			// This is what battery rules would check (simplified)
			isLowBattery := batteryRemaining < 20
			isCriticalBattery := batteryRemaining < 10
			
			assert.Equal(t, tc.expectLowAlert, isLowBattery, 
				"Low battery alert should match expectation")
			assert.Equal(t, tc.expectCriticalAlert, isCriticalBattery,
				"Critical battery alert should match expectation")
			
			t.Logf("Battery %d%% -> Low: %v, Critical: %v", 
				tc.batteryPercent, isLowBattery, isCriticalBattery)
		})
	}
}

// TestBatteryRemainingPrecision tests that we preserve the precision of battery_remaining
func TestBatteryRemainingPrecision(t *testing.T) {
	parser := parser.NewMAVLinkParser()
	
	// Test various battery levels including edge cases
	testLevels := []int8{0, 1, 5, 10, 15, 20, 25, 50, 75, 99, 100, -1} // -1 = invalid/unknown
	
	for _, level := range testLevels {
		t.Run("Battery_Level", func(t *testing.T) {
			// Generate realistic battery status with specific level
			seq := NewMessageSequence(1, 1)
			frame, err := seq.GenerateRealisticBatteryStatus(level)
			require.NoError(t, err)
			
			// Parse
			packets, err := parser.Parse(frame)
			require.NoError(t, err)
			require.Len(t, packets, 1)
			
			// Extract battery_remaining field
			batteryRemaining, ok := packets[0].ParsedFields["battery_remaining"].(int8)
			require.True(t, ok, "battery_remaining should be parsed as int8")
			
			// Verify exact precision is maintained
			assert.Equal(t, level, batteryRemaining, 
				"Battery remaining should preserve exact value for level %d", level)
		})
	}
}