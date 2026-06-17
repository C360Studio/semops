# Hosted Runtime Ownership Review

Date: 2026-06-17

Scope:

- `internal/app`
- `cmd/semops`
- `internal/stack.NewMAVLinkAdapter`

Evidence:

- Added a testable hosted runtime boundary in `internal/app`.
- `cmd/semops` now loads env config, connects to SemStreams NATS, registers first-phase COP ownership, starts owner
  heartbeat, and composes the MAVLink adapter with the registry-derived owner-token incarnation.
- Unit tests prove startup order, cleanup on adapter composition failure, MAVLink-disable behavior, and env parsing.
- Review caught and fixed heartbeat lifetime: owner heartbeat is stopped by `App.Close`, not by the startup timeout
  context used for NATS connect and initial ownership registration.
- Verification: `go test ./cmd/semops ./internal/app` and `go build ./cmd/semops`.

## Adversarial Findings

- Architect: This is the right composition root boundary. Ownership registration is no longer a smoke-test-only
  behavior, and low-level projectors still do not invent owner-token suffixes.
- Go reviewer: Startup cleanup is covered for adapter composition failure, but live process restart/replay
  reconciliation is still absent. A restarted process can register a new incarnation but cannot yet inspect graph
  state before deciding whether to rebirth already-known MAVLink entities.
- Go reviewer: The hosted MAVLink adapter is in-process only. It has no UDP/TCP/serial reader, so this is hosted
  graph-writer composition, not a complete feed service.
- Technical writer: The env surface is intentionally small. Do not document this as a one-command demo stack until
  Compose exists and active health/metrics polling wraps the process.

## Decision

Accept the hosted composition-root ownership wiring as the close for task `6.15`.

Follow-up: task `6.16` now passes through `scripts/cop-stack-smoke.sh`. Keep full-stack Compose expansion open for
API, UI, scenario runner, feed transports, and restart/replay reconciliation.
