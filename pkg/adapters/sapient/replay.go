package sapient

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

type RawMessageRecord struct {
	Ref           string
	Source        string
	ReceivedAt    time.Time
	Encoding      Encoding
	Content       ContentKind
	NodeID        string
	DestinationID string
	MessageAt     time.Time
	RawPayload    []byte
}

func (r RawMessageRecord) Message(descriptors *ProtoDescriptorSet) (Message, error) {
	switch r.Encoding {
	case EncodingJSON:
		return ParseJSONMessage(r.RawPayload)
	case EncodingProtobuf:
		return ParseBinaryMessage(r.RawPayload, descriptors)
	default:
		return Message{}, fmt.Errorf("sapient replay record %q has unsupported encoding %q", r.Ref, r.Encoding)
	}
}

type ReplayStore struct {
	path string
	mu   sync.Mutex
}

func NewReplayStore(path string) *ReplayStore {
	return &ReplayStore{path: path}
}

func (s *ReplayStore) Append(record RawMessageRecord) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("sapient replay store has no path")
	}
	if err := validateRawMessageRecord(record, "append"); err != nil {
		return err
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

func LoadReplay(path string) ([]RawMessageRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(bufio.NewReader(file))
	records := make([]RawMessageRecord, 0)
	for {
		var record RawMessageRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode replay record %d: %w", len(records)+1, err)
		}
		if err := validateRawMessageRecord(record, fmt.Sprintf("decode replay record %d", len(records)+1)); err != nil {
			return nil, err
		}
		records = append(records, cloneRawMessageRecord(record))
	}
	return records, nil
}

func validateRawMessageRecord(record RawMessageRecord, context string) error {
	if record.Ref == "" {
		return fmt.Errorf("%s: missing ref", context)
	}
	if !record.Encoding.Valid() {
		return fmt.Errorf("%s %q: unsupported encoding %q", context, record.Ref, record.Encoding)
	}
	if len(record.RawPayload) == 0 {
		return fmt.Errorf("%s %q: missing raw payload", context, record.Ref)
	}
	return nil
}
