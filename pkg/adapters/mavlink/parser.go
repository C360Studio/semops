package mavlink

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

var errChecksumMismatch = errors.New("mavlink checksum mismatch")

const maxBufferedBytes = 4096

// Packet is a decoded MAVLink frame with optional field-level decoding.
type Packet struct {
	Version      uint8
	Length       uint8
	Sequence     uint8
	SystemID     uint8
	ComponentID  uint8
	MessageID    uint32
	Payload      []byte
	Checksum     uint16
	Signature    []byte
	ParsedFields map[string]any
	Timestamp    time.Time
}

// Parser incrementally decodes MAVLink v1 and v2 frames from byte streams.
type Parser struct {
	buffer       []byte
	messageSpecs map[uint32]*MessageSpec
	stats        ParsingStats
}

// MessageSpec describes the fields needed for a MAVLink message projection.
type MessageSpec struct {
	ID          uint32
	Name        string
	Fields      []FieldSpec
	PayloadSize uint8
	CRCExtra    uint8
}

// FieldSpec describes a field in a MAVLink wire payload.
type FieldSpec struct {
	Name      string
	Type      string
	Size      uint8
	Offset    uint8
	ArraySize uint8
}

// ParsingStats exposes lightweight parser health counters.
type ParsingStats struct {
	TotalPackets    uint64
	ValidPackets    uint64
	InvalidPackets  uint64
	ChecksumErrors  uint64
	UnknownMessages uint64
	LastReset       time.Time
}

// NewParser returns a parser with the first SemOps COP message specs registered.
func NewParser() *Parser {
	p := &Parser{
		buffer:       make([]byte, 0, 512),
		messageSpecs: make(map[uint32]*MessageSpec),
		stats: ParsingStats{
			LastReset: time.Now(),
		},
	}
	p.registerStandardMessages()
	return p
}

// Parse appends data to the stream buffer and returns every complete valid packet.
func (p *Parser) Parse(data []byte) ([]*Packet, error) {
	if len(data) > 0 {
		p.buffer = append(p.buffer, data...)
	}

	packets := make([]*Packet, 0)
	for {
		syncOffset := findNextSync(p.buffer)
		if syncOffset < 0 {
			if len(p.buffer) > 0 {
				p.stats.InvalidPackets += uint64(len(p.buffer))
			}
			p.buffer = p.buffer[:0]
			break
		}
		if syncOffset > 0 {
			p.stats.InvalidPackets += uint64(syncOffset)
			p.buffer = p.buffer[syncOffset:]
		}

		packet, consumed, complete, err := p.parsePacket(p.buffer)
		if !complete {
			break
		}
		if err != nil {
			p.stats.InvalidPackets++
			if errors.Is(err, errChecksumMismatch) {
				p.stats.ChecksumErrors++
			}
			p.buffer = p.buffer[1:]
			continue
		}

		packets = append(packets, packet)
		p.stats.ValidPackets++
		p.buffer = p.buffer[consumed:]
	}

	if len(p.buffer) > maxBufferedBytes {
		p.stats.InvalidPackets += uint64(len(p.buffer))
		p.buffer = p.buffer[:0]
	}

	p.stats.TotalPackets += uint64(len(packets))
	return packets, nil
}

func (p *Parser) parsePacket(buffer []byte) (*Packet, int, bool, error) {
	if len(buffer) < 2 {
		return nil, 0, false, nil
	}

	switch buffer[0] {
	case STXV1:
		return p.parseV1Packet(buffer)
	case STXV2:
		return p.parseV2Packet(buffer)
	default:
		return nil, 1, true, fmt.Errorf("invalid sync byte: 0x%02x", buffer[0])
	}
}

func (p *Parser) parseV1Packet(buffer []byte) (*Packet, int, bool, error) {
	if len(buffer) < HeaderSizeV1 {
		return nil, 0, false, nil
	}

	payloadLength := buffer[1]
	totalLength := HeaderSizeV1 + int(payloadLength) + ChecksumSize
	if len(buffer) < totalLength {
		return nil, 0, false, nil
	}

	messageID := uint32(buffer[5])
	payloadStart := HeaderSizeV1
	checksumOffset := payloadStart + int(payloadLength)
	checksum := binary.LittleEndian.Uint16(buffer[checksumOffset : checksumOffset+ChecksumSize])
	calculated := p.calculateChecksum(buffer[1:checksumOffset], messageID)
	if checksum != calculated {
		return nil, 1, true, fmt.Errorf("%w: expected 0x%04x got 0x%04x", errChecksumMismatch, calculated, checksum)
	}

	packet := &Packet{
		Version:     Version1,
		Length:      payloadLength,
		Sequence:    buffer[2],
		SystemID:    buffer[3],
		ComponentID: buffer[4],
		MessageID:   messageID,
		Payload:     append([]byte(nil), buffer[payloadStart:checksumOffset]...),
		Checksum:    checksum,
		Timestamp:   time.Now(),
	}
	p.parseKnownFields(packet)

	return packet, totalLength, true, nil
}

func (p *Parser) parseV2Packet(buffer []byte) (*Packet, int, bool, error) {
	if len(buffer) < HeaderSizeV2 {
		return nil, 0, false, nil
	}

	payloadLength := buffer[1]
	incompatFlags := buffer[2]
	messageID := uint32(buffer[7]) | uint32(buffer[8])<<8 | uint32(buffer[9])<<16

	totalLength := HeaderSizeV2 + int(payloadLength) + ChecksumSize
	hasSignature := incompatFlags&0x01 != 0
	if hasSignature {
		totalLength += SignatureSize
	}
	if len(buffer) < totalLength {
		return nil, 0, false, nil
	}

	payloadStart := HeaderSizeV2
	checksumOffset := payloadStart + int(payloadLength)
	checksum := binary.LittleEndian.Uint16(buffer[checksumOffset : checksumOffset+ChecksumSize])
	calculated := p.calculateChecksum(buffer[1:checksumOffset], messageID)
	if checksum != calculated {
		return nil, 1, true, fmt.Errorf("%w: expected 0x%04x got 0x%04x", errChecksumMismatch, calculated, checksum)
	}

	packet := &Packet{
		Version:     Version2,
		Length:      payloadLength,
		Sequence:    buffer[4],
		SystemID:    buffer[5],
		ComponentID: buffer[6],
		MessageID:   messageID,
		Payload:     append([]byte(nil), buffer[payloadStart:checksumOffset]...),
		Checksum:    checksum,
		Timestamp:   time.Now(),
	}
	if hasSignature {
		signatureStart := checksumOffset + ChecksumSize
		packet.Signature = append([]byte(nil), buffer[signatureStart:signatureStart+SignatureSize]...)
	}
	p.parseKnownFields(packet)

	return packet, totalLength, true, nil
}

func (p *Parser) parseKnownFields(packet *Packet) {
	spec, ok := p.messageSpecs[packet.MessageID]
	if !ok {
		p.stats.UnknownMessages++
		return
	}

	fields := make(map[string]any, len(spec.Fields))
	for _, field := range spec.Fields {
		end := int(field.Offset + field.Size)
		if end > len(packet.Payload) {
			continue
		}
		value, err := parseFieldValue(packet.Payload[field.Offset:end], field)
		if err != nil {
			continue
		}
		fields[field.Name] = value
	}
	packet.ParsedFields = fields
}

func parseFieldValue(data []byte, field FieldSpec) (any, error) {
	if field.ArraySize > 1 {
		return parseArrayField(data, field)
	}

	switch field.Type {
	case "uint8":
		if len(data) < 1 {
			return nil, fmt.Errorf("field %q needs 1 byte", field.Name)
		}
		return data[0], nil
	case "int8":
		if len(data) < 1 {
			return nil, fmt.Errorf("field %q needs 1 byte", field.Name)
		}
		return int8(data[0]), nil
	case "uint16":
		if len(data) < 2 {
			return nil, fmt.Errorf("field %q needs 2 bytes", field.Name)
		}
		return binary.LittleEndian.Uint16(data), nil
	case "int16":
		if len(data) < 2 {
			return nil, fmt.Errorf("field %q needs 2 bytes", field.Name)
		}
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint32":
		if len(data) < 4 {
			return nil, fmt.Errorf("field %q needs 4 bytes", field.Name)
		}
		return binary.LittleEndian.Uint32(data), nil
	case "int32":
		if len(data) < 4 {
			return nil, fmt.Errorf("field %q needs 4 bytes", field.Name)
		}
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint64":
		if len(data) < 8 {
			return nil, fmt.Errorf("field %q needs 8 bytes", field.Name)
		}
		return binary.LittleEndian.Uint64(data), nil
	case "int64":
		if len(data) < 8 {
			return nil, fmt.Errorf("field %q needs 8 bytes", field.Name)
		}
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "float":
		if len(data) < 4 {
			return nil, fmt.Errorf("field %q needs 4 bytes", field.Name)
		}
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "double":
		if len(data) < 8 {
			return nil, fmt.Errorf("field %q needs 8 bytes", field.Name)
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	case "char":
		for i, b := range data {
			if b == 0 {
				return string(data[:i]), nil
			}
		}
		return string(data), nil
	default:
		return nil, fmt.Errorf("unsupported field type %q", field.Type)
	}
}

func parseArrayField(data []byte, field FieldSpec) (any, error) {
	switch field.Type {
	case "uint8":
		if len(data) < int(field.ArraySize) {
			return nil, fmt.Errorf("field %q needs %d bytes", field.Name, field.ArraySize)
		}
		values := make([]uint8, int(field.ArraySize))
		copy(values, data)
		return values, nil
	case "uint16":
		requiredBytes := int(field.ArraySize) * 2
		if len(data) < requiredBytes {
			return nil, fmt.Errorf("field %q needs %d bytes", field.Name, requiredBytes)
		}
		values := make([]uint16, int(field.ArraySize))
		for i := range values {
			values[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported array field type %q", field.Type)
	}
}

func findNextSync(buffer []byte) int {
	for i, b := range buffer {
		if b == STXV1 || b == STXV2 {
			return i
		}
	}
	return -1
}

func (p *Parser) registerStandardMessages() {
	heartbeatCRC, _ := standardCRCExtra(MessageIDHeartbeat)
	p.RegisterMessageSpec(&MessageSpec{
		ID:          MessageIDHeartbeat,
		Name:        "HEARTBEAT",
		PayloadSize: 9,
		CRCExtra:    heartbeatCRC,
		Fields: []FieldSpec{
			{Name: "custom_mode", Type: "uint32", Size: 4, Offset: 0},
			{Name: "type", Type: "uint8", Size: 1, Offset: 4},
			{Name: "autopilot", Type: "uint8", Size: 1, Offset: 5},
			{Name: "base_mode", Type: "uint8", Size: 1, Offset: 6},
			{Name: "system_status", Type: "uint8", Size: 1, Offset: 7},
			{Name: "mavlink_version", Type: "uint8", Size: 1, Offset: 8},
		},
	})

	globalPositionCRC, _ := standardCRCExtra(MessageIDGlobalPositionInt)
	p.RegisterMessageSpec(&MessageSpec{
		ID:          MessageIDGlobalPositionInt,
		Name:        "GLOBAL_POSITION_INT",
		PayloadSize: 28,
		CRCExtra:    globalPositionCRC,
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
	})

	attitudeCRC, _ := standardCRCExtra(MessageIDAttitude)
	p.RegisterMessageSpec(&MessageSpec{
		ID:          MessageIDAttitude,
		Name:        "ATTITUDE",
		PayloadSize: 28,
		CRCExtra:    attitudeCRC,
		Fields: []FieldSpec{
			{Name: "time_boot_ms", Type: "uint32", Size: 4, Offset: 0},
			{Name: "roll", Type: "float", Size: 4, Offset: 4},
			{Name: "pitch", Type: "float", Size: 4, Offset: 8},
			{Name: "yaw", Type: "float", Size: 4, Offset: 12},
			{Name: "rollspeed", Type: "float", Size: 4, Offset: 16},
			{Name: "pitchspeed", Type: "float", Size: 4, Offset: 20},
			{Name: "yawspeed", Type: "float", Size: 4, Offset: 24},
		},
	})

	batteryStatusCRC, _ := standardCRCExtra(MessageIDBatteryStatus)
	p.RegisterMessageSpec(&MessageSpec{
		ID:          MessageIDBatteryStatus,
		Name:        "BATTERY_STATUS",
		PayloadSize: 36,
		CRCExtra:    batteryStatusCRC,
		Fields: []FieldSpec{
			{Name: "current_consumed", Type: "int32", Size: 4, Offset: 0},
			{Name: "energy_consumed", Type: "int32", Size: 4, Offset: 4},
			{Name: "temperature", Type: "int16", Size: 2, Offset: 8},
			{Name: "voltages", Type: "uint16", Size: 20, Offset: 10, ArraySize: 10},
			{Name: "current_battery", Type: "int16", Size: 2, Offset: 30},
			{Name: "id", Type: "uint8", Size: 1, Offset: 32},
			{Name: "battery_function", Type: "uint8", Size: 1, Offset: 33},
			{Name: "type", Type: "uint8", Size: 1, Offset: 34},
			{Name: "battery_remaining", Type: "int8", Size: 1, Offset: 35},
		},
	})
}

// Stats returns a copy of the parser counters.
func (p *Parser) Stats() ParsingStats {
	return p.stats
}

// ResetStats resets parser counters without removing registered message specs.
func (p *Parser) ResetStats() {
	p.stats = ParsingStats{LastReset: time.Now()}
}

// RegisterMessageSpec adds or replaces a message specification.
func (p *Parser) RegisterMessageSpec(spec *MessageSpec) {
	if spec == nil {
		return
	}
	p.messageSpecs[spec.ID] = spec
}

func (p *Parser) calculateChecksum(dataWithoutSTX []byte, messageID uint32) uint16 {
	spec, ok := p.messageSpecs[messageID]
	if ok {
		return calculateChecksumWithExtra(dataWithoutSTX, spec.CRCExtra, true)
	}
	return calculateChecksum(dataWithoutSTX, messageID)
}

func (p *Parser) String() string {
	return fmt.Sprintf("Parser[specs=%d valid=%d invalid=%d unknown=%d]",
		len(p.messageSpecs), p.stats.ValidPackets, p.stats.InvalidPackets, p.stats.UnknownMessages)
}
