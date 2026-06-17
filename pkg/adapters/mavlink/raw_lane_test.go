package mavlink

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRawLaneCapturesFrameMetadataAndAnnotatesPacket(t *testing.T) {
	now := time.Date(2026, 6, 17, 13, 0, 0, 0, time.UTC)
	lane := NewRawLane(RawLaneConfig{
		Source:     "UDP :14550",
		MaxRecords: 8,
		MaxBytes:   1024,
		Clock:      func() time.Time { return now },
	})

	generator := NewGenerator(42, 7)
	frame, err := generator.GenerateHeartbeat(HeartbeatMessage{
		BaseMode:       ModeFlagSafetyArmed,
		SystemStatus:   StateActive,
		MavlinkVersion: Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	packet := parseOne(t, frame)

	record, err := lane.Capture(frame, packet)
	if err != nil {
		t.Fatalf("capture frame: %v", err)
	}

	if record.Ref != "mavlink://raw/udp-14550/00000001" {
		t.Fatalf("ref = %q", record.Ref)
	}
	if packet.SourceRef != record.Ref {
		t.Fatalf("packet source ref = %q, want %q", packet.SourceRef, record.Ref)
	}
	if record.ReceivedAt != now {
		t.Fatalf("received at = %s, want %s", record.ReceivedAt, now)
	}
	if record.SystemID != 42 || record.ComponentID != 7 {
		t.Fatalf("system/component = %d/%d, want 42/7", record.SystemID, record.ComponentID)
	}
	if record.MessageID != MessageIDHeartbeat {
		t.Fatalf("message id = %d, want %d", record.MessageID, MessageIDHeartbeat)
	}
	if !bytes.Equal(record.Frame, frame) {
		t.Fatal("captured frame bytes differ from input")
	}

	got, ok := lane.Get(record.Ref)
	if !ok {
		t.Fatalf("record %q was not replay-addressable", record.Ref)
	}
	if !bytes.Equal(got.Frame, frame) {
		t.Fatal("retrieved frame bytes differ from input")
	}
}

func TestRawLaneEvictsByRecordCap(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{
		Source:     "udp",
		MaxRecords: 2,
		MaxBytes:   64,
	})
	generator := NewGenerator(1, 1)

	first := captureHeartbeat(t, lane, generator)
	second := captureHeartbeat(t, lane, generator)
	third := captureHeartbeat(t, lane, generator)

	snapshot := lane.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("snapshot length = %d, want 2", len(snapshot))
	}
	if snapshot[0].Ref != second.Ref || snapshot[1].Ref != third.Ref {
		t.Fatalf("snapshot refs = %q, %q; want %q, %q", snapshot[0].Ref, snapshot[1].Ref, second.Ref, third.Ref)
	}
	if _, ok := lane.Get(first.Ref); ok {
		t.Fatalf("evicted record %q should not be retrievable", first.Ref)
	}
}

func TestRawLaneEvictsByByteCap(t *testing.T) {
	generator := NewGenerator(1, 1)
	frame, err := generator.GenerateHeartbeat(HeartbeatMessage{MavlinkVersion: Version2})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	lane := NewRawLane(RawLaneConfig{
		Source:     "udp",
		MaxRecords: 8,
		MaxBytes:   len(frame) * 2,
	})

	first := captureFrame(t, lane, frame)
	second := captureFrame(t, lane, frame)
	third := captureFrame(t, lane, frame)

	snapshot := lane.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("snapshot length = %d, want 2", len(snapshot))
	}
	if snapshot[0].Ref != second.Ref || snapshot[1].Ref != third.Ref {
		t.Fatalf("snapshot refs = %q, %q; want %q, %q", snapshot[0].Ref, snapshot[1].Ref, second.Ref, third.Ref)
	}
	if _, ok := lane.Get(first.Ref); ok {
		t.Fatalf("byte-evicted record %q should not be retrievable", first.Ref)
	}
}

func TestRawLaneRejectsFramesLargerThanLaneByteCap(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "udp", MaxRecords: 4, MaxBytes: 4})

	_, err := lane.Capture([]byte{1, 2, 3, 4, 5}, nil)
	if err == nil {
		t.Fatal("expected oversize frame rejection")
	}
	if !strings.Contains(err.Error(), "max lane bytes") {
		t.Fatalf("error = %v, want max lane bytes", err)
	}
	if len(lane.Snapshot()) != 0 {
		t.Fatal("oversize frame should not be retained")
	}
}

func TestRawLaneReturnsDefensiveCopies(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "udp", MaxRecords: 4, MaxBytes: 128})
	frame := []byte{STXV2, 0, 0, 0}

	record, err := lane.Capture(frame, nil)
	if err != nil {
		t.Fatalf("capture frame: %v", err)
	}
	frame[0] = 0xff
	if record.Frame[0] != STXV2 {
		t.Fatal("capture returned frame should not alias input")
	}

	snapshot := lane.Snapshot()
	snapshot[0].Frame[0] = 0xee
	got, ok := lane.Get(record.Ref)
	if !ok {
		t.Fatalf("record %q missing", record.Ref)
	}
	if got.Frame[0] != STXV2 {
		t.Fatal("snapshot frame should not alias retained record")
	}
}

func captureHeartbeat(t *testing.T, lane *RawLane, generator *Generator) RawFrameRecord {
	t.Helper()
	frame, err := generator.GenerateHeartbeat(HeartbeatMessage{MavlinkVersion: Version2})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	return captureFrame(t, lane, frame)
}

func captureFrame(t *testing.T, lane *RawLane, frame []byte) RawFrameRecord {
	t.Helper()
	packet := parseOne(t, frame)
	record, err := lane.Capture(frame, packet)
	if err != nil {
		t.Fatalf("capture frame: %v", err)
	}
	return record
}
