package mavlink

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/c360/semops/pkg/processors/mavlink/constants"
	"github.com/c360/semops/pkg/processors/mavlink/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_NewGenerator(t *testing.T) {
	systemID := uint8(1)
	componentID := uint8(1)
	
	gen := NewGenerator(systemID, componentID)
	
	assert.NotNil(t, gen)
	assert.Equal(t, systemID, gen.systemID)
	assert.Equal(t, componentID, gen.componentID)
	assert.Equal(t, uint32(0), gen.sequence)
}

func TestGenerator_SetSystemID(t *testing.T) {
	gen := NewGenerator(1, 1)
	newSystemID := uint8(42)
	
	gen.SetSystemID(newSystemID)
	
	assert.Equal(t, newSystemID, gen.systemID)
}

func TestGenerator_SetComponentID(t *testing.T) {
	gen := NewGenerator(1, 1)
	newComponentID := uint8(200)
	
	gen.SetComponentID(newComponentID)
	
	assert.Equal(t, newComponentID, gen.componentID)
}

func TestGenerator_nextSequence(t *testing.T) {
	gen := NewGenerator(1, 1)
	
	// Test sequence increments
	seq1 := gen.nextSequence()
	seq2 := gen.nextSequence()
	seq3 := gen.nextSequence()
	
	assert.Equal(t, uint8(1), seq1)
	assert.Equal(t, uint8(2), seq2)
	assert.Equal(t, uint8(3), seq3)
}

func TestGenerator_nextSequence_Wraparound(t *testing.T) {
	gen := NewGenerator(1, 1)
	gen.sequence = 254 // Near wrap-around
	
	seq1 := gen.nextSequence() // Should be 255
	seq2 := gen.nextSequence() // Should wrap to 0
	seq3 := gen.nextSequence() // Should be 1
	
	assert.Equal(t, uint8(255), seq1)
	assert.Equal(t, uint8(0), seq2)
	assert.Equal(t, uint8(1), seq3)
}

func TestGenerator_GenerateHeartbeat(t *testing.T) {
	gen := NewGenerator(1, 1)
	msg := HeartbeatMessage{
		VehicleType:    constants.MavTypeQuadrotor,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled,
		CustomMode:     0,
		SystemStatus:   constants.MavStateStandby,
		MavlinkVersion: constants.MavlinkV2,
	}
	
	frame, err := gen.GenerateHeartbeat(msg)
	
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Verify MAVLink v2 frame structure
	assert.Equal(t, uint8(constants.MavlinkStxV2), frame[0], "Start marker should be 0xFD")
	assert.Equal(t, uint8(9), frame[1], "Payload length should be 9 for heartbeat")
	assert.Equal(t, uint8(1), frame[5], "System ID should match")
	assert.Equal(t, uint8(1), frame[6], "Component ID should match")
	
	// Verify message ID (HEARTBEAT = 0)
	msgID := uint32(frame[7]) | (uint32(frame[8]) << 8) | (uint32(frame[9]) << 16)
	assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), msgID)
	
	// Verify payload content
	assert.Equal(t, msg.VehicleType, frame[14], "Vehicle type should match")
	assert.Equal(t, msg.Autopilot, frame[15], "Autopilot should match")
	assert.Equal(t, msg.BaseMode, frame[16], "Base mode should match")
	assert.Equal(t, msg.SystemStatus, frame[17], "System status should match")
	assert.Equal(t, msg.MavlinkVersion, frame[18], "MAVLink version should match")
	
	// Verify frame can be parsed
	validateWithParser(t, frame)
}

func TestGenerator_GenerateBatteryStatus(t *testing.T) {
	gen := NewGenerator(1, 1)
	msg := BatteryMessage{
		BatteryID:        0,
		BatteryFunction:  0,
		BatteryType:      1,
		Temperature:      2500, // 25°C
		CurrentBattery:   -1500, // -15A
		CurrentConsumed:  1000,
		EnergyConsumed:   144000,
		BatteryRemaining: 85,
	}
	
	// Set some realistic cell voltages
	for i := 0; i < 4; i++ {
		msg.Voltages[i] = 3700 // 3.7V per cell in mV
	}
	
	frame, err := gen.GenerateBatteryStatus(msg)
	
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Verify MAVLink v2 frame structure
	assert.Equal(t, uint8(constants.MavlinkStxV2), frame[0])
	assert.Equal(t, uint8(36), frame[1], "Payload length should be 36 for battery status")
	
	// Verify message ID (BATTERY_STATUS = 147)
	msgID := uint32(frame[7]) | (uint32(frame[8]) << 8) | (uint32(frame[9]) << 16)
	assert.Equal(t, uint32(constants.MavlinkMsgIdBatteryStatus), msgID)
	
	// Verify some key payload fields
	assert.Equal(t, msg.BatteryID, frame[10], "Battery ID should match") // payload[0]
	assert.Equal(t, uint8(msg.BatteryRemaining), frame[45], "Battery remaining should match") // payload[35]
	
	// Verify frame can be parsed
	validateWithParser(t, frame)
}

func TestGenerator_GenerateGlobalPosition(t *testing.T) {
	gen := NewGenerator(1, 1)
	msg := PositionMessage{
		TimeBootMs:  12345,
		Lat:         -353621474, // Sydney coordinates in degE7
		Lon:         1513379054,
		Alt:         50000,  // 50m in mm
		RelativeAlt: 50000,  // 50m in mm
		Vx:          500,    // 5 m/s in cm/s
		Vy:          0,
		Vz:          0,
		Hdg:         9000,   // 90 degrees in cdeg
	}
	
	frame, err := gen.GenerateGlobalPosition(msg)
	
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Verify MAVLink v2 frame structure
	assert.Equal(t, uint8(constants.MavlinkStxV2), frame[0])
	assert.Equal(t, uint8(28), frame[1], "Payload length should be 28 for global position")
	
	// Verify message ID (GLOBAL_POSITION_INT = 33)
	msgID := uint32(frame[7]) | (uint32(frame[8]) << 8) | (uint32(frame[9]) << 16)
	assert.Equal(t, uint32(constants.MavlinkMsgIdGlobalPositionInt), msgID)
	
	// Verify timestamp in payload
	timeBootMs := binary.LittleEndian.Uint32(frame[10:14]) // payload[0:4]
	assert.Equal(t, msg.TimeBootMs, timeBootMs)
	
	// Verify frame can be parsed
	validateWithParser(t, frame)
}

func TestGenerator_GenerateAttitude(t *testing.T) {
	gen := NewGenerator(1, 1)
	msg := AttitudeMessage{
		TimeBootMs: 12345,
		Roll:       0.1,  // radians
		Pitch:      0.05, // radians
		Yaw:        1.57, // ~90 degrees
		Rollspeed:  0.01,
		Pitchspeed: 0.02,
		Yawspeed:   0.03,
	}
	
	frame, err := gen.GenerateAttitude(msg)
	
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Verify MAVLink v2 frame structure
	assert.Equal(t, uint8(constants.MavlinkStxV2), frame[0])
	assert.Equal(t, uint8(28), frame[1], "Payload length should be 28 for attitude")
	
	// Verify message ID (ATTITUDE = 30)
	msgID := uint32(frame[7]) | (uint32(frame[8]) << 8) | (uint32(frame[9]) << 16)
	assert.Equal(t, uint32(constants.MavlinkMsgIdAttitude), msgID)
	
	// Verify timestamp in payload
	timeBootMs := binary.LittleEndian.Uint32(frame[10:14]) // payload[0:4]
	assert.Equal(t, msg.TimeBootMs, timeBootMs)
	
	// Verify frame can be parsed
	validateWithParser(t, frame)
}

func TestGenerator_ThreadSafety(t *testing.T) {
	gen := NewGenerator(1, 1)
	const numGoroutines = 10
	const messagesPerGoroutine = 100
	
	messages := make(chan []byte, numGoroutines*messagesPerGoroutine)
	
	// Launch multiple goroutines generating messages concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := HeartbeatMessage{
					VehicleType:    constants.MavTypeQuadrotor,
					Autopilot:      constants.MavAutopilotArdupilotmega,
					BaseMode:       constants.MavModeFlagStabilizeEnabled,
					SystemStatus:   constants.MavStateStandby,
					MavlinkVersion: constants.MavlinkV2,
				}
				
				frame, err := gen.GenerateHeartbeat(msg)
				require.NoError(t, err)
				messages <- frame
			}
		}()
	}
	
	// Collect all messages and verify they're all valid
	sequences := make(map[uint8]int)
	for i := 0; i < numGoroutines*messagesPerGoroutine; i++ {
		frame := <-messages
		require.NotNil(t, frame)
		
		// Extract sequence number
		seq := frame[4]
		sequences[seq]++
		
		// Verify frame structure
		assert.Equal(t, uint8(constants.MavlinkStxV2), frame[0])
		validateWithParser(t, frame)
	}
	
	// Verify we got the expected total number of messages
	totalMessages := 0
	for _, count := range sequences {
		totalMessages += count
	}
	assert.Equal(t, numGoroutines*messagesPerGoroutine, totalMessages)
}

func TestMessageSequence(t *testing.T) {
	seq := NewMessageSequence(1, 1)
	
	// Test initial state
	assert.Equal(t, uint32(0), seq.GetTimeBootMs())
	
	// Test time advancement
	seq.AdvanceTime(1 * time.Second)
	assert.Equal(t, uint32(1000), seq.GetTimeBootMs())
	
	seq.AdvanceTime(500 * time.Millisecond)
	assert.Equal(t, uint32(1500), seq.GetTimeBootMs())
}

func TestMessageSequence_RealisticMessages(t *testing.T) {
	seq := NewMessageSequence(42, 1)
	
	// Test realistic heartbeat
	heartbeat, err := seq.GenerateRealisticHeartbeat(constants.MavTypeQuadrotor)
	require.NoError(t, err)
	require.NotNil(t, heartbeat)
	validateWithParser(t, heartbeat)
	
	// Test realistic battery status
	battery, err := seq.GenerateRealisticBatteryStatus(75)
	require.NoError(t, err)
	require.NotNil(t, battery)
	validateWithParser(t, battery)
	
	// Test realistic position
	position, err := seq.GenerateRealisticPosition(-33.86785, 151.20732, 100) // Sydney coords
	require.NoError(t, err)
	require.NotNil(t, position)
	validateWithParser(t, position)
	
	// Test realistic attitude
	attitude, err := seq.GenerateRealisticAttitude(0.1, 0.05, 1.57)
	require.NoError(t, err)
	require.NotNil(t, attitude)
	validateWithParser(t, attitude)
}

func TestQuadcopterScenario(t *testing.T) {
	scenario := NewQuadcopterScenario(1, -33.86785, 151.20732, 100)
	
	// Generate a sequence of messages
	for i := 0; i < 5; i++ {
		// Advance time
		scenario.AdvanceTime(1 * time.Second)
		
		// Generate all message types
		heartbeat, err := scenario.NextHeartbeat()
		require.NoError(t, err)
		validateWithParser(t, heartbeat)
		
		battery, err := scenario.NextBatteryStatus()
		require.NoError(t, err)
		validateWithParser(t, battery)
		
		position, err := scenario.NextPosition()
		require.NoError(t, err)
		validateWithParser(t, position)
		
		attitude, err := scenario.NextAttitude()
		require.NoError(t, err)
		validateWithParser(t, attitude)
		
		// Verify time advancement
		assert.Equal(t, uint32((i+1)*1000), scenario.GetCurrentTime())
	}
}

func TestGenerator_InvalidPayloadSize(t *testing.T) {
	gen := NewGenerator(1, 1)
	
	// Create a payload that's too large
	largePayload := make([]byte, constants.MavlinkMaxPayloadLen+1)
	
	frame, err := gen.buildMAVLinkV2Frame(constants.MavlinkMsgIdHeartbeat, largePayload)
	
	assert.Error(t, err)
	assert.Nil(t, frame)
	assert.Contains(t, err.Error(), "payload too large")
}

func TestBatteryMessage_EdgeCases(t *testing.T) {
	gen := NewGenerator(1, 1)
	
	// Test with empty/invalid values
	msg := BatteryMessage{
		BatteryID:        255,
		BatteryFunction:  0,
		BatteryType:      0,
		Temperature:      -1, // Not measured
		CurrentBattery:   -1, // Not measured
		CurrentConsumed:  0,
		EnergyConsumed:   0,
		BatteryRemaining: -1, // Not measured
	}
	
	frame, err := gen.GenerateBatteryStatus(msg)
	require.NoError(t, err)
	validateWithParser(t, frame)
}

func TestGenerator_CRCValidation(t *testing.T) {
	gen := NewGenerator(1, 1)
	
	// Generate a heartbeat message
	msg := HeartbeatMessage{
		VehicleType:    constants.MavTypeQuadrotor,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled,
		SystemStatus:   constants.MavStateStandby,
		MavlinkVersion: constants.MavlinkV2,
	}
	
	frame1, err := gen.GenerateHeartbeat(msg)
	require.NoError(t, err)
	
	frame2, err := gen.GenerateHeartbeat(msg)
	require.NoError(t, err)
	
	// CRCs should be the same for identical messages (except sequence)
	crc1 := binary.LittleEndian.Uint16(frame1[len(frame1)-2:])
	crc2 := binary.LittleEndian.Uint16(frame2[len(frame2)-2:])
	
	// CRCs will differ because sequence numbers differ
	assert.NotEqual(t, crc1, crc2, "CRCs should differ due to different sequence numbers")
	
	// But both frames should be valid
	validateWithParser(t, frame1)
	validateWithParser(t, frame2)
}

// validateWithParser uses the existing MAVLink parser to validate generated frames
func validateWithParser(t *testing.T, frame []byte) {
	parser := parser.NewMAVLinkParser()
	
	// Parse the frame
	packets, err := parser.Parse(frame)
	assert.NoError(t, err, "Generated MAVLink frame should be parseable")
	assert.NotEmpty(t, packets, "Parser should return at least one packet")
	
	// Verify the first packet is valid
	packet := packets[0]
	assert.NotNil(t, packet, "Parser should return valid packet")
	assert.Equal(t, uint8(2), packet.Version, "Should be MAVLink v2")
	assert.NotNil(t, packet.Payload, "Packet should have payload")
}

// Benchmarks
func BenchmarkGenerator_GenerateHeartbeat(b *testing.B) {
	gen := NewGenerator(1, 1)
	msg := HeartbeatMessage{
		VehicleType:    constants.MavTypeQuadrotor,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled,
		SystemStatus:   constants.MavStateStandby,
		MavlinkVersion: constants.MavlinkV2,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.GenerateHeartbeat(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerator_GenerateBatteryStatus(b *testing.B) {
	gen := NewGenerator(1, 1)
	msg := BatteryMessage{
		BatteryID:        0,
		BatteryFunction:  0,
		BatteryType:      1,
		Temperature:      2500,
		CurrentBattery:   -1500,
		CurrentConsumed:  1000,
		EnergyConsumed:   144000,
		BatteryRemaining: 85,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.GenerateBatteryStatus(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuadcopterScenario_FullSequence(b *testing.B) {
	scenario := NewQuadcopterScenario(1, -33.86785, 151.20732, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario.AdvanceTime(100 * time.Millisecond)
		
		_, _ = scenario.NextHeartbeat()
		_, _ = scenario.NextBatteryStatus()
		_, _ = scenario.NextPosition()
		_, _ = scenario.NextAttitude()
	}
}