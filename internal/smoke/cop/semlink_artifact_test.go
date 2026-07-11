package cop

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/c360studio/semops/internal/adapters/semlinkdemo"
	copapi "github.com/c360studio/semops/internal/api/cop"
)

const liveSemLinkArtifactPathEnv = "SEMOPS_COP_SMOKE_SEMLINK_ARTIFACT_PATH"

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

	provider := copapi.NewSemLinkArtifactProvider(artifact, copapi.NewFixtureProvider(nil))
	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("build COP snapshot from SemLink artifact: %v", err)
	}
	if snapshot.Summary.ActiveCompanionNodes < 3 {
		t.Fatalf("active companion nodes = %d, want at least 3", snapshot.Summary.ActiveCompanionNodes)
	}
	if len(snapshot.CompanionFleets) == 0 {
		t.Fatal("snapshot has no companion fleet")
	}
	fleet := snapshot.CompanionFleets[0]
	if fleet.Source != "semlink" || fleet.SourceFidelity == "" {
		t.Fatalf("fleet source metadata = %+v", fleet)
	}
	if fleet.NodeCount < 3 || len(fleet.Nodes) < 3 {
		t.Fatalf("fleet nodes = %d/%d, want at least 3/3", fleet.NodeCount, len(fleet.Nodes))
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
