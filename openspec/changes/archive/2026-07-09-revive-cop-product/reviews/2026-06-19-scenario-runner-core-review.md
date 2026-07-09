# Scenario Runner Core Review

Date: 2026-06-19

## Decision

Accept the first `internal/scenario` runner core as structural replay infrastructure.

This is not accepted as the final scenario-runner service, orchestration UI, or shared-airspace vignette. It is a
testable in-process runner that replays the Phase 1 HADR flood/evacuation feed mix through the same seams used by
hosted adapters and graph writers.

## Evidence Checked

- `internal/scenario/runner.go`
- `internal/scenario/runner_test.go`
- `go test ./internal/scenario`
- `go test ./...`

## Adversarial Findings

- Product value: accepted. A deterministic runner is useful because it lets MAVLink, TAK/CoT, and CAP replay together
  without inventing a COP shell or topology/orchestration feature.
- Service boundary: not accepted yet. The runner is currently a library because container placement, API control, and
  operator start/stop semantics are not proven.
- False-green risk: mitigated. The runner fails preflight when a fixture includes a feed but the matching sink is
  missing.
- Framework compliance: accepted for this stage. The runner uses real MAVLink and CoT adapter seams and the CAP
  projector/writer seam, so it preserves born-first and owner-token behavior instead of creating a parallel projection
  path.
- Scenario fidelity: partial. The HADR flood/evacuation seed exists, but the shared-airspace vignette, pause/resume,
  acceleration, reset, and expected-state checkpointing remain open.
- Indexing pressure: partial. The runner exercises the mixed feed set, but it does not yet assert index cardinality or
  graph query cost.

## Accepted Risks

- The runner does not currently wait on offsets or expose a controllable demo clock beyond deterministic fixture
  timestamps.
- CAP runs through direct projector/writer seams; hosted CAP polling/webhook behavior remains unproven.
- No container or API endpoint starts the scenario yet.

## Follow-Ups

- Add a hosted `semops-scenario-runner` service or CLI only after the API/control semantics are clear.
- Add shared-airspace playback with MAVLink plus ADS-B/SAPIENT once those feed boundaries exist.
- Add expected-state checkpoints and graph readback assertions so scenario playback can become a full-stack gate.
- Add cardinality/index-profile assertions once the mixed-feed scenario runs against live SemStreams.
