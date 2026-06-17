# MAVLink Structural Wiring Review

Date: 2026-06-17

Scope:

- `internal/stack.NewMAVLinkAdapter`
- `internal/stack/mavlink_test.go`
- COP-004 structural-stack status update

Decision:

- Accept the structural wiring factory as Phase 1 composition evidence.
- Do not treat it as a live adapter service or Compose milestone.

Evidence Checked:

- The factory composes parser, bounded raw lane, projector, graph writer, retry-aware NATS requester, adapter harness,
  clock, write timeout, and retry config.
- Tests drive real MAVLink heartbeat and position frames through the composed path.
- Tests assert create/create/update subjects, default/custom retry propagation, write timeout propagation, raw-lane
  capture, source-edge birth behavior, and writer injection.

Adversarial Findings:

- Architect: The package proves composition but not process ownership. A future `semops-adapter-mavlink` entrypoint
  still needs config loading, NATS connection lifecycle, signal handling, health endpoints, and active polling.
- Go reviewer: The adapter's in-memory projector birth cache is not restart-safe. Live hosting must add read-back,
  checkpointing, or idempotent create behavior before restart/replay claims.
- Go reviewer: SemStreams `RequestWithRetry` documents a classified-error gap for legacy handler failures. The current
  graph writer will surface bad JSON bodies, but a future SemStreams classified retry API may be safer for production.
- Technical writer: The docs must keep "structural wiring" separate from "container stack" so the demo plan does not
  overclaim before one-command Compose and graph-state smoke tests exist.

Accepted Risks:

- The wiring package intentionally has no UDP/TCP listener. Protocol binding remains a service-hosting task.
- The tests fake SemStreams graph responses and do not prove a live graph responder or ownership lease behavior.
- Raw-lane replay records are still not connected to scenario-runner playback or retention policy.
- Later update: MAVLink now has narrow restart create-conflict reconciliation for existing asset/track births, but no
  durable checkpoint/read-back seeding.

Follow-Up:

- Host the wiring in `cmd/semops` or a dedicated adapter service entrypoint.
- Add durable checkpoint/read-back reconciliation before calling MAVLink graph writes production-grade.
- Revisit classified retry handling when the live SemStreams graph responder is wired.
- Add graph-state polling smoke tests after the first two-feed stack exists.
