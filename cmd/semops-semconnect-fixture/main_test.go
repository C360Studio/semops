package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunDryRunPrintsReadSideFixturePlan(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"-dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("run dry-run: %v\nstderr: %s", err, stderr.String())
	}

	var evidence map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatalf("decode evidence: %v\n%s", err, stdout.String())
	}
	if evidence["status"] != "planned" ||
		evidence["fixture_id"] != "semops.csapi.read-side.hadr.v1" {
		t.Fatalf("dry-run evidence = %#v", evidence)
	}
	if !strings.Contains(stdout.String(), `"claim_scope":"read-side-egress-only"`) {
		t.Fatalf("dry-run missing read-side claim scope:\n%s", stdout.String())
	}
}

func TestRunPostsFixtureToSemConnectBoundary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"-base-url", server.URL + "/csapi"}, &stdout, &stderr); err != nil {
		t.Fatalf("run fixture: %v\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	var evidence map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatalf("decode evidence: %v\n%s", err, stdout.String())
	}
	if evidence["status"] != "passed" {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestRunRequiresBaseURLUnlessDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected missing base-url error")
	}
	if !strings.Contains(err.Error(), "base URL required") {
		t.Fatalf("error = %v", err)
	}
}
