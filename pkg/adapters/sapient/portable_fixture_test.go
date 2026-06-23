package sapient

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPortableSAPIENTFixturesMatchRuntimePayloads(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		runtime []byte
		content ContentKind
	}{
		{
			name:    "task ack",
			path:    "../../../fixtures/sapient/task-ack.json",
			runtime: TaskAckFixtureJSON(),
			content: ContentTaskAck,
		},
		{
			name:    "absolute detection",
			path:    "../../../fixtures/sapient/absolute-detection.json",
			runtime: DetectionFixtureJSON(),
			content: ContentDetectionReport,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portable, err := os.ReadFile(filepath.Clean(tt.path))
			if err != nil {
				t.Fatalf("read portable fixture: %v", err)
			}
			requireSameJSON(t, portable, tt.runtime)

			msg, err := ParseJSONMessage(portable)
			if err != nil {
				t.Fatalf("parse portable fixture: %v", err)
			}
			if msg.Content != tt.content {
				t.Fatalf("content = %s, want %s", msg.Content, tt.content)
			}
		})
	}
}

func requireSameJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var gotCompact bytes.Buffer
	if err := json.Compact(&gotCompact, got); err != nil {
		t.Fatalf("compact got JSON: %v", err)
	}
	var wantCompact bytes.Buffer
	if err := json.Compact(&wantCompact, want); err != nil {
		t.Fatalf("compact want JSON: %v", err)
	}
	if !bytes.Equal(gotCompact.Bytes(), wantCompact.Bytes()) {
		t.Fatalf("portable fixture JSON differs from runtime payload\ngot:  %s\nwant: %s", gotCompact.String(), wantCompact.String())
	}
}
