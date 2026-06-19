package cot

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
	records    []RawEventRecord
}

type RawEventRecord struct {
	Ref        string
	Source     string
	ReceivedAt time.Time
	UID        string
	Type       string
	Callsign   string
	StaleAt    time.Time
	RawXML     []byte
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
		records:    make([]RawEventRecord, 0, cfg.MaxRecords),
	}
}

func (l *RawLane) Capture(raw []byte, event *Event) (RawEventRecord, error) {
	if l == nil {
		return RawEventRecord{}, fmt.Errorf("cot raw lane is nil")
	}
	if len(raw) == 0 {
		return RawEventRecord{}, fmt.Errorf("cot raw event is empty")
	}
	if len(raw) > l.maxBytes {
		return RawEventRecord{}, fmt.Errorf("cot raw event is %d bytes; max lane bytes is %d", len(raw), l.maxBytes)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.next++
	record := RawEventRecord{
		Ref:        fmt.Sprintf("cot://raw/%s/%08d", l.source, l.next),
		Source:     l.source,
		ReceivedAt: l.clock().UTC(),
		RawXML:     append([]byte(nil), raw...),
	}
	if event != nil {
		record.UID = event.UID
		record.Type = event.Type
		record.Callsign = event.Callsign
		record.StaleAt = event.Stale
	}

	l.records = append(l.records, record)
	l.totalBytes += len(record.RawXML)
	l.evict()
	return cloneRawEventRecord(record), nil
}

func (l *RawLane) Snapshot() []RawEventRecord {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]RawEventRecord, len(l.records))
	for i, record := range l.records {
		out[i] = cloneRawEventRecord(record)
	}
	return out
}

func (l *RawLane) Get(ref string) (RawEventRecord, bool) {
	if l == nil {
		return RawEventRecord{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, record := range l.records {
		if record.Ref == ref {
			return cloneRawEventRecord(record), true
		}
	}
	return RawEventRecord{}, false
}

func (l *RawLane) evict() {
	for len(l.records) > l.maxRecords || l.totalBytes > l.maxBytes {
		l.totalBytes -= len(l.records[0].RawXML)
		copy(l.records, l.records[1:])
		l.records = l.records[:len(l.records)-1]
	}
}

func cloneRawEventRecord(record RawEventRecord) RawEventRecord {
	record.RawXML = append([]byte(nil), record.RawXML...)
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
