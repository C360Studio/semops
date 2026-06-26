package scenario

import (
	"strings"
	"testing"
)

func TestCheckpointManifestLoadsPhase1HADRClaims(t *testing.T) {
	manifest, err := LoadCheckpointManifest("../../scenarios/phase1-hadr.checkpoints.json")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if manifest.Version != CheckpointManifestVersion {
		t.Fatalf("version = %q", manifest.Version)
	}
	if manifest.ScenarioID != Phase1HADRScenarioID {
		t.Fatalf("scenario id = %q, want %q", manifest.ScenarioID, Phase1HADRScenarioID)
	}
	if len(manifest.Checkpoints) != 3 {
		t.Fatalf("checkpoints = %d, want 3", len(manifest.Checkpoints))
	}

	product := manifest.Checkpoints[0]
	if product.ClaimScope != ClaimProductE2E ||
		product.IngressMode != IngressModeFeedBoundary ||
		product.Boundary.Kind != BoundaryExternalFeed {
		t.Fatalf("product checkpoint = %+v", product)
	}
	if len(product.ExpectedCOP.Entities) == 0 || len(product.RuntimeEvidence.ComponentFeeds) == 0 {
		t.Fatalf("product checkpoint missing expected COP/runtime evidence: %+v", product)
	}
	for _, evidence := range []string{
		EvidenceDirectGraphMutation,
		EvidenceDecodedNATSPayload,
		EvidenceProjectedPayloadInjection,
	} {
		if !contains(product.DisallowedEvidence, evidence) {
			t.Fatalf("product checkpoint should disallow %s: %+v", evidence, product.DisallowedEvidence)
		}
	}

	preflight := manifest.Checkpoints[1]
	if preflight.ClaimScope != ClaimComponentPreflight ||
		preflight.Boundary.Kind != BoundarySemStreamsInput ||
		len(preflight.AllowedOwners) != 0 {
		t.Fatalf("SAPIENT preflight checkpoint = %+v", preflight)
	}

	contract := manifest.Checkpoints[2]
	if contract.ClaimScope != ClaimContractReplay ||
		contract.IngressMode != IngressModeDirectGraphContract ||
		contract.Boundary.Kind != BoundaryDirectGraph {
		t.Fatalf("contract checkpoint = %+v", contract)
	}
}

func TestCheckpointManifestRejectsProductDirectGraphEvidence(t *testing.T) {
	_, err := DecodeCheckpointManifest(strings.NewReader(`{
		"version": "semops.scenario-checkpoints.v1",
		"scenario_id": "bad-product",
		"checkpoints": [{
			"id": "bad",
			"claim_scope": "product_e2e",
			"ingress_mode": "direct-graph-contract",
			"boundary": {"kind": "direct_graph_contract"},
			"expected_cop": {"snapshot_url": "/api/cop/snapshot", "entities": ["track-1"]},
			"runtime_evidence": {"runtime_url": "/api/cop/runtime", "component_feeds": ["mavlink"]},
			"allowed_owners": ["semops.feed.mavlink"],
			"disallowed_evidence": ["direct_graph_mutation", "decoded_nats_payload_injection", "projected_payload_injection"]
		}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "direct graph boundary is contract/replay evidence only") {
		t.Fatalf("error = %v, want direct graph rejection", err)
	}
}

func TestCheckpointManifestRejectsProductMissingAntiCheatEvidence(t *testing.T) {
	_, err := DecodeCheckpointManifest(strings.NewReader(`{
		"version": "semops.scenario-checkpoints.v1",
		"scenario_id": "bad-product",
		"checkpoints": [{
			"id": "bad",
			"claim_scope": "product_e2e",
			"ingress_mode": "feed-boundary",
			"boundary": {"kind": "semstreams_input_component", "feeds": ["adsb"]},
			"expected_cop": {"snapshot_url": "/api/cop/snapshot", "entities": ["track-1"]},
			"runtime_evidence": {"runtime_url": "/api/cop/runtime", "component_feeds": ["adsb"]},
			"allowed_owners": ["semops.feed.adsb"],
			"disallowed_evidence": ["direct_graph_mutation"]
		}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "decoded_nats_payload_injection") {
		t.Fatalf("error = %v, want missing disallowed evidence rejection", err)
	}
}

func TestCheckpointManifestRejectsProductMissingAllowedOwners(t *testing.T) {
	_, err := DecodeCheckpointManifest(strings.NewReader(`{
		"version": "semops.scenario-checkpoints.v1",
		"scenario_id": "bad-product",
		"checkpoints": [{
			"id": "bad",
			"claim_scope": "product_e2e",
			"ingress_mode": "feed-boundary",
			"boundary": {"kind": "semstreams_input_component", "feeds": ["adsb"]},
			"expected_cop": {"snapshot_url": "/api/cop/snapshot", "entities": ["track-1"]},
			"runtime_evidence": {"runtime_url": "/api/cop/runtime", "component_feeds": ["adsb"]},
			"disallowed_evidence": ["direct_graph_mutation", "decoded_nats_payload_injection", "projected_payload_injection"]
		}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "product_e2e checkpoint requires allowed_owners") {
		t.Fatalf("error = %v, want missing allowed owners rejection", err)
	}
}

func TestCheckpointManifestRejectsTrailingJSON(t *testing.T) {
	_, err := DecodeCheckpointManifest(strings.NewReader(`{
		"version": "semops.scenario-checkpoints.v1",
		"scenario_id": "bad-json",
		"checkpoints": [{
			"id": "ok",
			"claim_scope": "contract_replay",
			"ingress_mode": "direct-graph-contract",
			"boundary": {"kind": "direct_graph_contract"},
			"allowed_owners": ["semops.feed.mavlink"]
		}]
	} {"version": "extra"}`))
	if err == nil || !strings.Contains(err.Error(), "extra JSON value") {
		t.Fatalf("error = %v, want trailing JSON rejection", err)
	}
}

func TestCheckpointManifestRequiresCommandControlFeedbackLoop(t *testing.T) {
	_, err := DecodeCheckpointManifest(strings.NewReader(`{
		"version": "semops.scenario-checkpoints.v1",
		"scenario_id": "bad-command",
		"checkpoints": [{
			"id": "bad-command-control",
			"claim_scope": "command_control",
			"ingress_mode": "feed-boundary",
			"boundary": {"kind": "native_transmitter", "feeds": ["mavlink"]},
			"expected_cop": {},
			"runtime_evidence": {},
			"allowed_owners": ["semops.command.intent"],
			"disallowed_evidence": [],
			"command_evidence": {
				"native_transmitter": "mavlink-request-message",
				"safety_review": "openspec/reviews/mavlink-command.md",
				"ack_status_correlation": true
			}
		}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "post-command COP readback") {
		t.Fatalf("error = %v, want command feedback rejection", err)
	}
}

func TestCheckpointManifestRequiresCSAPIReconciliation(t *testing.T) {
	_, err := DecodeCheckpointManifest(strings.NewReader(`{
		"version": "semops.scenario-checkpoints.v1",
		"scenario_id": "bad-csapi",
		"checkpoints": [{
			"id": "bad-csapi",
			"claim_scope": "cs_api",
			"ingress_mode": "feed-boundary",
			"boundary": {"kind": "cs_api_boundary", "feeds": ["mavlink"]},
			"expected_cop": {},
			"runtime_evidence": {},
			"allowed_owners": ["semops.command.intent"],
			"disallowed_evidence": [],
			"cs_api_evidence": {
				"boundary_review": "openspec/reviews/csapi.md"
			}
		}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "desired/actual-state reconciliation") {
		t.Fatalf("error = %v, want CS API reconciliation rejection", err)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
