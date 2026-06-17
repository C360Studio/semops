package mavlink

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync/atomic"
	"time"
)

// Generator builds MAVLink v2 frames for deterministic adapter and scenario tests.
type Generator struct {
	systemID    uint8
	componentID uint8
	sequence    uint32
}

func NewGenerator(systemID, componentID uint8) *Generator {
	return &Generator{
		systemID:    systemID,
		componentID: componentID,
	}
}

func (g *Generator) SetSystemID(systemID uint8) {
	g.systemID = systemID
}

func (g *Generator) SetComponentID(componentID uint8) {
	g.componentID = componentID
}

func (g *Generator) nextSequence() uint8 {
	seq := atomic.AddUint32(&g.sequence, 1)
	return uint8(seq % 256)
}

type HeartbeatMessage struct {
	VehicleType    uint8
	Autopilot      uint8
	BaseMode       uint8
	CustomMode     uint32
	SystemStatus   uint8
	MavlinkVersion uint8
}

func (g *Generator) GenerateHeartbeat(msg HeartbeatMessage) ([]byte, error) {
	payload := make([]byte, 9)
	binary.LittleEndian.PutUint32(payload[0:4], msg.CustomMode)
	payload[4] = msg.VehicleType
	payload[5] = msg.Autopilot
	payload[6] = msg.BaseMode
	payload[7] = msg.SystemStatus
	payload[8] = msg.MavlinkVersion

	return g.buildV2Frame(MessageIDHeartbeat, payload)
}

type BatteryMessage struct {
	BatteryID        uint8
	BatteryFunction  uint8
	BatteryType      uint8
	Temperature      int16
	Voltages         [10]uint16
	CurrentBattery   int16
	CurrentConsumed  int32
	EnergyConsumed   int32
	BatteryRemaining int8
}

func (g *Generator) GenerateBatteryStatus(msg BatteryMessage) ([]byte, error) {
	payload := make([]byte, 36)
	binary.LittleEndian.PutUint32(payload[0:4], uint32(msg.CurrentConsumed))
	binary.LittleEndian.PutUint32(payload[4:8], uint32(msg.EnergyConsumed))
	binary.LittleEndian.PutUint16(payload[8:10], uint16(msg.Temperature))
	for i := 0; i < len(msg.Voltages); i++ {
		binary.LittleEndian.PutUint16(payload[10+i*2:12+i*2], msg.Voltages[i])
	}
	binary.LittleEndian.PutUint16(payload[30:32], uint16(msg.CurrentBattery))
	payload[32] = msg.BatteryID
	payload[33] = msg.BatteryFunction
	payload[34] = msg.BatteryType
	payload[35] = uint8(msg.BatteryRemaining)

	return g.buildV2Frame(MessageIDBatteryStatus, payload)
}

type PositionMessage struct {
	TimeBootMs  uint32
	Lat         int32
	Lon         int32
	Alt         int32
	RelativeAlt int32
	Vx          int16
	Vy          int16
	Vz          int16
	Hdg         uint16
}

func (g *Generator) GenerateGlobalPosition(msg PositionMessage) ([]byte, error) {
	payload := make([]byte, 28)
	binary.LittleEndian.PutUint32(payload[0:4], msg.TimeBootMs)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(msg.Lat))
	binary.LittleEndian.PutUint32(payload[8:12], uint32(msg.Lon))
	binary.LittleEndian.PutUint32(payload[12:16], uint32(msg.Alt))
	binary.LittleEndian.PutUint32(payload[16:20], uint32(msg.RelativeAlt))
	binary.LittleEndian.PutUint16(payload[20:22], uint16(msg.Vx))
	binary.LittleEndian.PutUint16(payload[22:24], uint16(msg.Vy))
	binary.LittleEndian.PutUint16(payload[24:26], uint16(msg.Vz))
	binary.LittleEndian.PutUint16(payload[26:28], msg.Hdg)

	return g.buildV2Frame(MessageIDGlobalPositionInt, payload)
}

type AttitudeMessage struct {
	TimeBootMs uint32
	Roll       float32
	Pitch      float32
	Yaw        float32
	Rollspeed  float32
	Pitchspeed float32
	Yawspeed   float32
}

func (g *Generator) GenerateAttitude(msg AttitudeMessage) ([]byte, error) {
	payload := make([]byte, 28)
	binary.LittleEndian.PutUint32(payload[0:4], msg.TimeBootMs)
	binary.LittleEndian.PutUint32(payload[4:8], math.Float32bits(msg.Roll))
	binary.LittleEndian.PutUint32(payload[8:12], math.Float32bits(msg.Pitch))
	binary.LittleEndian.PutUint32(payload[12:16], math.Float32bits(msg.Yaw))
	binary.LittleEndian.PutUint32(payload[16:20], math.Float32bits(msg.Rollspeed))
	binary.LittleEndian.PutUint32(payload[20:24], math.Float32bits(msg.Pitchspeed))
	binary.LittleEndian.PutUint32(payload[24:28], math.Float32bits(msg.Yawspeed))

	return g.buildV2Frame(MessageIDAttitude, payload)
}

func (g *Generator) buildV2Frame(messageID uint32, payload []byte) ([]byte, error) {
	payloadLength := len(payload)
	if payloadLength > MaxPayloadLength {
		return nil, fmt.Errorf("payload too large: %d bytes (max %d)", payloadLength, MaxPayloadLength)
	}

	frame := make([]byte, HeaderSizeV2+payloadLength+ChecksumSize)
	frame[0] = STXV2
	frame[1] = uint8(payloadLength)
	frame[2] = 0
	frame[3] = 0
	frame[4] = g.nextSequence()
	frame[5] = g.systemID
	frame[6] = g.componentID
	frame[7] = uint8(messageID & 0xff)
	frame[8] = uint8((messageID >> 8) & 0xff)
	frame[9] = uint8((messageID >> 16) & 0xff)
	copy(frame[HeaderSizeV2:], payload)

	checksum := calculateChecksum(frame[1:HeaderSizeV2+payloadLength], messageID)
	binary.LittleEndian.PutUint16(frame[HeaderSizeV2+payloadLength:], checksum)

	return frame, nil
}

type MessageSequence struct {
	generator   *Generator
	currentTime uint32
}

func NewMessageSequence(systemID, componentID uint8) *MessageSequence {
	return &MessageSequence{generator: NewGenerator(systemID, componentID)}
}

func (ms *MessageSequence) AdvanceTime(duration time.Duration) {
	ms.currentTime += uint32(duration.Milliseconds())
}

func (ms *MessageSequence) TimeBootMs() uint32 {
	return ms.currentTime
}

func (ms *MessageSequence) GenerateRealisticHeartbeat(vehicleType uint8) ([]byte, error) {
	return ms.generator.GenerateHeartbeat(HeartbeatMessage{
		VehicleType:    vehicleType,
		Autopilot:      AutopilotArduPilotMega,
		BaseMode:       ModeFlagStabilizeEnabled | ModeFlagManualInput,
		CustomMode:     0,
		SystemStatus:   StateStandby,
		MavlinkVersion: Version2,
	})
}

func (ms *MessageSequence) GenerateRealisticBatteryStatus(batteryPercent int8) ([]byte, error) {
	percent := int(batteryPercent)
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	cellVoltage := uint16(3300 + (4200-3300)*percent/100)

	msg := BatteryMessage{
		BatteryID:        0,
		BatteryFunction:  0,
		BatteryType:      1,
		Temperature:      2500,
		CurrentBattery:   -1500,
		CurrentConsumed:  1000,
		EnergyConsumed:   144000,
		BatteryRemaining: batteryPercent,
	}
	for i := 0; i < 4; i++ {
		msg.Voltages[i] = cellVoltage + uint16((int(ms.currentTime)/100+i*7)%50)
	}

	return ms.generator.GenerateBatteryStatus(msg)
}

func (ms *MessageSequence) GenerateRealisticPosition(lat, lon float64, alt int32) ([]byte, error) {
	return ms.generator.GenerateGlobalPosition(PositionMessage{
		TimeBootMs:  ms.currentTime,
		Lat:         int32(lat * 1e7),
		Lon:         int32(lon * 1e7),
		Alt:         alt * 1000,
		RelativeAlt: alt * 1000,
		Vx:          500,
		Vy:          0,
		Vz:          0,
		Hdg:         0,
	})
}

func (ms *MessageSequence) GenerateRealisticAttitude(roll, pitch, yaw float32) ([]byte, error) {
	return ms.generator.GenerateAttitude(AttitudeMessage{
		TimeBootMs: ms.currentTime,
		Roll:       roll,
		Pitch:      pitch,
		Yaw:        yaw,
		Rollspeed:  0.01,
		Pitchspeed: 0.01,
		Yawspeed:   0.02,
	})
}

type QuadcopterScenario struct {
	sequence       *MessageSequence
	startLat       float64
	startLon       float64
	startAlt       int32
	batteryPercent int8
}

func NewQuadcopterScenario(systemID uint8, lat, lon float64, alt int32) *QuadcopterScenario {
	return &QuadcopterScenario{
		sequence:       NewMessageSequence(systemID, 1),
		startLat:       lat,
		startLon:       lon,
		startAlt:       alt,
		batteryPercent: 100,
	}
}

func (qs *QuadcopterScenario) NextHeartbeat() ([]byte, error) {
	return qs.sequence.GenerateRealisticHeartbeat(TypeQuadrotor)
}

func (qs *QuadcopterScenario) NextBatteryStatus() ([]byte, error) {
	if qs.batteryPercent > 0 {
		qs.batteryPercent--
	}
	return qs.sequence.GenerateRealisticBatteryStatus(qs.batteryPercent)
}

func (qs *QuadcopterScenario) NextPosition() ([]byte, error) {
	radius := 0.001
	angle := float64(qs.sequence.currentTime) / 10000.0
	return qs.sequence.GenerateRealisticPosition(
		qs.startLat+radius*math.Sin(angle),
		qs.startLon+radius*math.Cos(angle),
		qs.startAlt,
	)
}

func (qs *QuadcopterScenario) NextAttitude() ([]byte, error) {
	return qs.sequence.GenerateRealisticAttitude(
		float32(0.05*math.Sin(float64(qs.sequence.currentTime)/1000.0)),
		float32(0.03*math.Sin(float64(qs.sequence.currentTime)/1500.0)),
		float32(float64(qs.sequence.currentTime)/10000.0),
	)
}

func (qs *QuadcopterScenario) AdvanceTime(duration time.Duration) {
	qs.sequence.AdvanceTime(duration)
}

func (qs *QuadcopterScenario) TimeBootMs() uint32 {
	return qs.sequence.TimeBootMs()
}
