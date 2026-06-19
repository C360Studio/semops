package mavlink

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	DefaultRawLaneMaxRecords = 1024
	DefaultRawLaneMaxBytes   = 8 * 1024 * 1024
)

type RawLaneConfig struct {
	Source     string
	MaxRecords int
	MaxBytes   int
	Clock      func() time.Time
}

type RawLane struct {
	mu         sync.Mutex
	source     string
	maxRecords int
	maxBytes   int
	clock      func() time.Time
	next       uint64
	totalBytes int
	records    []RawFrameRecord
}

type RawFrameRecord struct {
	Ref         string
	Source      string
	ReceivedAt  time.Time
	Version     uint8
	Sequence    uint8
	SystemID    uint8
	ComponentID uint8
	MessageID   uint32
	Checksum    uint16
	Frame       []byte
}

func NewRawLane(cfg RawLaneConfig) *RawLane {
	if cfg.Source == "" {
		cfg.Source = "default"
	}
	if cfg.MaxRecords <= 0 {
		cfg.MaxRecords = DefaultRawLaneMaxRecords
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = DefaultRawLaneMaxBytes
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &RawLane{
		source:     sanitizeSourceToken(cfg.Source),
		maxRecords: cfg.MaxRecords,
		maxBytes:   cfg.MaxBytes,
		clock:      cfg.Clock,
		records:    make([]RawFrameRecord, 0, cfg.MaxRecords),
	}
}

func (l *RawLane) Capture(frame []byte, packet *Packet) (RawFrameRecord, error) {
	if l == nil {
		return RawFrameRecord{}, fmt.Errorf("mavlink raw lane is nil")
	}
	if len(frame) == 0 {
		return RawFrameRecord{}, fmt.Errorf("mavlink raw frame is empty")
	}
	if len(frame) > l.maxBytes {
		return RawFrameRecord{}, fmt.Errorf("mavlink raw frame is %d bytes; max lane bytes is %d", len(frame), l.maxBytes)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.next++
	record := RawFrameRecord{
		Ref:        fmt.Sprintf("mavlink://raw/%s/%08d", l.source, l.next),
		Source:     l.source,
		ReceivedAt: l.clock().UTC(),
		Frame:      append([]byte(nil), frame...),
	}
	if packet != nil {
		record.Version = packet.Version
		record.Sequence = packet.Sequence
		record.SystemID = packet.SystemID
		record.ComponentID = packet.ComponentID
		record.MessageID = packet.MessageID
		record.Checksum = packet.Checksum
		packet.SourceRef = record.Ref
	}

	l.records = append(l.records, record)
	l.totalBytes += len(record.Frame)
	l.evict()
	return cloneRawFrameRecord(record), nil
}

func (l *RawLane) Snapshot() []RawFrameRecord {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]RawFrameRecord, len(l.records))
	for i, record := range l.records {
		out[i] = cloneRawFrameRecord(record)
	}
	return out
}

func (l *RawLane) Get(ref string) (RawFrameRecord, bool) {
	if l == nil {
		return RawFrameRecord{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, record := range l.records {
		if record.Ref == ref {
			return cloneRawFrameRecord(record), true
		}
	}
	return RawFrameRecord{}, false
}

func (l *RawLane) evict() {
	for len(l.records) > l.maxRecords || l.totalBytes > l.maxBytes {
		l.totalBytes -= len(l.records[0].Frame)
		copy(l.records, l.records[1:])
		l.records = l.records[:len(l.records)-1]
	}
}

func cloneRawFrameRecord(record RawFrameRecord) RawFrameRecord {
	record.Frame = append([]byte(nil), record.Frame...)
	return record
}

func sanitizeSourceToken(source string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range source {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' {
			builder.WriteRune(unicode.ToLower(r))
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "default"
	}
	return token
}
