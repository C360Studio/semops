package cot

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"
)

func TestSeedEventsMarshalToExpectedFixtureSet(t *testing.T) {
	events := SeedEvents(time.Date(2026, 6, 19, 14, 5, 0, 0, time.UTC))
	raw, err := MarshalEvents(events)
	if err != nil {
		t.Fatalf("marshal seed events: %v", err)
	}
	if len(raw) != 4 {
		t.Fatalf("seed raw events = %d, want 4", len(raw))
	}
	uids := make([]string, 0, len(raw))
	for _, next := range raw {
		event, err := Unmarshal(next)
		if err != nil {
			t.Fatalf("unmarshal seed XML: %v", err)
		}
		uids = append(uids, event.UID)
	}
	want := []string{"ANDROID-ALPHA", "ANDROID-BRAVO", "MARKER-NORTH-GATE", "CHAT-ALPHA-1"}
	for i := range want {
		if uids[i] != want[i] {
			t.Fatalf("seed uid[%d] = %q, want %q", i, uids[i], want[i])
		}
	}
}

func TestReplayStoreAppendsAndLoadsRawEventRecords(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "tak:fixture", MaxRecords: 8, MaxBytes: 4096})
	rawEvents := mustMarshalEvents(t, SeedEvents(time.Date(2026, 6, 19, 14, 6, 0, 0, time.UTC))[:2])

	first := captureRaw(t, lane, rawEvents[0])
	second := captureRaw(t, lane, rawEvents[1])

	path := filepath.Join(t.TempDir(), "fixtures", "cot.jsonl")
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
	if !bytes.Equal(records[0].RawXML, rawEvents[0]) || !bytes.Equal(records[1].RawXML, rawEvents[1]) {
		t.Fatal("loaded raw XML differs from input")
	}

	for _, record := range records {
		event, err := Unmarshal(record.RawXML)
		if err != nil {
			t.Fatalf("parse replay record %q: %v", record.Ref, err)
		}
		if event.UID != record.UID {
			t.Fatalf("replay record %q uid = %q, want %q", record.Ref, event.UID, record.UID)
		}
	}
}

func TestReplayStoreRejectsIncompleteRecords(t *testing.T) {
	store := NewReplayStore(filepath.Join(t.TempDir(), "cot.jsonl"))
	if err := store.Append(RawEventRecord{}); err == nil {
		t.Fatal("expected empty record rejection")
	}
	if err := store.Append(RawEventRecord{Ref: "cot://raw/test/00000001"}); err == nil {
		t.Fatal("expected raw-XML-less record rejection")
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
