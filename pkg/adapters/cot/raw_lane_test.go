package cot

import (
	"bytes"
	"testing"
	"time"
)

func TestRawLaneCapturesEventMetadata(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 0, 0, 0, time.UTC)
	lane := NewRawLane(RawLaneConfig{
		Source:     "TAK UDP :6970",
		MaxRecords: 8,
		MaxBytes:   4096,
		Clock:      func() time.Time { return now },
	})
	event := SeedEvents(now)[0]
	raw, err := Marshal(event)
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}

	record, err := lane.Capture(raw, &event)
	if err != nil {
		t.Fatalf("capture event: %v", err)
	}
	if record.Ref != "cot://raw/tak-udp-6970/00000001" {
		t.Fatalf("ref = %q", record.Ref)
	}
	if record.ReceivedAt != now {
		t.Fatalf("received at = %s, want %s", record.ReceivedAt, now)
	}
	if record.UID != "ANDROID-ALPHA" || record.Type != TypeOperatorPosition || record.Callsign != "ALPHA" {
		t.Fatalf("record identity = %+v", record)
	}
	if !bytes.Equal(record.RawXML, raw) {
		t.Fatal("captured XML differs from input")
	}

	got, ok := lane.Get(record.Ref)
	if !ok {
		t.Fatalf("record %q was not replay-addressable", record.Ref)
	}
	if !bytes.Equal(got.RawXML, raw) {
		t.Fatal("retrieved XML differs from input")
	}
}

func TestRawLaneCapturesMalformedInput(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "tak", MaxRecords: 4, MaxBytes: 64})
	record, err := lane.Capture([]byte("<event"), nil)
	if err != nil {
		t.Fatalf("capture malformed input: %v", err)
	}
	if record.UID != "" || record.Type != "" {
		t.Fatalf("malformed record identity should be empty: %+v", record)
	}
}

func TestRawLaneEvictsByRecordCap(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "tak", MaxRecords: 2, MaxBytes: 4096})
	rawEvents := mustMarshalEvents(t, SeedEvents(time.Date(2026, 6, 19, 14, 1, 0, 0, time.UTC))[:3])

	first := captureRaw(t, lane, rawEvents[0])
	second := captureRaw(t, lane, rawEvents[1])
	third := captureRaw(t, lane, rawEvents[2])

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
	lane := NewRawLane(RawLaneConfig{Source: "tak", MaxRecords: 8, MaxBytes: 420})
	rawEvents := mustMarshalEvents(t, SeedEvents(time.Date(2026, 6, 19, 14, 2, 0, 0, time.UTC))[:2])

	first := captureRaw(t, lane, rawEvents[0])
	second := captureRaw(t, lane, rawEvents[1])

	snapshot := lane.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("snapshot length = %d, want 1", len(snapshot))
	}
	if snapshot[0].Ref != second.Ref {
		t.Fatalf("remaining ref = %q, want %q after evicting %q", snapshot[0].Ref, second.Ref, first.Ref)
	}
}

func TestRawLaneRejectsInvalidInput(t *testing.T) {
	if _, err := ((*RawLane)(nil)).Capture([]byte("<event/>"), nil); err == nil {
		t.Fatal("expected nil lane rejection")
	}
	lane := NewRawLane(RawLaneConfig{Source: "tak", MaxBytes: 4})
	if _, err := lane.Capture(nil, nil); err == nil {
		t.Fatal("expected empty raw event rejection")
	}
	if _, err := lane.Capture([]byte("<event/>"), nil); err == nil {
		t.Fatal("expected oversize raw event rejection")
	}
}

func captureRaw(t *testing.T, lane *RawLane, raw []byte) RawEventRecord {
	t.Helper()
	event, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("unmarshal raw fixture: %v", err)
	}
	record, err := lane.Capture(raw, &event)
	if err != nil {
		t.Fatalf("capture raw fixture: %v", err)
	}
	return record
}

func mustMarshalEvents(t *testing.T, events []Event) [][]byte {
	t.Helper()
	raw, err := MarshalEvents(events)
	if err != nil {
		t.Fatalf("marshal events: %v", err)
	}
	return raw
}
