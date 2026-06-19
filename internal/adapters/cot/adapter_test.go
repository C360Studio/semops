package cot

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
)

func TestAdapterIngestCapturesReplayAndHealth(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 20, 0, 0, time.UTC)
	events := cotcodec.SeedEvents(now)
	raw, err := cotcodec.Marshal(events[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	storePath := filepath.Join(t.TempDir(), "cot.jsonl")
	seen := make(chan IngestResult, 1)
	adapter, err := NewAdapter(Config{
		Source: "tak:unit",
		Replay: cotcodec.NewReplayStore(storePath),
		Clock:  func() time.Time { return now },
		OnEvent: func(result IngestResult) {
			seen <- result
		},
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), raw)
	if err != nil {
		t.Fatalf("ingest event: %v", err)
	}
	if result.RawRef != "cot://raw/tak-unit/00000001" || result.Event.UID != "ANDROID-ALPHA" {
		t.Fatalf("result = %+v", result)
	}
	select {
	case got := <-seen:
		if got.Event.UID != "ANDROID-ALPHA" {
			t.Fatalf("seen uid = %q", got.Event.UID)
		}
	default:
		t.Fatal("expected OnEvent notification")
	}

	health := adapter.Health()
	if !health.Ready || health.EventsReceived != 1 || health.EventsCaptured != 1 || health.EventsDecoded != 1 {
		t.Fatalf("health = %+v", health)
	}
	if health.LastUID != "ANDROID-ALPHA" || health.LastType != cotcodec.TypeOperatorPosition || health.LastRawRef != result.RawRef {
		t.Fatalf("last health fields = %+v", health)
	}

	records, err := cotcodec.LoadReplay(storePath)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(records) != 1 || records[0].UID != "ANDROID-ALPHA" {
		t.Fatalf("replay records = %+v", records)
	}
}

func TestAdapterCapturesMalformedInputBeforeRejecting(t *testing.T) {
	adapter, err := NewAdapter(Config{Source: "tak:unit"})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	result, err := adapter.IngestEvent(context.Background(), []byte("<event"))
	if err == nil {
		t.Fatal("expected malformed CoT rejection")
	}
	if result.RawRef == "" {
		t.Fatal("malformed input should still be replay-addressable")
	}
	health := adapter.Health()
	if health.Ready || health.EventsReceived != 1 || health.EventsCaptured != 1 || health.ParseErrors != 1 {
		t.Fatalf("health = %+v", health)
	}
	if len(adapter.RawLane().Snapshot()) != 1 {
		t.Fatalf("raw lane records = %d, want 1", len(adapter.RawLane().Snapshot()))
	}
}

func TestAdapterReportsReplayAppendErrors(t *testing.T) {
	adapter, err := NewAdapter(Config{
		Source: "tak:unit",
		Replay: failingReplayAppender{},
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	raw, err := cotcodec.Marshal(cotcodec.SeedEvents(time.Now())[0])
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}

	if _, err := adapter.IngestEvent(context.Background(), raw); err == nil {
		t.Fatal("expected replay append error")
	}
	health := adapter.Health()
	if health.ReplayErrors != 1 || health.Ready {
		t.Fatalf("health = %+v", health)
	}
}

type failingReplayAppender struct{}

func (failingReplayAppender) Append(cotcodec.RawEventRecord) error {
	return errors.New("disk full")
}
