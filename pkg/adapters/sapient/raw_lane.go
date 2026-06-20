package sapient

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

type Encoding string

const (
	EncodingJSON     Encoding = "json"
	EncodingProtobuf Encoding = "protobuf"
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
	records    []RawMessageRecord
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
		records:    make([]RawMessageRecord, 0, cfg.MaxRecords),
	}
}

func (l *RawLane) Capture(raw []byte, encoding Encoding, msg *Message) (RawMessageRecord, error) {
	if l == nil {
		return RawMessageRecord{}, fmt.Errorf("sapient raw lane is nil")
	}
	if len(raw) == 0 {
		return RawMessageRecord{}, fmt.Errorf("sapient raw message is empty")
	}
	if !encoding.Valid() {
		return RawMessageRecord{}, fmt.Errorf("sapient raw encoding %q is unsupported", encoding)
	}
	if len(raw) > l.maxBytes {
		return RawMessageRecord{}, fmt.Errorf("sapient raw message is %d bytes; max lane bytes is %d", len(raw), l.maxBytes)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.next++
	record := RawMessageRecord{
		Ref:        fmt.Sprintf("sapient://raw/%s/%s/%08d", l.source, encoding, l.next),
		Source:     l.source,
		ReceivedAt: l.clock().UTC(),
		Encoding:   encoding,
		RawPayload: append([]byte(nil), raw...),
	}
	if msg != nil {
		record.Content = msg.Content
		record.NodeID = msg.NodeID
		record.MessageAt = msg.Timestamp
		if msg.DestinationID != nil {
			record.DestinationID = *msg.DestinationID
		}
	}

	l.records = append(l.records, record)
	l.totalBytes += len(record.RawPayload)
	l.evict()
	return cloneRawMessageRecord(record), nil
}

func (l *RawLane) Snapshot() []RawMessageRecord {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]RawMessageRecord, len(l.records))
	for i, record := range l.records {
		out[i] = cloneRawMessageRecord(record)
	}
	return out
}

func (l *RawLane) Get(ref string) (RawMessageRecord, bool) {
	if l == nil {
		return RawMessageRecord{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, record := range l.records {
		if record.Ref == ref {
			return cloneRawMessageRecord(record), true
		}
	}
	return RawMessageRecord{}, false
}

func (l *RawLane) evict() {
	for len(l.records) > l.maxRecords || l.totalBytes > l.maxBytes {
		l.totalBytes -= len(l.records[0].RawPayload)
		copy(l.records, l.records[1:])
		l.records = l.records[:len(l.records)-1]
	}
}

func (e Encoding) Valid() bool {
	switch e {
	case EncodingJSON, EncodingProtobuf:
		return true
	default:
		return false
	}
}

func cloneRawMessageRecord(record RawMessageRecord) RawMessageRecord {
	record.RawPayload = append([]byte(nil), record.RawPayload...)
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
