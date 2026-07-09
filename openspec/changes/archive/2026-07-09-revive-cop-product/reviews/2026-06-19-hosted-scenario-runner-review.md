# Hosted Scenario Runner Review

Date: 2026-06-19

## Decision

Accept `cmd/semops-scenario-runner` and the `semops-scenario-runner` Compose service as the first hosted scenario
playback boundary.

This is a one-shot demo service with health/status polling. It is not an operator orchestration surface, timeline UI,
or full scenario-control plane.

## Evidence Checked

- `cmd/semops-scenario-runner/main.go`
- `internal/scenario/http.go`
- `compose.cop.yml`
- `Dockerfile`
- `scripts/cop-stack-smoke.sh`
- `go test ./internal/scenario`
- `go test ./...`
- `go build ./cmd/semops`
- `go build ./cmd/semops-scenario-runner`
- `bash scripts/cop-stack-smoke.sh`

## Adversarial Findings

- Product value: accepted. A hosted runner gives the demo a repeatable HADR feed mix and lets the smoke poll concrete
  playback state instead of treating logs as proof.
- Orchestration risk: contained. The service runs one fixture on startup and exposes status; it does not add topology,
  tier controls, manual flow wiring, or a new operator shell.
- Framework compliance: accepted. The service registers the first-phase COP owners, uses registry-derived typed owner
  tokens, and writes through the same MAVLink, TAK/CoT, and CAP graph mutation seams as the tested adapters.
- Health semantics: accepted for this stage. `/healthz` stays unavailable until the replay succeeds, while
  `/scenario/status` exposes completed steps, failed steps, mutation count, and last error for active polling.
- Service boundary: accepted. Unlike deterministic mappers, the runner is a deployment artifact because it owns demo
  playback timing and cross-feed startup behavior in the Compose stack.
- Scenario fidelity: partial. The flood/evacuation fixture is hosted; shared-airspace playback, pause/resume/reset,
  expected-state checkpoints, and a durable replay policy remain open.

## Accepted Risks

- CAP is still scenario replay, not a hosted CAP/NWS poller or webhook service.
- The runner reuses first-phase ownership registration; append-evidence-only CAP still emits the known SemStreams
  warning until the evidence-contribution declaration split matures.
- The smoke asserts scenario success, but does not yet assert graph index/cardinality deltas for the mixed-feed
  scenario.

## Follow-Ups

- Add expected graph-state checkpoints after scenario replay.
- Add operator/API controls only after the workflow proves value.
- Add shared-airspace playback once ADS-B/SAPIENT boundaries exist.
- Add mixed-feed cardinality/index-profile assertions before Phase 1 signoff.
