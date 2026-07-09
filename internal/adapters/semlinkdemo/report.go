package semlinkdemo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	SingleNodeReportKind = "single-node-companion-demo"
	SimpleMeshReportKind = "simple-mesh-companion-demo"
)

const maxReportBytes = 1 << 20

type Report struct {
	Kind                string
	GeneratedAt         time.Time
	VehicleProfile      string
	ExpectedSummaries   int
	Nodes               []Node
	Probes              []EndpointProbe
	Evidence            EvidenceSummary
	RawMAVLinkExclusion RawMAVLinkExclusion
	CommandPosture      CommandPosture
	SimulatorCommand    SimulatorCommand
	SemOpsReadback      Readback
	Assertions          []Assertion
}

type Node struct {
	NodeID              string   `json:"node_id"`
	VehicleCount        int      `json:"vehicle_count"`
	AlertCount          int      `json:"alert_count,omitempty"`
	FrameCount          int      `json:"frame_count,omitempty"`
	DecodedCount        int      `json:"decoded_count,omitempty"`
	GraphEntities       int      `json:"graph_entities,omitempty"`
	PeerCount           int      `json:"peer_count,omitempty"`
	PeerURLs            []string `json:"peer_urls,omitempty"`
	InitialSummaryCount int      `json:"initial_summary_count,omitempty"`
	FinalSummaryCount   int      `json:"final_summary_count,omitempty"`
	WatermarkCount      int      `json:"watermark_count,omitempty"`
	AppliedDiffCount    int      `json:"applied_diff_count,omitempty"`
	DiffItemCount       int      `json:"diff_item_count,omitempty"`
	TTLMergePosture     string   `json:"ttl_merge_posture,omitempty"`
}

type EndpointProbe struct {
	Path       string `json:"path"`
	HTTPStatus int    `json:"http_status"`
	Status     string `json:"status"`
}

type EvidenceSummary struct {
	ContractName                  string
	ContractVersion               string
	VehicleCount                  int
	MeshStatus                    string
	MeshPosture                   string
	SummaryCount                  int
	WatermarkCount                int
	RawMAVLinkReplicatesByDefault bool
	RawMAVLinkReplicationPolicy   string
}

type RawMAVLinkExclusion struct {
	RejectedBySummaryIndex bool   `json:"rejected_by_summary_index"`
	Policy                 string `json:"policy,omitempty"`
	Error                  string `json:"error,omitempty"`
}

type CommandPosture struct {
	HTTPStatus      int    `json:"http_status,omitempty"`
	Status          string `json:"status,omitempty"`
	HardwareBlocked bool   `json:"hardware_blocked,omitempty"`
}

type SimulatorCommand struct {
	Accepted                   bool   `json:"accepted,omitempty"`
	PreflightAccepted          bool   `json:"preflight_accepted,omitempty"`
	SimulatorOnly              bool   `json:"simulator_only,omitempty"`
	ACKAccepted                bool   `json:"ack_accepted,omitempty"`
	PostStateObserved          bool   `json:"post_state_observed,omitempty"`
	HardwareTransmitAuthorized bool   `json:"hardware_transmit_authorized,omitempty"`
	Status                     string `json:"status,omitempty"`
	FrameCount                 int    `json:"frame_count,omitempty"`
	ACKCount                   int    `json:"ack_count,omitempty"`
}

type Readback struct {
	Request          ReadbackRequest  `json:"request"`
	Response         ReadbackResponse `json:"response"`
	ReceivedRequests int              `json:"received_requests"`
	CommandACK       CommandACKStatus `json:"command_ack,omitempty"`
	Result           ReadbackResult   `json:"result,omitempty"`
}

type ReadbackRequest struct {
	Contract           string    `json:"contract"`
	CompanionNodeID    string    `json:"companion_node_id"`
	TargetSystemID     uint8     `json:"target_system_id"`
	TargetComponentID  uint8     `json:"target_component_id"`
	CommandID          uint16    `json:"command_id"`
	RequestedMessageID uint32    `json:"requested_message_id"`
	CorrelationID      string    `json:"correlation_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	RequestedAt        time.Time `json:"requested_at"`
	TTLSeconds         int       `json:"ttl_seconds"`
	SourceRef          string    `json:"source_ref,omitempty"`
}

type ReadbackResponse struct {
	Contract                 string    `json:"contract"`
	Accepted                 bool      `json:"accepted"`
	Status                   string    `json:"status"`
	Duplicate                bool      `json:"duplicate"`
	CorrelationID            string    `json:"correlation_id,omitempty"`
	IdempotencyKey           string    `json:"idempotency_key,omitempty"`
	CompanionNodeID          string    `json:"companion_node_id,omitempty"`
	AuthorityScope           string    `json:"authority_scope,omitempty"`
	TargetAssetID            string    `json:"target_asset_id,omitempty"`
	EntityID                 string    `json:"entity_id,omitempty"`
	NativeID                 string    `json:"native_id,omitempty"`
	ClaimScope               string    `json:"claim_scope,omitempty"`
	SourceRef                string    `json:"source_ref,omitempty"`
	RequestedAt              time.Time `json:"requested_at,omitempty"`
	NativeExecutionAllowed   bool      `json:"native_execution_allowed"`
	CompanionTransmitAllowed bool      `json:"companion_transmit_allowed"`
	Mutations                int       `json:"mutations"`
	Error                    string    `json:"error,omitempty"`
}

type CommandACKStatus struct {
	Status     string    `json:"status,omitempty"`
	CommandID  uint16    `json:"command_id,omitempty"`
	Result     string    `json:"result,omitempty"`
	Accepted   bool      `json:"accepted,omitempty"`
	ObservedAt time.Time `json:"observed_at,omitempty"`
}

type ReadbackResult struct {
	Status           string         `json:"status,omitempty"`
	ObservedProperty string         `json:"observed_property,omitempty"`
	MessageID        uint32         `json:"message_id,omitempty"`
	ObservedAt       time.Time      `json:"observed_at,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
}

type Assertion struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type singleNodeReport struct {
	Kind             string           `json:"kind"`
	GeneratedAt      time.Time        `json:"generated_at"`
	VehicleProfile   string           `json:"vehicle_profile"`
	Node             Node             `json:"node"`
	Probes           []EndpointProbe  `json:"probes"`
	Evidence         evidenceBundle   `json:"evidence"`
	CommandPosture   CommandPosture   `json:"command_posture"`
	SimulatorCommand SimulatorCommand `json:"simulator_command"`
	SemOpsReadback   Readback         `json:"semops_readback"`
	Assertions       []Assertion      `json:"assertions"`
}

type simpleMeshReport struct {
	Kind                string              `json:"kind"`
	GeneratedAt         time.Time           `json:"generated_at"`
	VehicleProfile      string              `json:"vehicle_profile"`
	ExpectedSummaries   int                 `json:"expected_summaries"`
	Nodes               []Node              `json:"nodes"`
	RawMAVLinkExclusion RawMAVLinkExclusion `json:"raw_mavlink_exclusion"`
	Assertions          []Assertion         `json:"assertions"`
}

type evidenceBundle struct {
	Contract evidenceContract  `json:"contract"`
	Vehicles []vehicleEvidence `json:"vehicles"`
	Mesh     meshEvidence      `json:"mesh"`
}

type evidenceContract struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type vehicleEvidence struct {
	EntityID    string `json:"entity_id"`
	Callsign    string `json:"callsign"`
	SystemID    uint8  `json:"system_id"`
	VehicleType string `json:"vehicle_type,omitempty"`
}

type meshEvidence struct {
	Status                        string `json:"status"`
	Posture                       string `json:"posture"`
	SummaryCount                  int    `json:"summary_count"`
	WatermarkCount                int    `json:"watermark_count"`
	RawMAVLinkReplicatesByDefault bool   `json:"raw_mavlink_replicates_by_default"`
	RawMAVLinkReplicationPolicy   string `json:"raw_mavlink_replication_policy"`
}

func DecodeReport(r io.Reader) (Report, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxReportBytes+1))
	if err != nil {
		return Report{}, fmt.Errorf("read semlink demo report: %w", err)
	}
	if len(data) > maxReportBytes {
		return Report{}, fmt.Errorf("semlink demo report exceeds %d bytes", maxReportBytes)
	}
	return ParseReport(data)
}

func ParseReport(data []byte) (Report, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Report{}, fmt.Errorf("semlink demo report is empty")
	}
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return Report{}, fmt.Errorf("decode semlink demo report kind: %w", err)
	}
	switch strings.TrimSpace(header.Kind) {
	case SingleNodeReportKind:
		var report singleNodeReport
		if err := json.Unmarshal(data, &report); err != nil {
			return Report{}, fmt.Errorf("decode single-node semlink demo report: %w", err)
		}
		out := Report{
			Kind:                report.Kind,
			GeneratedAt:         report.GeneratedAt,
			VehicleProfile:      report.VehicleProfile,
			Nodes:               []Node{report.Node},
			Probes:              report.Probes,
			Evidence:            summarizeEvidence(report.Evidence),
			RawMAVLinkExclusion: singleNodeRawMAVLinkExclusion(report.Evidence.Mesh),
			CommandPosture:      report.CommandPosture,
			SimulatorCommand:    report.SimulatorCommand,
			SemOpsReadback:      report.SemOpsReadback,
			Assertions:          report.Assertions,
		}
		if out.Nodes[0].VehicleCount == 0 && len(report.Evidence.Vehicles) > 0 {
			out.Nodes[0].VehicleCount = len(report.Evidence.Vehicles)
		}
		if err := validateReport(out); err != nil {
			return Report{}, err
		}
		return out, nil
	case SimpleMeshReportKind:
		var report simpleMeshReport
		if err := json.Unmarshal(data, &report); err != nil {
			return Report{}, fmt.Errorf("decode simple-mesh semlink demo report: %w", err)
		}
		out := Report{
			Kind:                report.Kind,
			GeneratedAt:         report.GeneratedAt,
			VehicleProfile:      report.VehicleProfile,
			ExpectedSummaries:   report.ExpectedSummaries,
			Nodes:               report.Nodes,
			RawMAVLinkExclusion: report.RawMAVLinkExclusion,
			Assertions:          report.Assertions,
		}
		if err := validateReport(out); err != nil {
			return Report{}, err
		}
		return out, nil
	default:
		if strings.TrimSpace(header.Kind) == "" {
			return Report{}, fmt.Errorf("semlink demo report kind is required")
		}
		return Report{}, fmt.Errorf("unsupported semlink demo report kind %q", header.Kind)
	}
}

func (r Report) NodeCount() int {
	return len(r.Nodes)
}

func (r Report) VehicleCount() int {
	var total int
	for _, node := range r.Nodes {
		total += node.VehicleCount
	}
	return total
}

func (r Report) AssertionsPassed() bool {
	if len(r.Assertions) == 0 {
		return false
	}
	for _, assertion := range r.Assertions {
		if !assertion.Passed {
			return false
		}
	}
	return true
}

func (r Report) RawMAVLinkExcluded() bool {
	if r.RawMAVLinkExclusion.RejectedBySummaryIndex {
		return true
	}
	if r.Kind == SingleNodeReportKind && r.Evidence.RawMAVLinkReplicationPolicy != "" {
		return !r.Evidence.RawMAVLinkReplicatesByDefault
	}
	return false
}

func validateReport(report Report) error {
	if strings.TrimSpace(report.Kind) == "" {
		return fmt.Errorf("semlink demo report kind is required")
	}
	if report.GeneratedAt.IsZero() {
		return fmt.Errorf("semlink demo report generated_at is required")
	}
	if strings.TrimSpace(report.VehicleProfile) == "" {
		return fmt.Errorf("semlink demo report vehicle_profile is required")
	}
	if len(report.Nodes) == 0 {
		return fmt.Errorf("semlink demo report must include at least one companion node")
	}
	seen := make(map[string]struct{}, len(report.Nodes))
	for index, node := range report.Nodes {
		nodeID := strings.TrimSpace(node.NodeID)
		if nodeID == "" {
			return fmt.Errorf("semlink demo report node %d is missing node_id", index)
		}
		if _, ok := seen[nodeID]; ok {
			return fmt.Errorf("semlink demo report has duplicate node_id %q", nodeID)
		}
		seen[nodeID] = struct{}{}
	}
	return nil
}

func summarizeEvidence(evidence evidenceBundle) EvidenceSummary {
	return EvidenceSummary{
		ContractName:                  evidence.Contract.Name,
		ContractVersion:               evidence.Contract.Version,
		VehicleCount:                  len(evidence.Vehicles),
		MeshStatus:                    evidence.Mesh.Status,
		MeshPosture:                   evidence.Mesh.Posture,
		SummaryCount:                  evidence.Mesh.SummaryCount,
		WatermarkCount:                evidence.Mesh.WatermarkCount,
		RawMAVLinkReplicatesByDefault: evidence.Mesh.RawMAVLinkReplicatesByDefault,
		RawMAVLinkReplicationPolicy:   evidence.Mesh.RawMAVLinkReplicationPolicy,
	}
}

func singleNodeRawMAVLinkExclusion(mesh meshEvidence) RawMAVLinkExclusion {
	if mesh.RawMAVLinkReplicationPolicy == "" {
		return RawMAVLinkExclusion{}
	}
	return RawMAVLinkExclusion{
		RejectedBySummaryIndex: !mesh.RawMAVLinkReplicatesByDefault,
		Policy:                 mesh.RawMAVLinkReplicationPolicy,
	}
}
