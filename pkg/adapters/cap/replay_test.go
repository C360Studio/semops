package cap

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"
)

func TestLifecycleFixtureRecordsParseIntoExpectedSequence(t *testing.T) {
	start := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	records, err := LifecycleFixtureRecords(start)
	if err != nil {
		t.Fatalf("lifecycle fixtures: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("records = %d, want 4", len(records))
	}
	want := []struct {
		refSuffix  string
		identifier string
		msgType    string
	}{
		{"0001-alert", "nws-demo-flood-warning", "Alert"},
		{"0002-update", "nws-demo-flood-warning", "Update"},
		{"0003-cancel", "nws-demo-flood-warning", "Cancel"},
		{"0004-expired", "nws-demo-flood-expired", "Alert"},
	}
	for i, record := range records {
		if record.Ref != "cap://fixture/hadr-flood/"+want[i].refSuffix ||
			record.Identifier != want[i].identifier ||
			record.MsgType != want[i].msgType {
			t.Fatalf("record[%d] = %+v", i, record)
		}
		alert, err := Parse(record.RawXML)
		if err != nil {
			t.Fatalf("parse record %q: %v", record.Ref, err)
		}
		if alert.Identifier != want[i].identifier || alert.MsgType != want[i].msgType {
			t.Fatalf("alert[%d] = %+v", i, alert)
		}
		if record.SentAt != alert.Sent {
			t.Fatalf("sent[%d] = %s, want %s", i, record.SentAt, alert.Sent)
		}
	}
	expired, _ := records[3].Alert()
	info, ok := expired.PrimaryInfo()
	if !ok {
		t.Fatal("expired fixture missing info")
	}
	if !info.Expires.Before(start) {
		t.Fatalf("expired fixture expires at %s, want before %s", info.Expires, start)
	}
}

func TestReplayStoreAppendsAndLoadsRawAlertRecords(t *testing.T) {
	records, err := LifecycleFixtureRecords(time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("lifecycle fixtures: %v", err)
	}
	path := filepath.Join(t.TempDir(), "fixtures", "cap.jsonl")
	store := NewReplayStore(path)
	for _, record := range records[:3] {
		if err := store.Append(record); err != nil {
			t.Fatalf("append %s: %v", record.Ref, err)
		}
	}

	loaded, err := LoadReplay(path)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("loaded = %d, want 3", len(loaded))
	}
	for i, record := range loaded {
		if record.Ref != records[i].Ref ||
			record.Identifier != records[i].Identifier ||
			record.MsgType != records[i].MsgType {
			t.Fatalf("loaded[%d] = %+v, want %+v", i, record, records[i])
		}
		if !bytes.Equal(record.RawXML, records[i].RawXML) {
			t.Fatalf("loaded raw XML %d differs", i)
		}
		if _, err := Parse(record.RawXML); err != nil {
			t.Fatalf("parse loaded %q: %v", record.Ref, err)
		}
	}
}

func TestReplayStoreRejectsIncompleteRecords(t *testing.T) {
	store := NewReplayStore(filepath.Join(t.TempDir(), "cap.jsonl"))
	if err := store.Append(RawAlertRecord{}); err == nil {
		t.Fatal("expected empty record rejection")
	}
	if err := store.Append(RawAlertRecord{Ref: "cap://fixture/test/0001"}); err == nil {
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
