package cot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ReplayStore struct {
	path string
	mu   sync.Mutex
}

func NewReplayStore(path string) *ReplayStore {
	return &ReplayStore{path: path}
}

func (s *ReplayStore) Append(record RawEventRecord) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("cot replay store has no path")
	}
	if record.Ref == "" {
		return fmt.Errorf("cot replay record has no ref")
	}
	if len(record.RawXML) == 0 {
		return fmt.Errorf("cot replay record %q has no raw XML", record.Ref)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create replay directory: %w", err)
		}
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(record); err != nil {
		return fmt.Errorf("write replay record %q: %w", record.Ref, err)
	}
	return nil
}

func LoadReplay(path string) ([]RawEventRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(bufio.NewReader(file))
	records := make([]RawEventRecord, 0)
	for {
		var record RawEventRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode replay record %d: %w", len(records)+1, err)
		}
		if record.Ref == "" {
			return nil, fmt.Errorf("decode replay record %d: missing ref", len(records)+1)
		}
		if len(record.RawXML) == 0 {
			return nil, fmt.Errorf("decode replay record %q: missing raw XML", record.Ref)
		}
		records = append(records, cloneRawEventRecord(record))
	}
	return records, nil
}

func SeedEvents(now time.Time) []Event {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stale := now.Add(2 * time.Minute)
	return []Event{
		{
			UID:      "ANDROID-ALPHA",
			Type:     TypeOperatorPosition,
			How:      DefaultHow,
			Time:     now,
			Start:    now,
			Stale:    stale,
			Point:    &Point{Lat: 38.8920, Lon: -77.0350, HAE: 24, CE: 5, LE: 5},
			Callsign: "ALPHA",
		},
		{
			UID:      "ANDROID-BRAVO",
			Type:     TypeOperatorPosition,
			How:      DefaultHow,
			Time:     now,
			Start:    now,
			Stale:    stale,
			Point:    &Point{Lat: 38.8910, Lon: -77.0410, HAE: 22, CE: 5, LE: 5},
			Callsign: "BRAVO",
		},
		{
			UID:      "MARKER-NORTH-GATE",
			Type:     TypeMarker,
			How:      DefaultHow,
			Time:     now,
			Start:    now,
			Stale:    stale,
			Point:    &Point{Lat: 38.8940, Lon: -77.0380, CE: 5, LE: 5},
			Callsign: "North Gate",
			Remarks:  "checkpoint",
		},
		{
			UID:       "CHAT-ALPHA-1",
			Type:      TypeGeoChat,
			How:       "h-g-i-g-o",
			Time:      now,
			Start:     now,
			Stale:     stale,
			Point:     &Point{Lat: 38.8920, Lon: -77.0350, HAE: 24, CE: 5, LE: 5},
			Callsign:  "ALPHA",
			SenderUID: "ANDROID-ALPHA",
			Remarks:   "hold at checkpoint",
			ChatText:  "hold at checkpoint",
		},
	}
}

func MarshalEvents(events []Event) ([][]byte, error) {
	raw := make([][]byte, 0, len(events))
	for i, event := range events {
		next, err := Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("marshal cot event %d: %w", i+1, err)
		}
		raw = append(raw, next)
	}
	return raw, nil
}
