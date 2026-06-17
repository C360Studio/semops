# MAVLink Adapter Harness Adversarial Review

Date: 2026-06-17
Scope: `COP-004` in-process MAVLink adapter harness in `internal/adapters/mavlink`

## Finding Summary

- Severity: Medium. The harness proves parse, raw capture, projection, graph-plan write ordering, and health counters,
  but it is not a UDP/TCP listener and is not hosted in Compose.
- Severity: Medium. The graph writer is still backed by a test double in harness tests. Live NATS/SemStreams
  request/reply wiring remains open.
- Severity: Medium. Invalid frames are captured and stopped before graph writes, but durable replay storage and
  restart reconciliation are still missing.
- Severity: Low. Command ACK packets are captured without current-state graph writes. Command lifecycle entities
  remain a later control-profile projection, not part of this harness slice.

## Role Review

architect:

- Accepts `internal/adapters/mavlink` as the adapter service core because it composes existing library boundaries
  without making transport, Compose, or orchestration assumptions.
- Requires the future service wrapper to preserve the same ordering: parse, bounded raw capture, projection, graph
  writer, and pollable health state.

go-reviewer:

- Accepts tests for telemetry success, unsupported command ACK capture, corrupt-frame stop-before-write behavior, and
  writer failure health.
- Requires the live stack smoke to poll graph state and health endpoints instead of relying on logs or sleeps.

technical-writer:

- Requires docs to call this an in-process harness, not the containerized structural stack.
- Requires remaining COP-004 work to keep NATS/SemStreams wiring, durable replay, and two-feed smoke open.

## Decision

Accept the MAVLink adapter harness as the first COP-004 implementation increment. Do not claim `semops-adapter-mavlink`
or Phase 1 stack readiness until transport hosting, live graph writes, durable replay, and active health polling pass.
