package cop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/semops/internal/adapters/semlinkdemo"
	copapi "github.com/c360studio/semops/internal/api/cop"
)

const liveSemLinkArtifactPathEnv = "SEMOPS_COP_SMOKE_SEMLINK_ARTIFACT_PATH"

func TestSemLinkGeneratedOneToOneMeshArtifactFixtureSmoke(t *testing.T) {
	artifactPath := semLinkArtifactFixturePath("generated-one-to-one-mesh.artifact.json")
	artifact, err := semlinkdemo.LoadArtifactFile(artifactPath, semlinkdemo.ArtifactOptions{})
	if err != nil {
		t.Fatalf("load SemLink artifact %s: %v", artifactPath, err)
	}

	fleet := requireSemLinkArtifactCOPSnapshot(t, artifact)
	if artifact.Report.Kind != semlinkdemo.SimpleMeshReportKind ||
		artifact.Source.SourceFidelity != semlinkdemo.FidelityDeterministic ||
		artifact.Report.ExpectedSummaries != 3 {
		t.Fatalf("fixture is not the one-to-one mesh proof: artifact=%+v report=%+v", artifact.Source, artifact.Report)
	}
	if fleet.NodeCount != 3 || fleet.VehicleCount != 3 || fleet.ExpectedSummaries != 3 || fleet.SITLBackedNodes != 0 {
		t.Fatalf("one-to-one mesh fleet = %+v", fleet)
	}
	for _, node := range fleet.Nodes {
		if node.VehicleCount != 1 || node.FinalSummaryCount != 3 || node.WatermarkCount != 3 {
			t.Fatalf("one-to-one node evidence = %+v", node)
		}
	}
}

func TestSemLinkGeneratedArtifactSmoke(t *testing.T) {
	artifactPath := strings.TrimSpace(os.Getenv(liveSemLinkArtifactPathEnv))
	if artifactPath == "" {
		t.Skipf("set %s to run the SemLink-generated companion artifact smoke", liveSemLinkArtifactPathEnv)
	}

	artifact, err := semlinkdemo.LoadArtifactFile(artifactPath, semlinkdemo.ArtifactOptions{})
	if err != nil {
		t.Fatalf("load SemLink artifact %s: %v", artifactPath, err)
	}
	if artifact.ArtifactKind != semlinkdemo.ArtifactKind {
		t.Fatalf("artifact kind = %q, want %q", artifact.ArtifactKind, semlinkdemo.ArtifactKind)
	}
	if !artifact.GeneratedAt.Equal(artifact.Report.GeneratedAt) {
		t.Fatalf("artifact generated_at %s != report generated_at %s", artifact.GeneratedAt, artifact.Report.GeneratedAt)
	}
	if !hasReleaseSourceRef(artifact.Source) {
		t.Fatalf("artifact source lacks real SemLink source ref: %+v", artifact.Source)
	}

	requireSemLinkArtifactCOPSnapshot(t, artifact)
}

func requireSemLinkArtifactCOPSnapshot(t *testing.T, artifact semlinkdemo.Artifact) copapi.CompanionFleet {
	t.Helper()
	provider := copapi.NewSemLinkArtifactProvider(artifact, copapi.NewFixtureProvider(nil))
	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("build COP snapshot from SemLink artifact: %v", err)
	}
	fleet := generatedSemLinkFleet(t, snapshot.CompanionFleets)
	if snapshot.Summary.ActiveCompanionNodes != fleet.NodeCount {
		t.Fatalf(
			"active companion nodes = %d, want artifact fleet node count %d",
			snapshot.Summary.ActiveCompanionNodes,
			fleet.NodeCount,
		)
	}
	if fleet.Source != "semlink" || fleet.SourceFidelity == "" {
		t.Fatalf("fleet source metadata = %+v", fleet)
	}
	if fleet.NodeCount != artifact.Report.NodeCount() || len(fleet.Nodes) != artifact.Report.NodeCount() {
		t.Fatalf(
			"fleet nodes = %d/%d, want artifact report node count %d",
			fleet.NodeCount,
			len(fleet.Nodes),
			artifact.Report.NodeCount(),
		)
	}
	if !fleet.RawMAVLinkExcluded {
		t.Fatalf("raw MAVLink exclusion not preserved: %+v", fleet.RawMAVLinkExclusion)
	}
	if fleet.Readback.NativeExecutionAllowed || fleet.Readback.CompanionTransmitAllowed {
		t.Fatalf("artifact smoke must not grant transmit authority: %+v", fleet.Readback)
	}
	if !strings.Contains(fleet.NoTransmitPosture, "no native") ||
		!strings.Contains(fleet.NoTransmitPosture, "no companion") {
		t.Fatalf("no-transmit posture missing expected boundaries: %q", fleet.NoTransmitPosture)
	}
	requireFidelitySpecificSmoke(t, artifact, fleet)
	return fleet
}

func hasReleaseSourceRef(source semlinkdemo.ArtifactSource) bool {
	commit := strings.TrimSpace(source.SemLinkCommit)
	if commit != "" && !isPlaceholderSourceRef(commit) {
		return true
	}
	version := strings.TrimSpace(source.SemLinkVersion)
	return version != "" && !isPlaceholderSourceRef(version)
}

func isPlaceholderSourceRef(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "dev", "(devel)", "local", "semlink-demo-local":
		return true
	default:
		return false
	}
}

func generatedSemLinkFleet(t *testing.T, fleets []copapi.CompanionFleet) copapi.CompanionFleet {
	t.Helper()
	for _, fleet := range fleets {
		if fleet.Source == "semlink" && fleet.Provenance.Owner == "semlink.generated.artifact" {
			return fleet
		}
	}
	t.Fatalf("snapshot has no generated SemLink fleet: %+v", fleets)
	return copapi.CompanionFleet{}
}

func requireFidelitySpecificSmoke(t *testing.T, artifact semlinkdemo.Artifact, fleet copapi.CompanionFleet) {
	t.Helper()
	switch artifact.Source.SourceFidelity {
	case semlinkdemo.FidelitySITLBacked:
		if artifact.Report.Kind != semlinkdemo.SingleNodeReportKind {
			t.Fatalf("SITL-backed smoke expects single-node report, got %q", artifact.Report.Kind)
		}
		if fleet.SITLBackedNodes < 1 {
			t.Fatalf("SITL-backed artifact produced no SITL-backed COP nodes: %+v", fleet)
		}
		requireSITLNodeMetadata(t, fleet)
	case semlinkdemo.FidelityDeterministic, semlinkdemo.FidelityGenerated:
		if fleet.SITLBackedNodes != sourceSITLNodeCount(artifact.Source.Nodes) {
			t.Fatalf(
				"SITL-backed node count = %d, want source metadata count %d",
				fleet.SITLBackedNodes,
				sourceSITLNodeCount(artifact.Source.Nodes),
			)
		}
	case semlinkdemo.FidelityHardwareAdjacent:
		t.Fatalf("hardware-adjacent artifact smoke requires a separate acceptance lane: %+v", artifact.Source)
	default:
		t.Fatalf("unsupported artifact source fidelity %q", artifact.Source.SourceFidelity)
	}
}

func requireSITLNodeMetadata(t *testing.T, fleet copapi.CompanionFleet) {
	t.Helper()
	for _, node := range fleet.Nodes {
		if node.SourceFidelity != semlinkdemo.FidelitySITLBacked {
			continue
		}
		if node.SimulatorFamily == "" ||
			node.VehicleSource == "" ||
			node.MAVLinkSystemID <= 0 ||
			node.Route == "" {
			t.Fatalf("SITL-backed node metadata incomplete: %+v", node)
		}
		if !strings.Contains(node.LiveSourcePosture, "MAVLink system") {
			t.Fatalf("SITL-backed node posture missing MAVLink system evidence: %+v", node)
		}
		return
	}
	t.Fatalf("no SITL-backed node found in fleet: %+v", fleet)
}

func sourceSITLNodeCount(nodes []semlinkdemo.NodeSource) int {
	var count int
	for _, node := range nodes {
		if semlinkdemo.NormalizeFidelity(node.SourceFidelity) == semlinkdemo.FidelitySITLBacked {
			count++
		}
	}
	return count
}

func semLinkArtifactFixturePath(name string) string {
	return filepath.Join("..", "..", "..", "testdata", "contracts", "semlink-companion-demo-v0", name)
}
