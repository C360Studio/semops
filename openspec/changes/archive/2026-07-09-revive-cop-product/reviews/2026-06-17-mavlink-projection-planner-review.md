# MAVLink Projection Planner Adversarial Review

Date: 2026-06-17
Scope: `COP-003` current-state projection planner in `internal/projectors/mavlink`

## Finding Summary

- Severity: Medium. The planner emits SemStreams graph mutation request shapes, but it does not write to the live graph
  yet. Phase 1 must still prove NATS request/reply wiring, graph health, and scenario state polling.
- Severity: Medium. The planner keeps track birth state in memory. That is acceptable for a library-level planner, but
  the live adapter must reconcile with graph state or replay checkpoints before restart/replay claims.
- Severity: Medium. Raw MAVLink frames are still not on a bounded lane. The planner must not become the ingest boundary
  or create one graph entity per native packet.
- Severity: Low. Battery remaining, roll, pitch, and yaw are now MAVLink-owned track signal predicates. Low-battery
  alert state still belongs to a derived rule or fusion owner, not to the feed owner.

## Role Review

Architect:

- Accepts the born-first ordering: source asset create, then track create with `cop.track.source`, then updates.
- Requires live graph wiring to keep ADR-055/056 behavior explicit and not fall back to `triple.add` auto-vivify.

go-reviewer:

- Accepts real-frame projector tests for heartbeat, global position, attitude, and battery packets.
- Requires restart/replay behavior before treating the in-memory birth cache as production adapter state.

technical-writer:

- Requires docs to call this a projection planner, not complete feed integration.
- Requires remaining evidence gates to keep raw-lane, SITL, PX4, replay, and live graph writes open.

## Decision

Accept the planner as the next `COP-003` slice. Do not call MAVLink Phase 1 complete until live graph writes, raw-lane
behavior, replay evidence, SITL/PX4 gates, and stack health checks are proven.
