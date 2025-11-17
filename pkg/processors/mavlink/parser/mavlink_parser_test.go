package parser

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	message "github.com/c360/semstreams/message"
	"github.com/c360/semops/pkg/processors/mavlink/constants"
	roboticsPayloads "github.com/c360/semops/pkg/processors/mavlink/payloads"
)

func TestNewMAVLinkParser(t *testing.T) {
	parser := NewMAVLinkParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.buffer)
	assert.NotNil(t, parser.messageSpecs)
	assert.Equal(t, 0, len(parser.buffer))
	assert.Greater(t, len(parser.messageSpecs), 0, "Should have registered standard messages")

	// Check that standard messages are registered
	assert.Contains(t, parser.messageSpecs, uint32(constants.MavlinkMsgIdHeartbeat))
	assert.Contains(t, parser.messageSpecs, uint32(constants.MavlinkMsgIdGlobalPositionInt))
	assert.Contains(t, parser.messageSpecs, uint32(constants.MavlinkMsgIdAttitude))
	assert.Contains(t, parser.messageSpecs, uint32(constants.MAVLINK_MSG_ID_BATTERY_STATUS))
}

func TestParseV1Heartbeat(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create a valid MAVLink v1 heartbeat packet
	packet := createV1HeartbeatPacket(t)

	packets, err := parser.Parse(packet)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	parsed := packets[0]
	assert.Equal(t, uint8(constants.MavlinkV1), parsed.Version)
	assert.Equal(t, uint8(9), parsed.Length) // Heartbeat payload length
	assert.Equal(t, uint8(1), parsed.SystemID)
	assert.Equal(t, uint8(1), parsed.ComponentID)
	assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), parsed.MessageID)
	assert.NotNil(t, parsed.ParsedFields)

	// Check parsed fields
	assert.Equal(t, uint8(constants.MavTypeQuadrotor), parsed.ParsedFields["type"])
	assert.Equal(t, uint8(constants.MavAutopilotArdupilotmega), parsed.ParsedFields["autopilot"])
	assert.Equal(t, uint8(constants.MavModeFlagSafetyArmed), parsed.ParsedFields["base_mode"])
}

func TestParseV2GlobalPosition(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create a valid MAVLink v2 global position packet
	packet := createV2GlobalPositionPacket(t)

	packets, err := parser.Parse(packet)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	parsed := packets[0]
	assert.Equal(t, uint8(constants.MavlinkV2), parsed.Version)
	assert.Equal(t, uint8(28), parsed.Length) // Global position payload length
	assert.Equal(t, uint32(constants.MavlinkMsgIdGlobalPositionInt), parsed.MessageID)

	// Check parsed fields
	lat, ok := parsed.ParsedFields["lat"].(int32)
	require.True(t, ok)
	assert.Equal(t, int32(374946240), lat) // 37.4946240 degrees * 1e7

	lon, ok := parsed.ParsedFields["lon"].(int32)
	require.True(t, ok)
	assert.Equal(t, int32(-1223073280), lon) // -122.3073280 degrees * 1e7
}

func TestParseMultiplePackets(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create multiple packets in sequence
	heartbeat := createV1HeartbeatPacket(t)
	position := createV2GlobalPositionPacket(t)
	combined := append(heartbeat, position...)

	packets, err := parser.Parse(combined)
	require.NoError(t, err)
	require.Len(t, packets, 2)

	// First packet should be heartbeat
	assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), packets[0].MessageID)
	assert.Equal(t, uint8(constants.MavlinkV1), packets[0].Version)

	// Second packet should be global position
	assert.Equal(t, uint32(constants.MavlinkMsgIdGlobalPositionInt), packets[1].MessageID)
	assert.Equal(t, uint8(constants.MavlinkV2), packets[1].Version)
}

func TestParseFragmentedPacket(t *testing.T) {
	parser := NewMAVLinkParser()

	packet := createV1HeartbeatPacket(t)

	// Send packet in fragments
	fragment1 := packet[:5]
	fragment2 := packet[5:]

	// First fragment should return no packets
	packets1, err := parser.Parse(fragment1)
	require.NoError(t, err)
	assert.Len(t, packets1, 0)

	// Second fragment should complete the packet
	packets2, err := parser.Parse(fragment2)
	require.NoError(t, err)
	require.Len(t, packets2, 1)

	assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), packets2[0].MessageID)
}

func TestParseInvalidSyncByte(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create packet with invalid sync byte
	invalidPacket := []byte{0xFF, 0x09, 0x01, 0x01, 0x01, 0x00}

	packets, err := parser.Parse(invalidPacket)
	require.NoError(t, err)
	assert.Len(t, packets, 0) // Should skip invalid bytes
}

func TestParseChecksumError(t *testing.T) {
	parser := NewMAVLinkParser()

	packet := createV1HeartbeatPacket(t)
	// Corrupt the checksum
	packet[len(packet)-1] = 0xFF

	packets, err := parser.Parse(packet)
	require.NoError(t, err)
	assert.Len(t, packets, 0) // Should reject packet with bad checksum
	assert.Greater(t, parser.parsingStats.ChecksumErrors, uint64(0))
}

func TestConvertToSemStreamsHeartbeat(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create and parse heartbeat packet
	packet := createV1HeartbeatPacket(t)
	packets, err := parser.Parse(packet)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	// Convert to SemStreams message
	msg, err := parser.ConvertToSemStreamsMessage(packets[0])
	require.NoError(t, err)

	baseMsg, ok := msg.(message.Message)
	require.True(t, ok)

	assert.Equal(t, "robotics.heartbeat.v1", baseMsg.Type().String())
	heartbeatPayload, ok := baseMsg.Payload().(*roboticsPayloads.HeartbeatPayload)
	require.True(t, ok)
	assert.Equal(t, uint8(1), heartbeatPayload.SystemID)
	assert.Equal(t, uint8(1), heartbeatPayload.ComponentID)
	assert.Equal(t, uint8(constants.MavTypeQuadrotor), heartbeatPayload.VehicleType)
	assert.Equal(t, uint8(constants.MavAutopilotArdupilotmega), heartbeatPayload.Autopilot)
}

func TestConvertToSemStreamsPosition(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create and parse position packet
	packet := createV2GlobalPositionPacket(t)
	packets, err := parser.Parse(packet)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	// Convert to SemStreams message
	msg, err := parser.ConvertToSemStreamsMessage(packets[0])
	require.NoError(t, err)

	baseMsg, ok := msg.(message.Message)
	require.True(t, ok)

	assert.Equal(t, "robotics.position.v1", baseMsg.Type().String())
	positionPayload, ok := baseMsg.Payload().(*roboticsPayloads.PositionPayload)
	require.True(t, ok)
	assert.Equal(t, uint8(1), positionPayload.SystemID)
	// Convert from int32 back to float64 degrees (divide by 1e7)
	expectedLat := float64(positionPayload.Lat) / 1e7
	expectedLon := float64(positionPayload.Lon) / 1e7
	assert.InDelta(t, 37.494624, expectedLat, 0.0000001)
	assert.InDelta(t, -122.307328, expectedLon, 0.0000001)
}

func TestCRCCalculation(t *testing.T) {
	parser := NewMAVLinkParser()

	// Test CRC calculation with known values
	data := []byte{0x09, 0x00, 0x01, 0x01, 0x00}
	payload := []byte{0x02, 0x03, 0x89, 0x00, 0x00, 0x00, 0x00, 0x04, 0x03}
	fullData := append(data, payload...)

	crc := parser.calculateChecksum(fullData, constants.MavlinkMsgIdHeartbeat)
	assert.Greater(t, crc, uint16(0)) // Should produce non-zero CRC
}

func TestStreamParsingWithNoise(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create stream with noise before valid packet
	noise := []byte{0x12, 0x34, 0x56, 0x78}
	validPacket := createV1HeartbeatPacket(t)
	stream := append(noise, validPacket...)

	packets, err := parser.Parse(stream)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	assert.Equal(t, uint32(constants.MavlinkMsgIdHeartbeat), packets[0].MessageID)
}

func TestV2WithSignature(t *testing.T) {
	parser := NewMAVLinkParser()

	// Create MAVLink v2 packet with signature
	packet := createV2PacketWithSignature(t)

	packets, err := parser.Parse(packet)
	require.NoError(t, err)
	require.Len(t, packets, 1)

	parsed := packets[0]
	assert.Equal(t, uint8(constants.MavlinkV2), parsed.Version)
	assert.NotNil(t, parsed.Signature)
	assert.Len(t, parsed.Signature, constants.MavlinkSignatureSize)
}

func TestParserStats(t *testing.T) {
	parser := NewMAVLinkParser()

	// Parse valid packet
	validPacket := createV1HeartbeatPacket(t)
	_, err := parser.Parse(validPacket)
	require.NoError(t, err)

	// Parse invalid packet
	invalidPacket := createV1HeartbeatPacket(t)
	invalidPacket[len(invalidPacket)-1] = 0xFF // Corrupt checksum
	_, err = parser.Parse(invalidPacket)
	require.NoError(t, err)

	stats := parser.GetStats()
	assert.Equal(t, uint64(1), stats.ValidPackets)
	assert.Equal(t, uint64(1), stats.ChecksumErrors)
	assert.Equal(t, uint64(1), stats.TotalPackets)
}

func TestRegisterCustomMessageSpec(t *testing.T) {
	parser := NewMAVLinkParser()

	customSpec := &MessageSpec{
		ID:          999,
		Name:        "CUSTOM_TEST",
		PayloadSize: 4,
		CRCExtra:    100,
		Fields: []FieldSpec{
			{Name: "test_field", Type: "uint32", Size: 4, Offset: 0},
		},
	}

	parser.RegisterMessageSpec(customSpec)

	assert.Contains(t, parser.messageSpecs, uint32(999))
	assert.Equal(t, customSpec, parser.messageSpecs[uint32(999)])
}

func TestFieldTypeParsing(t *testing.T) {
	parser := NewMAVLinkParser()

	tests := []struct {
		name      string
		fieldType string
		data      []byte
		expected  any
	}{
		{"uint8", "uint8", []byte{0xFF}, uint8(255)},
		{"int8", "int8", []byte{0xFF}, int8(-1)},
		{"uint16", "uint16", []byte{0x34, 0x12}, uint16(0x1234)},
		{"int16", "int16", []byte{0xFF, 0xFF}, int16(-1)},
		{"uint32", "uint32", []byte{0x78, 0x56, 0x34, 0x12}, uint32(0x12345678)},
		{"float", "float", []byte{0x00, 0x00, 0x80, 0x3F}, float32(1.0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldSpec := FieldSpec{
				Name: "test",
				Type: tt.fieldType,
				Size: uint8(len(tt.data)),
			}

			value, err := parser.parseFieldValue(tt.data, fieldSpec)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}
}

// Helper functions for creating test packets

func createV1HeartbeatPacket(t *testing.T) []byte {
	var buf bytes.Buffer

	// Header
	buf.WriteByte(constants.MavlinkStxV1)          // Start byte
	buf.WriteByte(9)                               // Payload length
	buf.WriteByte(0)                               // Sequence
	buf.WriteByte(1)                               // System ID
	buf.WriteByte(1)                               // Component ID
	buf.WriteByte(constants.MavlinkMsgIdHeartbeat) // Message ID

	// Payload (heartbeat) - correct MAVLink field order
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // custom_mode (offset 0-3)
	buf.WriteByte(constants.MavTypeQuadrotor)          // type (offset 4)
	buf.WriteByte(constants.MavAutopilotArdupilotmega) // autopilot (offset 5)
	buf.WriteByte(constants.MavModeFlagSafetyArmed)    // base_mode (offset 6)
	buf.WriteByte(constants.MavStateActive)            // system_status (offset 7)
	buf.WriteByte(3)                                   // mavlink_version (offset 8)

	// Calculate and append checksum
	data := buf.Bytes()[1:] // Exclude start byte from CRC
	parser := NewMAVLinkParser()
	crc := parser.calculateChecksum(data, constants.MavlinkMsgIdHeartbeat)
	binary.Write(&buf, binary.LittleEndian, crc)

	return buf.Bytes()
}

func createV2GlobalPositionPacket(t *testing.T) []byte {
	var buf bytes.Buffer

	// Header
	buf.WriteByte(constants.MavlinkStxV2) // Start byte
	buf.WriteByte(28)                     // Payload length
	buf.WriteByte(0)                      // Incompat flags
	buf.WriteByte(0)                      // Compat flags
	buf.WriteByte(0)                      // Sequence
	buf.WriteByte(1)                      // System ID
	buf.WriteByte(1)                      // Component ID

	// Message ID (24-bit)
	msgID := constants.MavlinkMsgIdGlobalPositionInt
	buf.WriteByte(byte(msgID))
	buf.WriteByte(byte(msgID >> 8))
	buf.WriteByte(byte(msgID >> 16))

	// Payload (global position)
	binary.Write(&buf, binary.LittleEndian, uint32(12345))      // time_boot_ms
	binary.Write(&buf, binary.LittleEndian, int32(374946240))   // lat (37.4946240 * 1e7)
	binary.Write(&buf, binary.LittleEndian, int32(-1223073280)) // lon (-122.3073280 * 1e7)
	binary.Write(&buf, binary.LittleEndian, int32(50000))       // alt (50m)
	binary.Write(&buf, binary.LittleEndian, int32(100000))      // relative_alt (100m)
	binary.Write(&buf, binary.LittleEndian, int16(100))         // vx (1 m/s)
	binary.Write(&buf, binary.LittleEndian, int16(0))           // vy
	binary.Write(&buf, binary.LittleEndian, int16(-10))         // vz (-0.1 m/s)
	binary.Write(&buf, binary.LittleEndian, uint16(9000))       // hdg (90 degrees)

	// Calculate and append checksum
	data := buf.Bytes()[1:] // Exclude start byte from CRC
	parser := NewMAVLinkParser()
	crc := parser.calculateChecksum(data, uint32(msgID))
	binary.Write(&buf, binary.LittleEndian, crc)

	return buf.Bytes()
}

func createV2PacketWithSignature(t *testing.T) []byte {
	var buf bytes.Buffer

	// Header with signature flag
	buf.WriteByte(constants.MavlinkStxV2) // Start byte
	buf.WriteByte(9)                      // Payload length
	buf.WriteByte(0x01)                   // Incompat flags (signature enabled)
	buf.WriteByte(0)                      // Compat flags
	buf.WriteByte(0)                      // Sequence
	buf.WriteByte(1)                      // System ID
	buf.WriteByte(1)                      // Component ID

	// Message ID (24-bit)
	msgID := constants.MavlinkMsgIdHeartbeat
	buf.WriteByte(byte(msgID))
	buf.WriteByte(byte(msgID >> 8))
	buf.WriteByte(byte(msgID >> 16))

	// Payload (heartbeat) - correct MAVLink field order
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // custom_mode (offset 0-3)
	buf.WriteByte(constants.MavTypeQuadrotor)          // type (offset 4)
	buf.WriteByte(constants.MavAutopilotArdupilotmega) // autopilot (offset 5)
	buf.WriteByte(constants.MavModeFlagSafetyArmed)    // base_mode (offset 6)
	buf.WriteByte(constants.MavStateActive)            // system_status (offset 7)
	buf.WriteByte(3)                                   // mavlink_version (offset 8)

	// Calculate and append checksum
	data := buf.Bytes()[1:] // Exclude start byte from CRC
	parser := NewMAVLinkParser()
	crc := parser.calculateChecksum(data, uint32(msgID))
	binary.Write(&buf, binary.LittleEndian, crc)

	// Append signature (13 bytes)
	signature := make([]byte, constants.MavlinkSignatureSize)
	for i := range signature {
		signature[i] = byte(i + 1) // Simple test signature
	}
	buf.Write(signature)

	return buf.Bytes()
}

func createV1HeartbeatPacketBench() []byte {
	var buf bytes.Buffer

	// Header
	buf.WriteByte(constants.MavlinkStxV1)          // Start byte
	buf.WriteByte(9)                               // Payload length
	buf.WriteByte(0)                               // Sequence
	buf.WriteByte(1)                               // System ID
	buf.WriteByte(1)                               // Component ID
	buf.WriteByte(constants.MavlinkMsgIdHeartbeat) // Message ID

	// Payload (heartbeat) - correct MAVLink field order
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // custom_mode (offset 0-3)
	buf.WriteByte(constants.MavTypeQuadrotor)          // type (offset 4)
	buf.WriteByte(constants.MavAutopilotArdupilotmega) // autopilot (offset 5)
	buf.WriteByte(constants.MavModeFlagSafetyArmed)    // base_mode (offset 6)
	buf.WriteByte(constants.MavStateActive)            // system_status (offset 7)
	buf.WriteByte(3)                                   // mavlink_version (offset 8)

	// Calculate and append checksum
	data := buf.Bytes()[1:] // Exclude start byte from CRC
	parser := NewMAVLinkParser()
	crc := parser.calculateChecksum(data, constants.MavlinkMsgIdHeartbeat)
	binary.Write(&buf, binary.LittleEndian, crc)

	return buf.Bytes()
}

func createV2GlobalPositionPacketBench() []byte {
	var buf bytes.Buffer

	// Header
	buf.WriteByte(constants.MavlinkStxV2) // Start byte
	buf.WriteByte(28)                     // Payload length
	buf.WriteByte(0)                      // Incompat flags
	buf.WriteByte(0)                      // Compat flags
	buf.WriteByte(0)                      // Sequence
	buf.WriteByte(1)                      // System ID
	buf.WriteByte(1)                      // Component ID

	// Message ID (24-bit)
	msgID := constants.MavlinkMsgIdGlobalPositionInt
	buf.WriteByte(byte(msgID))
	buf.WriteByte(byte(msgID >> 8))
	buf.WriteByte(byte(msgID >> 16))

	// Payload (global position)
	binary.Write(&buf, binary.LittleEndian, uint32(12345))      // time_boot_ms
	binary.Write(&buf, binary.LittleEndian, int32(374946240))   // lat (37.4946240 * 1e7)
	binary.Write(&buf, binary.LittleEndian, int32(-1223073280)) // lon (-122.3073280 * 1e7)
	binary.Write(&buf, binary.LittleEndian, int32(50000))       // alt (50m)
	binary.Write(&buf, binary.LittleEndian, int32(100000))      // relative_alt (100m)
	binary.Write(&buf, binary.LittleEndian, int16(100))         // vx (1 m/s)
	binary.Write(&buf, binary.LittleEndian, int16(0))           // vy
	binary.Write(&buf, binary.LittleEndian, int16(-10))         // vz (-0.1 m/s)
	binary.Write(&buf, binary.LittleEndian, uint16(9000))       // hdg (90 degrees)

	// Calculate and append checksum
	data := buf.Bytes()[1:] // Exclude start byte from CRC
	parser := NewMAVLinkParser()
	crc := parser.calculateChecksum(data, uint32(msgID))
	binary.Write(&buf, binary.LittleEndian, crc)

	return buf.Bytes()
}

func BenchmarkParseV1Heartbeat(b *testing.B) {
	parser := NewMAVLinkParser()
	packet := createV1HeartbeatPacketBench()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.buffer = parser.buffer[:0] // Reset buffer
		_, err := parser.Parse(packet)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseV2GlobalPosition(b *testing.B) {
	parser := NewMAVLinkParser()
	packet := createV2GlobalPositionPacketBench()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.buffer = parser.buffer[:0] // Reset buffer
		_, err := parser.Parse(packet)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertToSemStreams(b *testing.B) {
	parser := NewMAVLinkParser()
	packet := createV1HeartbeatPacketBench()

	packets, err := parser.Parse(packet)
	if err != nil || len(packets) != 1 {
		b.Fatal("Failed to parse test packet")
	}

	mavlinkPacket := packets[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ConvertToSemStreamsMessage(mavlinkPacket)
		if err != nil {
			b.Fatal(err)
		}
	}
}
