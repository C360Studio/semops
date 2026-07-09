package semlinkdemo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseArtifactAcceptsGeneratedMixedSITLArtifact(t *testing.T) {
	artifact := mustParseArtifactFixture(t, "generated-mixed-sitl.artifact.json", ArtifactOptions{})

	if artifact.ArtifactKind != ArtifactKind {
		t.Fatalf("artifact kind = %q", artifact.ArtifactKind)
	}
	if artifact.Source.SourceFidelity != FidelityGenerated {
		t.Fatalf("source fidelity = %q", artifact.Source.SourceFidelity)
	}
	if artifact.Source.SemLinkVersion == "" || artifact.Source.GeneratorProfile != "ardurover-sitl-mixed" {
		t.Fatalf("source metadata missing: %+v", artifact.Source)
	}
	if artifact.Report.Kind != SimpleMeshReportKind || artifact.Report.NodeCount() != 3 {
		t.Fatalf("report = %+v", artifact.Report)
	}
	var sitlNodes int
	for _, node := range artifact.Source.Nodes {
		if node.SourceFidelity == FidelitySITLBacked {
			sitlNodes++
			if node.NodeID != "semlink-node-alpha" ||
				node.SimulatorFamily != "ardupilot" ||
				node.MAVLinkSystemID != 42 ||
				!strings.Contains(node.VehicleSource, "ArduRover SITL") {
				t.Fatalf("SITL node metadata = %+v", node)
			}
		}
	}
	if sitlNodes != 1 {
		t.Fatalf("SITL node count = %d, want 1", sitlNodes)
	}
}

func TestParseArtifactWrapsCommittedReportAsFixtureEvidence(t *testing.T) {
	raw := mustReadArtifactFixture(t, "simple-mesh.report.json")
	artifact, err := ParseArtifact(raw, ArtifactOptions{})
	if err != nil {
		t.Fatalf("parse raw fixture report: %v", err)
	}

	if artifact.ArtifactKind != RawReportArtifactKind {
		t.Fatalf("artifact kind = %q", artifact.ArtifactKind)
	}
	if artifact.Source.SourceFidelity != FidelityFixture {
		t.Fatalf("source fidelity = %q", artifact.Source.SourceFidelity)
	}
	if len(artifact.Source.Nodes) != 3 {
		t.Fatalf("source nodes = %d, want 3", len(artifact.Source.Nodes))
	}
	if artifact.Source.Nodes[0].SourceFidelity != FidelityDeterministic {
		t.Fatalf("node source fidelity = %q", artifact.Source.Nodes[0].SourceFidelity)
	}
}

func TestParseArtifactRejectsStaleArtifactWhenFreshnessConfigured(t *testing.T) {
	raw := mustReadArtifactFixture(t, "generated-mixed-sitl.artifact.json")
	_, err := ParseArtifact(raw, ArtifactOptions{
		Now:    func() time.Time { return time.Date(2026, 7, 9, 12, 10, 0, 0, time.UTC) },
		MaxAge: time.Minute,
	})
	if err == nil || !strings.Contains(err.Error(), "stale semlink demo artifact") {
		t.Fatalf("err = %v, want stale artifact rejection", err)
	}
}

func TestParseArtifactRejectsSITLClaimWithoutNodeSimulatorMetadata(t *testing.T) {
	raw := string(mustReadArtifactFixture(t, "generated-mixed-sitl.artifact.json"))
	raw = strings.Replace(
		raw,
		"        \"simulator_family\": \"ardupilot\",\n        \"vehicle_source\": \"ArduRover SITL without Gazebo\",",
		"        \"vehicle_source\": \"ArduRover SITL without Gazebo\",",
		1,
	)

	_, err := ParseArtifact([]byte(raw), ArtifactOptions{})
	if err == nil || !strings.Contains(err.Error(), "SITL-backed node") {
		t.Fatalf("err = %v, want SITL metadata rejection", err)
	}
}

func TestParseArtifactRejectsUnsupportedReportKind(t *testing.T) {
	raw := string(mustReadArtifactFixture(t, "generated-mixed-sitl.artifact.json"))
	raw = strings.Replace(raw, "\"kind\": \"simple-mesh-companion-demo\"", "\"kind\": \"generic-robotics-demo\"", 1)

	_, err := ParseArtifact([]byte(raw), ArtifactOptions{})
	if err == nil || !strings.Contains(err.Error(), "unsupported semlink demo report kind") {
		t.Fatalf("err = %v, want unsupported report kind rejection", err)
	}
}

func TestParseArtifactRejectsUnknownSourceNode(t *testing.T) {
	raw := string(mustReadArtifactFixture(t, "generated-mixed-sitl.artifact.json"))
	raw = strings.Replace(
		raw,
		"        \"node_id\": \"semlink-node-charlie\",\n        \"source_fidelity\": \"deterministic\",",
		"        \"node_id\": \"semlink-node-delta\",\n        \"source_fidelity\": \"deterministic\",",
		1,
	)

	_, err := ParseArtifact([]byte(raw), ArtifactOptions{})
	if err == nil || !strings.Contains(err.Error(), "is not present in report") {
		t.Fatalf("err = %v, want unknown source node rejection", err)
	}
}

func mustParseArtifactFixture(t *testing.T, name string, opts ArtifactOptions) Artifact {
	t.Helper()
	artifact, err := ParseArtifact(mustReadArtifactFixture(t, name), opts)
	if err != nil {
		t.Fatalf("parse artifact fixture %s: %v", name, err)
	}
	return artifact
}

func mustReadArtifactFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "contracts", "semlink-companion-demo-v0", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return raw
}
