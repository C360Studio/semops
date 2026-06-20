package adsb

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenSkyFixtureRecordsParseIntoExpectedSequence(t *testing.T) {
	start := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	records, err := OpenSkyFixtureRecords(start)
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	if records[0].Ref != "adsb://fixture/opensky-hadr/0001-snapshot" ||
		records[1].Ref != "adsb://fixture/opensky-hadr/0002-snapshot" {
		t.Fatalf("refs = %q/%q", records[0].Ref, records[1].Ref)
	}

	first, err := records[0].Snapshot()
	if err != nil {
		t.Fatalf("parse first snapshot: %v", err)
	}
	if first.Time != start.Add(20*time.Second) || len(first.States) != 2 {
		t.Fatalf("first snapshot = %+v", first)
	}
	if first.States[0].ICAO24 != "a1b2c3" ||
		!first.States[0].HasPosition() ||
		first.States[1].ICAO24 != "d4e5f6" ||
		first.States[1].HasPosition() {
		t.Fatalf("first states = %+v", first.States)
	}

	second, err := records[1].Snapshot()
	if err != nil {
		t.Fatalf("parse second snapshot: %v", err)
	}
	if second.Time != start.Add(35*time.Second) || len(second.States) != 2 {
		t.Fatalf("second snapshot = %+v", second)
	}
	if second.States[0].ICAO24 != "a1b2c3" ||
		second.States[1].PositionSource != PositionSourceMLAT ||
		second.States[1].PositionSourceLabel() != "mlat" {
		t.Fatalf("second states = %+v", second.States)
	}
}

func TestReplayStoreAppendsAndLoadsRawSnapshotRecords(t *testing.T) {
	records, err := OpenSkyFixtureRecords(time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	path := filepath.Join(t.TempDir(), "adsb", "opensky.jsonl")
	store := NewReplayStore(path)
	for _, record := range records {
		if err := store.Append(record); err != nil {
			t.Fatalf("append %s: %v", record.Ref, err)
		}
	}

	loaded, err := LoadReplay(path)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(loaded) != len(records) {
		t.Fatalf("loaded records = %d, want %d", len(loaded), len(records))
	}
	if loaded[0].Ref != records[0].Ref || string(loaded[0].RawJSON) != string(records[0].RawJSON) {
		t.Fatalf("loaded first = %+v, want %+v", loaded[0], records[0])
	}

	loaded[0].RawJSON[0] = '{'
	reloaded, err := LoadReplay(path)
	if err != nil {
		t.Fatalf("reload replay: %v", err)
	}
	if string(reloaded[0].RawJSON) != string(records[0].RawJSON) {
		t.Fatal("loaded replay records should be cloned from stored data")
	}
}

func TestReplayStoreRejectsIncompleteRecords(t *testing.T) {
	store := NewReplayStore(filepath.Join(t.TempDir(), "adsb.jsonl"))
	if err := store.Append(RawSnapshotRecord{}); err == nil || !strings.Contains(err.Error(), "ref") {
		t.Fatalf("append missing ref error = %v", err)
	}
	if err := store.Append(RawSnapshotRecord{Ref: "adsb://fixture/empty"}); err == nil ||
		!strings.Contains(err.Error(), "raw JSON") {
		t.Fatalf("append missing raw JSON error = %v", err)
	}
}

func TestLoadReplayMissingFileReturnsEmpty(t *testing.T) {
	records, err := LoadReplay(filepath.Join(t.TempDir(), "missing.jsonl"))
	if err != nil {
		t.Fatalf("load missing replay: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records = %+v, want empty", records)
	}
}
