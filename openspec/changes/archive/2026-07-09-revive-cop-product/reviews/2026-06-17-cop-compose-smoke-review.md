# COP Compose Smoke Review

Date: 2026-06-17

Scope:

- `Dockerfile`
- `compose.cop.yml`
- `scripts/cop-stack-smoke.sh`
- `internal/smoke/mavlink/live_graph_test.go`

Evidence:

- Added a SemOps Dockerfile that builds `cmd/semops` with the sibling SemStreams checkout as a named BuildKit
  context.
- Added `compose.cop.yml` with NATS, SemStreams graph backend, and SemOps runtime.
- Added `scripts/cop-stack-smoke.sh` to start the stack, actively poll SemStreams health and metrics, run the MAVLink
  live graph smoke with both NATS and metrics URLs, and tear the stack down unless `SEMOPS_COP_KEEP_STACK=true`.
- Ran `bash scripts/cop-stack-smoke.sh` on 2026-06-17. Result: pass. The script built both images, started all three
  services, ran `TestLiveGraphMAVLinkBornFirstSmoke`, asserted zero SemOps-specific metrics deltas, and removed the
  stack.

## Adversarial Findings

- Architect: This is a graph scaffold, not the full COP stack. It intentionally lacks SemOps API, UI, scenario
  runner, TAK/CAP adapters, and MAVLink transport listeners.
- Go reviewer: The smoke still injects generated MAVLink frames from the test process. That is the right
  born-first/metrics gate, but feed-fidelity and transport readiness remain open.
- Ops reviewer: The SemStreams sibling build context is large. This is acceptable for local revival proof, but a
  pinned image or slimmer context should replace it before repeated developer/demo runs.
- Technical writer: The script actively polls `/healthz` and `/metrics`, which is the correct monitoring shape. Do
  not replace this with log-grep-only readiness.

## Decision

Accept the one-command graph smoke as the close for task `6.16`.

Keep `6.1` open until the Compose stack includes SemOps API, UI, scenario runner, and at least Phase 1 feed transport
services.
