//go:build ignore
// +build ignore

package parser

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c360/semops/pkg/processors/mavlink/constants"
	"github.com/c360/semops/pkg/processors/mavlink/payloads"
	"github.com/c360/semops/pkg/processors/mavlink/testing/mavlink"
)

// TestMAVLinkParserIntegration tests the full pipeline from generation to parsing
func TestMAVLinkParserIntegration(t *testing.T) {
	parser := NewMAVLinkParser()
	generator := mavlink.NewGenerator(1, 1)

	t.Run("EndToEndPipeline", func(t *testing.T) {
		// Test all supported message types
		testCases := []struct {
			name         string
			generateFunc func() ([]byte, error)
			expectedType uint32
			validateFunc func(t *testing.T, packet *MAVLinkPacket)
		}{
			{
				name: "Heartbeat",
				generateFunc: func() ([]byte, error) {
					return generator.GenerateHeartbeat(mavlink.HeartbeatMessage{
						VehicleType:    constants.MavTypeQuadrotor,
						Autopilot:      constants.MavAutopilotArdupilotmega,
						BaseMode:       constants.MavModeFlagStabilizeEnabled,
						CustomMode:     12345,
						SystemStatus:   constants.MavStateActive,
						MavlinkVersion: constants.MavlinkV2,
					})
				},
				expectedType: constants.MavlinkMsgIdHeartbeat,
				validateFunc: func(t *testing.T, packet *MAVLinkPacket) {
					assert.Equal(t, uint8(constants.MavTypeQuadrotor), packet.ParsedFields["type"])
					assert.Equal(t, uint8(constants.MavAutopilotArdupilotmega), packet.ParsedFields["autopilot"])
					assert.Equal(t, uint8(constants.MavModeFlagStabilizeEnabled), packet.ParsedFields["base_mode"])
					assert.Equal(t, uint32(12345), packet.ParsedFields["custom_mode"])
					assert.Equal(t, uint8(constants.MavStateActive), packet.ParsedFields["system_status"])
					assert.Equal(t, uint8(constants.MavlinkV2), packet.ParsedFields["mavlink_version"])
				},
			},
			{
				name: "GlobalPosition",
				generateFunc: func() ([]byte, error) {
					return generator.GenerateGlobalPosition(mavlink.PositionMessage{
						TimeBootMs:  12345,
						Lat:         int32(37.7749 * 1e7), // San Francisco
						Lon:         int32(-122.4194 * 1e7),
						Alt:         1000 * 1000, // 1km in mm
						RelativeAlt: 500 * 1000,  // 500m in mm
						Vx:          100,         // 1m/s in cm/s
						Vy:          -50,         // -0.5m/s in cm/s
						Vz:          10,          // 0.1m/s in cm/s
						Hdg:         9000,        // 90 degrees in cdeg
					})
				},
				expectedType: constants.MavlinkMsgIdGlobalPositionInt,
				validateFunc: func(t *testing.T, packet *MAVLinkPacket) {
					assert.Equal(t, uint32(12345), packet.ParsedFields["time_boot_ms"])
					assert.Equal(t, int32(37.7749*1e7), packet.ParsedFields["lat"])
					assert.Equal(t, int32(-122.4194*1e7), packet.ParsedFields["lon"])
					assert.Equal(t, int32(1000*1000), packet.ParsedFields["alt"])
					assert.Equal(t, int32(500*1000), packet.ParsedFields["relative_alt"])
					assert.Equal(t, int16(100), packet.ParsedFields["vx"])
					assert.Equal(t, int16(-50), packet.ParsedFields["vy"])
					assert.Equal(t, int16(10), packet.ParsedFields["vz"])
					assert.Equal(t, uint16(9000), packet.ParsedFields["hdg"])
				},
			},
			{
				name: "Attitude",
				generateFunc: func() ([]byte, error) {
					return generator.GenerateAttitude(mavlink.AttitudeMessage{
						TimeBootMs: 54321,
						Roll:       0.1,
						Pitch:      -0.2,
						Yaw:        1.57, // 90 degrees
						Rollspeed:  0.01,
						Pitchspeed: -0.02,
						Yawspeed:   0.05,
					})
				},
				expectedType: constants.MavlinkMsgIdAttitude,
				validateFunc: func(t *testing.T, packet *MAVLinkPacket) {
					assert.Equal(t, uint32(54321), packet.ParsedFields["time_boot_ms"])
					assert.InDelta(t, 0.1, packet.ParsedFields["roll"], 0.001)
					assert.InDelta(t, -0.2, packet.ParsedFields["pitch"], 0.001)
					assert.InDelta(t, 1.57, packet.ParsedFields["yaw"], 0.001)
					assert.InDelta(t, 0.01, packet.ParsedFields["rollspeed"], 0.001)
					assert.InDelta(t, -0.02, packet.ParsedFields["pitchspeed"], 0.001)
					assert.InDelta(t, 0.05, packet.ParsedFields["yawspeed"], 0.001)
				},
			},
			{
				name: "BatteryStatus",
				generateFunc: func() ([]byte, error) {
					return generator.GenerateBatteryStatus(mavlink.BatteryMessage{
						BatteryID:        2,
						BatteryFunction:  1,
						BatteryType:      3,
						Temperature:      2500, // 25°C in centigrade
						Voltages:         [10]uint16{4200, 4180, 4190, 4170, 65535, 65535, 65535, 65535, 65535, 65535},
						CurrentBattery:   -1500,  // -15A discharge
						CurrentConsumed:  2000,   // 2Ah consumed
						EnergyConsumed:   288000, // 8Wh in hJ
						BatteryRemaining: 75,     // 75%
					})
				},
				expectedType: constants.MavlinkMsgIdBatteryStatus,
				validateFunc: func(t *testing.T, packet *MAVLinkPacket) {
					assert.Equal(t, uint8(2), packet.ParsedFields["id"])
					assert.Equal(t, uint8(1), packet.ParsedFields["battery_function"])
					assert.Equal(t, uint8(3), packet.ParsedFields["type"])
					assert.Equal(t, int16(2500), packet.ParsedFields["temperature"])

					voltages, ok := packet.ParsedFields["voltages"].([]uint16)
					require.True(t, ok, "Voltages should be []uint16")
					require.Len(t, voltages, 10, "Should have 10 voltage readings")
					assert.Equal(t, uint16(4200), voltages[0])
					assert.Equal(t, uint16(4180), voltages[1])
					assert.Equal(t, uint16(4190), voltages[2])
					assert.Equal(t, uint16(4170), voltages[3])
					assert.Equal(t, uint16(65535), voltages[4]) // Invalid cell

					assert.Equal(t, int16(-1500), packet.ParsedFields["current_battery"])
					assert.Equal(t, int32(2000), packet.ParsedFields["current_consumed"])
					assert.Equal(t, int32(288000), packet.ParsedFields["energy_consumed"])
					assert.Equal(t, int8(75), packet.ParsedFields["battery_remaining"])
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Generate message
				data, err := tc.generateFunc()
				require.NoError(t, err, "Failed to generate message")

				// Parse message
				packets, err := parser.Parse(data)
				require.NoError(t, err, "Failed to parse message")
				require.Len(t, packets, 1, "Expected exactly one packet")

				packet := packets[0]
				assert.Equal(t, tc.expectedType, packet.MessageID)
				assert.Equal(t, uint8(1), packet.SystemID)
				assert.Equal(t, uint8(1), packet.ComponentID)
				assert.Equal(t, uint8(constants.MavlinkV2), packet.Version)
				assert.NotNil(t, packet.ParsedFields, "ParsedFields should not be nil")

				// Run message-specific validation
				tc.validateFunc(t, packet)
			})
		}
	})
}

// TestBatteryStatusFieldOffsets verifies each field is at the correct byte offset
func TestBatteryStatusFieldOffsets(t *testing.T) {
	generator := mavlink.NewGenerator(1, 1)
	parser := NewMAVLinkParser()

	// Test battery message with specific values to verify field positions
	batteryMsg := mavlink.BatteryMessage{
		BatteryID:        0x12,                                                                                       // offset 0, should be at byte 0
		BatteryFunction:  0x34,                                                                                       // offset 1, should be at byte 1
		BatteryType:      0x56,                                                                                       // offset 2, should be at byte 2
		Temperature:      0x789A,                                                                                     // offset 3-4, should be at bytes 3-4
		Voltages:         [10]uint16{0xBCDE, 0x1122, 0x3344, 0x5566, 0x7788, 0x99AA, 0xBBCC, 0xDDEE, 0xFF00, 0x1234}, // offset 5-24
		CurrentBattery:   0x5678,                                                                                     // offset 25-26, should be at bytes 25-26
		CurrentConsumed:  0x12345678,                                                                                 // offset 27-30, should be at bytes 27-30
		EnergyConsumed:   0x1ABCDEF0,                                                                                 // offset 31-34, should be at bytes 31-34
		BatteryRemaining: 0x42,                                                                                       // offset 35, should be at byte 35
	}

	data, err := generator.GenerateBatteryStatus(batteryMsg)
	require.NoError(t, err)

	packets, err := parser.Parse(data)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	packet := packets[0]
	require.NotNil(t, packet.ParsedFields)

	t.Run("FieldOffsetVerification", func(t *testing.T) {
		// Verify payload structure directly
		payload := packet.Payload
		require.Len(t, payload, 36, "BATTERY_STATUS payload should be 36 bytes")

		// Test each field at its expected offset
		assert.Equal(t, uint8(0x12), payload[0], "BatteryID at offset 0")
		assert.Equal(t, uint8(0x34), payload[1], "BatteryFunction at offset 1")
		assert.Equal(t, uint8(0x56), payload[2], "BatteryType at offset 2")

		// Temperature at offset 3-4 (int16, little endian)
		tempBytes := payload[3:5]
		temp := binary.LittleEndian.Uint16(tempBytes)
		assert.Equal(t, uint16(0x789A), temp, "Temperature at offset 3-4")

		// Voltages array at offset 5-24 (10 x uint16, little endian)
		for i := 0; i < 10; i++ {
			offset := 5 + i*2
			voltageBytes := payload[offset : offset+2]
			voltage := binary.LittleEndian.Uint16(voltageBytes)
			expectedVoltages := []uint16{0xBCDE, 0x1122, 0x3344, 0x5566, 0x7788, 0x99AA, 0xBBCC, 0xDDEE, 0xFF00, 0x1234}
			assert.Equal(t, expectedVoltages[i], voltage, "Voltage[%d] at offset %d", i, offset)
		}

		// CurrentBattery at offset 25-26 (int16, little endian)
		currentBytes := payload[25:27]
		current := binary.LittleEndian.Uint16(currentBytes)
		assert.Equal(t, uint16(0x5678), current, "CurrentBattery at offset 25-26")

		// CurrentConsumed at offset 27-30 (int32, little endian)
		consumedBytes := payload[27:31]
		consumed := binary.LittleEndian.Uint32(consumedBytes)
		assert.Equal(t, uint32(0x12345678), consumed, "CurrentConsumed at offset 27-30")

		// EnergyConsumed at offset 31-34 (int32, little endian)
		energyBytes := payload[31:35]
		energy := binary.LittleEndian.Uint32(energyBytes)
		assert.Equal(t, uint32(0x1ABCDEF0), energy, "EnergyConsumed at offset 31-34")

		// BatteryRemaining at offset 35 (int8)
		assert.Equal(t, uint8(0x42), payload[35], "BatteryRemaining at offset 35")
	})

	t.Run("ParsedFieldsCorrectness", func(t *testing.T) {
		// Verify parsed fields match expected values
		assert.Equal(t, uint8(0x12), packet.ParsedFields["id"])
		assert.Equal(t, uint8(0x34), packet.ParsedFields["battery_function"])
		assert.Equal(t, uint8(0x56), packet.ParsedFields["type"])
		assert.Equal(t, int16(0x789A), packet.ParsedFields["temperature"])

		voltages, ok := packet.ParsedFields["voltages"].([]uint16)
		require.True(t, ok)
		require.Len(t, voltages, 10)
		expectedVoltages := []uint16{0xBCDE, 0x1122, 0x3344, 0x5566, 0x7788, 0x99AA, 0xBBCC, 0xDDEE, 0xFF00, 0x1234}
		for i, expected := range expectedVoltages {
			assert.Equal(t, expected, voltages[i], "Voltage[%d]", i)
		}

		assert.Equal(t, int16(0x5678), packet.ParsedFields["current_battery"])
		assert.Equal(t, int32(0x12345678), packet.ParsedFields["current_consumed"])
		assert.Equal(t, int32(0x1ABCDEF0), packet.ParsedFields["energy_consumed"])
		assert.Equal(t, int8(0x42), packet.ParsedFields["battery_remaining"])
	})
}

// TestHeartbeatParsing tests heartbeat message parsing with various configurations
func TestHeartbeatParsing(t *testing.T) {
	generator := mavlink.NewGenerator(255, 250)
	parser := NewMAVLinkParser()

	testCases := []struct {
		name string
		msg  mavlink.HeartbeatMessage
	}{
		{
			name: "QuadcopterArduPilot",
			msg: mavlink.HeartbeatMessage{
				VehicleType:    constants.MavTypeQuadrotor,
				Autopilot:      constants.MavAutopilotArdupilotmega,
				BaseMode:       constants.MavModeFlagStabilizeEnabled | constants.MavModeFlagManualInputEnabled,
				CustomMode:     uint32(100),
				SystemStatus:   constants.MavStateActive,
				MavlinkVersion: constants.MavlinkV2,
			},
		},
		{
			name: "GCSSystem",
			msg: mavlink.HeartbeatMessage{
				VehicleType:    constants.MavTypeGcs,
				Autopilot:      constants.MavAutopilotInvalid,
				BaseMode:       0,
				CustomMode:     uint32(0),
				SystemStatus:   constants.MavStateStandby,
				MavlinkVersion: constants.MavlinkV2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := generator.GenerateHeartbeat(tc.msg)
			require.NoError(t, err)

			packets, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, packets, 1)

			packet := packets[0]
			assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), packet.MessageID)
			assert.Equal(t, uint8(255), packet.SystemID)
			assert.Equal(t, uint8(250), packet.ComponentID)

			// Verify all fields are parsed correctly
			assert.Equal(t, tc.msg.VehicleType, packet.ParsedFields["type"])
			assert.Equal(t, tc.msg.Autopilot, packet.ParsedFields["autopilot"])
			assert.Equal(t, tc.msg.BaseMode, packet.ParsedFields["base_mode"])
			assert.Equal(t, tc.msg.CustomMode, packet.ParsedFields["custom_mode"])
			assert.Equal(t, tc.msg.SystemStatus, packet.ParsedFields["system_status"])
			assert.Equal(t, tc.msg.MavlinkVersion, packet.ParsedFields["mavlink_version"])
		})
	}
}

// TestPositionParsing tests GPS position message parsing
func TestPositionParsing(t *testing.T) {
	generator := mavlink.NewGenerator(1, 1)
	parser := NewMAVLinkParser()

	testCases := []struct {
		name string
		msg  mavlink.PositionMessage
	}{
		{
			name: "SanFrancisco",
			msg: mavlink.PositionMessage{
				TimeBootMs:  123456,
				Lat:         int32(37.7749 * 1e7),
				Lon:         int32(-122.4194 * 1e7),
				Alt:         100000, // 100m in mm
				RelativeAlt: 50000,  // 50m in mm
				Vx:          1000,   // 10 m/s north
				Vy:          -500,   // -5 m/s east
				Vz:          200,    // 2 m/s down
				Hdg:         27000,  // 270 degrees
			},
		},
		{
			name: "Equator",
			msg: mavlink.PositionMessage{
				TimeBootMs:  0,
				Lat:         0,
				Lon:         0,
				Alt:         0,
				RelativeAlt: 0,
				Vx:          0,
				Vy:          0,
				Vz:          0,
				Hdg:         0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := generator.GenerateGlobalPosition(tc.msg)
			require.NoError(t, err)

			packets, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, packets, 1)

			packet := packets[0]
			assert.Equal(t, uint32(constants.MavlinkMsgIdGlobalPositionInt), packet.MessageID)

			// Verify all position fields
			assert.Equal(t, tc.msg.TimeBootMs, packet.ParsedFields["time_boot_ms"])
			assert.Equal(t, tc.msg.Lat, packet.ParsedFields["lat"])
			assert.Equal(t, tc.msg.Lon, packet.ParsedFields["lon"])
			assert.Equal(t, tc.msg.Alt, packet.ParsedFields["alt"])
			assert.Equal(t, tc.msg.RelativeAlt, packet.ParsedFields["relative_alt"])
			assert.Equal(t, tc.msg.Vx, packet.ParsedFields["vx"])
			assert.Equal(t, tc.msg.Vy, packet.ParsedFields["vy"])
			assert.Equal(t, tc.msg.Vz, packet.ParsedFields["vz"])
			assert.Equal(t, tc.msg.Hdg, packet.ParsedFields["hdg"])
		})
	}
}

// TestAttitudeParsing tests attitude message parsing with various orientations
func TestAttitudeParsing(t *testing.T) {
	generator := mavlink.NewGenerator(42, 1)
	parser := NewMAVLinkParser()

	testCases := []struct {
		name string
		msg  mavlink.AttitudeMessage
	}{
		{
			name: "LevelFlight",
			msg: mavlink.AttitudeMessage{
				TimeBootMs: 987654,
				Roll:       0.0,
				Pitch:      0.0,
				Yaw:        0.0,
				Rollspeed:  0.0,
				Pitchspeed: 0.0,
				Yawspeed:   0.0,
			},
		},
		{
			name: "BankedTurn",
			msg: mavlink.AttitudeMessage{
				TimeBootMs: 1000000,
				Roll:       math.Pi / 6, // 30 degrees
				Pitch:      0.1,         // slight nose up
				Yaw:        math.Pi / 2, // 90 degrees (east)
				Rollspeed:  0.5,         // rolling right
				Pitchspeed: 0.1,         // pitching up slowly
				Yawspeed:   0.3,         // yawing right
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := generator.GenerateAttitude(tc.msg)
			require.NoError(t, err)

			packets, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, packets, 1)

			packet := packets[0]
			assert.Equal(t, uint32(constants.MavlinkMsgIdAttitude), packet.MessageID)
			assert.Equal(t, uint8(42), packet.SystemID)

			// Verify all attitude fields with appropriate precision
			assert.Equal(t, tc.msg.TimeBootMs, packet.ParsedFields["time_boot_ms"])
			assert.InDelta(t, tc.msg.Roll, packet.ParsedFields["roll"], 0.0001)
			assert.InDelta(t, tc.msg.Pitch, packet.ParsedFields["pitch"], 0.0001)
			assert.InDelta(t, tc.msg.Yaw, packet.ParsedFields["yaw"], 0.0001)
			assert.InDelta(t, tc.msg.Rollspeed, packet.ParsedFields["rollspeed"], 0.0001)
			assert.InDelta(t, tc.msg.Pitchspeed, packet.ParsedFields["pitchspeed"], 0.0001)
			assert.InDelta(t, tc.msg.Yawspeed, packet.ParsedFields["yawspeed"], 0.0001)
		})
	}
}

// TestBatteryEdgeCases tests battery parsing with edge cases
func TestBatteryEdgeCases(t *testing.T) {
	generator := mavlink.NewGenerator(1, 1)
	parser := NewMAVLinkParser()

	testCases := []struct {
		name            string
		batteryPercent  int8
		expectedPercent int8
	}{
		{"FullBattery", 100, 100},
		{"HalfBattery", 50, 50},
		{"LowBattery", 20, 20},
		{"CriticalBattery", 10, 10},
		{"EmptyBattery", 0, 0},
		{"InvalidBattery", -1, -1}, // -1 indicates invalid/unknown
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create realistic battery message
			cellVoltage := uint16(3300 + (4200-3300)*int(tc.batteryPercent)/100)
			if tc.batteryPercent < 0 {
				cellVoltage = 0 // Invalid voltage for invalid battery
			}

			batteryMsg := mavlink.BatteryMessage{
				BatteryID:        0,
				BatteryFunction:  0, // Main battery
				BatteryType:      1, // LiPo
				Temperature:      2500,
				CurrentBattery:   -1000, // 10A discharge
				CurrentConsumed:  500,
				EnergyConsumed:   72000,
				BatteryRemaining: tc.batteryPercent,
			}

			// Set cell voltages (4S battery)
			for i := 0; i < 4; i++ {
				batteryMsg.Voltages[i] = cellVoltage
			}
			// Mark unused cells as invalid
			for i := 4; i < 10; i++ {
				batteryMsg.Voltages[i] = 65535
			}

			data, err := generator.GenerateBatteryStatus(batteryMsg)
			require.NoError(t, err)

			packets, err := parser.Parse(data)
			require.NoError(t, err)
			require.Len(t, packets, 1)

			packet := packets[0]
			assert.Equal(t, uint32(constants.MavlinkMsgIdBatteryStatus), packet.MessageID)
			assert.Equal(t, tc.expectedPercent, packet.ParsedFields["battery_remaining"])

			// Verify cell voltages
			voltages, ok := packet.ParsedFields["voltages"].([]uint16)
			require.True(t, ok)
			for i := 0; i < 4; i++ {
				assert.Equal(t, cellVoltage, voltages[i], "Cell %d voltage", i)
			}
			for i := 4; i < 10; i++ {
				assert.Equal(t, uint16(65535), voltages[i], "Unused cell %d", i)
			}
		})
	}
}

// TestConvertToSemStreamsMessage tests the conversion of parsed MAVLink packets to SemStreams messages
func TestConvertToSemStreamsMessage(t *testing.T) {
	generator := mavlink.NewGenerator(1, 1)
	parser := NewMAVLinkParser()

	t.Run("BatteryConversion", func(t *testing.T) {
		// Generate and parse a battery message
		batteryMsg := mavlink.BatteryMessage{
			BatteryID:        1,
			BatteryFunction:  0,
			BatteryType:      1,
			Temperature:      2750,   // 27.5°C
			CurrentBattery:   -2000,  // 20A discharge
			CurrentConsumed:  1500,   // 1.5Ah consumed
			EnergyConsumed:   216000, // 6Wh in hJ
			BatteryRemaining: 45,     // 45%
		}

		// Set realistic cell voltages for 45% battery
		for i := 0; i < 4; i++ {
			batteryMsg.Voltages[i] = 3850 // 3.85V per cell (45% charge)
		}

		data, err := generator.GenerateBatteryStatus(batteryMsg)
		require.NoError(t, err)

		packets, err := parser.Parse(data)
		require.NoError(t, err)
		require.Len(t, packets, 1)

		// Convert to SemStreams message
		semMsg, err := parser.ConvertToSemStreamsMessage(packets[0])
		require.NoError(t, err)

		// Verify message type and payload
		assert.Equal(t, "robotics.battery.v1", semMsg.Type().String())

		batteryPayload, ok := semMsg.Payload().(*payloads.BatteryPayload)
		require.True(t, ok, "Payload should be BatteryPayload")

		assert.Equal(t, uint8(1), batteryPayload.SystemID)
		assert.Equal(t, uint8(1), batteryPayload.BatteryID)
		assert.Equal(t, int8(45), batteryPayload.BatteryRemaining)
		assert.Equal(t, int16(-2000), batteryPayload.CurrentBattery)
		assert.Equal(t, int32(1500), batteryPayload.CurrentConsumed)
		assert.Equal(t, int32(216000), batteryPayload.EnergyConsumed)
		assert.Equal(t, int16(2750), batteryPayload.Temperature)

		// Verify cell voltages (all 10 are included from MAVLink message)
		require.Len(t, batteryPayload.Voltages, 10)
		for i := 0; i < 4; i++ {
			assert.Equal(t, uint16(3850), batteryPayload.Voltages[i])
		}
		// Unused cells should be 0
		for i := 4; i < 10; i++ {
			assert.Equal(t, uint16(0), batteryPayload.Voltages[i])
		}
	})
}

// TestMultipleMessagesInBuffer tests parsing multiple messages from a single buffer
func TestMultipleMessagesInBuffer(t *testing.T) {
	generator := mavlink.NewGenerator(1, 1)
	parser := NewMAVLinkParser()

	// Generate multiple different message types
	heartbeatData, err := generator.GenerateHeartbeat(mavlink.HeartbeatMessage{
		VehicleType:    constants.MavTypeQuadrotor,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled,
		CustomMode:     0,
		SystemStatus:   constants.MavStateActive,
		MavlinkVersion: constants.MavlinkV2,
	})
	require.NoError(t, err)

	batteryData, err := generator.GenerateBatteryStatus(mavlink.BatteryMessage{
		BatteryID:        0,
		BatteryFunction:  0,
		BatteryType:      1,
		Temperature:      2500,
		BatteryRemaining: 80,
	})
	require.NoError(t, err)

	positionData, err := generator.GenerateGlobalPosition(mavlink.PositionMessage{
		TimeBootMs: 12345,
		Lat:        int32(37.7749 * 1e7),
		Lon:        int32(-122.4194 * 1e7),
		Alt:        100000,
		Hdg:        0,
	})
	require.NoError(t, err)

	// Concatenate all messages into one buffer
	combinedBuffer := append(append(heartbeatData, batteryData...), positionData...)

	// Parse all messages at once
	packets, err := parser.Parse(combinedBuffer)
	require.NoError(t, err)
	require.Len(t, packets, 3, "Should parse all three messages")

	// Verify message types in order
	assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), packets[0].MessageID)
	assert.Equal(t, uint32(constants.MavlinkMsgIdBatteryStatus), packets[1].MessageID)
	assert.Equal(t, uint32(constants.MavlinkMsgIdGlobalPositionInt), packets[2].MessageID)

	// Verify sequence numbers increment
	assert.Equal(t, uint8(1), packets[0].Sequence)
	assert.Equal(t, uint8(2), packets[1].Sequence)
	assert.Equal(t, uint8(3), packets[2].Sequence)
}

// TestCorruptedMessage tests parser behavior with corrupted data
func TestCorruptedMessage(t *testing.T) {
	generator := mavlink.NewGenerator(1, 1)
	parser := NewMAVLinkParser()

	// Generate a valid message first
	validData, err := generator.GenerateHeartbeat(mavlink.HeartbeatMessage{
		VehicleType:    constants.MavTypeQuadrotor,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled,
		CustomMode:     0,
		SystemStatus:   constants.MavStateActive,
		MavlinkVersion: constants.MavlinkV2,
	})
	require.NoError(t, err)

	t.Run("CorruptedChecksum", func(t *testing.T) {
		// Corrupt the checksum (last 2 bytes)
		corruptedData := make([]byte, len(validData))
		copy(corruptedData, validData)
		corruptedData[len(corruptedData)-1] ^= 0xFF // Flip all bits in high byte of checksum
		corruptedData[len(corruptedData)-2] ^= 0xFF // Flip all bits in low byte of checksum

		packets, err := parser.Parse(corruptedData)
		require.NoError(t, err) // Parser should not error, but should skip invalid packets
		assert.Empty(t, packets, "Should not return packets with corrupted checksums")

		stats := parser.GetStats()
		assert.Greater(t, stats.ChecksumErrors, uint64(0), "Should record checksum errors")
	})

	t.Run("RecoveryAfterCorruption", func(t *testing.T) {
		// Add some garbage data, then a valid message
		garbageData := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
		mixedData := append(garbageData, validData...)

		packets, err := parser.Parse(mixedData)
		require.NoError(t, err)
		require.Len(t, packets, 1, "Should recover and parse the valid message")

		packet := packets[0]
		assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), packet.MessageID)
		assert.Equal(t, uint8(constants.MavTypeQuadrotor), packet.ParsedFields["type"])
	})
}

// BenchmarkParserPerformance benchmarks the parser performance
func BenchmarkParserPerformance(t *testing.B) {
	generator := mavlink.NewGenerator(1, 1)

	// Pre-generate test data
	heartbeatData, _ := generator.GenerateHeartbeat(mavlink.HeartbeatMessage{
		VehicleType:    constants.MavTypeQuadrotor,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled,
		SystemStatus:   constants.MavStateActive,
		MavlinkVersion: constants.MavlinkV2,
	})

	batteryData, _ := generator.GenerateBatteryStatus(mavlink.BatteryMessage{
		BatteryID:        0,
		BatteryRemaining: 75,
		Temperature:      2500,
	})

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		parser := NewMAVLinkParser()

		// Parse heartbeat
		packets, err := parser.Parse(heartbeatData)
		if err != nil || len(packets) != 1 {
			t.Fatalf("Failed to parse heartbeat: %v", err)
		}

		// Parse battery
		packets, err = parser.Parse(batteryData)
		if err != nil || len(packets) != 1 {
			t.Fatalf("Failed to parse battery: %v", err)
		}
	}
}

// TestBinaryDumpOnFailure provides detailed binary analysis for debugging failures
func TestBinaryDumpOnFailure(t *testing.T) {
	// This test helps debug field offset issues by dumping binary data
	generator := mavlink.NewGenerator(1, 1)

	batteryMsg := mavlink.BatteryMessage{
		BatteryID:        0xAA,
		BatteryFunction:  0xBB,
		BatteryType:      0xCC,
		Temperature:      0x1122,
		Voltages:         [10]uint16{0x3344, 0x5566, 0x7788, 0x99AA, 0xBBCC, 0xDDEE, 0xFF00, 0x1122, 0x3344, 0x5566},
		CurrentBattery:   0x7788,
		CurrentConsumed:  0x19AABBCC,
		EnergyConsumed:   0x5DEEFF00,
		BatteryRemaining: 0x19,
	}

	data, err := generator.GenerateBatteryStatus(batteryMsg)
	require.NoError(t, err)

	t.Logf("Generated MAVLink frame (%d bytes):", len(data))
	for i, b := range data {
		if i%16 == 0 {
			t.Logf("%04X:", i)
		}
		t.Logf(" %02X", b)
		if i%16 == 15 || i == len(data)-1 {
			t.Logf("")
		}
	}

	// Extract payload (skip header)
	payload := data[constants.MavlinkHeaderSizeV2 : len(data)-constants.MavlinkChecksumSize]
	t.Logf("\nPayload analysis (%d bytes):", len(payload))
	t.Logf("Offset 0 (BatteryID): 0x%02X", payload[0])
	t.Logf("Offset 1 (BatteryFunction): 0x%02X", payload[1])
	t.Logf("Offset 2 (BatteryType): 0x%02X", payload[2])
	t.Logf("Offset 3-4 (Temperature): 0x%04X", binary.LittleEndian.Uint16(payload[3:5]))

	for i := 0; i < 10; i++ {
		offset := 5 + i*2
		voltage := binary.LittleEndian.Uint16(payload[offset : offset+2])
		t.Logf("Offset %d-%d (Voltage[%d]): 0x%04X", offset, offset+1, i, voltage)
	}

	t.Logf("Offset 25-26 (CurrentBattery): 0x%04X", binary.LittleEndian.Uint16(payload[25:27]))
	t.Logf("Offset 27-30 (CurrentConsumed): 0x%08X", binary.LittleEndian.Uint32(payload[27:31]))
	t.Logf("Offset 31-34 (EnergyConsumed): 0x%08X", binary.LittleEndian.Uint32(payload[31:35]))
	t.Logf("Offset 35 (BatteryRemaining): 0x%02X", payload[35])
}
