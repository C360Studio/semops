//go:build ignore
// +build ignore

package parser

import (
	"testing"

	"github.com/c360/semops/pkg/processors/mavlink/testing/mavlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratorParserIntegration verifies that the MAVLink generator and parser work together
func TestGeneratorParserIntegration(t *testing.T) {
	// Create generator
	gen := mavlink.NewGenerator(1, 1) // systemID=1, componentID=1

	// Create parser
	parser := NewMAVLinkParser()

	t.Run("heartbeat message round-trip", func(t *testing.T) {
		// Generate a heartbeat message
		heartbeatMsg := mavlink.HeartbeatMessage{
			VehicleType:    2,  // MAV_TYPE_QUADROTOR
			Autopilot:      3,  // MAV_AUTOPILOT_ARDUPILOTMEGA
			BaseMode:       81, // ARMED + MANUAL
			CustomMode:     0,
			SystemStatus:   4, // MAV_STATE_ACTIVE
			MavlinkVersion: 3, // MAVLink v1 (for v2 compatible frame)
		}
		heartbeatData, err := gen.GenerateHeartbeat(heartbeatMsg)
		require.NoError(t, err, "generator should produce heartbeat data")
		require.NotNil(t, heartbeatData, "heartbeat data should not be nil")
		require.NotEmpty(t, heartbeatData, "heartbeat data should not be empty")

		// Parse the generated message
		messages, err := parser.Parse(heartbeatData)
		require.NoError(t, err, "parser should handle generator output")
		require.Len(t, messages, 1, "should parse exactly one heartbeat message")

		// Verify the parsed message
		msg := messages[0]
		assert.Equal(t, uint32(0), msg.MessageID, "heartbeat message ID should be 0")
		assert.Equal(t, uint8(1), msg.SystemID, "system ID should match generator")
		assert.Equal(t, uint8(1), msg.ComponentID, "component ID should match generator")
	})

	t.Run("position message round-trip", func(t *testing.T) {
		// Generate a position message
		posMsg := mavlink.PositionMessage{
			TimeBootMs:  1000,
			Lat:         int32(47.3977 * 1e7), // Seattle area
			Lon:         int32(-122.0316 * 1e7),
			Alt:         100000, // 100m MSL in mm
			RelativeAlt: 50000,  // 50m AGL in mm
			Vx:          1050,   // 10.5 m/s in cm/s
			Vy:          -230,   // -2.3 m/s
			Vz:          10,     // 0.1 m/s
			Hdg:         9000,   // 90 degrees in cdeg
		}
		posData, err := gen.GenerateGlobalPosition(posMsg)
		require.NoError(t, err, "generator should produce position data")
		require.NotNil(t, posData, "position data should not be nil")
		require.NotEmpty(t, posData, "position data should not be empty")

		// Parse the generated message
		messages, err := parser.Parse(posData)
		require.NoError(t, err, "parser should handle position data")
		require.Len(t, messages, 1, "should parse exactly one position message")

		// Verify the parsed message
		msg := messages[0]
		assert.Equal(t, uint32(33), msg.MessageID, "position message ID should be 33")
		assert.Equal(t, uint8(1), msg.SystemID, "system ID should match generator")
		assert.Equal(t, uint8(1), msg.ComponentID, "component ID should match generator")
	})

	t.Run("attitude message round-trip", func(t *testing.T) {
		// Generate an attitude message
		attMsg := mavlink.AttitudeMessage{
			TimeBootMs: 2000,
			Roll:       0.1,
			Pitch:      0.2,
			Yaw:        0.3,
			Rollspeed:  0.01,
			Pitchspeed: 0.02,
			Yawspeed:   0.03,
		}
		attData, err := gen.GenerateAttitude(attMsg)
		require.NoError(t, err, "generator should produce attitude data")
		require.NotNil(t, attData, "attitude data should not be nil")
		require.NotEmpty(t, attData, "attitude data should not be empty")

		// Parse the generated message
		messages, err := parser.Parse(attData)
		require.NoError(t, err, "parser should handle attitude data")
		require.Len(t, messages, 1, "should parse exactly one attitude message")

		// Verify the parsed message
		msg := messages[0]
		assert.Equal(t, uint32(30), msg.MessageID, "attitude message ID should be 30")
		assert.Equal(t, uint8(1), msg.SystemID, "system ID should match generator")
		assert.Equal(t, uint8(1), msg.ComponentID, "component ID should match generator")
	})

	t.Run("battery status round-trip", func(t *testing.T) {
		// Generate a battery status message
		battMsg := mavlink.BatteryMessage{
			BatteryID:        0,
			BatteryFunction:  0,
			BatteryType:      0,
			Temperature:      2500, // 25°C in cdegC
			CurrentBattery:   -200, // -2A discharge in 10*mA
			CurrentConsumed:  1000, // 1000 mAh
			EnergyConsumed:   3600, // 3600 hJ
			BatteryRemaining: 75,   // 75%
		}
		// Set some cell voltages
		for i := 0; i < 4; i++ {
			battMsg.Voltages[i] = 3700 // 3.7V per cell in mV
		}

		battData, err := gen.GenerateBatteryStatus(battMsg)
		require.NoError(t, err, "generator should produce battery data")
		require.NotNil(t, battData, "battery data should not be nil")
		require.NotEmpty(t, battData, "battery data should not be empty")

		// Parse the generated message
		messages, err := parser.Parse(battData)
		require.NoError(t, err, "parser should handle battery data")
		require.Len(t, messages, 1, "should parse exactly one battery message")

		// Verify the parsed message
		msg := messages[0]
		assert.Equal(t, uint32(147), msg.MessageID, "battery message ID should be 147")
		assert.Equal(t, uint8(1), msg.SystemID, "system ID should match generator")
		assert.Equal(t, uint8(1), msg.ComponentID, "component ID should match generator")
	})

	t.Run("multiple messages in buffer", func(t *testing.T) {
		// Generate multiple messages
		heartbeat, err := gen.GenerateHeartbeat(mavlink.HeartbeatMessage{
			VehicleType:    2,
			Autopilot:      3,
			BaseMode:       81,
			CustomMode:     0,
			SystemStatus:   4,
			MavlinkVersion: 3,
		})
		require.NoError(t, err)

		position, err := gen.GenerateGlobalPosition(mavlink.PositionMessage{
			TimeBootMs:  1000,
			Lat:         int32(47.3977 * 1e7),
			Lon:         int32(-122.0316 * 1e7),
			Alt:         100000,
			RelativeAlt: 50000,
			Vx:          0,
			Vy:          0,
			Vz:          0,
			Hdg:         0,
		})
		require.NoError(t, err)

		attitude, err := gen.GenerateAttitude(mavlink.AttitudeMessage{
			TimeBootMs: 1000,
			Roll:       0.1,
			Pitch:      0.2,
			Yaw:        0.3,
			Rollspeed:  0,
			Pitchspeed: 0,
			Yawspeed:   0,
		})
		require.NoError(t, err)

		battery, err := gen.GenerateBatteryStatus(mavlink.BatteryMessage{
			BatteryRemaining: 50,
		})
		require.NoError(t, err)

		// Combine all messages into a single buffer
		combined := append(heartbeat, position...)
		combined = append(combined, attitude...)
		combined = append(combined, battery...)

		// Parse all messages at once
		messages, err := parser.Parse(combined)
		require.NoError(t, err, "parser should handle multiple messages")
		assert.Len(t, messages, 4, "should parse all 4 messages")

		// Verify message IDs
		messageIDs := make([]uint32, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.MessageID
		}
		assert.Contains(t, messageIDs, uint32(0), "should have heartbeat")
		assert.Contains(t, messageIDs, uint32(33), "should have position")
		assert.Contains(t, messageIDs, uint32(30), "should have attitude")
		assert.Contains(t, messageIDs, uint32(147), "should have battery")
	})

	t.Run("streaming messages", func(t *testing.T) {
		// Simulate streaming data by feeding messages in chunks
		gen := mavlink.NewGenerator(2, 3)
		parser := NewMAVLinkParser()

		// Generate a large message that might be split
		position, err := gen.GenerateGlobalPosition(mavlink.PositionMessage{
			TimeBootMs:  5000,
			Lat:         int32(47.3977 * 1e7),
			Lon:         int32(-122.0316 * 1e7),
			Alt:         100000,
			RelativeAlt: 50000,
			Vx:          100,
			Vy:          200,
			Vz:          300,
			Hdg:         18000,
		})
		require.NoError(t, err)

		// Feed it in small chunks to simulate streaming
		chunkSize := 10
		totalParsed := 0

		for i := 0; i < len(position); i += chunkSize {
			end := i + chunkSize
			if end > len(position) {
				end = len(position)
			}

			chunk := position[i:end]
			messages, err := parser.Parse(chunk)
			require.NoError(t, err, "parser should handle chunked data")
			totalParsed += len(messages)
		}

		// Should eventually parse the complete message
		assert.Equal(t, 1, totalParsed, "should parse exactly one message from chunks")
	})

	t.Run("corrupted data recovery", func(t *testing.T) {
		parser := NewMAVLinkParser()

		// Valid heartbeat
		heartbeat1, err := gen.GenerateHeartbeat(mavlink.HeartbeatMessage{
			VehicleType:    2,
			Autopilot:      3,
			BaseMode:       81,
			CustomMode:     0,
			SystemStatus:   4,
			MavlinkVersion: 3,
		})
		require.NoError(t, err)

		// Corrupt data (random bytes)
		corrupt := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

		// Another valid heartbeat
		heartbeat2, err := gen.GenerateHeartbeat(mavlink.HeartbeatMessage{
			VehicleType:    2,
			Autopilot:      3,
			BaseMode:       81,
			CustomMode:     0,
			SystemStatus:   4,
			MavlinkVersion: 3,
		})
		require.NoError(t, err)

		// Combine: valid + corrupt + valid
		combined := append(heartbeat1, corrupt...)
		combined = append(combined, heartbeat2...)

		// Parser should recover and parse valid messages
		messages, err := parser.Parse(combined)
		require.NoError(t, err, "parser should handle corrupted data")

		// Should parse at least one valid message (possibly both)
		assert.GreaterOrEqual(t, len(messages), 1, "should parse at least one valid message despite corruption")

		// All parsed messages should be valid heartbeats
		for _, msg := range messages {
			assert.Equal(t, uint32(0), msg.MessageID, "recovered messages should be heartbeats")
		}
	})

	t.Run("message sequence generator", func(t *testing.T) {
		// Test the realistic message sequence generator
		seq := mavlink.NewMessageSequence(3, 4)
		parser := NewMAVLinkParser()

		// Generate realistic heartbeat
		heartbeat, err := seq.GenerateRealisticHeartbeat(2) // quadrotor
		require.NoError(t, err)

		messages, err := parser.Parse(heartbeat)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, uint32(0), messages[0].MessageID)

		// Generate realistic battery status
		battery, err := seq.GenerateRealisticBatteryStatus(85)
		require.NoError(t, err)

		messages, err = parser.Parse(battery)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, uint32(147), messages[0].MessageID)

		// Generate realistic position
		position, err := seq.GenerateRealisticPosition(47.3977, -122.0316, 100000)
		require.NoError(t, err)

		messages, err = parser.Parse(position)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, uint32(33), messages[0].MessageID)

		// Generate realistic attitude
		attitude, err := seq.GenerateRealisticAttitude(0.1, 0.2, 0.3)
		require.NoError(t, err)

		messages, err = parser.Parse(attitude)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, uint32(30), messages[0].MessageID)
	})
}
