//go:build ignore
// +build ignore

// Package sitl provides programmatic control of ArduPilot SITL for testing and demonstration.
// It implements MAVLink command protocol for flight control operations with proper safety
// features and timeout handling.
package sitl

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
	"unsafe"

	"github.com/c360studio/semops/pkg/processors/mavlink/constants"
	"github.com/c360studio/semops/pkg/processors/mavlink/parser"
)

// Controller provides programmatic control of ArduPilot SITL drone
type Controller struct {
	conn     net.Conn
	parser   *parser.MAVLinkParser
	shutdown chan struct{} // Signal shutdown
	done     chan struct{} // Signal completion

	// Connection parameters
	systemID     uint8
	componentID  uint8
	targetSysID  uint8 // SITL system ID (usually 1)
	targetCompID uint8 // SITL component ID (usually 1)

	// Sequence management
	sequence    uint8
	sequenceMux sync.Mutex

	// State tracking
	state    *DroneState
	stateMux sync.RWMutex

	// Command acknowledgment tracking
	commandAcks map[uint16]chan CommandResult
	ackMux      sync.Mutex

	// Message handlers
	messageHandlers map[uint32]func(*parser.MAVLinkPacket)
	handlerMux      sync.RWMutex
}

// DroneState represents the current state of the drone
type DroneState struct {
	// Connection status
	Connected     bool      `json:"connected"`
	LastHeartbeat time.Time `json:"last_heartbeat"`

	// Flight status
	Armed        bool   `json:"armed"`
	FlightMode   string `json:"flight_mode"`
	SystemStatus uint8  `json:"system_status"`

	// Position
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Altitude    float64 `json:"altitude"`     // meters MSL
	RelativeAlt float64 `json:"relative_alt"` // meters above home

	// Attitude
	Roll  float32 `json:"roll"`  // radians
	Pitch float32 `json:"pitch"` // radians
	Yaw   float32 `json:"yaw"`   // radians

	// Velocity
	VelocityX float32 `json:"velocity_x"` // m/s north
	VelocityY float32 `json:"velocity_y"` // m/s east
	VelocityZ float32 `json:"velocity_z"` // m/s down

	// Battery
	BatteryPercent int8    `json:"battery_percent"`
	BatteryVoltage float32 `json:"battery_voltage"`

	// Last update times
	LastPosition time.Time `json:"last_position"`
	LastAttitude time.Time `json:"last_attitude"`
	LastBattery  time.Time `json:"last_battery"`
}

// CommandResult represents the result of a MAVLink command
type CommandResult struct {
	Command uint16 `json:"command"`
	Result  uint8  `json:"result"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ControllerConfig provides configuration for the Controller
type ControllerConfig struct {
	Address           string        `json:"address"`             // "localhost:5760" (SITL TCP port)
	SystemID          uint8         `json:"system_id"`           // GCS system ID (255)
	ComponentID       uint8         `json:"component_id"`        // GCS component ID (1)
	TargetSystemID    uint8         `json:"target_system_id"`    // SITL system ID (1)
	TargetComponentID uint8         `json:"target_component_id"` // SITL component ID (1)
	CommandTimeout    time.Duration `json:"command_timeout"`     // Default 5s
	HeartbeatTimeout  time.Duration `json:"heartbeat_timeout"`   // Default 10s
}

// DefaultConfig returns default controller configuration for SITL
func DefaultConfig() ControllerConfig {
	return ControllerConfig{
		Address:           "localhost:5760", // SITL TCP port, not UDP
		SystemID:          255,              // Ground Control Station
		ComponentID:       1,
		TargetSystemID:    1, // SITL drone
		TargetComponentID: 1,
		CommandTimeout:    5 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
	}
}

// NewController creates a new SITL controller with the given configuration
func NewController(ctx context.Context, config ControllerConfig) (*Controller, error) {
	// Create TCP connection to SITL
	conn, err := net.Dial("tcp", config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SITL at %s: %w", config.Address, err)
	}

	controller := &Controller{
		conn:            conn,
		parser:          parser.NewMAVLinkParser(),
		shutdown:        make(chan struct{}),
		done:            make(chan struct{}),
		systemID:        config.SystemID,
		componentID:     config.ComponentID,
		targetSysID:     config.TargetSystemID,
		targetCompID:    config.TargetComponentID,
		state:           &DroneState{},
		commandAcks:     make(map[uint16]chan CommandResult),
		messageHandlers: make(map[uint32]func(*parser.MAVLinkPacket)),
	}

	// Register default message handlers
	controller.registerDefaultHandlers()

	// Start message processing with context
	go controller.processMessages(ctx)
	go controller.sendHeartbeat(ctx)
	go controller.monitorConnection(ctx)

	// Send initial heartbeat to establish communication
	controller.sendInitialHeartbeat()

	return controller, nil
}

// Close closes the controller and releases resources
func (c *Controller) Close() error {
	// Signal shutdown
	select {
	case <-c.shutdown:
		// Already closed
		return nil
	default:
		close(c.shutdown)
	}

	// Close connection to trigger read exit
	var connErr error
	if c.conn != nil {
		connErr = c.conn.Close()
	}

	// Wait for goroutines to finish with timeout
	select {
	case <-c.done:
		// Clean shutdown
	case <-time.After(5 * time.Second):
		return fmt.Errorf("shutdown timeout")
	}

	return connErr
}

// GetState returns a copy of the current drone state
func (c *Controller) GetState() DroneState {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	return *c.state
}

// IsConnected returns true if we have recent heartbeat from drone
func (c *Controller) IsConnected() bool {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	return c.state.Connected && time.Since(c.state.LastHeartbeat) < 10*time.Second
}

// nextSequence returns the next sequence number in a thread-safe manner
func (c *Controller) nextSequence() uint8 {
	c.sequenceMux.Lock()
	defer c.sequenceMux.Unlock()
	c.sequence++
	return c.sequence
}

// sendCommand sends a MAVLink COMMAND_LONG message and waits for acknowledgment
func (c *Controller) sendCommand(command uint16, params [7]float32, timeout time.Duration) (*CommandResult, error) {
	// Create command message
	payload := c.encodeCommandLong(command, params)
	frame, err := c.buildMAVLinkFrame(constants.MavlinkMsgIdCommandLong, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to build command frame: %w", err)
	}

	// Setup acknowledgment channel
	ackChan := make(chan CommandResult, 1)
	c.ackMux.Lock()
	c.commandAcks[command] = ackChan
	c.ackMux.Unlock()

	// Cleanup acknowledgment channel
	defer func() {
		c.ackMux.Lock()
		delete(c.commandAcks, command)
		c.ackMux.Unlock()
		close(ackChan)
	}()

	// Send command
	if err := c.sendFrame(frame); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Wait for acknowledgment with timeout
	select {
	case result := <-ackChan:
		return &result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("command %d timeout after %v", command, timeout)
	case <-c.shutdown:
		return nil, fmt.Errorf("controller shutting down")
	}
}

// encodeCommandLong creates a COMMAND_LONG payload
func (c *Controller) encodeCommandLong(command uint16, params [7]float32) []byte {
	// COMMAND_LONG payload: 33 bytes
	payload := make([]byte, 33)

	// param1-7 (floats, 4 bytes each) = 28 bytes
	for i, param := range params {
		binary.LittleEndian.PutUint32(payload[i*4:], binary.LittleEndian.Uint32((*[4]byte)(unsafe.Pointer(&param))[:]))
	}

	// command (uint16) at offset 28
	binary.LittleEndian.PutUint16(payload[28:], command)

	// target_system (uint8) at offset 30
	payload[30] = c.targetSysID

	// target_component (uint8) at offset 31
	payload[31] = c.targetCompID

	// confirmation (uint8) at offset 32
	payload[32] = 0

	return payload
}

// buildMAVLinkFrame creates a complete MAVLink v2 frame
func (c *Controller) buildMAVLinkFrame(msgID uint32, payload []byte) ([]byte, error) {
	payloadLen := len(payload)
	if payloadLen > constants.MavlinkMaxPayloadLen {
		return nil, fmt.Errorf("payload too large: %d bytes", payloadLen)
	}

	// MAVLink v2 frame size
	frameLen := constants.MavlinkHeaderSizeV2 + payloadLen + constants.MavlinkChecksumSize
	frame := make([]byte, frameLen)

	// Build header
	frame[0] = constants.MavlinkStxV2      // Start marker
	frame[1] = uint8(payloadLen)           // Payload length
	frame[2] = 0                           // Incompatibility flags
	frame[3] = 0                           // Compatibility flags
	frame[4] = c.nextSequence()            // Sequence
	frame[5] = c.systemID                  // System ID
	frame[6] = c.componentID               // Component ID
	frame[7] = uint8(msgID & 0xFF)         // Message ID low byte
	frame[8] = uint8((msgID >> 8) & 0xFF)  // Message ID middle byte
	frame[9] = uint8((msgID >> 16) & 0xFF) // Message ID high byte

	// Copy payload
	copy(frame[constants.MavlinkHeaderSizeV2:], payload)

	// Calculate and append CRC
	crc := c.calculateCRC(frame[:constants.MavlinkHeaderSizeV2+payloadLen], msgID)
	binary.LittleEndian.PutUint16(frame[constants.MavlinkHeaderSizeV2+payloadLen:], crc)

	return frame, nil
}

// calculateCRC computes MAVLink CRC-16 checksum
func (c *Controller) calculateCRC(data []byte, msgID uint32) uint16 {
	crc := uint16(0xFFFF)

	// Process data starting from byte 1 (skip STX)
	for i := 1; i < len(data); i++ {
		tmp := data[i] ^ uint8(crc)
		tmp ^= tmp << 4
		crc = (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
	}

	// Add CRC extra byte for COMMAND_LONG
	if msgID == constants.MavlinkMsgIdCommandLong {
		tmp := uint8(152) ^ uint8(crc) // CRC extra for COMMAND_LONG
		tmp ^= tmp << 4
		crc = (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
	}

	return crc
}

// sendFrame sends a MAVLink frame over the UDP connection
func (c *Controller) sendFrame(frame []byte) error {
	_, err := c.conn.Write(frame)
	return err
}
