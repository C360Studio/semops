package mavlink

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ReplayStore struct {
	path string
	mu   sync.Mutex
}

func NewReplayStore(path string) *ReplayStore {
	return &ReplayStore{path: path}
}

func (s *ReplayStore) Append(record RawFrameRecord) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("mavlink replay store has no path")
	}
	if record.Ref == "" {
		return fmt.Errorf("mavlink replay record has no ref")
	}
	if len(record.Frame) == 0 {
		return fmt.Errorf("mavlink replay record %q has no frame bytes", record.Ref)
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

func LoadReplay(path string) ([]RawFrameRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(bufio.NewReader(file))
	records := make([]RawFrameRecord, 0)
	for {
		var record RawFrameRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode replay record %d: %w", len(records)+1, err)
		}
		if record.Ref == "" {
			return nil, fmt.Errorf("decode replay record %d: missing ref", len(records)+1)
		}
		if len(record.Frame) == 0 {
			return nil, fmt.Errorf("decode replay record %q: missing frame bytes", record.Ref)
		}
		records = append(records, record)
	}
	return records, nil
}
