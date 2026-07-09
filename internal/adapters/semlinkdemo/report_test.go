package semlinkdemo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeReportAcceptsSemLinkDemoFixtures(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		kind      string
		nodeCount int
	}{
		{
			name:      "single node",
			file:      "single-node.report.json",
			kind:      SingleNodeReportKind,
			nodeCount: 1,
		},
		{
			name:      "simple mesh",
			file:      "simple-mesh.report.json",
			kind:      SimpleMeshReportKind,
			nodeCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := mustDecodeFixture(t, tt.file)
			if report.Kind != tt.kind {
				t.Fatalf("kind = %q, want %q", report.Kind, tt.kind)
			}
			if report.GeneratedAt.IsZero() {
				t.Fatalf("generated_at missing")
			}
			if report.VehicleProfile == "" {
				t.Fatalf("vehicle_profile missing")
			}
			if report.NodeCount() != tt.nodeCount {
				t.Fatalf("node count = %d, want %d", report.NodeCount(), tt.nodeCount)
			}
			if report.VehicleCount() != tt.nodeCount {
				t.Fatalf("vehicle count = %d, want %d", report.VehicleCount(), tt.nodeCount)
			}
			if !report.AssertionsPassed() {
				t.Fatalf("expected fixture assertions to pass: %+v", report.Assertions)
			}
		})
	}
}

func TestDecodeReportRejectsUnknownKind(t *testing.T) {
	_, err := ParseReport([]byte(`{"kind":"blueos-magic-mesh","generated_at":"2026-07-09T12:00:00Z"}`))
	if err == nil || !strings.Contains(err.Error(), "unsupported semlink demo report kind") {
		t.Fatalf("err = %v, want unsupported kind", err)
	}
}

func TestDecodeReportSurfacesFailedAssertions(t *testing.T) {
	report, err := ParseReport([]byte(`{
		"kind":"simple-mesh-companion-demo",
		"generated_at":"2026-07-09T12:00:00Z",
		"vehicle_profile":"ardurover-blueboat",
		"expected_summaries":1,
		"nodes":[{"node_id":"semlink-node-alpha","vehicle_count":1}],
		"raw_mavlink_exclusion":{"rejected_by_summary_index":true,"policy":"raw MAVLink not summarized"},
		"assertions":[{"name":"watermark-catch-up","passed":false,"detail":"node beta missing"}]
	}`))
	if err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.AssertionsPassed() {
		t.Fatalf("assertions passed unexpectedly: %+v", report.Assertions)
	}
}

func TestDecodeReportRejectsMissingNodeIdentity(t *testing.T) {
	_, err := ParseReport([]byte(`{
		"kind":"single-node-companion-demo",
		"generated_at":"2026-07-09T12:00:00Z",
		"vehicle_profile":"ardurover-blueboat",
		"node":{"vehicle_count":1},
		"assertions":[{"name":"node-ready","passed":true}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "missing node_id") {
		t.Fatalf("err = %v, want missing node_id", err)
	}
}

func TestDecodeReportPreservesRawMAVLinkExclusionPosture(t *testing.T) {
	report := mustDecodeFixture(t, "simple-mesh.report.json")
	if !report.RawMAVLinkExcluded() {
		t.Fatalf("fixture should preserve raw MAVLink exclusion posture: %+v", report.RawMAVLinkExclusion)
	}

	report, err := ParseReport([]byte(`{
		"kind":"simple-mesh-companion-demo",
		"generated_at":"2026-07-09T12:00:00Z",
		"vehicle_profile":"ardurover-blueboat",
		"expected_summaries":1,
		"nodes":[{"node_id":"semlink-node-alpha","vehicle_count":1}],
		"raw_mavlink_exclusion":{"rejected_by_summary_index":false,"policy":"not enforced"},
		"assertions":[{"name":"node-ready","passed":true}]
	}`))
	if err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.RawMAVLinkExcluded() {
		t.Fatalf("raw MAVLink exclusion should not be inferred when SemLink reports it false")
	}
}

func mustDecodeFixture(t *testing.T, name string) Report {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "contracts", "semlink-companion-demo-v0", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	report, err := DecodeReport(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return report
}
