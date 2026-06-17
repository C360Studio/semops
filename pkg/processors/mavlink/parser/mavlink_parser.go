//go:build ignore
// +build ignore

package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/c360studio/semops/pkg/processors/mavlink/constants"
)

// MAVLinkPacket represents a parsed MAVLink packet
type MAVLinkPacket struct {
	Version      uint8          `json:"version"`                 // MAVLink version (1 or 2)
	Length       uint8          `json:"length"`                  // Payload length
	Sequence     uint8          `json:"sequence"`                // Packet sequence number
	SystemID     uint8          `json:"system_id"`               // System ID
	ComponentID  uint8          `json:"component_id"`            // Component ID
	MessageID    uint32         `json:"message_id"`              // Message ID (24-bit in v1, 32-bit in v2)
	Payload      []byte         `json:"payload"`                 // Raw payload data
	Checksum     uint16         `json:"checksum"`                // Packet checksum
	Signature    []byte         `json:"signature,omitempty"`     // v2 signature (13 bytes)
	ParsedFields map[string]any `json:"parsed_fields,omitempty"` // Parsed field data
	Timestamp    time.Time      `json:"timestamp"`               // Parse timestamp
}

// MAVLinkParser parses MAVLink binary protocol messages
type MAVLinkParser struct {
	buffer        []byte
	checksumTable [256]uint16             // CRC-16-CCITT table
	messageSpecs  map[uint32]*MessageSpec // Message specifications
	parsingStats  ParsingStats
}

// MessageSpec defines the structure of a MAVLink message
type MessageSpec struct {
	ID          uint32      `json:"id"`
	Name        string      `json:"name"`
	Fields      []FieldSpec `json:"fields"`
	PayloadSize uint8       `json:"payload_size"`
	CRCExtra    uint8       `json:"crc_extra"`
}

// FieldSpec defines a field within a MAVLink message
type FieldSpec struct {
	Name      string `json:"name"`
	Type      string `json:"type"`                 // "uint8", "int16", "float", etc.
	Size      uint8  `json:"size"`                 // Size in bytes
	Offset    uint8  `json:"offset"`               // Offset in payload
	ArraySize uint8  `json:"array_size,omitempty"` // For array fields
}

// ParsingStats tracks parser performance
type ParsingStats struct {
	TotalPackets    uint64    `json:"total_packets"`
	ValidPackets    uint64    `json:"valid_packets"`
	InvalidPackets  uint64    `json:"invalid_packets"`
	ChecksumErrors  uint64    `json:"checksum_errors"`
	UnknownMessages uint64    `json:"unknown_messages"`
	LastReset       time.Time `json:"last_reset"`
}

// NewMAVLinkParser creates a new MAVLink parser
func NewMAVLinkParser() *MAVLinkParser {
	parser := &MAVLinkParser{
		buffer:       make([]byte, 0, 512),
		messageSpecs: make(map[uint32]*MessageSpec),
		parsingStats: ParsingStats{
			LastReset: time.Now(),
		},
	}

	// Initialize CRC table
	parser.initCRCTable()

	// Register standard message specifications
	parser.registerStandardMessages()

	return parser
}

// Parse processes incoming binary data and extracts MAVLink messages
func (p *MAVLinkParser) Parse(data []byte) ([]*MAVLinkPacket, error) {
	// Append new data to buffer
	p.buffer = append(p.buffer, data...)

	packets := make([]*MAVLinkPacket, 0)

	// Process buffer for complete packets
	for {
		packet, bytesConsumed, err := p.parsePacket(p.buffer)
		if err != nil {
			// Try to find next sync byte and continue
			if syncOffset := p.findNextSync(p.buffer); syncOffset > 0 {
				p.buffer = p.buffer[syncOffset:]
				continue
			}
			break
		}

		if packet == nil {
			break // Need more data
		}

		packets = append(packets, packet)

		// SAFETY: Prevent slice bounds panic if bytesConsumed is larger than buffer
		// This can happen with corrupted MAVLink data where payload length is incorrect
		if bytesConsumed > len(p.buffer) {
			// Clear the corrupted buffer and stop processing
			// The parser will attempt to resync on the next Parse() call
			p.parsingStats.InvalidPackets++
			p.buffer = p.buffer[:0]
			break
		}

		p.buffer = p.buffer[bytesConsumed:]
		p.parsingStats.ValidPackets++
	}

	// Prevent buffer from growing too large
	if len(p.buffer) > 1024 {
		p.buffer = p.buffer[:0]
	}

	p.parsingStats.TotalPackets += uint64(len(packets))
	return packets, nil
}

// parsePacket attempts to parse a single MAVLink packet from buffer
func (p *MAVLinkParser) parsePacket(buffer []byte) (*MAVLinkPacket, int, error) {
	if len(buffer) < 8 {
		return nil, 0, nil // Need more data
	}

	// Check for MAVLink sync byte
	if buffer[0] != constants.MavlinkStxV1 && buffer[0] != constants.MavlinkStxV2 {
		return nil, 0, fmt.Errorf("invalid sync byte: 0x%02X", buffer[0])
	}

	version := constants.MavlinkV1
	if buffer[0] == constants.MavlinkStxV2 {
		version = constants.MavlinkV2
	}

	// Parse header based on version
	if version == constants.MavlinkV1 {
		return p.parseV1Packet(buffer)
	}
	return p.parseV2Packet(buffer)
}

// parseV1Packet parses a MAVLink v1 packet
func (p *MAVLinkParser) parseV1Packet(buffer []byte) (*MAVLinkPacket, int, error) {
	if len(buffer) < constants.MavlinkHeaderSizeV1 {
		return nil, 0, nil // Need more data
	}

	payloadLength := buffer[1]
	sequence := buffer[2]
	systemID := buffer[3]
	componentID := buffer[4]
	messageID := uint32(buffer[5])

	totalLength := constants.MavlinkHeaderSizeV1 + int(payloadLength) + constants.MavlinkChecksumSize

	if len(buffer) < totalLength {
		return nil, 0, nil // Need more data
	}

	// Extract payload
	payload := buffer[6 : 6+payloadLength]

	// Extract checksum
	checksumBytes := buffer[6+payloadLength : 6+payloadLength+2]
	checksum := binary.LittleEndian.Uint16(checksumBytes)

	// Verify checksum
	calculatedChecksum := p.calculateChecksum(buffer[1:6+payloadLength], messageID)
	if checksum != calculatedChecksum {
		p.parsingStats.ChecksumErrors++
		return nil, 1, fmt.Errorf("checksum mismatch: expected 0x%04X, got 0x%04X", calculatedChecksum, checksum)
	}

	packet := &MAVLinkPacket{
		Version:     constants.MavlinkV1,
		Length:      payloadLength,
		Sequence:    sequence,
		SystemID:    systemID,
		ComponentID: componentID,
		MessageID:   messageID,
		Payload:     payload,
		Checksum:    checksum,
		Timestamp:   time.Now(),
	}

	// Parse payload fields if message spec is known
	if spec, exists := p.messageSpecs[messageID]; exists {
		fields, err := p.parsePayloadFields(payload, spec)
		if err == nil {
			packet.ParsedFields = fields
		}
	} else {
		p.parsingStats.UnknownMessages++
	}

	return packet, totalLength, nil
}

// parseV2Packet parses a MAVLink v2 packet
func (p *MAVLinkParser) parseV2Packet(buffer []byte) (*MAVLinkPacket, int, error) {
	if len(buffer) < constants.MavlinkHeaderSizeV2 {
		return nil, 0, nil // Need more data
	}

	payloadLength := buffer[1]
	incompatFlags := buffer[2]
	_ = buffer[3] // compatFlags - reserved for future use
	sequence := buffer[4]
	systemID := buffer[5]
	componentID := buffer[6]

	// Message ID is 24-bit in v2
	messageID := uint32(buffer[7]) | (uint32(buffer[8]) << 8) | (uint32(buffer[9]) << 16)

	totalLength := constants.MavlinkHeaderSizeV2 + int(payloadLength) + constants.MavlinkChecksumSize

	// Check for signature (indicated by flag)
	hasSignature := (incompatFlags & 0x01) != 0
	if hasSignature {
		totalLength += constants.MavlinkSignatureSize
	}

	if len(buffer) < totalLength {
		return nil, 0, nil // Need more data
	}

	// Extract payload
	payload := buffer[10 : 10+payloadLength]

	// Extract checksum
	checksumOffset := 10 + int(payloadLength)
	checksumBytes := buffer[checksumOffset : checksumOffset+2]
	checksum := binary.LittleEndian.Uint16(checksumBytes)

	// Extract signature if present
	var signature []byte
	if hasSignature {
		signatureOffset := checksumOffset + 2
		signature = buffer[signatureOffset : signatureOffset+constants.MavlinkSignatureSize]
	}

	// Verify checksum
	calculatedChecksum := p.calculateChecksum(buffer[1:checksumOffset], messageID)
	if checksum != calculatedChecksum {
		p.parsingStats.ChecksumErrors++
		return nil, 1, fmt.Errorf("checksum mismatch: expected 0x%04X, got 0x%04X", calculatedChecksum, checksum)
	}

	packet := &MAVLinkPacket{
		Version:     constants.MavlinkV2,
		Length:      payloadLength,
		Sequence:    sequence,
		SystemID:    systemID,
		ComponentID: componentID,
		MessageID:   messageID,
		Payload:     payload,
		Checksum:    checksum,
		Signature:   signature,
		Timestamp:   time.Now(),
	}

	// Parse payload fields if message spec is known
	if spec, exists := p.messageSpecs[messageID]; exists {
		fields, err := p.parsePayloadFields(payload, spec)
		if err == nil {
			packet.ParsedFields = fields
		}
	} else {
		p.parsingStats.UnknownMessages++
	}

	return packet, totalLength, nil
}

// parsePayloadFields parses payload data according to message specification
func (p *MAVLinkParser) parsePayloadFields(payload []byte, spec *MessageSpec) (map[string]any, error) {
	fields := make(map[string]any)

	for _, fieldSpec := range spec.Fields {
		if int(fieldSpec.Offset+fieldSpec.Size) > len(payload) {
			continue // Skip fields that extend beyond payload
		}

		fieldData := payload[fieldSpec.Offset : fieldSpec.Offset+fieldSpec.Size]
		value, err := p.parseFieldValue(fieldData, fieldSpec)
		if err != nil {
			continue // Skip invalid fields
		}

		fields[fieldSpec.Name] = value
	}

	return fields, nil
}

// parseFieldValue parses a field value based on its type
func (p *MAVLinkParser) parseFieldValue(data []byte, fieldSpec FieldSpec) (any, error) {
	buf := bytes.NewReader(data)

	switch fieldSpec.Type {
	case "uint8":
		if fieldSpec.ArraySize > 1 {
			values := make([]uint8, fieldSpec.ArraySize)
			for i := 0; i < int(fieldSpec.ArraySize); i++ {
				if err := binary.Read(buf, binary.LittleEndian, &values[i]); err != nil {
					return nil, err
				}
			}
			return values, nil
		}
		var value uint8
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "int8":
		var value int8
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "uint16":
		if fieldSpec.ArraySize > 1 {
			values := make([]uint16, fieldSpec.ArraySize)
			for i := 0; i < int(fieldSpec.ArraySize); i++ {
				if err := binary.Read(buf, binary.LittleEndian, &values[i]); err != nil {
					return nil, err
				}
			}
			return values, nil
		}
		var value uint16
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "int16":
		var value int16
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "uint32":
		var value uint32
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "int32":
		var value int32
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "uint64":
		var value uint64
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "int64":
		var value int64
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "float":
		var value float32
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "double":
		var value float64
		err := binary.Read(buf, binary.LittleEndian, &value)
		return value, err

	case "char":
		// Handle string fields
		chars := make([]byte, fieldSpec.Size)
		n, err := buf.Read(chars)
		if err != nil || n != int(fieldSpec.Size) {
			return "", err
		}
		// Find null terminator
		nullIndex := bytes.IndexByte(chars, 0)
		if nullIndex >= 0 {
			chars = chars[:nullIndex]
		}
		return string(chars), nil

	default:
		return nil, fmt.Errorf("unsupported field type: %s", fieldSpec.Type)
	}
}

// findNextSync finds the next MAVLink sync byte in buffer
func (p *MAVLinkParser) findNextSync(buffer []byte) int {
	for i := 1; i < len(buffer); i++ {
		if buffer[i] == constants.MavlinkStxV1 || buffer[i] == constants.MavlinkStxV2 {
			return i
		}
	}
	return -1
}

// calculateChecksum calculates MAVLink CRC-16-CCITT checksum
func (p *MAVLinkParser) calculateChecksum(data []byte, messageID uint32) uint16 {
	crc := uint16(0xFFFF)

	// Include data in checksum
	for _, b := range data {
		crc = p.crcAccumulate(b, crc)
	}

	// Include CRC extra byte if message spec is known
	if spec, exists := p.messageSpecs[messageID]; exists {
		crc = p.crcAccumulate(spec.CRCExtra, crc)
	}

	return crc
}

// crcAccumulate accumulates CRC with one byte
func (p *MAVLinkParser) crcAccumulate(b byte, crc uint16) uint16 {
	tmp := b ^ uint8(crc&0xFF)
	tmp ^= tmp << 4
	return ((crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4))
}

// initCRCTable initializes the CRC lookup table
func (p *MAVLinkParser) initCRCTable() {
	// This is a simplified implementation
	// In practice, you'd want the full CRC-16-CCITT table
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc = crc << 1
			}
		}
		p.checksumTable[i] = crc
	}
}

// registerStandardMessages registers common MAVLink message specifications
func (p *MAVLinkParser) registerStandardMessages() {
	// Heartbeat message
	p.messageSpecs[constants.MavlinkMsgIdHeartbeat] = &MessageSpec{
		ID:          constants.MavlinkMsgIdHeartbeat,
		Name:        "HEARTBEAT",
		PayloadSize: 9,
		CRCExtra:    50,
		Fields: []FieldSpec{
			{Name: "custom_mode", Type: "uint32", Size: 4, Offset: 0},
			{Name: "type", Type: "uint8", Size: 1, Offset: 4},
			{Name: "autopilot", Type: "uint8", Size: 1, Offset: 5},
			{Name: "base_mode", Type: "uint8", Size: 1, Offset: 6},
			{Name: "system_status", Type: "uint8", Size: 1, Offset: 7},
			{Name: "mavlink_version", Type: "uint8", Size: 1, Offset: 8},
		},
	}

	// Global position message
	p.messageSpecs[constants.MavlinkMsgIdGlobalPositionInt] = &MessageSpec{
		ID:          constants.MavlinkMsgIdGlobalPositionInt,
		Name:        "GLOBAL_POSITION_INT",
		PayloadSize: 28,
		CRCExtra:    104,
		Fields: []FieldSpec{
			{Name: "time_boot_ms", Type: "uint32", Size: 4, Offset: 0},
			{Name: "lat", Type: "int32", Size: 4, Offset: 4},
			{Name: "lon", Type: "int32", Size: 4, Offset: 8},
			{Name: "alt", Type: "int32", Size: 4, Offset: 12},
			{Name: "relative_alt", Type: "int32", Size: 4, Offset: 16},
			{Name: "vx", Type: "int16", Size: 2, Offset: 20},
			{Name: "vy", Type: "int16", Size: 2, Offset: 22},
			{Name: "vz", Type: "int16", Size: 2, Offset: 24},
			{Name: "hdg", Type: "uint16", Size: 2, Offset: 26},
		},
	}

	// Attitude message
	p.messageSpecs[constants.MavlinkMsgIdAttitude] = &MessageSpec{
		ID:          constants.MavlinkMsgIdAttitude,
		Name:        "ATTITUDE",
		PayloadSize: 28,
		CRCExtra:    39,
		Fields: []FieldSpec{
			{Name: "time_boot_ms", Type: "uint32", Size: 4, Offset: 0},
			{Name: "roll", Type: "float", Size: 4, Offset: 4},
			{Name: "pitch", Type: "float", Size: 4, Offset: 8},
			{Name: "yaw", Type: "float", Size: 4, Offset: 12},
			{Name: "rollspeed", Type: "float", Size: 4, Offset: 16},
			{Name: "pitchspeed", Type: "float", Size: 4, Offset: 20},
			{Name: "yawspeed", Type: "float", Size: 4, Offset: 24},
		},
	}

	// Battery status message
	p.messageSpecs[constants.MavlinkMsgIdBatteryStatus] = &MessageSpec{
		ID:          constants.MavlinkMsgIdBatteryStatus,
		Name:        "BATTERY_STATUS",
		PayloadSize: 36,
		CRCExtra:    154,
		Fields: []FieldSpec{
			{Name: "id", Type: "uint8", Size: 1, Offset: 0},
			{Name: "battery_function", Type: "uint8", Size: 1, Offset: 1},
			{Name: "type", Type: "uint8", Size: 1, Offset: 2},
			{Name: "temperature", Type: "int16", Size: 2, Offset: 3},
			{Name: "voltages", Type: "uint16", Size: 20, Offset: 5, ArraySize: 10},
			{Name: "current_battery", Type: "int16", Size: 2, Offset: 25},
			{Name: "current_consumed", Type: "int32", Size: 4, Offset: 27},
			{Name: "energy_consumed", Type: "int32", Size: 4, Offset: 31},
			{Name: "battery_remaining", Type: "int8", Size: 1, Offset: 35},
		},
	}
}

// GetStats returns parser statistics
func (p *MAVLinkParser) GetStats() ParsingStats {
	return p.parsingStats
}

// ResetStats resets parser statistics
func (p *MAVLinkParser) ResetStats() {
	p.parsingStats = ParsingStats{
		LastReset: time.Now(),
	}
}

// RegisterMessageSpec registers a new message specification
func (p *MAVLinkParser) RegisterMessageSpec(spec *MessageSpec) {
	p.messageSpecs[spec.ID] = spec
}

// String returns string representation of the parser
func (p *MAVLinkParser) String() string {
	return fmt.Sprintf("MAVLinkParser[specs=%d valid=%d invalid=%d unknown=%d]",
		len(p.messageSpecs), p.parsingStats.ValidPackets, p.parsingStats.InvalidPackets, p.parsingStats.UnknownMessages)
}
