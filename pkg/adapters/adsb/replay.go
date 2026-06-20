package adsb

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

type RawSnapshotRecord struct {
	Ref        string
	Source     string
	ReceivedAt time.Time
	SnapshotAt time.Time
	RawJSON    []byte
}

func (r RawSnapshotRecord) Snapshot() (OpenSkySnapshot, error) {
	return ParseOpenSkySnapshot(r.RawJSON)
}

type ReplayStore struct {
	path string
	mu   sync.Mutex
}

func NewReplayStore(path string) *ReplayStore {
	return &ReplayStore{path: path}
}

func (s *ReplayStore) Append(record RawSnapshotRecord) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("adsb replay store has no path")
	}
	if record.Ref == "" {
		return fmt.Errorf("adsb replay record has no ref")
	}
	if len(record.RawJSON) == 0 {
		return fmt.Errorf("adsb replay record %q has no raw JSON", record.Ref)
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

func LoadReplay(path string) ([]RawSnapshotRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open replay file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(bufio.NewReader(file))
	records := make([]RawSnapshotRecord, 0)
	for {
		var record RawSnapshotRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode replay record %d: %w", len(records)+1, err)
		}
		if record.Ref == "" {
			return nil, fmt.Errorf("decode replay record %d: missing ref", len(records)+1)
		}
		if len(record.RawJSON) == 0 {
			return nil, fmt.Errorf("decode replay record %q: missing raw JSON", record.Ref)
		}
		records = append(records, cloneRawSnapshotRecord(record))
	}
	return records, nil
}

func OpenSkyFixtureRecords(start time.Time) ([]RawSnapshotRecord, error) {
	if start.IsZero() {
		start = time.Now().UTC()
	}
	start = start.UTC()

	firstAt := start.Add(20 * time.Second)
	firstJSON, err := marshalOpenSkySnapshot(firstAt,
		openSkyStateRow(
			"a1b2c3",
			"N123AB  ",
			"United States",
			&firstAt,
			firstAt.Add(7*time.Second),
			ptrFloat64(-77.0400),
			ptrFloat64(38.9000),
			false,
			ptrFloat64(71.5),
			ptrFloat64(180.25),
			ptrFloat64(-1.2),
			[]int{101, 202},
			ptrInt(int(PositionSourceADSB)),
			ptrInt(2),
		),
		openSkyStateRow(
			"d4e5f6",
			"",
			"Canada",
			nil,
			firstAt.Add(5*time.Second),
			nil,
			nil,
			false,
			nil,
			nil,
			nil,
			nil,
			ptrInt(int(PositionSourceADSB)),
			nil,
		),
	)
	if err != nil {
		return nil, err
	}

	secondAt := start.Add(35 * time.Second)
	secondJSON, err := marshalOpenSkySnapshot(secondAt,
		openSkyStateRow(
			"a1b2c3",
			"N123AB  ",
			"United States",
			&secondAt,
			secondAt.Add(5*time.Second),
			ptrFloat64(-77.0410),
			ptrFloat64(38.9010),
			false,
			ptrFloat64(75.0),
			ptrFloat64(181.0),
			ptrFloat64(-0.5),
			[]int{101, 202},
			ptrInt(int(PositionSourceADSB)),
			ptrInt(2),
		),
		openSkyStateRow(
			"b7c8d9",
			"MLAT01  ",
			"United States",
			&secondAt,
			secondAt.Add(4*time.Second),
			ptrFloat64(-77.0600),
			ptrFloat64(38.9150),
			false,
			ptrFloat64(102.25),
			ptrFloat64(90.0),
			ptrFloat64(0.0),
			[]int{303},
			ptrInt(int(PositionSourceMLAT)),
			ptrInt(14),
		),
	)
	if err != nil {
		return nil, err
	}

	return []RawSnapshotRecord{
		{
			Ref:        "adsb://fixture/opensky-hadr/0001-snapshot",
			Source:     "opensky-fixture",
			ReceivedAt: firstAt,
			SnapshotAt: firstAt,
			RawJSON:    firstJSON,
		},
		{
			Ref:        "adsb://fixture/opensky-hadr/0002-snapshot",
			Source:     "opensky-fixture",
			ReceivedAt: secondAt,
			SnapshotAt: secondAt,
			RawJSON:    secondJSON,
		},
	}, nil
}

func marshalOpenSkySnapshot(snapshotAt time.Time, rows ...[]any) ([]byte, error) {
	body := struct {
		Time   int64   `json:"time"`
		States [][]any `json:"states"`
	}{
		Time:   snapshotAt.Unix(),
		States: rows,
	}
	return json.Marshal(body)
}

func openSkyStateRow(
	icao24 string,
	callsign string,
	originCountry string,
	timePosition *time.Time,
	lastContact time.Time,
	longitude *float64,
	latitude *float64,
	onGround bool,
	velocity *float64,
	trueTrack *float64,
	verticalRate *float64,
	sensorIDs []int,
	positionSource *int,
	category *int,
) []any {
	return []any{
		icao24,
		nullableString(callsign),
		originCountry,
		nullableUnix(timePosition),
		lastContact.Unix(),
		nullableFloat64(longitude),
		nullableFloat64(latitude),
		nil,
		onGround,
		nullableFloat64(velocity),
		nullableFloat64(trueTrack),
		nullableFloat64(verticalRate),
		nullableIntSlice(sensorIDs),
		nil,
		nil,
		false,
		nullableInt(positionSource),
		nullableInt(category),
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableUnix(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Unix()
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableIntSlice(value []int) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func ptrFloat64(value float64) *float64 {
	return &value
}

func ptrInt(value int) *int {
	return &value
}

func cloneRawSnapshotRecord(record RawSnapshotRecord) RawSnapshotRecord {
	record.RawJSON = append([]byte(nil), record.RawJSON...)
	return record
}
