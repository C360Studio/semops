package scenario

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
)

const CheckpointManifestVersion = "semops.scenario-checkpoints.v1"

type ClaimScope string

const (
	ClaimProductE2E           ClaimScope = "product_e2e"
	ClaimComponentPreflight   ClaimScope = "component_preflight"
	ClaimContractReplay       ClaimScope = "contract_replay"
	ClaimCommandControl       ClaimScope = "command_control"
	ClaimCSAPI                ClaimScope = "cs_api"
	ClaimProviderIntegration  ClaimScope = "provider_integration"
	ClaimSimulatorFidelity    ClaimScope = "simulator_fidelity"
	ClaimStandardsConformance ClaimScope = "standards_conformance"
	ClaimOperatorControl      ClaimScope = "operator_scenario_control"
)

type BoundaryKind string

const (
	BoundaryExternalFeed       BoundaryKind = "external_feed_boundary"
	BoundarySemStreamsInput    BoundaryKind = "semstreams_input_component"
	BoundaryDirectGraph        BoundaryKind = "direct_graph_contract"
	BoundaryNativeTransmitter  BoundaryKind = "native_transmitter"
	BoundaryCSAPI              BoundaryKind = "cs_api_boundary"
	BoundaryProviderEndpoint   BoundaryKind = "provider_endpoint"
	BoundarySimulatorFamily    BoundaryKind = "simulator_family"
	BoundaryConformanceHarness BoundaryKind = "conformance_harness"
	BoundaryReviewedControl    BoundaryKind = "reviewed_operator_control"
)

const (
	EvidenceDirectGraphMutation       = "direct_graph_mutation"
	EvidenceDecodedNATSPayload        = "decoded_nats_payload_injection"
	EvidenceProjectedPayloadInjection = "projected_payload_injection"
)

type CheckpointManifest struct {
	Version     string               `json:"version"`
	ScenarioID  string               `json:"scenario_id"`
	Checkpoints []ScenarioCheckpoint `json:"checkpoints"`
}

type ScenarioCheckpoint struct {
	ID                 string                  `json:"id"`
	Label              string                  `json:"label,omitempty"`
	ClaimScope         ClaimScope              `json:"claim_scope"`
	IngressMode        IngressMode             `json:"ingress_mode"`
	Boundary           CheckpointBoundary      `json:"boundary"`
	ExpectedCOP        ExpectedCOPState        `json:"expected_cop"`
	RuntimeEvidence    RuntimeEvidence         `json:"runtime_evidence"`
	AllowedOwners      []string                `json:"allowed_owners"`
	DisallowedEvidence []string                `json:"disallowed_evidence"`
	CommandEvidence    CommandEvidence         `json:"command_evidence,omitempty"`
	CSAPIEvidence      CSAPIEvidence           `json:"cs_api_evidence,omitempty"`
	ProviderEvidence   ProviderEvidence        `json:"provider_evidence,omitempty"`
	SimulatorEvidence  SimulatorEvidence       `json:"simulator_evidence,omitempty"`
	StandardsEvidence  StandardsEvidence       `json:"standards_evidence,omitempty"`
	OperatorEvidence   OperatorControlEvidence `json:"operator_control_evidence,omitempty"`
}

type CheckpointBoundary struct {
	Kind       BoundaryKind `json:"kind"`
	Feeds      []string     `json:"feeds,omitempty"`
	Components []string     `json:"components,omitempty"`
	Endpoints  []string     `json:"endpoints,omitempty"`
}

type ExpectedCOPState struct {
	SnapshotURL       string   `json:"snapshot_url,omitempty"`
	ScenarioStatusURL string   `json:"scenario_status_url,omitempty"`
	ScenarioState     State    `json:"scenario_state,omitempty"`
	Entities          []string `json:"entities,omitempty"`
	Feeds             []string `json:"feeds,omitempty"`
}

type RuntimeEvidence struct {
	RuntimeURL        string   `json:"runtime_url,omitempty"`
	MetricsURL        string   `json:"metrics_url,omitempty"`
	ComponentFeeds    []string `json:"component_feeds,omitempty"`
	PrometheusMetrics []string `json:"prometheus_metrics,omitempty"`
}

type CommandEvidence struct {
	NativeTransmitter      string `json:"native_transmitter,omitempty"`
	SafetyReview           string `json:"safety_review,omitempty"`
	AckStatusCorrelation   bool   `json:"ack_status_correlation,omitempty"`
	PostCommandCOPReadback bool   `json:"post_command_cop_readback,omitempty"`
}

type CSAPIEvidence struct {
	BoundaryReview              string `json:"boundary_review,omitempty"`
	DesiredActualReconciliation bool   `json:"desired_actual_reconciliation,omitempty"`
}

type ProviderEvidence struct {
	Endpoint string `json:"endpoint,omitempty"`
	Review   string `json:"review,omitempty"`
}

type SimulatorEvidence struct {
	Family string `json:"family,omitempty"`
	Review string `json:"review,omitempty"`
}

type StandardsEvidence struct {
	Harness string `json:"harness,omitempty"`
	Review  string `json:"review,omitempty"`
}

type OperatorControlEvidence struct {
	Review string `json:"review,omitempty"`
}

func LoadCheckpointManifest(path string) (CheckpointManifest, error) {
	if path == "" {
		return CheckpointManifest{}, fmt.Errorf("scenario checkpoint manifest path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return CheckpointManifest{}, fmt.Errorf("open scenario checkpoint manifest %q: %w", path, err)
	}
	defer file.Close()
	manifest, err := DecodeCheckpointManifest(file)
	if err != nil {
		return CheckpointManifest{}, fmt.Errorf("load scenario checkpoint manifest %q: %w", path, err)
	}
	return manifest, nil
}

func DecodeCheckpointManifest(reader io.Reader) (CheckpointManifest, error) {
	if reader == nil {
		return CheckpointManifest{}, fmt.Errorf("scenario checkpoint manifest reader is nil")
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var manifest CheckpointManifest
	if err := decoder.Decode(&manifest); err != nil {
		return CheckpointManifest{}, fmt.Errorf("decode scenario checkpoint manifest: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("extra JSON value after manifest")
		}
		return CheckpointManifest{}, fmt.Errorf("decode scenario checkpoint manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return CheckpointManifest{}, err
	}
	return manifest, nil
}

func (m CheckpointManifest) Validate() error {
	if m.Version != CheckpointManifestVersion {
		return fmt.Errorf("scenario checkpoint manifest version = %q, want %q", m.Version, CheckpointManifestVersion)
	}
	if m.ScenarioID == "" {
		return fmt.Errorf("scenario checkpoint manifest requires scenario_id")
	}
	if len(m.Checkpoints) == 0 {
		return fmt.Errorf("scenario checkpoint manifest %q requires at least one checkpoint", m.ScenarioID)
	}
	seen := map[string]struct{}{}
	for i, checkpoint := range m.Checkpoints {
		if err := checkpoint.Validate(); err != nil {
			return fmt.Errorf("checkpoint %d: %w", i+1, err)
		}
		if _, ok := seen[checkpoint.ID]; ok {
			return fmt.Errorf("checkpoint %q is duplicated", checkpoint.ID)
		}
		seen[checkpoint.ID] = struct{}{}
	}
	return nil
}

func (c ScenarioCheckpoint) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("requires id")
	}
	if !c.ClaimScope.valid() {
		return fmt.Errorf("%q claim_scope is unsupported", c.ClaimScope)
	}
	if !c.IngressMode.valid() {
		return fmt.Errorf("%q ingress_mode is unsupported", c.IngressMode)
	}
	if !c.Boundary.Kind.valid() {
		return fmt.Errorf("%q boundary kind is unsupported", c.Boundary.Kind)
	}
	if c.Boundary.Kind == BoundaryDirectGraph && c.ClaimScope != ClaimContractReplay {
		return fmt.Errorf("direct graph boundary is contract/replay evidence only")
	}
	if c.IngressMode == IngressModeDirectGraphContract && c.ClaimScope != ClaimContractReplay {
		return fmt.Errorf("direct graph ingress is contract/replay evidence only")
	}
	if requiresOwners(c.ClaimScope) && len(c.AllowedOwners) == 0 {
		return fmt.Errorf("%s checkpoint requires allowed_owners", c.ClaimScope)
	}
	switch c.ClaimScope {
	case ClaimProductE2E:
		return c.validateProductE2E()
	case ClaimComponentPreflight:
		return c.validateComponentPreflight()
	case ClaimContractReplay:
		return nil
	case ClaimCommandControl:
		return c.validateCommandControl()
	case ClaimCSAPI:
		return c.validateCSAPI()
	case ClaimProviderIntegration:
		return c.validateProviderIntegration()
	case ClaimSimulatorFidelity:
		return c.validateSimulatorFidelity()
	case ClaimStandardsConformance:
		return c.validateStandardsConformance()
	case ClaimOperatorControl:
		return c.validateOperatorControl()
	default:
		return fmt.Errorf("%q claim_scope is unsupported", c.ClaimScope)
	}
}

func (c ScenarioCheckpoint) validateComponentPreflight() error {
	if c.IngressMode != IngressModeFeedBoundary {
		return fmt.Errorf("component preflight checkpoint must use feed-boundary ingress")
	}
	if c.Boundary.Kind != BoundaryExternalFeed && c.Boundary.Kind != BoundarySemStreamsInput {
		return fmt.Errorf("component preflight checkpoint must enter through an external feed boundary or SemStreams input component")
	}
	if len(c.Boundary.Feeds) == 0 && len(c.Boundary.Components) == 0 {
		return fmt.Errorf("component preflight checkpoint requires boundary feeds or components")
	}
	if c.RuntimeEvidence.RuntimeURL == "" && c.RuntimeEvidence.MetricsURL == "" {
		return fmt.Errorf("component preflight checkpoint requires runtime or metrics evidence")
	}
	return nil
}

func (c ScenarioCheckpoint) validateProductE2E() error {
	if c.IngressMode != IngressModeFeedBoundary {
		return fmt.Errorf("product e2e checkpoint must use feed-boundary ingress")
	}
	if c.Boundary.Kind != BoundaryExternalFeed && c.Boundary.Kind != BoundarySemStreamsInput {
		return fmt.Errorf("product e2e checkpoint must enter through an external feed boundary or SemStreams input component")
	}
	if len(c.Boundary.Feeds) == 0 && len(c.Boundary.Components) == 0 {
		return fmt.Errorf("product e2e checkpoint requires boundary feeds or components")
	}
	if c.ExpectedCOP.SnapshotURL == "" && c.ExpectedCOP.ScenarioStatusURL == "" {
		return fmt.Errorf("product e2e checkpoint requires Caddy-routed snapshot or scenario status readback")
	}
	if len(c.ExpectedCOP.Entities) == 0 && len(c.ExpectedCOP.Feeds) == 0 {
		return fmt.Errorf("product e2e checkpoint requires expected COP entities or feeds")
	}
	if c.RuntimeEvidence.RuntimeURL == "" && c.RuntimeEvidence.MetricsURL == "" {
		return fmt.Errorf("product e2e checkpoint requires runtime or metrics evidence")
	}
	if len(c.RuntimeEvidence.ComponentFeeds) == 0 && len(c.RuntimeEvidence.PrometheusMetrics) == 0 {
		return fmt.Errorf("product e2e checkpoint requires component feeds or Prometheus metrics")
	}
	for _, evidence := range []string{
		EvidenceDirectGraphMutation,
		EvidenceDecodedNATSPayload,
		EvidenceProjectedPayloadInjection,
	} {
		if !slices.Contains(c.DisallowedEvidence, evidence) {
			return fmt.Errorf("product e2e checkpoint must disallow %s", evidence)
		}
	}
	return nil
}

func (c ScenarioCheckpoint) validateCommandControl() error {
	if c.Boundary.Kind != BoundaryNativeTransmitter {
		return fmt.Errorf("command-control checkpoint requires a native transmitter or driver boundary")
	}
	if c.CommandEvidence.NativeTransmitter == "" || c.CommandEvidence.SafetyReview == "" {
		return fmt.Errorf("command-control checkpoint requires native transmitter and safety review evidence")
	}
	if !c.CommandEvidence.AckStatusCorrelation || !c.CommandEvidence.PostCommandCOPReadback {
		return fmt.Errorf("command-control checkpoint requires ACK/status correlation and post-command COP readback")
	}
	return nil
}

func (c ScenarioCheckpoint) validateCSAPI() error {
	if c.Boundary.Kind != BoundaryCSAPI {
		return fmt.Errorf("CS API checkpoint requires a CS API boundary")
	}
	if c.CSAPIEvidence.BoundaryReview == "" || !c.CSAPIEvidence.DesiredActualReconciliation {
		return fmt.Errorf("CS API checkpoint requires boundary review and desired/actual-state reconciliation")
	}
	return nil
}

func (c ScenarioCheckpoint) validateProviderIntegration() error {
	if c.Boundary.Kind != BoundaryProviderEndpoint {
		return fmt.Errorf("provider checkpoint requires a provider endpoint boundary")
	}
	if c.ProviderEvidence.Endpoint == "" || c.ProviderEvidence.Review == "" {
		return fmt.Errorf("provider checkpoint requires endpoint and review evidence")
	}
	return nil
}

func (c ScenarioCheckpoint) validateSimulatorFidelity() error {
	if c.Boundary.Kind != BoundarySimulatorFamily {
		return fmt.Errorf("simulator checkpoint requires a simulator family boundary")
	}
	if c.SimulatorEvidence.Family == "" || c.SimulatorEvidence.Review == "" {
		return fmt.Errorf("simulator checkpoint requires family and review evidence")
	}
	return nil
}

func (c ScenarioCheckpoint) validateStandardsConformance() error {
	if c.Boundary.Kind != BoundaryConformanceHarness {
		return fmt.Errorf("standards checkpoint requires a conformance harness boundary")
	}
	if c.StandardsEvidence.Harness == "" || c.StandardsEvidence.Review == "" {
		return fmt.Errorf("standards checkpoint requires harness and review evidence")
	}
	return nil
}

func (c ScenarioCheckpoint) validateOperatorControl() error {
	if c.Boundary.Kind != BoundaryReviewedControl {
		return fmt.Errorf("operator scenario-control checkpoint requires a reviewed control boundary")
	}
	if c.OperatorEvidence.Review == "" {
		return fmt.Errorf("operator scenario-control checkpoint requires review evidence")
	}
	return nil
}

func (s ClaimScope) valid() bool {
	switch s {
	case ClaimProductE2E,
		ClaimComponentPreflight,
		ClaimContractReplay,
		ClaimCommandControl,
		ClaimCSAPI,
		ClaimProviderIntegration,
		ClaimSimulatorFidelity,
		ClaimStandardsConformance,
		ClaimOperatorControl:
		return true
	default:
		return false
	}
}

func (k BoundaryKind) valid() bool {
	switch k {
	case BoundaryExternalFeed,
		BoundarySemStreamsInput,
		BoundaryDirectGraph,
		BoundaryNativeTransmitter,
		BoundaryCSAPI,
		BoundaryProviderEndpoint,
		BoundarySimulatorFamily,
		BoundaryConformanceHarness,
		BoundaryReviewedControl:
		return true
	default:
		return false
	}
}

func (m IngressMode) valid() bool {
	switch m {
	case IngressModeFeedBoundary, IngressModeDirectGraphContract:
		return true
	default:
		return false
	}
}

func requiresOwners(scope ClaimScope) bool {
	switch scope {
	case ClaimProductE2E, ClaimContractReplay, ClaimCommandControl, ClaimCSAPI:
		return true
	default:
		return false
	}
}
