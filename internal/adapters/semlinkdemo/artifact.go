package semlinkdemo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	ArtifactKind          = "semlink-companion-demo-artifact-v0"
	RawReportArtifactKind = "semlink-companion-demo-raw-report-v0"

	FidelityFixture          = "fixture"
	FidelityGenerated        = "generated"
	FidelityDeterministic    = "deterministic"
	FidelitySITLBacked       = "sitl-backed"
	FidelityHardwareAdjacent = "hardware-adjacent"
)

type Artifact struct {
	ArtifactKind string
	GeneratedAt  time.Time
	Source       ArtifactSource
	Report       Report
}

type ArtifactSource struct {
	SourceFidelity    string       `json:"source_fidelity"`
	SemLinkVersion    string       `json:"semlink_version,omitempty"`
	SemLinkCommit     string       `json:"semlink_commit,omitempty"`
	GeneratorCommand  string       `json:"generator_command,omitempty"`
	GeneratorProfile  string       `json:"generator_profile,omitempty"`
	SimulatorFamily   string       `json:"simulator_family,omitempty"`
	NoTransmitPosture string       `json:"no_transmit_posture,omitempty"`
	Nodes             []NodeSource `json:"nodes,omitempty"`
}

type NodeSource struct {
	NodeID          string `json:"node_id"`
	SourceFidelity  string `json:"source_fidelity"`
	SimulatorFamily string `json:"simulator_family,omitempty"`
	VehicleSource   string `json:"vehicle_source,omitempty"`
	MAVLinkSystemID int    `json:"mavlink_system_id,omitempty"`
	Route           string `json:"route,omitempty"`
}

type ArtifactOptions struct {
	Now    func() time.Time
	MaxAge time.Duration
}

type artifactEnvelope struct {
	ArtifactKind string          `json:"artifact_kind"`
	GeneratedAt  time.Time       `json:"generated_at"`
	Source       ArtifactSource  `json:"source"`
	Report       json.RawMessage `json:"report"`
}

func LoadArtifactFile(path string, opts ArtifactOptions) (Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return Artifact{}, fmt.Errorf("open semlink demo artifact: %w", err)
	}
	defer file.Close()
	return DecodeArtifact(file, opts)
}

func DecodeArtifact(r io.Reader, opts ArtifactOptions) (Artifact, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxReportBytes+1))
	if err != nil {
		return Artifact{}, fmt.Errorf("read semlink demo artifact: %w", err)
	}
	if len(data) > maxReportBytes {
		return Artifact{}, fmt.Errorf("semlink demo artifact exceeds %d bytes", maxReportBytes)
	}
	return ParseArtifact(data, opts)
}

func ParseArtifact(data []byte, opts ArtifactOptions) (Artifact, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Artifact{}, fmt.Errorf("semlink demo artifact is empty")
	}
	var header struct {
		ArtifactKind string `json:"artifact_kind"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return Artifact{}, fmt.Errorf("decode semlink demo artifact kind: %w", err)
	}
	if strings.TrimSpace(header.ArtifactKind) == "" {
		report, err := ParseReport(data)
		if err != nil {
			return Artifact{}, err
		}
		artifact := Artifact{
			ArtifactKind: RawReportArtifactKind,
			GeneratedAt:  report.GeneratedAt,
			Source: ArtifactSource{
				SourceFidelity:    FidelityFixture,
				GeneratorProfile:  "semops-committed-fixture",
				NoTransmitPosture: "fixture evidence only; no native or companion hardware transmit authority",
				Nodes:             fixtureNodeSources(report),
			},
			Report: report,
		}
		if err := validateArtifact(artifact, opts, false); err != nil {
			return Artifact{}, err
		}
		return artifact, nil
	}
	if strings.TrimSpace(header.ArtifactKind) != ArtifactKind {
		return Artifact{}, fmt.Errorf("unsupported semlink demo artifact kind %q", header.ArtifactKind)
	}

	var envelope artifactEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Artifact{}, fmt.Errorf("decode semlink demo artifact: %w", err)
	}
	if len(bytes.TrimSpace(envelope.Report)) == 0 {
		return Artifact{}, fmt.Errorf("semlink demo artifact report is required")
	}
	report, err := ParseReport(envelope.Report)
	if err != nil {
		return Artifact{}, err
	}
	artifact := Artifact{
		ArtifactKind: envelope.ArtifactKind,
		GeneratedAt:  envelope.GeneratedAt,
		Source:       envelope.Source,
		Report:       report,
	}
	if err := validateArtifact(artifact, opts, true); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func fixtureNodeSources(report Report) []NodeSource {
	nodes := make([]NodeSource, 0, len(report.Nodes))
	for _, node := range report.Nodes {
		nodes = append(nodes, NodeSource{
			NodeID:         node.NodeID,
			SourceFidelity: FidelityDeterministic,
		})
	}
	return nodes
}

func validateArtifact(artifact Artifact, opts ArtifactOptions, generatedEnvelope bool) error {
	if strings.TrimSpace(artifact.ArtifactKind) == "" {
		return fmt.Errorf("semlink demo artifact kind is required")
	}
	if artifact.GeneratedAt.IsZero() {
		return fmt.Errorf("semlink demo artifact generated_at is required")
	}
	if opts.MaxAge < 0 {
		return fmt.Errorf("semlink demo artifact max age must be non-negative")
	}
	if opts.MaxAge > 0 && opts.Now != nil {
		now := opts.Now().UTC()
		if now.After(artifact.GeneratedAt) && now.Sub(artifact.GeneratedAt) > opts.MaxAge {
			return fmt.Errorf("stale semlink demo artifact generated_at=%s exceeds max age %s", artifact.GeneratedAt.Format(time.RFC3339), opts.MaxAge)
		}
	}
	fidelity := NormalizeFidelity(artifact.Source.SourceFidelity)
	if fidelity == "" {
		return fmt.Errorf("semlink demo artifact source_fidelity is required")
	}
	if !IsKnownFidelity(fidelity) {
		return fmt.Errorf("unsupported semlink demo artifact source_fidelity %q", artifact.Source.SourceFidelity)
	}
	artifact.Source.SourceFidelity = fidelity
	if generatedEnvelope {
		if !artifact.GeneratedAt.Equal(artifact.Report.GeneratedAt) {
			return fmt.Errorf(
				"semlink demo artifact generated_at %s does not match report generated_at %s",
				artifact.GeneratedAt.Format(time.RFC3339),
				artifact.Report.GeneratedAt.Format(time.RFC3339),
			)
		}
		if !hasSemLinkSourceRef(artifact.Source) {
			return fmt.Errorf("semlink demo artifact source requires semlink_version or semlink_commit")
		}
		if !hasGeneratorRef(artifact.Source) {
			return fmt.Errorf("semlink demo artifact source requires generator_command or generator_profile")
		}
		if strings.TrimSpace(artifact.Source.NoTransmitPosture) == "" {
			return fmt.Errorf("semlink demo artifact source requires no_transmit_posture")
		}
	}

	reportNodes := make(map[string]struct{}, len(artifact.Report.Nodes))
	for _, node := range artifact.Report.Nodes {
		reportNodes[node.NodeID] = struct{}{}
	}
	sourceNodes := make(map[string]NodeSource, len(artifact.Source.Nodes))
	for _, node := range artifact.Source.Nodes {
		nodeID := strings.TrimSpace(node.NodeID)
		if nodeID == "" {
			return fmt.Errorf("semlink demo artifact source node_id is required")
		}
		if _, ok := reportNodes[nodeID]; !ok {
			return fmt.Errorf("semlink demo artifact source node %q is not present in report", nodeID)
		}
		if _, ok := sourceNodes[nodeID]; ok {
			return fmt.Errorf("semlink demo artifact source has duplicate node %q", nodeID)
		}
		node.SourceFidelity = NormalizeFidelity(node.SourceFidelity)
		if node.SourceFidelity == "" {
			node.SourceFidelity = fidelity
		}
		if !IsKnownFidelity(node.SourceFidelity) {
			return fmt.Errorf("unsupported semlink demo artifact node source_fidelity %q", node.SourceFidelity)
		}
		if err := validateNodeSource(node); err != nil {
			return err
		}
		sourceNodes[nodeID] = node
	}
	if (fidelity == FidelitySITLBacked || fidelity == FidelityHardwareAdjacent) && len(sourceNodes) == 0 {
		return fmt.Errorf("semlink demo artifact source_fidelity %q requires node source metadata", fidelity)
	}
	return nil
}

func validateNodeSource(node NodeSource) error {
	switch node.SourceFidelity {
	case FidelitySITLBacked:
		if strings.TrimSpace(node.SimulatorFamily) == "" {
			return fmt.Errorf("semlink demo artifact SITL-backed node %q requires simulator_family", node.NodeID)
		}
		if strings.TrimSpace(node.VehicleSource) == "" {
			return fmt.Errorf("semlink demo artifact SITL-backed node %q requires vehicle_source", node.NodeID)
		}
		if node.MAVLinkSystemID <= 0 || node.MAVLinkSystemID > 255 {
			return fmt.Errorf("semlink demo artifact SITL-backed node %q requires mavlink_system_id 1..255", node.NodeID)
		}
	case FidelityHardwareAdjacent:
		if strings.TrimSpace(node.VehicleSource) == "" {
			return fmt.Errorf("semlink demo artifact hardware-adjacent node %q requires vehicle_source", node.NodeID)
		}
		if node.MAVLinkSystemID <= 0 || node.MAVLinkSystemID > 255 {
			return fmt.Errorf("semlink demo artifact hardware-adjacent node %q requires mavlink_system_id 1..255", node.NodeID)
		}
	}
	return nil
}

func NormalizeFidelity(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func IsKnownFidelity(value string) bool {
	switch NormalizeFidelity(value) {
	case FidelityFixture, FidelityGenerated, FidelityDeterministic, FidelitySITLBacked, FidelityHardwareAdjacent:
		return true
	default:
		return false
	}
}

func hasSemLinkSourceRef(source ArtifactSource) bool {
	return strings.TrimSpace(source.SemLinkVersion) != "" || strings.TrimSpace(source.SemLinkCommit) != ""
}

func hasGeneratorRef(source ArtifactSource) bool {
	return strings.TrimSpace(source.GeneratorCommand) != "" || strings.TrimSpace(source.GeneratorProfile) != ""
}
