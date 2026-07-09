# MAVLink Graph Writer Boundary Adversarial Review

Date: 2026-06-17
Scope: `COP-003` graph request/reply writer in `internal/projectors/mavlink`

## Finding Summary

- Severity: Medium. The writer sends current SemStreams graph mutation request shapes, but it is not yet wired into
  the containerized SemOps stack or exercised against a live graph-ingest service.
- Severity: Medium. Restart/replay reconciliation is still missing. A restarted adapter may attempt create births for
  already-born entities and should fail closed until a read-back or checkpoint path is designed.
- Severity: Medium. Raw MAVLink frames still need a bounded lane. The writer must remain downstream of decode and
  projection, not become a raw ingest surface.
- Severity: Low. The writer correctly treats committed-but-degraded mutation responses as committed writes. Demo
  health checks still need a separate read path to recover post-write state when SemStreams returns degraded success.

## Role Review

architect:

- Accepts the writer as the minimum SemStreams graph boundary: typed create-with-triples and update-with-triples only.
- Requires the live adapter to preserve born-first ordering when transport retries, restarts, or replay are added.

go-reviewer:

- Accepts unit coverage for subject selection, request ordering, owner-token transit, context cancellation, mutation
  response failures, and committed-but-degraded responses.
- Requires live graph-ingest integration tests before calling projection writes production-ready.

technical-writer:

- Requires docs to call this a graph writer boundary, not a full feed service.
- Requires remaining gates to keep raw-lane, container health, restart/replay, SITL/PX4, and scenario evidence open.

## Decision

Accept the writer boundary as the next `COP-003` increment. Do not claim MAVLink live-feed completion until it runs in
the structural stack with raw-lane capture, graph health polling, replay reconciliation, and SITL/PX4 evidence.
