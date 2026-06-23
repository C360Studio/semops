package adsb

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const committedOpenSkyFixturePath = "../../../fixtures/adsb/opensky-hadr.jsonl"

func TestCommittedOpenSkyFixtureMatchesGenerator(t *testing.T) {
	start := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	want, err := OpenSkyFixtureRecords(start)
	if err != nil {
		t.Fatalf("generate OpenSky fixture: %v", err)
	}
	if os.Getenv("SEMOPS_UPDATE_FIXTURES") == "1" {
		writeOpenSkyFixture(t, committedOpenSkyFixturePath, want)
	}

	got, err := LoadReplay(committedOpenSkyFixturePath)
	if err != nil {
		t.Fatalf("load committed OpenSky fixture: %v", err)
	}
	requireRawSnapshotRecordsEqual(t, got, want)
}

func writeOpenSkyFixture(t *testing.T, path string, records []RawSnapshotRecord) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create OpenSky fixture dir: %v", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove old OpenSky fixture: %v", err)
	}
	store := NewReplayStore(path)
	for _, record := range records {
		if err := store.Append(record); err != nil {
			t.Fatalf("write OpenSky fixture record %s: %v", record.Ref, err)
		}
	}
}

func requireRawSnapshotRecordsEqual(t *testing.T, got, want []RawSnapshotRecord) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("records = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Ref != want[i].Ref ||
			got[i].Source != want[i].Source ||
			!got[i].ReceivedAt.Equal(want[i].ReceivedAt) ||
			!got[i].SnapshotAt.Equal(want[i].SnapshotAt) ||
			!bytes.Equal(got[i].RawJSON, want[i].RawJSON) {
			t.Fatalf("record[%d] = %+v, want %+v", i, got[i], want[i])
		}
		if _, err := ParseOpenSkySnapshot(got[i].RawJSON); err != nil {
			t.Fatalf("parse committed record %s: %v", got[i].Ref, err)
		}
	}
}
