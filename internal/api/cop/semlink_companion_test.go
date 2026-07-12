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

func TestCompanionFleetFromSemLinkGeneratedArtifactLabelsMixedSITL(t *testing.T) {
	artifact := mustDecodeSemLinkArtifactFixture(t, "generated-mixed-sitl.artifact.json")

	fleet := CompanionFleetFromSemLinkArtifact(artifact)

	if fleet.SourceFidelity != semlinkdemo.FidelityGenerated {
		t.Fatalf("source fidelity = %q", fleet.SourceFidelity)
	}
	if fleet.SemLinkVersion != "v0.0.0-e2e" ||
		fleet.GeneratorProfile != "ardurover-sitl-mixed" ||
		fleet.SimulatorFamily != "ardupilot" {
		t.Fatalf("generated metadata missing: %+v", fleet)
	}
	if fleet.SITLBackedNodes != 1 || !strings.Contains(fleet.LiveSourceSummary, "1 SITL-backed ardupilot node") {
		t.Fatalf("live source summary = %q sitl=%d", fleet.LiveSourceSummary, fleet.SITLBackedNodes)
	}
	if !strings.Contains(fleet.DemoEvidenceLabel, "not live BlueOS") ||
		!strings.Contains(fleet.DemoEvidenceLabel, "not n live ArduPilot vehicles") {
		t.Fatalf("demo evidence label overclaims: %q", fleet.DemoEvidenceLabel)
	}
	alpha := fleet.Nodes[0]
	if alpha.ID != "semlink-node-alpha" ||
		alpha.SourceFidelity != semlinkdemo.FidelitySITLBacked ||
		alpha.SimulatorFamily != "ardupilot" ||
		alpha.MAVLinkSystemID != 42 ||
		!strings.Contains(alpha.LiveSourcePosture, "MAVLink system 42") {
		t.Fatalf("alpha source metadata = %+v", alpha)
	}
	if fleet.Readback.CommandACKStatus != "unavailable" || fleet.Readback.ResultStatus != "unavailable" {
		t.Fatalf("generated artifact should not synthesize readback status: %+v", fleet.Readback)
	}
}

func TestCompanionFleetFromSemLinkGeneratedOneToOneMeshArtifact(t *testing.T) {
	artifact := mustDecodeSemLinkArtifactFixture(t, "generated-one-to-one-mesh.artifact.json")

	fleet := CompanionFleetFromSemLinkArtifact(artifact)

	if fleet.EvidenceKind != semlinkdemo.SimpleMeshReportKind ||
		fleet.SourceFidelity != semlinkdemo.FidelityDeterministic ||
		fleet.NodeCount != 3 ||
		fleet.VehicleCount != 3 ||
		fleet.ExpectedSummaries != 3 {
		t.Fatalf("one-to-one mesh fleet = %+v", fleet)
	}
	if fleet.SemLinkCommit != "11f7e6dafea06898f1262877b7d0300e311237f1" ||
		fleet.GeneratorCommand != "semlink-demo -mode mesh -nodes 3 -vehicle-profile ardurover" ||
		fleet.GeneratorProfile != "mesh-deterministic" {
		t.Fatalf("generated source metadata missing: %+v", fleet)
	}
	if fleet.SITLBackedNodes != 0 ||
		!strings.Contains(fleet.LiveSourceSummary, "SemLink-generated deterministic evidence") ||
		!strings.Contains(fleet.DemoEvidenceLabel, "not live BlueOS") ||
		!strings.Contains(fleet.DemoEvidenceLabel, "not live BlueOS, Navigator, hardware, radio") {
		t.Fatalf("one-to-one source labels overclaim: %+v", fleet)
	}
	if !fleet.RawMAVLinkExcluded ||
		!strings.Contains(fleet.RawMAVLinkPolicy, "local-only") ||
		!strings.Contains(fleet.NoTransmitPosture, "no native transmit authority") ||
		!strings.Contains(fleet.NoTransmitPosture, "no companion hardware transmit authority") {
		t.Fatalf("mesh safety posture missing: %+v", fleet)
	}
	for _, node := range fleet.Nodes {
		if node.VehicleCount != 1 ||
			node.InitialSummaryCount != 1 ||
			node.FinalSummaryCount != 3 ||
			node.WatermarkCount != 3 ||
			node.AppliedDiffCount != 2 ||
			node.DiffItemCount != 2 ||
			node.SourceFidelity != semlinkdemo.FidelityDeterministic {
			t.Fatalf("one-to-one node evidence = %+v", node)
		}
	}
}

func TestSemLinkArtifactProviderOverlaysFixtureFleet(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 6, 0, 0, time.UTC)
	provider, err := NewSemLinkArtifactProviderFromFile(
		semLinkArtifactFixturePath("generated-mixed-sitl.artifact.json"),
		NewFixtureProvider(func() time.Time { return now }),
		semlinkdemo.ArtifactOptions{
			Now:    func() time.Time { return now },
			MaxAge: 5 * time.Minute,
		},
	)
	if err != nil {
		t.Fatalf("artifact provider: %v", err)
	}

	snapshot, err := provider.Snapshot(t.Context())
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
	if fleet.SourceFidelity != semlinkdemo.FidelityGenerated || fleet.SITLBackedNodes != 1 {
		t.Fatalf("provider fleet = %+v", fleet)
	}
	if len(snapshot.Tracks) == 0 || len(snapshot.Tasks) == 0 {
		t.Fatalf(
			"provider should preserve fallback COP content: tracks=%d tasks=%d",
			len(snapshot.Tracks),
			len(snapshot.Tasks),
		)
	}
}

func TestSemLinkArtifactProviderOverlaysOneToOneMeshArtifact(t *testing.T) {
	provider, err := NewSemLinkArtifactProviderFromFile(
		semLinkArtifactFixturePath("generated-one-to-one-mesh.artifact.json"),
		NewFixtureProvider(func() time.Time { return time.Unix(20, 0).UTC() }),
		semlinkdemo.ArtifactOptions{},
	)
	if err != nil {
		t.Fatalf("artifact provider: %v", err)
	}

	snapshot, err := provider.Snapshot(t.Context())
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
	if fleet.SourceFidelity != semlinkdemo.FidelityDeterministic ||
		fleet.SITLBackedNodes != 0 ||
		fleet.NodeCount != 3 ||
		fleet.ExpectedSummaries != 3 ||
		!fleet.RawMAVLinkExcluded {
		t.Fatalf("provider one-to-one mesh fleet = %+v", fleet)
	}
	if len(snapshot.Tracks) == 0 || len(snapshot.Tasks) == 0 {
		t.Fatalf(
			"provider should preserve fallback COP content: tracks=%d tasks=%d",
			len(snapshot.Tracks),
			len(snapshot.Tasks),
		)
	}
}

func TestSemLinkArtifactProviderReplacesFixtureFleetWhenArtifactKindChanges(t *testing.T) {
	now := time.Date(2026, 7, 11, 15, 1, 0, 0, time.UTC)
	provider, err := NewSemLinkArtifactProviderFromFile(
		semLinkArtifactFixturePath("generated-sitl-single.artifact.json"),
		NewFixtureProvider(func() time.Time { return now }),
		semlinkdemo.ArtifactOptions{
			Now:    func() time.Time { return now },
			MaxAge: 5 * time.Minute,
		},
	)
	if err != nil {
		t.Fatalf("artifact provider: %v", err)
	}

	snapshot, err := provider.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Summary.ActiveCompanionNodes != 1 {
		t.Fatalf("active companion nodes = %d, want 1", snapshot.Summary.ActiveCompanionNodes)
	}
	if len(snapshot.CompanionFleets) != 1 {
		t.Fatalf("companion fleets = %d, want 1", len(snapshot.CompanionFleets))
	}
	fleet := snapshot.CompanionFleets[0]
	if fleet.EvidenceKind != semlinkdemo.SingleNodeReportKind ||
		fleet.SourceFidelity != semlinkdemo.FidelitySITLBacked ||
		fleet.SITLBackedNodes != 1 ||
		fleet.NodeCount != 1 {
		t.Fatalf("provider fleet = %+v", fleet)
	}
	if len(snapshot.Tracks) == 0 || len(snapshot.Tasks) == 0 {
		t.Fatalf(
			"provider should preserve fallback COP content: tracks=%d tasks=%d",
			len(snapshot.Tracks),
			len(snapshot.Tasks),
		)
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

func mustDecodeSemLinkArtifactFixture(t *testing.T, name string) semlinkdemo.Artifact {
	t.Helper()
	path := semLinkArtifactFixturePath(name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	artifact, err := semlinkdemo.DecodeArtifact(strings.NewReader(string(raw)), semlinkdemo.ArtifactOptions{})
	if err != nil {
		t.Fatalf("decode artifact fixture %s: %v", name, err)
	}
	return artifact
}

func mustDecodeSemLinkDemoFixture(t *testing.T, name string) semlinkdemo.Report {
	t.Helper()
	path := semLinkArtifactFixturePath(name)
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

func semLinkArtifactFixturePath(name string) string {
	return filepath.Join("..", "..", "..", "testdata", "contracts", "semlink-companion-demo-v0", name)
}
