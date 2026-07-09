package cop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/internal/adapters/semlinkdemo"
)

func TestCompanionFleetFromSemLinkMeshReport(t *testing.T) {
	report := mustDecodeSemLinkDemoFixture(t, "simple-mesh.report.json")

	fleet := CompanionFleetFromSemLinkDemoReport(report)

	if fleet.EvidenceKind != semlinkdemo.SimpleMeshReportKind {
		t.Fatalf("kind = %q", fleet.EvidenceKind)
	}
	if fleet.NodeCount != 3 || len(fleet.Nodes) != 3 {
		t.Fatalf("nodes = %d/%d, want 3/3", fleet.NodeCount, len(fleet.Nodes))
	}
	if fleet.VehicleCount != 3 || fleet.VehicleProfile != "ardurover-blueboat" {
		t.Fatalf("fleet vehicle view = %+v", fleet)
	}
	if !fleet.RawMAVLinkExcluded || !strings.Contains(fleet.NoTransmitPosture, "no native transmit authority") {
		t.Fatalf("no-transmit posture missing: %+v", fleet)
	}
	if strings.Contains(fleet.RawMAVLinkPolicy, "nats://") {
		t.Fatalf("raw peer URLs leaked into COP policy: %q", fleet.RawMAVLinkPolicy)
	}
	if fleet.Readback.AdapterStatus != "unavailable" ||
		fleet.Readback.CommandACKStatus != "unavailable" ||
		fleet.Readback.ResultStatus != "unavailable" {
		t.Fatalf("absent readback fields should be unavailable: %+v", fleet.Readback)
	}
}

func TestCompanionFleetReadbackKeepsAdapterACKAndResultSeparate(t *testing.T) {
	observed := time.Date(2026, 7, 9, 12, 1, 0, 0, time.UTC)
	report := semlinkdemo.Report{
		Kind:           semlinkdemo.SingleNodeReportKind,
		GeneratedAt:    observed,
		VehicleProfile: "ardurover-blueboat",
		Nodes:          []semlinkdemo.Node{{NodeID: "semlink-node-alpha", VehicleCount: 1}},
		RawMAVLinkExclusion: semlinkdemo.RawMAVLinkExclusion{
			RejectedBySummaryIndex: true,
			Policy:                 "raw MAVLink excluded",
		},
		SemOpsReadback: semlinkdemo.Readback{
			Response: semlinkdemo.ReadbackResponse{
				Accepted:                 true,
				Status:                   "accepted",
				CorrelationID:            "readback-1",
				NativeExecutionAllowed:   false,
				CompanionTransmitAllowed: false,
			},
			ReceivedRequests: 1,
			CommandACK: semlinkdemo.CommandACKStatus{
				Status:     "accepted",
				CommandID:  512,
				Result:     "MAV_RESULT_ACCEPTED",
				Accepted:   true,
				ObservedAt: observed.Add(1 * time.Second),
			},
			Result: semlinkdemo.ReadbackResult{
				Status:           "observed",
				ObservedProperty: "AUTOPILOT_VERSION",
				MessageID:        148,
				ObservedAt:       observed.Add(2 * time.Second),
			},
		},
		Assertions: []semlinkdemo.Assertion{{Name: "readback", Passed: true}},
	}

	fleet := CompanionFleetFromSemLinkDemoReport(report)

	if fleet.Readback.AdapterStatus != "accepted" ||
		fleet.Readback.CommandACKStatus != "accepted" ||
		fleet.Readback.CommandACKResult != "MAV_RESULT_ACCEPTED" ||
		fleet.Readback.ResultStatus != "observed" ||
		fleet.Readback.ResultObservedProperty != "AUTOPILOT_VERSION" {
		t.Fatalf("readback separation lost: %+v", fleet.Readback)
	}
	if fleet.Readback.NativeExecutionAllowed || fleet.Readback.CompanionTransmitAllowed {
		t.Fatalf("readback should not imply transmit authority: %+v", fleet.Readback)
	}
}

func TestFixtureProviderIncludesSemLinkCompanionFleet(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 3, 0, 0, time.UTC)
	snapshot, err := NewFixtureProvider(func() time.Time { return now }).Snapshot(t.Context())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Summary.ActiveCompanionNodes != 3 {
		t.Fatalf("active companion nodes = %d, want 3", snapshot.Summary.ActiveCompanionNodes)
	}
	if len(snapshot.CompanionFleets) != 1 {
		t.Fatalf("companion fleets = %d, want 1", len(snapshot.CompanionFleets))
	}
	fleet := snapshot.CompanionFleets[0]
	if fleet.NodeCount != 3 || fleet.VehicleCount != 3 || !fleet.RawMAVLinkExcluded {
		t.Fatalf("fixture companion fleet = %+v", fleet)
	}
	if !strings.Contains(fleet.DemoEvidenceLabel, "not live BlueOS") {
		t.Fatalf("demo evidence label should avoid live hardware claim: %q", fleet.DemoEvidenceLabel)
	}
}

func mustDecodeSemLinkDemoFixture(t *testing.T, name string) semlinkdemo.Report {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "contracts", "semlink-companion-demo-v0", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	report, err := semlinkdemo.DecodeReport(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return report
}
