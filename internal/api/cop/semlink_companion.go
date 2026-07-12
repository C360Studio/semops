package cop

import (
	"fmt"
	"strings"

	"github.com/c360studio/semops/internal/adapters/semlinkdemo"
)

const semLinkDemoEvidenceLabel = "SemLink-produced deterministic demo evidence; not live BlueOS, Navigator, hardware, radio, or mesh reliability evidence"

func CompanionFleetFromSemLinkDemoReport(report semlinkdemo.Report) CompanionFleet {
	nodes := make([]CompanionNode, 0, len(report.Nodes))
	for _, node := range report.Nodes {
		nodes = append(nodes, CompanionNode{
			ID:                  node.NodeID,
			VehicleCount:        node.VehicleCount,
			PeerCount:           node.PeerCount,
			InitialSummaryCount: node.InitialSummaryCount,
			FinalSummaryCount:   node.FinalSummaryCount,
			WatermarkCount:      node.WatermarkCount,
			AppliedDiffCount:    node.AppliedDiffCount,
			DiffItemCount:       node.DiffItemCount,
			TTLMergePosture:     node.TTLMergePosture,
			SourceFidelity:      semlinkdemo.FidelityDeterministic,
			LiveSourcePosture:   "deterministic SemLink fixture summary evidence",
		})
	}

	rawPolicy := report.RawMAVLinkExclusion.Policy
	if rawPolicy == "" {
		rawPolicy = report.Evidence.RawMAVLinkReplicationPolicy
	}
	fleet := CompanionFleet{
		ID:                 semLinkFleetID(report.Kind),
		Label:              semLinkFleetLabel(report),
		Source:             "semlink",
		Status:             semLinkFleetStatus(report),
		EvidenceKind:       report.Kind,
		VehicleProfile:     report.VehicleProfile,
		NodeCount:          report.NodeCount(),
		VehicleCount:       report.VehicleCount(),
		ExpectedSummaries:  report.ExpectedSummaries,
		AssertionState:     semLinkAssertionState(report),
		SourceFidelity:     semlinkdemo.FidelityFixture,
		LiveSourceSummary:  semLinkLiveSourceSummary(report.NodeCount(), 0, semlinkdemo.FidelityFixture, ""),
		NoTransmitPosture:  semLinkNoTransmitPosture(report),
		RawMAVLinkExcluded: report.RawMAVLinkExcluded(),
		RawMAVLinkPolicy:   rawPolicy,
		DemoEvidenceLabel:  semLinkDemoEvidenceLabel,
		Confidence:         1,
		UpdatedAt:          report.GeneratedAt,
		Nodes:              nodes,
		Readback:           semLinkReadback(report),
		CommandPosture:     semLinkCommandPosture(report),
		RawMAVLinkExclusion: CompanionRawMAVLinkExclusion{
			RejectedBySummaryIndex: report.RawMAVLinkExclusion.RejectedBySummaryIndex,
			Policy:                 report.RawMAVLinkExclusion.Policy,
			Error:                  report.RawMAVLinkExclusion.Error,
		},
		Assertions: semLinkAssertions(report),
		Provenance: Provenance{
			Owner:     "semlink.e2e.demo",
			SourceRef: fmt.Sprintf("semlink-demo://%s/%s", report.Kind, report.GeneratedAt.Format("20060102T150405Z")),
			Observed:  report.GeneratedAt,
		},
	}
	if fleet.RawMAVLinkExclusion.Policy == "" {
		fleet.RawMAVLinkExclusion.Policy = rawPolicy
	}
	return fleet
}

func CompanionFleetFromSemLinkArtifact(artifact semlinkdemo.Artifact) CompanionFleet {
	fleet := CompanionFleetFromSemLinkDemoReport(artifact.Report)
	source := artifact.Source
	sourceFidelity := semlinkdemo.NormalizeFidelity(source.SourceFidelity)
	if sourceFidelity == "" {
		sourceFidelity = semlinkdemo.FidelityGenerated
	}
	fleet.SourceFidelity = sourceFidelity
	fleet.SemLinkVersion = source.SemLinkVersion
	fleet.SemLinkCommit = source.SemLinkCommit
	fleet.GeneratorCommand = source.GeneratorCommand
	fleet.GeneratorProfile = source.GeneratorProfile
	fleet.SimulatorFamily = source.SimulatorFamily
	fleet.UpdatedAt = artifact.GeneratedAt
	fleet.DemoEvidenceLabel = semLinkArtifactEvidenceLabel(artifact)
	fleet.NoTransmitPosture = semLinkAppendUniquePosture(fleet.NoTransmitPosture, source.NoTransmitPosture)
	fleet.Provenance = Provenance{
		Owner:     "semlink.generated.artifact",
		SourceRef: fmt.Sprintf("semlink-artifact://%s/%s/%s", artifact.ArtifactKind, artifact.Report.Kind, artifact.GeneratedAt.Format("20060102T150405Z")),
		Observed:  artifact.GeneratedAt,
	}

	sourceByNode := make(map[string]semlinkdemo.NodeSource, len(source.Nodes))
	for _, node := range source.Nodes {
		sourceByNode[node.NodeID] = node
	}
	var sitlNodes int
	for index := range fleet.Nodes {
		node := &fleet.Nodes[index]
		nodeSource, ok := sourceByNode[node.ID]
		if !ok {
			node.SourceFidelity = sourceFidelity
			node.LiveSourcePosture = semLinkNodeSourcePosture(semlinkdemo.NodeSource{
				NodeID:         node.ID,
				SourceFidelity: sourceFidelity,
			})
			continue
		}
		nodeFidelity := semlinkdemo.NormalizeFidelity(nodeSource.SourceFidelity)
		if nodeFidelity == "" {
			nodeFidelity = sourceFidelity
		}
		node.SourceFidelity = nodeFidelity
		node.SimulatorFamily = nodeSource.SimulatorFamily
		node.VehicleSource = nodeSource.VehicleSource
		node.MAVLinkSystemID = nodeSource.MAVLinkSystemID
		node.Route = nodeSource.Route
		node.LiveSourcePosture = semLinkNodeSourcePosture(nodeSource)
		if nodeFidelity == semlinkdemo.FidelitySITLBacked {
			sitlNodes++
		}
	}
	fleet.SITLBackedNodes = sitlNodes
	fleet.LiveSourceSummary = semLinkLiveSourceSummary(fleet.NodeCount, sitlNodes, sourceFidelity, source.SimulatorFamily)
	return fleet
}

func semLinkFleetID(kind string) string {
	switch kind {
	case semlinkdemo.SingleNodeReportKind:
		return "c360.edge.cop.semlink.fleet.single-node-demo"
	case semlinkdemo.SimpleMeshReportKind:
		return "c360.edge.cop.semlink.fleet.simple-mesh-demo"
	default:
		return "c360.edge.cop.semlink.fleet.demo"
	}
}

func semLinkArtifactEvidenceLabel(artifact semlinkdemo.Artifact) string {
	sitlNodes := semLinkSITLNodeCount(artifact.Source.Nodes)
	if sitlNodes > 0 {
		return fmt.Sprintf(
			"SemLink-generated mixed-fidelity evidence with %d ArduPilot SITL-backed %s; not live BlueOS, not Navigator, not hardware, not radio, and not n live ArduPilot vehicles",
			sitlNodes,
			semLinkNodeWord(sitlNodes),
		)
	}
	if semlinkdemo.NormalizeFidelity(artifact.Source.SourceFidelity) == semlinkdemo.FidelityHardwareAdjacent {
		return "SemLink-generated hardware-adjacent evidence; not BlueOS, Navigator, radio, or mesh reliability evidence unless separately proven"
	}
	return "SemLink-generated deterministic demo evidence; not live BlueOS, Navigator, hardware, radio, or mesh reliability evidence"
}

func semLinkLiveSourceSummary(nodeCount int, sitlNodes int, sourceFidelity string, simulatorFamily string) string {
	if sitlNodes > 0 {
		family := semLinkFirstNonEmpty(simulatorFamily, "ArduPilot")
		return fmt.Sprintf(
			"%d SemLink nodes; %d SITL-backed %s %s; remaining nodes are deterministic companion summary evidence",
			nodeCount,
			sitlNodes,
			family,
			semLinkNodeWord(sitlNodes),
		)
	}
	switch semlinkdemo.NormalizeFidelity(sourceFidelity) {
	case semlinkdemo.FidelityFixture:
		return fmt.Sprintf("%d SemLink nodes; deterministic fixture evidence; 0 SITL-backed nodes", nodeCount)
	case semlinkdemo.FidelityGenerated:
		return fmt.Sprintf("%d SemLink nodes; SemLink-generated deterministic evidence; 0 SITL-backed nodes", nodeCount)
	case semlinkdemo.FidelityDeterministic:
		return fmt.Sprintf("%d SemLink nodes; SemLink-generated deterministic evidence; 0 SITL-backed nodes", nodeCount)
	case semlinkdemo.FidelityHardwareAdjacent:
		return fmt.Sprintf("%d SemLink nodes; hardware-adjacent source evidence; 0 SITL-backed nodes", nodeCount)
	default:
		return fmt.Sprintf("%d SemLink nodes; %s evidence; 0 SITL-backed nodes", nodeCount, sourceFidelity)
	}
}

func semLinkNodeSourcePosture(source semlinkdemo.NodeSource) string {
	fidelity := semlinkdemo.NormalizeFidelity(source.SourceFidelity)
	switch fidelity {
	case semlinkdemo.FidelitySITLBacked:
		return fmt.Sprintf(
			"SITL-backed %s source; MAVLink system %d; no hardware transmit claim",
			semLinkFirstNonEmpty(source.SimulatorFamily, "simulator"),
			source.MAVLinkSystemID,
		)
	case semlinkdemo.FidelityDeterministic:
		return "deterministic SemLink companion summary evidence"
	case semlinkdemo.FidelityFixture:
		return "committed SemOps fixture evidence"
	case semlinkdemo.FidelityHardwareAdjacent:
		return "hardware-adjacent source evidence; requires separate field acceptance"
	case semlinkdemo.FidelityGenerated:
		return "SemLink-generated companion summary evidence"
	default:
		return semLinkFirstNonEmpty(fidelity, "unknown source fidelity")
	}
}

func semLinkSITLNodeCount(nodes []semlinkdemo.NodeSource) int {
	var count int
	for _, node := range nodes {
		if semlinkdemo.NormalizeFidelity(node.SourceFidelity) == semlinkdemo.FidelitySITLBacked {
			count++
		}
	}
	return count
}

func semLinkAppendUniquePosture(base string, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return base
	}
	if base == "" {
		return extra
	}
	if strings.Contains(base, extra) {
		return base
	}
	return base + "; " + extra
}

func semLinkNodeWord(count int) string {
	if count == 1 {
		return "node"
	}
	return "nodes"
}

func semLinkFleetLabel(report semlinkdemo.Report) string {
	if report.NodeCount() == 1 {
		return "SemLink companion node"
	}
	return "SemLink companion fleet"
}

func semLinkFleetStatus(report semlinkdemo.Report) string {
	if !report.AssertionsPassed() {
		return "demo-attention"
	}
	if !report.RawMAVLinkExcluded() {
		return "demo-attention"
	}
	if report.SimulatorCommand.HardwareTransmitAuthorized || report.SemOpsReadback.Response.CompanionTransmitAllowed {
		return "demo-attention"
	}
	return "demo-ready"
}

func semLinkAssertionState(report semlinkdemo.Report) string {
	if len(report.Assertions) == 0 {
		return "unavailable"
	}
	if report.AssertionsPassed() {
		return "passed"
	}
	return "failed"
}

func semLinkNoTransmitPosture(report semlinkdemo.Report) string {
	parts := []string{"demo evidence only", "no native transmit authority", "no companion hardware transmit authority"}
	if report.RawMAVLinkExcluded() {
		parts = append(parts, "raw MAVLink excluded from mesh summaries")
	} else {
		parts = append(parts, "raw MAVLink exclusion not proven")
	}
	if report.CommandPosture.HardwareBlocked {
		parts = append(parts, "hardware transmit blocked by SemLink")
	}
	if report.Kind == semlinkdemo.SimpleMeshReportKind {
		parts = append(parts, "SemOps exposes no mesh topology controls")
	}
	return strings.Join(parts, "; ")
}

func semLinkReadback(report semlinkdemo.Report) CompanionReadback {
	readback := CompanionReadback{
		AdapterStatus:            "unavailable",
		CommandACKStatus:         "unavailable",
		ResultStatus:             "unavailable",
		NativeExecutionAllowed:   false,
		CompanionTransmitAllowed: false,
	}
	if report.SemOpsReadback.Response.Status != "" {
		response := report.SemOpsReadback.Response
		readback.AdapterStatus = response.Status
		readback.Accepted = response.Accepted
		readback.ReceivedRequests = report.SemOpsReadback.ReceivedRequests
		readback.CorrelationID = semLinkFirstNonEmpty(response.CorrelationID, report.SemOpsReadback.Request.CorrelationID)
		readback.NativeExecutionAllowed = response.NativeExecutionAllowed
		readback.CompanionTransmitAllowed = response.CompanionTransmitAllowed
	}
	if report.SemOpsReadback.CommandACK.Status != "" {
		ack := report.SemOpsReadback.CommandACK
		readback.CommandACKStatus = ack.Status
		readback.CommandACKResult = ack.Result
		readback.CommandACKObservedAt = ack.ObservedAt
	}
	if report.SemOpsReadback.Result.Status != "" || report.SemOpsReadback.Result.ObservedProperty != "" {
		result := report.SemOpsReadback.Result
		readback.ResultStatus = result.Status
		readback.ResultObservedProperty = result.ObservedProperty
		readback.ResultObservedAt = result.ObservedAt
	}
	return readback
}

func semLinkCommandPosture(report semlinkdemo.Report) CompanionCommandPosture {
	status := semLinkFirstNonEmpty(report.SimulatorCommand.Status, report.CommandPosture.Status, "unavailable")
	return CompanionCommandPosture{
		Status:                     status,
		HardwareBlocked:            report.CommandPosture.HardwareBlocked,
		SimulatorOnly:              report.SimulatorCommand.SimulatorOnly,
		PreflightAccepted:          report.SimulatorCommand.PreflightAccepted,
		ACKAccepted:                report.SimulatorCommand.ACKAccepted,
		PostStateObserved:          report.SimulatorCommand.PostStateObserved,
		HardwareTransmitAuthorized: report.SimulatorCommand.HardwareTransmitAuthorized,
	}
}

func semLinkAssertions(report semlinkdemo.Report) []CompanionAssertion {
	assertions := make([]CompanionAssertion, 0, len(report.Assertions))
	for _, assertion := range report.Assertions {
		assertions = append(assertions, CompanionAssertion{
			Name:   assertion.Name,
			Passed: assertion.Passed,
			Detail: assertion.Detail,
		})
	}
	return assertions
}

func semLinkFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
