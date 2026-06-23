package cap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const committedLifecycleFixturePath = "../../../fixtures/cap/lifecycle/hadr-flood.jsonl"

func TestCommittedLifecycleFixtureMatchesGenerator(t *testing.T) {
	start := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	want, err := LifecycleFixtureRecords(start)
	if err != nil {
		t.Fatalf("generate lifecycle fixture: %v", err)
	}
	if os.Getenv("SEMOPS_UPDATE_FIXTURES") == "1" {
		writeLifecycleFixture(t, committedLifecycleFixturePath, want)
	}

	got, err := LoadReplay(committedLifecycleFixturePath)
	if err != nil {
		t.Fatalf("load committed lifecycle fixture: %v", err)
	}
	requireRawAlertRecordsEqual(t, got, want)
}

func writeLifecycleFixture(t *testing.T, path string, records []RawAlertRecord) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create lifecycle fixture dir: %v", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove old lifecycle fixture: %v", err)
	}
	store := NewReplayStore(path)
	for _, record := range records {
		if err := store.Append(record); err != nil {
			t.Fatalf("write lifecycle fixture record %s: %v", record.Ref, err)
		}
	}
}

func requireRawAlertRecordsEqual(t *testing.T, got, want []RawAlertRecord) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("records = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Ref != want[i].Ref ||
			got[i].Source != want[i].Source ||
			got[i].Identifier != want[i].Identifier ||
			got[i].MsgType != want[i].MsgType ||
			!got[i].ReceivedAt.Equal(want[i].ReceivedAt) ||
			!got[i].SentAt.Equal(want[i].SentAt) ||
			!bytes.Equal(got[i].RawXML, want[i].RawXML) {
			t.Fatalf("record[%d] = %+v, want %+v", i, got[i], want[i])
		}
		if _, err := Parse(got[i].RawXML); err != nil {
			t.Fatalf("parse committed record %s: %v", got[i].Ref, err)
		}
	}
}
