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
