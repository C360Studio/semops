# MAVLink Live Graph Smoke Adversarial Review

Date: 2026-06-17

Scope:

- `internal/smoke/mavlink/live_graph_test.go`
- SemStreams `configs/graph-backend.json`
- Generated MAVLink heartbeat and global-position frames

Evidence:

- Built the sibling SemStreams `cmd/semstreams` binary into `/private/tmp/semstreams-graph-backend`.
- Ran SemStreams graph backend with `SEMSTREAMS_NATS_URLS=nats://127.0.0.1:55438`.
- Verified `/health` and `/healthz` stayed healthy after the smoke.
- Ran
  `SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL=nats://127.0.0.1:55438 go test ./internal/smoke/mavlink -v`.
- Result: pass. The smoke created a source asset, created a MAVLink track, wrote the strict
  `cop.track.source` edge, updated `cop.track.position`, and read both entities back through `graph.query.entity`.
- The broker was an already-running local testcontainer, not a SemOps-owned stack. It became noisy after the smoke,
  with heartbeat and consumer warnings, so counter assertions must be repeated in a clean one-command stack.
- Follow-up result: pass against a clean temporary NATS/SemStreams stack at `nats://127.0.0.1:4222` after registering
  SemOps COP owners and using registry-derived `<owner>#<incarnation>` owner tokens.
- Metrics follow-up: pass with `SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL=http://localhost:9090/metrics`, asserting no
  SemOps-specific owner-token mismatch, foreign-edge, or indexing-profile-default counter deltas.
- Compose follow-up: pass through `bash scripts/cop-stack-smoke.sh`, which starts NATS, SemStreams graph backend,
  and SemOps runtime, polls health and metrics, and runs the same smoke with the metrics URL wired in.

Adversarial Findings:

- Go reviewer: The ADR-055 must-exist graph path is cleared for generated MAVLink frames. This is enough to keep
  PX4/SITL from blocking born-first graph compliance.
- Architect: The smoke proves graph mutation and readback behavior, not transport hosting. UDP/TCP listener work,
  Compose wiring, and scenario-runner replay still belong to `COP-004`.
- Go reviewer: The smoke sends owner tokens but does not prove explicit SemOps COP owner registration or heartbeat
  lifecycle. Add a structural-stack smoke for owner registry presence before claiming production ownership behavior.
- Update: the clean-stack follow-up now proves COP owner registration, heartbeat enrollment, owner lookup, and
  foreign-edge claim lookup for the MAVLink path.
- Go reviewer: Restart/replay remains open. The projector still keeps birth knowledge in memory, so a process restart
  needs read-back, checkpointing, or idempotent-create behavior before operational safety claims.
- Technical writer: The result should be described as generated-frame graph compliance, not MAVLink feed-fidelity or
  command/control evidence.
- Architect: Metrics need before/after delta assertions. The clean stack still exposed a baseline
  `message_type="unknown"` indexing-profile default counter, so a total-zero assertion would be misleading.
- Update: the live smoke now performs SemOps-specific before/after delta checks when a metrics URL is provided.

Decision:

Accept the smoke as the generated/replay ADR-055/056 must-exist gate for MAVLink. Keep COP-009 open for GitHub issue
evidence publication and restart/replay reconciliation.
