package adsb

import (
	"testing"
	"time"
)

func TestRawLaneCapturesAndBoundsOpenSkySnapshots(t *testing.T) {
	now := time.Date(2026, 6, 20, 15, 0, 0, 0, time.UTC)
	records, err := OpenSkyFixtureRecords(now)
	if err != nil {
		t.Fatalf("fixture records: %v", err)
	}
	snapshot, err := records[0].Snapshot()
	if err != nil {
		t.Fatalf("parse fixture snapshot: %v", err)
	}
	lane := NewRawLane(RawLaneConfig{
		Source:     "OpenSky Fixture",
		MaxRecords: 1,
		MaxBytes:   len(records[1].RawJSON) + 1,
		Clock:      func() time.Time { return now },
	})

	first, err := lane.Capture(records[0].RawJSON, &snapshot)
	if err != nil {
		t.Fatalf("capture first: %v", err)
	}
	if first.Ref != "adsb://raw/opensky-fixture/00000001" ||
		first.Source != "opensky-fixture" ||
		first.ReceivedAt != now ||
		first.SnapshotAt != snapshot.Time {
		t.Fatalf("first record = %+v", first)
	}
	first.RawJSON[0] = '{'
	stored, ok := lane.Get("adsb://raw/opensky-fixture/00000001")
	if !ok {
		t.Fatal("missing captured record")
	}
	if string(stored.RawJSON) != string(records[0].RawJSON) {
		t.Fatal("raw lane should clone captured JSON")
	}

	second, err := lane.Capture(records[1].RawJSON, nil)
	if err != nil {
		t.Fatalf("capture second: %v", err)
	}
	if second.Ref != "adsb://raw/opensky-fixture/00000002" {
		t.Fatalf("second ref = %q", second.Ref)
	}
	if _, ok := lane.Get("adsb://raw/opensky-fixture/00000001"); ok {
		t.Fatal("oldest record should be evicted when max records is exceeded")
	}
	if got := lane.Snapshot(); len(got) != 1 || got[0].Ref != second.Ref {
		t.Fatalf("lane snapshot = %+v", got)
	}
}

func TestRawLaneRejectsOversizedSnapshots(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{MaxBytes: 4})
	if _, err := lane.Capture([]byte(`{"time":1}`), nil); err == nil {
		t.Fatal("expected oversized snapshot rejection")
	}
}
