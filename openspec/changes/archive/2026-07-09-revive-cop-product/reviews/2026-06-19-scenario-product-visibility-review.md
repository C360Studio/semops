# Scenario Product Visibility Review

Date: 2026-06-19

## Decision

Accept the hosted scenario-runner product-visibility smoke as a Phase 1 stack gate.

This proves the runner is not merely healthy internally: its MAVLink, TAK/CoT,
and CAP writes are visible through the Caddy-routed COP snapshot. It is still
not full scenario control, shared-airspace playback, or index/cardinality
signoff.

## Evidence Checked

- `internal/smoke/cop/live_snapshot_test.go`
- `scripts/cop-stack-smoke.sh`
- `go test ./internal/smoke/cop`
- `go test ./...`
- `bash scripts/cop-stack-smoke.sh`

## Adversarial Findings

- Product visibility: accepted. The smoke waits for runner success and then
  verifies the same operator API surface includes MAVLink track, TAK/CoT
  task/advisory, and CAP hazard geometry/provenance.
- False-green risk: reduced. `/scenario/status` alone is insufficient; the
  snapshot assertion now catches graph-write or readback drift.
- Orchestration risk: contained. This remains a smoke gate, not scenario
  controls or topology UI.
- Coverage gap: partial. It checks representative entities but not expected
  graph revisions, exact event order, or cardinality/index deltas.
- CAP claim language: safe. The test verifies append-evidence hazard
  visibility, not authoritative CAP service behavior.

## Accepted Risks

- The smoke still uses the first deterministic HADR fixture only.
- No shared-airspace, ADS-B, or SAPIENT path exists yet.
- No expected-state checkpoint file or timeline replay clock is enforced yet.

## Follow-Ups

- Add expected-state checkpoints for scenario replay.
- Add mixed-feed graph cardinality/index-profile assertions before Phase 1
  signoff.
- Add shared-airspace playback after ADS-B/SAPIENT boundaries exist.
