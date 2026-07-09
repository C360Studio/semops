# TAK/CoT Graph Wiring Adversarial Review

Date: 2026-06-19

## Decision

Accept the TAK/CoT graph writer, adapter write path, stack constructor, and opt-in hosted runtime wiring as the next
gate after projection. Do not promote TAK/CoT to structural stack status until a live graph smoke and COP API/UI
readback prove the state is visible and operator-safe.

## Evidence Checked

- `internal/projectors/cot/writer.go` applies create/update mutation plans through SemStreams graph mutation subjects
  and surfaces typed mutation failures for restart reconciliation.
- `internal/adapters/cot/adapter.go` now optionally composes a projector and plan writer after raw capture/replay,
  preserving graph-free fixture behavior when no projector is configured.
- The adapter reconciles `entity_already_exists` create conflicts by marking born state and reprojecting the same CoT
  event with the same source reference.
- `internal/stack/cot.go` composes a retry-aware NATS graph writer or injected test writer.
- `internal/app` adds explicit `SEMOPS_COT_*` config and hosted UDP/TCP listener lifecycle only when
  `SEMOPS_COT_ENABLED=true`.
- `go test ./internal/projectors/cot ./internal/adapters/cot ./internal/stack ./internal/app` passes.

## Objections

- This is not yet a TAK live graph smoke. The path is tested with request/plan fakes and must still prove readback
  against a running SemStreams graph.
- The COP snapshot provider still queries configured MAVLink source asset and track entities only. CoT graph writes
  will not appear in the operator UI until the API/UI feed-state gate lands.
- UDP/TCP listener hosting is not TAK Server behavior. It has no auth, certificate, federation, team/user state, or
  mission-package support.
- Stale timestamps are parsed and projected as status inputs, but runtime downgrade/removal and UI freshness behavior
  remain unproven.

## Accepted Risks

- Keep CoT hosted runtime disabled by default and require explicit `SEMOPS_COT_ENABLED=true` plus listener addresses.
- Keep the first graph path in-process so the container/service boundary can follow evidence instead of creating a
  separate adapter process too early.
- Reuse MAVLink's born-first restart reconciliation shape rather than inventing feed-specific recovery semantics.

## Follow-Up Tasks

- Add a skipped-by-default TAK/CoT live graph smoke that verifies source asset, track, task, advisory, and source-ref
  readback.
- Extend the COP API snapshot provider and UI layer to include CoT tracks, tasks, advisories, source health, and
  provenance without treating native XML as prose.
- Add stale-data downgrade behavior before displaying CoT state as current operator truth.
