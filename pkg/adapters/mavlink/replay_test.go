package mavlink

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestReplayStoreAppendsAndLoadsRawFrameRecords(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "udp:14550", MaxRecords: 8, MaxBytes: 4096})
	generator := NewGenerator(42, 7)

	heartbeatFrame, err := generator.GenerateHeartbeat(HeartbeatMessage{
		BaseMode:       ModeFlagSafetyArmed,
		SystemStatus:   StateActive,
		MavlinkVersion: Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	positionFrame, err := generator.GenerateGlobalPosition(PositionMessage{
		Lat: 389000001,
		Lon: -770000002,
		Vx:  321,
	})
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}

	first := captureReplayRecord(t, lane, heartbeatFrame)
	second := captureReplayRecord(t, lane, positionFrame)

	path := filepath.Join(t.TempDir(), "fixtures", "mavlink.jsonl")
	store := NewReplayStore(path)
	if err := store.Append(first); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.Append(second); err != nil {
		t.Fatalf("append second: %v", err)
	}

	records, err := LoadReplay(path)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	if records[0].Ref != first.Ref || records[1].Ref != second.Ref {
		t.Fatalf("refs = %q/%q, want %q/%q", records[0].Ref, records[1].Ref, first.Ref, second.Ref)
	}
	if !bytes.Equal(records[0].Frame, heartbeatFrame) {
		t.Fatal("loaded heartbeat frame differs from input")
	}
	if !bytes.Equal(records[1].Frame, positionFrame) {
		t.Fatal("loaded position frame differs from input")
	}

	parser := NewParser()
	for _, record := range records {
		packets, err := parser.Parse(record.Frame)
		if err != nil {
			t.Fatalf("parse replay record %q: %v", record.Ref, err)
		}
		if len(packets) != 1 {
			t.Fatalf("replay record %q packets = %d, want 1", record.Ref, len(packets))
		}
	}
}

func TestReplayStoreRejectsIncompleteRecords(t *testing.T) {
	store := NewReplayStore(filepath.Join(t.TempDir(), "mavlink.jsonl"))

	if err := store.Append(RawFrameRecord{}); err == nil {
		t.Fatal("expected empty record rejection")
	}
	if err := store.Append(RawFrameRecord{Ref: "mavlink://raw/test/00000001"}); err == nil {
		t.Fatal("expected frame-less record rejection")
	}
}

func TestLoadReplayMissingFileReturnsEmpty(t *testing.T) {
	records, err := LoadReplay(filepath.Join(t.TempDir(), "missing.jsonl"))
	if err != nil {
		t.Fatalf("load missing replay: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records = %d, want 0", len(records))
	}
}

func captureReplayRecord(t *testing.T, lane *RawLane, frame []byte) RawFrameRecord {
	t.Helper()
	packet := parseOne(t, frame)
	record, err := lane.Capture(frame, packet)
	if err != nil {
		t.Fatalf("capture replay frame: %v", err)
	}
	return record
}
