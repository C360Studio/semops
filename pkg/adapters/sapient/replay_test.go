package sapient

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayStoreAppendsAndLoadsRawMessageRecords(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "sapient:fixture", MaxRecords: 8, MaxBytes: 8192})
	descriptors, err := EmbeddedProtoDescriptorSet(context.Background())
	if err != nil {
		t.Fatalf("compile embedded descriptors: %v", err)
	}

	jsonMessage, err := ParseJSONMessage([]byte(sampleDetectionReport))
	if err != nil {
		t.Fatalf("parse JSON fixture: %v", err)
	}
	jsonRecord, err := lane.Capture([]byte(sampleDetectionReport), EncodingJSON, &jsonMessage)
	if err != nil {
		t.Fatalf("capture JSON record: %v", err)
	}

	binaryRaw := mustBinaryPayload(t, descriptors, binaryRegistrationFixture)
	binaryMessage, err := ParseBinaryMessage(binaryRaw, descriptors)
	if err != nil {
		t.Fatalf("parse binary fixture: %v", err)
	}
	binaryRecord, err := lane.Capture(binaryRaw, EncodingProtobuf, &binaryMessage)
	if err != nil {
		t.Fatalf("capture binary record: %v", err)
	}

	path := filepath.Join(t.TempDir(), "fixtures", "sapient.jsonl")
	store := NewReplayStore(path)
	if err := store.Append(jsonRecord); err != nil {
		t.Fatalf("append JSON record: %v", err)
	}
	if err := store.Append(binaryRecord); err != nil {
		t.Fatalf("append binary record: %v", err)
	}

	records, err := LoadReplay(path)
	if err != nil {
		t.Fatalf("load replay: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	if records[0].Ref != jsonRecord.Ref || records[1].Ref != binaryRecord.Ref {
		t.Fatalf("refs = %q/%q, want %q/%q", records[0].Ref, records[1].Ref, jsonRecord.Ref, binaryRecord.Ref)
	}
	if !bytes.Equal(records[0].RawPayload, []byte(sampleDetectionReport)) {
		t.Fatal("loaded JSON payload differs from input")
	}
	if !bytes.Equal(records[1].RawPayload, binaryRaw) {
		t.Fatal("loaded binary payload differs from input")
	}

	for _, record := range records {
		message, err := record.Message(descriptors)
		if err != nil {
			t.Fatalf("parse replay record %q: %v", record.Ref, err)
		}
		if message.Content != record.Content {
			t.Fatalf("replay record %q content = %q, want %q", record.Ref, message.Content, record.Content)
		}
	}
}

func TestReplayStoreRejectsIncompleteRecords(t *testing.T) {
	store := NewReplayStore(filepath.Join(t.TempDir(), "sapient.jsonl"))

	if err := store.Append(RawMessageRecord{}); err == nil {
		t.Fatal("expected empty record rejection")
	}
	if err := store.Append(RawMessageRecord{Ref: "sapient://raw/test/json/00000001", Encoding: EncodingJSON}); err == nil {
		t.Fatal("expected payload-less record rejection")
	}
	if err := store.Append(RawMessageRecord{Ref: "sapient://raw/test/yaml/00000002", Encoding: Encoding("yaml"), RawPayload: []byte("{}")}); err == nil {
		t.Fatal("expected unsupported encoding rejection")
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

func TestLoadReplayRejectsMalformedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sapient.jsonl")
	if err := os.WriteFile(path, []byte("{not-json\n"), 0o644); err != nil {
		t.Fatalf("write malformed replay: %v", err)
	}
	if _, err := LoadReplay(path); err == nil {
		t.Fatal("expected malformed replay rejection")
	}
}
