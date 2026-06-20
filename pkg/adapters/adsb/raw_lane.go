package adsb

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
	records    []RawSnapshotRecord
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
		records:    make([]RawSnapshotRecord, 0, cfg.MaxRecords),
	}
}

func (l *RawLane) Capture(raw []byte, snapshot *OpenSkySnapshot) (RawSnapshotRecord, error) {
	if l == nil {
		return RawSnapshotRecord{}, fmt.Errorf("adsb raw lane is nil")
	}
	if len(raw) == 0 {
		return RawSnapshotRecord{}, fmt.Errorf("adsb raw snapshot is empty")
	}
	if len(raw) > l.maxBytes {
		return RawSnapshotRecord{}, fmt.Errorf("adsb raw snapshot is %d bytes; max lane bytes is %d", len(raw), l.maxBytes)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.next++
	record := RawSnapshotRecord{
		Ref:        fmt.Sprintf("adsb://raw/%s/%08d", l.source, l.next),
		Source:     l.source,
		ReceivedAt: l.clock().UTC(),
		RawJSON:    append([]byte(nil), raw...),
	}
	if snapshot != nil {
		record.SnapshotAt = snapshot.Time
	}

	l.records = append(l.records, record)
	l.totalBytes += len(record.RawJSON)
	l.evict()
	return cloneRawSnapshotRecord(record), nil
}

func (l *RawLane) Snapshot() []RawSnapshotRecord {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]RawSnapshotRecord, len(l.records))
	for i, record := range l.records {
		out[i] = cloneRawSnapshotRecord(record)
	}
	return out
}

func (l *RawLane) Get(ref string) (RawSnapshotRecord, bool) {
	if l == nil {
		return RawSnapshotRecord{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, record := range l.records {
		if record.Ref == ref {
			return cloneRawSnapshotRecord(record), true
		}
	}
	return RawSnapshotRecord{}, false
}

func (l *RawLane) evict() {
	for len(l.records) > l.maxRecords || l.totalBytes > l.maxBytes {
		l.totalBytes -= len(l.records[0].RawJSON)
		copy(l.records, l.records[1:])
		l.records = l.records[:len(l.records)-1]
	}
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
