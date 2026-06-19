# Hosted Runtime Ownership Review

Date: 2026-06-17

Scope:

- `internal/app`
- `cmd/semops`
- `internal/stack.NewMAVLinkAdapter`

Evidence:

- Added a testable hosted runtime boundary in `internal/app`.
- `cmd/semops` now loads env config, connects to SemStreams NATS, registers first-phase COP ownership, starts owner
  heartbeat, and composes the MAVLink adapter with registry-derived typed owner tokens.
- Unit tests prove startup order, cleanup on adapter composition failure, MAVLink-disable behavior, and env parsing.
- Review caught and fixed heartbeat lifetime: owner heartbeat is stopped by `App.Close`, not by the startup timeout
  context used for NATS connect and initial ownership registration.
- Verification: `go test ./cmd/semops ./internal/app` and `go build ./cmd/semops`.

## Adversarial Findings

- Architect: This is the right composition root boundary. Ownership registration is no longer a smoke-test-only
  behavior, and low-level projectors now consume typed owner tokens instead of suffix config.
- Go reviewer: Startup cleanup is covered for adapter composition failure. Later restart work added narrow
  `entity_already_exists` create-conflict reconciliation for existing MAVLink asset/track births, but durable
  checkpoint/read-back seeding remains absent.
- Go reviewer: The hosted MAVLink adapter is in-process only. Later work added an opt-in UDP datagram listener, but
  TCP/serial readers and dedicated adapter-process packaging remain open.
- Technical writer: The env surface is intentionally small. Do not document this as a one-command demo stack until
  Compose exists and active health/metrics polling wraps the process.

## Decision

Accept the hosted composition-root ownership wiring as the close for task `6.15`.

Follow-up: task `6.16` now passes through `scripts/cop-stack-smoke.sh`. Keep full-stack Compose expansion open for
API, UI, scenario runner, feed transports, and durable checkpoint/read-back reconciliation.
