//go:build ignore
// +build ignore

// Package mavlink provides MAVLink message generation utilities for testing drone communication protocols.
// It implements the MAVLink v2 protocol specification for generating realistic test data
// in E2E testing scenarios.
package mavlink

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/c360/semops/pkg/processors/mavlink/constants"
)

// CRCTable stores MAVLink CRC extra values for message validation
var crcTable = map[uint32]uint8{
	constants.MavlinkMsgIdHeartbeat:         50,  // HEARTBEAT
	constants.MavlinkMsgIdBatteryStatus:     154, // BATTERY_STATUS
	constants.MavlinkMsgIdGlobalPositionInt: 104, // GLOBAL_POSITION_INT
	constants.MavlinkMsgIdAttitude:          39,  // ATTITUDE
}

// Generator provides thread-safe MAVLink message generation for testing
type Generator struct {
	systemID    uint8
	componentID uint8
	sequence    uint32 // atomic counter for thread safety
}

// NewGenerator creates a new MAVLink message generator
func NewGenerator(systemID, componentID uint8) *Generator {
	return &Generator{
		systemID:    systemID,
		componentID: componentID,
		sequence:    0,
	}
}

// SetSystemID sets the system ID for generated messages
func (g *Generator) SetSystemID(systemID uint8) {
	g.systemID = systemID
}

// SetComponentID sets the component ID for generated messages
func (g *Generator) SetComponentID(componentID uint8) {
	g.componentID = componentID
}

// nextSequence returns the next sequence number in a thread-safe manner
func (g *Generator) nextSequence() uint8 {
	seq := atomic.AddUint32(&g.sequence, 1)
	return uint8(seq % 256)
}

// HeartbeatMessage represents a MAVLink HEARTBEAT message payload
type HeartbeatMessage struct {
	VehicleType    uint8  // MAV_TYPE
	Autopilot      uint8  // MAV_AUTOPILOT
	BaseMode       uint8  // MAV_MODE_FLAG
	CustomMode     uint32 // Custom mode
	SystemStatus   uint8  // MAV_STATE
	MavlinkVersion uint8  // MAVLink protocol version
}

// GenerateHeartbeat creates a valid MAVLink HEARTBEAT message
func (g *Generator) GenerateHeartbeat(msg HeartbeatMessage) ([]byte, error) {
	// MAVLink v2 HEARTBEAT payload: 9 bytes
	payload := make([]byte, 9)

	// Pack payload according to MAVLink HEARTBEAT message format
	binary.LittleEndian.PutUint32(payload[0:4], msg.CustomMode)
	payload[4] = msg.VehicleType
	payload[5] = msg.Autopilot
	payload[6] = msg.BaseMode
	payload[7] = msg.SystemStatus
	payload[8] = msg.MavlinkVersion

	return g.buildMAVLinkV2Frame(constants.MavlinkMsgIdHeartbeat, payload)
}

// BatteryMessage represents a MAVLink BATTERY_STATUS message payload
// According to parser specification (36 bytes total)
type BatteryMessage struct {
	BatteryID        uint8      // Battery ID
	BatteryFunction  uint8      // Battery function
	BatteryType      uint8      // Battery type
	Temperature      int16      // Temperature (cdegC)
	Voltages         [10]uint16 // Cell voltages (mV)
	CurrentBattery   int16      // Current (10*mA)
	CurrentConsumed  int32      // Current consumed charge (mAh)
	EnergyConsumed   int32      // Consumed energy (hJ)
	BatteryRemaining int8       // Battery remaining (%)
}

// GenerateBatteryStatus creates a valid MAVLink BATTERY_STATUS message
func (g *Generator) GenerateBatteryStatus(msg BatteryMessage) ([]byte, error) {
	// MAVLink v2 BATTERY_STATUS payload: 36 bytes (matches parser specification)
	// Field layout (matching parser.go):
	// Offset  Size  Field
	// 0       1     id (uint8)
	// 1       1     battery_function (uint8)
	// 2       1     type (uint8)
	// 3       2     temperature (int16, centiC)
	// 5       20    voltages[10] (uint16 array, mV)
	// 25      2     current_battery (int16, cA)
	// 27      4     current_consumed (int32, mAh)
	// 31      4     energy_consumed (int32, hJ)
	// 35      1     battery_remaining (int8, %)
	payload := make([]byte, 36)

	// Pack payload according to parser specification layout
	payload[0] = msg.BatteryID
	payload[1] = msg.BatteryFunction
	payload[2] = msg.BatteryType
	binary.LittleEndian.PutUint16(payload[3:5], uint16(msg.Temperature))

	// Pack cell voltages (10 cells * 2 bytes each = 20 bytes, starting at offset 5)
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint16(payload[5+i*2:7+i*2], msg.Voltages[i])
	}

	// Pack remaining fields at correct offsets
	binary.LittleEndian.PutUint16(payload[25:27], uint16(msg.CurrentBattery))
	binary.LittleEndian.PutUint32(payload[27:31], uint32(msg.CurrentConsumed))
	binary.LittleEndian.PutUint32(payload[31:35], uint32(msg.EnergyConsumed))
	payload[35] = uint8(msg.BatteryRemaining)

	return g.buildMAVLinkV2Frame(constants.MavlinkMsgIdBatteryStatus, payload)
}

// PositionMessage represents a MAVLink GLOBAL_POSITION_INT message payload
type PositionMessage struct {
	TimeBootMs  uint32 // Timestamp (ms since system boot)
	Lat         int32  // Latitude (degE7)
	Lon         int32  // Longitude (degE7)
	Alt         int32  // Altitude (mm)
	RelativeAlt int32  // Relative altitude (mm)
	Vx          int16  // Ground X velocity (cm/s)
	Vy          int16  // Ground Y velocity (cm/s)
	Vz          int16  // Ground Z velocity (cm/s)
	Hdg         uint16 // Heading (cdeg)
}

// GenerateGlobalPosition creates a valid MAVLink GLOBAL_POSITION_INT message
func (g *Generator) GenerateGlobalPosition(msg PositionMessage) ([]byte, error) {
	// MAVLink v2 GLOBAL_POSITION_INT payload: 28 bytes
	payload := make([]byte, 28)

	// Pack payload according to MAVLink GLOBAL_POSITION_INT message format
	binary.LittleEndian.PutUint32(payload[0:4], msg.TimeBootMs)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(msg.Lat))
	binary.LittleEndian.PutUint32(payload[8:12], uint32(msg.Lon))
	binary.LittleEndian.PutUint32(payload[12:16], uint32(msg.Alt))
	binary.LittleEndian.PutUint32(payload[16:20], uint32(msg.RelativeAlt))
	binary.LittleEndian.PutUint16(payload[20:22], uint16(msg.Vx))
	binary.LittleEndian.PutUint16(payload[22:24], uint16(msg.Vy))
	binary.LittleEndian.PutUint16(payload[24:26], uint16(msg.Vz))
	binary.LittleEndian.PutUint16(payload[26:28], msg.Hdg)

	return g.buildMAVLinkV2Frame(constants.MavlinkMsgIdGlobalPositionInt, payload)
}

// AttitudeMessage represents a MAVLink ATTITUDE message payload
type AttitudeMessage struct {
	TimeBootMs uint32  // Timestamp (ms since system boot)
	Roll       float32 // Roll angle (rad)
	Pitch      float32 // Pitch angle (rad)
	Yaw        float32 // Yaw angle (rad)
	Rollspeed  float32 // Roll angular speed (rad/s)
	Pitchspeed float32 // Pitch angular speed (rad/s)
	Yawspeed   float32 // Yaw angular speed (rad/s)
}

// GenerateAttitude creates a valid MAVLink ATTITUDE message
func (g *Generator) GenerateAttitude(msg AttitudeMessage) ([]byte, error) {
	// MAVLink v2 ATTITUDE payload: 28 bytes
	payload := make([]byte, 28)

	// Pack payload according to MAVLink ATTITUDE message format
	binary.LittleEndian.PutUint32(payload[0:4], msg.TimeBootMs)
	binary.LittleEndian.PutUint32(payload[4:8], math.Float32bits(msg.Roll))
	binary.LittleEndian.PutUint32(payload[8:12], math.Float32bits(msg.Pitch))
	binary.LittleEndian.PutUint32(payload[12:16], math.Float32bits(msg.Yaw))
	binary.LittleEndian.PutUint32(payload[16:20], math.Float32bits(msg.Rollspeed))
	binary.LittleEndian.PutUint32(payload[20:24], math.Float32bits(msg.Pitchspeed))
	binary.LittleEndian.PutUint32(payload[24:28], math.Float32bits(msg.Yawspeed))

	return g.buildMAVLinkV2Frame(constants.MavlinkMsgIdAttitude, payload)
}

// buildMAVLinkV2Frame constructs a complete MAVLink v2 frame with proper CRC
func (g *Generator) buildMAVLinkV2Frame(msgID uint32, payload []byte) ([]byte, error) {
	payloadLen := len(payload)
	if payloadLen > constants.MavlinkMaxPayloadLen {
		return nil, fmt.Errorf("payload too large: %d bytes (max %d)", payloadLen, constants.MavlinkMaxPayloadLen)
	}

	// MAVLink v2 header: STX(1) + LEN(1) + INCOMPAT(1) + COMPAT(1) + SEQ(1) + SYSID(1) + COMPID(1) + MSGID(3) = 10 bytes
	frameLen := constants.MavlinkHeaderSizeV2 + payloadLen + constants.MavlinkChecksumSize
	frame := make([]byte, frameLen)

	// Build header
	frame[0] = constants.MavlinkStxV2      // Start marker
	frame[1] = uint8(payloadLen)           // Payload length
	frame[2] = 0                           // Incompatibility flags
	frame[3] = 0                           // Compatibility flags
	frame[4] = g.nextSequence()            // Sequence
	frame[5] = g.systemID                  // System ID
	frame[6] = g.componentID               // Component ID
	frame[7] = uint8(msgID & 0xFF)         // Message ID low byte
	frame[8] = uint8((msgID >> 8) & 0xFF)  // Message ID middle byte
	frame[9] = uint8((msgID >> 16) & 0xFF) // Message ID high byte

	// Copy payload
	copy(frame[constants.MavlinkHeaderSizeV2:], payload)

	// Calculate and append CRC
	crc := g.calculateCRC(frame[:constants.MavlinkHeaderSizeV2+payloadLen], msgID)
	binary.LittleEndian.PutUint16(frame[constants.MavlinkHeaderSizeV2+payloadLen:], crc)

	return frame, nil
}

// calculateCRC computes the MAVLink CRC-16/MCRF4XX checksum
func (g *Generator) calculateCRC(data []byte, msgID uint32) uint16 {
	// MAVLink uses CRC-16-CCITT with polynomial 0x1021
	// Implementation based on MAVLink specification
	crc := uint16(0xFFFF)

	// Process data starting from byte 1 (skip STX)
	for i := 1; i < len(data); i++ {
		tmp := data[i] ^ uint8(crc)
		tmp ^= tmp << 4
		crc = (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
	}

	// Add CRC extra byte if available for this message type
	if crcExtra, ok := crcTable[msgID]; ok {
		tmp := crcExtra ^ uint8(crc)
		tmp ^= tmp << 4
		crc = (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
	}

	return crc
}

// MessageSequence provides utilities for generating realistic message sequences
type MessageSequence struct {
	generator   *Generator
	startTime   time.Time
	currentTime uint32 // milliseconds since start
}

// NewMessageSequence creates a new message sequence generator
func NewMessageSequence(systemID, componentID uint8) *MessageSequence {
	return &MessageSequence{
		generator: NewGenerator(systemID, componentID),
		startTime: time.Now(),
	}
}

// AdvanceTime advances the internal timestamp by the specified duration
func (ms *MessageSequence) AdvanceTime(duration time.Duration) {
	ms.currentTime += uint32(duration.Milliseconds())
}

// GetTimeBootMs returns the current time in milliseconds since sequence start
func (ms *MessageSequence) GetTimeBootMs() uint32 {
	return ms.currentTime
}

// GenerateRealisticHeartbeat creates a realistic heartbeat message
func (ms *MessageSequence) GenerateRealisticHeartbeat(vehicleType uint8) ([]byte, error) {
	msg := HeartbeatMessage{
		VehicleType:    vehicleType,
		Autopilot:      constants.MavAutopilotArdupilotmega,
		BaseMode:       constants.MavModeFlagStabilizeEnabled | constants.MavModeFlagManualInputEnabled,
		CustomMode:     0,
		SystemStatus:   constants.MavStateStandby,
		MavlinkVersion: constants.MavlinkV2,
	}
	return ms.generator.GenerateHeartbeat(msg)
}

// GenerateRealisticBatteryStatus creates a realistic battery status message
func (ms *MessageSequence) GenerateRealisticBatteryStatus(batteryPercent int8) ([]byte, error) {
	// Generate realistic cell voltages based on battery percentage
	cellVoltage := uint16(3300 + (4200-3300)*int(batteryPercent)/100) // 3.3V to 4.2V per cell

	msg := BatteryMessage{
		BatteryID:        0,
		BatteryFunction:  0,      // Main battery
		BatteryType:      1,      // LiPo
		Temperature:      2500,   // 25°C in centigrade
		CurrentBattery:   -1500,  // -15A discharge
		CurrentConsumed:  1000,   // 1Ah consumed
		EnergyConsumed:   144000, // 4Wh in hJ
		BatteryRemaining: batteryPercent,
	}

	// Set realistic cell voltages (4S battery)
	for i := 0; i < 4; i++ {
		msg.Voltages[i] = cellVoltage + uint16(rand.Intn(50)) // Add some variation
	}

	return ms.generator.GenerateBatteryStatus(msg)
}

// GenerateRealisticPosition creates a realistic GPS position message
func (ms *MessageSequence) GenerateRealisticPosition(lat, lon float64, alt int32) ([]byte, error) {
	msg := PositionMessage{
		TimeBootMs:  ms.currentTime,
		Lat:         int32(lat * 1e7), // Convert to degE7
		Lon:         int32(lon * 1e7), // Convert to degE7
		Alt:         alt * 1000,       // Convert to mm
		RelativeAlt: alt * 1000,       // Convert to mm
		Vx:          500,              // 5 m/s north
		Vy:          0,                // 0 m/s east
		Vz:          0,                // 0 m/s down
		Hdg:         uint16(0 * 100),  // 0 degrees in cdeg
	}
	return ms.generator.GenerateGlobalPosition(msg)
}

// GenerateRealisticAttitude creates a realistic attitude message
func (ms *MessageSequence) GenerateRealisticAttitude(roll, pitch, yaw float32) ([]byte, error) {
	msg := AttitudeMessage{
		TimeBootMs: ms.currentTime,
		Roll:       roll,
		Pitch:      pitch,
		Yaw:        yaw,
		Rollspeed:  0.01, // Small angular velocity
		Pitchspeed: 0.01,
		Yawspeed:   0.02,
	}
	return ms.generator.GenerateAttitude(msg)
}

// QuadcopterScenario generates a realistic flight scenario for a quadcopter
type QuadcopterScenario struct {
	sequence       *MessageSequence
	startLat       float64
	startLon       float64
	startAlt       int32
	batteryPercent int8
}

// NewQuadcopterScenario creates a new quadcopter flight scenario
func NewQuadcopterScenario(systemID uint8, lat, lon float64, alt int32) *QuadcopterScenario {
	return &QuadcopterScenario{
		sequence:       NewMessageSequence(systemID, 1),
		startLat:       lat,
		startLon:       lon,
		startAlt:       alt,
		batteryPercent: 100,
	}
}

// NextHeartbeat generates the next heartbeat message in the scenario
func (qs *QuadcopterScenario) NextHeartbeat() ([]byte, error) {
	return qs.sequence.GenerateRealisticHeartbeat(constants.MavTypeQuadrotor)
}

// NextBatteryStatus generates the next battery status message in the scenario
func (qs *QuadcopterScenario) NextBatteryStatus() ([]byte, error) {
	// Gradually drain battery over time
	if qs.batteryPercent > 0 {
		qs.batteryPercent--
	}
	return qs.sequence.GenerateRealisticBatteryStatus(qs.batteryPercent)
}

// NextPosition generates the next position message in the scenario
func (qs *QuadcopterScenario) NextPosition() ([]byte, error) {
	// Simple circular pattern around start position
	radius := 0.001                                     // ~100m in degrees
	angle := float64(qs.sequence.currentTime) / 10000.0 // Slow circular motion

	lat := qs.startLat + radius*math.Sin(angle)
	lon := qs.startLon + radius*math.Cos(angle)

	return qs.sequence.GenerateRealisticPosition(lat, lon, qs.startAlt)
}

// NextAttitude generates the next attitude message in the scenario
func (qs *QuadcopterScenario) NextAttitude() ([]byte, error) {
	// Small attitude variations for stability
	roll := float32(0.05 * math.Sin(float64(qs.sequence.currentTime)/1000.0))
	pitch := float32(0.03 * math.Sin(float64(qs.sequence.currentTime)/1500.0))
	yaw := float32(float64(qs.sequence.currentTime) / 10000.0) // Slowly rotating

	return qs.sequence.GenerateRealisticAttitude(roll, pitch, yaw)
}

// AdvanceTime advances the scenario time
func (qs *QuadcopterScenario) AdvanceTime(duration time.Duration) {
	qs.sequence.AdvanceTime(duration)
}

// GetCurrentTime returns the current scenario time
func (qs *QuadcopterScenario) GetCurrentTime() uint32 {
	return qs.sequence.GetTimeBootMs()
}
