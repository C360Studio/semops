package sapient

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRawLaneCapturesJSONAndBinaryMessageMetadata(t *testing.T) {
	now := time.Date(2026, 6, 20, 16, 0, 0, 0, time.UTC)
	lane := NewRawLane(RawLaneConfig{
		Source:     "SAPIENT TCP :9000",
		MaxRecords: 8,
		MaxBytes:   8192,
		Clock:      func() time.Time { return now },
	})

	jsonRaw := []byte(sampleDetectionReport)
	jsonMessage, err := ParseJSONMessage(jsonRaw)
	if err != nil {
		t.Fatalf("parse JSON fixture: %v", err)
	}
	jsonRecord, err := lane.Capture(jsonRaw, EncodingJSON, &jsonMessage)
	if err != nil {
		t.Fatalf("capture JSON message: %v", err)
	}
	if jsonRecord.Ref != "sapient://raw/sapient-tcp-9000/json/00000001" {
		t.Fatalf("json ref = %q", jsonRecord.Ref)
	}
	if jsonRecord.ReceivedAt != now {
		t.Fatalf("json received at = %s, want %s", jsonRecord.ReceivedAt, now)
	}
	if jsonRecord.Encoding != EncodingJSON || jsonRecord.Content != ContentDetectionReport {
		t.Fatalf("json record metadata = %+v", jsonRecord)
	}
	if jsonRecord.NodeID != jsonMessage.NodeID || jsonRecord.MessageAt != jsonMessage.Timestamp {
		t.Fatalf("json identity = %+v, want node %q at %s", jsonRecord, jsonMessage.NodeID, jsonMessage.Timestamp)
	}
	if !bytes.Equal(jsonRecord.RawPayload, jsonRaw) {
		t.Fatal("captured JSON differs from input")
	}

	descriptors, err := EmbeddedProtoDescriptorSet(context.Background())
	if err != nil {
		t.Fatalf("compile embedded descriptors: %v", err)
	}
	binaryRaw := mustBinaryPayload(t, descriptors, binaryRegistrationFixture)
	binaryMessage, err := ParseBinaryMessage(binaryRaw, descriptors)
	if err != nil {
		t.Fatalf("parse binary fixture: %v", err)
	}
	binaryRecord, err := lane.Capture(binaryRaw, EncodingProtobuf, &binaryMessage)
	if err != nil {
		t.Fatalf("capture binary message: %v", err)
	}
	if binaryRecord.Ref != "sapient://raw/sapient-tcp-9000/protobuf/00000002" {
		t.Fatalf("binary ref = %q", binaryRecord.Ref)
	}
	if binaryRecord.Encoding != EncodingProtobuf || binaryRecord.Content != ContentRegistration {
		t.Fatalf("binary record metadata = %+v", binaryRecord)
	}

	got, ok := lane.Get(binaryRecord.Ref)
	if !ok {
		t.Fatalf("record %q was not replay-addressable", binaryRecord.Ref)
	}
	if !bytes.Equal(got.RawPayload, binaryRaw) {
		t.Fatal("retrieved binary payload differs from input")
	}
	replayed, err := got.Message(descriptors)
	if err != nil {
		t.Fatalf("decode binary replay record: %v", err)
	}
	if replayed.Content != ContentRegistration {
		t.Fatalf("replayed content = %q, want %q", replayed.Content, ContentRegistration)
	}
}

func TestRawLaneCapturesUnparsedBytesWithKnownEncoding(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "sapient", MaxRecords: 4, MaxBytes: 64})
	record, err := lane.Capture([]byte("{"), EncodingJSON, nil)
	if err != nil {
		t.Fatalf("capture malformed fixture: %v", err)
	}
	if record.Content != "" || record.NodeID != "" {
		t.Fatalf("malformed record identity should be empty: %+v", record)
	}
	if _, err := record.Message(nil); err == nil {
		t.Fatal("expected malformed replay decode failure")
	}
}

func TestRawLaneEvictsByRecordCap(t *testing.T) {
	lane := NewRawLane(RawLaneConfig{Source: "sapient", MaxRecords: 2, MaxBytes: 4096})

	first := captureRawMessage(t, lane, []byte(sampleDetectionReport))
	second := captureRawMessage(t, lane, []byte(sampleStatusReport))
	third := captureRawMessage(t, lane, []byte(sampleTaskAck))

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
	raw := []byte(sampleTaskAck)
	lane := NewRawLane(RawLaneConfig{Source: "sapient", MaxRecords: 8, MaxBytes: len(raw) * 2})

	first := captureRawMessage(t, lane, raw)
	second := captureRawMessage(t, lane, raw)
	third := captureRawMessage(t, lane, raw)

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

func TestRawLaneRejectsInvalidInputAndReturnsDefensiveCopies(t *testing.T) {
	if _, err := ((*RawLane)(nil)).Capture([]byte("{}"), EncodingJSON, nil); err == nil {
		t.Fatal("expected nil lane rejection")
	}

	lane := NewRawLane(RawLaneConfig{Source: "sapient", MaxRecords: 4, MaxBytes: 128})
	if _, err := lane.Capture(nil, EncodingJSON, nil); err == nil {
		t.Fatal("expected empty raw message rejection")
	}
	if _, err := lane.Capture([]byte("{}"), Encoding("yaml"), nil); err == nil {
		t.Fatal("expected unsupported encoding rejection")
	}
	if _, err := lane.Capture([]byte(strings.Repeat("x", 129)), EncodingJSON, nil); err == nil {
		t.Fatal("expected oversize raw message rejection")
	}

	raw := []byte("{}")
	record, err := lane.Capture(raw, EncodingJSON, nil)
	if err != nil {
		t.Fatalf("capture raw message: %v", err)
	}
	raw[0] = '['
	if string(record.RawPayload) != "{}" {
		t.Fatal("capture returned payload should not alias input")
	}

	snapshot := lane.Snapshot()
	snapshot[0].RawPayload[0] = '['
	got, ok := lane.Get(record.Ref)
	if !ok {
		t.Fatalf("record %q missing", record.Ref)
	}
	if string(got.RawPayload) != "{}" {
		t.Fatal("snapshot payload should not alias retained record")
	}
}

func captureRawMessage(t *testing.T, lane *RawLane, raw []byte) RawMessageRecord {
	t.Helper()
	message, err := ParseJSONMessage(raw)
	if err != nil {
		t.Fatalf("parse raw fixture: %v", err)
	}
	record, err := lane.Capture(raw, EncodingJSON, &message)
	if err != nil {
		t.Fatalf("capture raw fixture: %v", err)
	}
	return record
}
