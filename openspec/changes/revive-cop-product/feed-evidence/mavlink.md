# MAVLink Feed Evidence

Status: candidate Phase 1 feed with codec, bounded raw lane, projection planner, SemStreams graph writer boundary,
structural wiring, typed owner-token wiring, restart create-conflict reconciliation, opt-in UDP transport hosting, COP
owner-registration smoke evidence, generated-frame live graph smoke evidence, and a skipped-by-default external
PX4/MAVSDK/SITL telemetry smoke harness. Live feed integration remains blocked by running that harness against an
actual simulator, durable replay playback, TCP/serial transport work, and command/control fidelity work in `COP-004`.

## Decision

MAVLink should be the first feed because SemOps already contained parser, generator, payload, rule, and SITL material.
The active path now has a modern parser/generator package, bounded in-memory raw lane, COMMAND_LONG/COMMAND_ACK
coverage, current-state projection planner, tested graph request/reply writer boundary, retry-aware SemStreams NATS
requester boundary, in-process adapter harness, hosted runtime wiring, opt-in UDP datagram ingestion, and a one-command
graph scaffold. Live feed work still needs scenario-runner replay wiring, SITL/PX4 evidence, TCP/serial transport, and
full product-stack expansion.

SemOps GitHub issue #1 added a near-term breaking-tag gate: generated or replay MAVLink must prove the born-first
graph path against live SemStreams before PX4/SITL becomes the blocking milestone. The generated-frame smoke passed
locally on 2026-06-17. Clean-stack owner-registry smokes also passed on 2026-06-17 with typed SemStreams
`OwnerToken` values minted by the registry/bind path and serialized only at graph mutation requests.

## Local Evidence

- `pkg/adapters/mavlink/parser.go` parses MAVLink v1/v2 frames, validates checksums, handles stream buffering and
  resync, and registers the first COP message specs.
- `pkg/adapters/mavlink/generator.go` generates MAVLink v2 heartbeat, battery status, global position, attitude, and
  deterministic quadcopter scenario frames with CRC.
- `pkg/adapters/mavlink/parser_test.go` validates generator/parser compatibility, split buffers, noisy resync,
  checksum rejection, concurrent sequence generation, scenario frame generation, and canonical battery wire order.
- `pkg/adapters/mavlink/commands.go` provides COMMAND_LONG/COMMAND_ACK support, MAV_RESULT naming, and ArduCopter
  mode mapping.
- `pkg/adapters/mavlink/commands_test.go` proves COMMAND_LONG and COMMAND_ACK frame generation/parsing from real
  MAVLink bytes.
- `pkg/adapters/mavlink/raw_lane.go` keeps copied native frames in a bounded lane and annotates decoded packets with
  replay-addressable source references.
- `pkg/adapters/mavlink/raw_lane_test.go` proves metadata capture, record and byte eviction, oversize rejection,
  replay lookup, and defensive copies.
- `pkg/adapters/mavlink/replay.go` appends durable raw-lane records as JSON Lines fixtures and loads them back.
- `pkg/adapters/mavlink/replay_test.go` proves appended fixtures load back into parseable MAVLink frame bytes.
- `internal/projectors/mavlink` maps decoded heartbeat, global position, attitude, and battery packets into ordered
  SemStreams graph mutation requests.
- `internal/projectors/mavlink/projector_test.go` proves source asset birth before strict `cop.track.source` edges,
  signal-profiled track current state, source-reference projection, pure projection before committed birth marking, and
  update-only behavior after first birth.
- `internal/projectors/mavlink/writer.go` sends plans to SemStreams `graph.mutation.entity.create_with_triples` and
  `graph.mutation.entity.update_with_triples` request/reply subjects.
- `internal/projectors/mavlink/writer_test.go` proves write ordering, owner-token transit,
  committed-but-degraded response handling, cancellation, failure stops, and unsupported mutation rejection.
- `internal/graphrequest` adapts SemStreams `natsclient.Client.RequestWithRetry` into the graph writer requester
  interface so mutation writers do not use bare query-style request calls.
- `internal/adapters/mavlink` composes parser, raw lane, projector, graph plan writer, and pollable health counters
  for the future adapter service boundary.
- `internal/adapters/mavlink/adapter_test.go` proves valid telemetry writes graph plans, command ACK frames are
  captured without graph writes, corrupt frames stop before graph writes, writer failures are reflected in health, and
  strict `entity_already_exists` birth conflicts after restart are reconciled into update-only writes.
- `internal/adapters/mavlink/udp_listener.go` hosts an opt-in UDP datagram loop that feeds real datagrams into the
  adapter without letting corrupt frames terminate the listener.
- `internal/adapters/mavlink/udp_listener_test.go` sends generated MAVLink frames over localhost UDP and proves invalid
  datagrams are recorded in adapter health before later valid datagrams still write graph plans.
- `internal/stack` wires the MAVLink parser, bounded raw lane, projector, retry-aware NATS requester, graph writer,
  adapter harness, and health state from config.
- `internal/stack/mavlink_test.go` proves custom and default retry config propagation, write timeout propagation,
  create/update graph subjects, born-first source edge behavior, raw-lane capture, and writer injection for tests.
- `internal/copownership` registers first-phase SemOps COP contracts through SemStreams `projection.BindAndHeartbeat`
  and returns typed `ownership.OwnerToken` values minted by the registry/bind path.
- `internal/app` and `cmd/semops` connect to SemStreams, register first-phase COP ownership, enroll heartbeat, and
  compose the hosted MAVLink adapter with registry-derived owner tokens.
- `internal/smoke/mavlink/live_graph_test.go` drives generated heartbeat and position frames through the configured
  stack, registers COP ownership, polls SemStreams graph state, and asserts source asset, track, `cop.track.source`,
  `cop.track.position`, owner lookup, and foreign-edge claim readback.
- `internal/smoke/mavlink/external_sitl_test.go` is a skipped-by-default external simulator smoke. Given a hosted COP
  snapshot URL and a simulator emitting MAVLink telemetry into the SemOps UDP input, it observes a real MAVLink track
  without injecting generated frames itself, requires live feed health, provenance, source refs, velocity evidence, and
  repeated simulator updates, and can optionally require position motion.
- `scripts/cop-stack-smoke.sh` can opt into that external simulator smoke with
  `SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true`. In that mode it allows COP discovery for systems `1,42` by default
  and expects the external simulator track `c360.edge-compose.cop.mavlink.track.system-1`.
- `compose.cop.yml` now respects `SEMOPS_COP_MAVLINK_SYSTEM_IDS`, so PX4-style system `1` can be included without
  editing the Compose file.
- See `openspec/changes/revive-cop-product/reviews/2026-06-23-mavlink-external-sitl-smoke-review.md` for the
  simulator-fidelity harness review.
- 2026-06-23 local readiness preflight found no `px4`, `mavsdk_server`, or `sim_vehicle.py` on PATH and no local
  PX4/MAVSDK/ArduPilot simulator Docker image. `go test ./internal/smoke/mavlink -run
  TestExternalSITLTelemetryCOPSnapshot -count=1 -v` skipped as designed without a snapshot URL, and focused MAVLink
  parser/projector/component tests passed. This is readiness-gap evidence only; it does not close the simulator
  fidelity gate.
- `scripts/mavlink-sitl-gate.sh` now wraps the external SITL gate in `preflight`, `focused`, and `stack` modes. The
  focused and stack modes require a named simulator source and refuse to run without local simulator tooling or an
  explicit remote-source override.
- 2026-06-23 PX4 Docker review selected `jonasvautherin/px4-gazebo-headless:1.17.0` as the preferred dev/demo SITL
  image because it packages a runnable headless PX4/Gazebo simulator, defaults to `gz_x500`, targets host UDP `14550`
  and `14540`, carries Apache-2.0 repo licensing, and has active Docker Hub `1.17.0`/`latest` tags. The image is
  unofficial and large, so the helper will not pull it unless `SEMOPS_MAVLINK_SITL_DOCKER_PULL=true`.
- `scripts/mavlink-sitl-gate.sh` now has `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack`, which starts the preferred
  PX4/Gazebo headless container, waits for boot, runs the full hosted COP stack smoke with the external SITL telemetry
  assertion enabled, and stops only the simulator container it started unless `SEMOPS_MAVLINK_SITL_KEEP_SIMULATOR=true`.
- Ignored ArduPilot SITL controller/scenario reference files were deleted after command encoding and ACK parsing moved
  into the active adapter and the live controller was rejected as legacy scaffolding.

## External Evidence

- MAVLink developer guide documents MAVLink as a lightweight messaging protocol for drones and onboard components.
- PX4 documents a simulator path and a Simulator MAVLink API for exchanging simulated sensor and actuator data.
- PX4's current Docker docs recommend `px4io/px4-dev:<version>` for builds, note that a dedicated `px4-sim` container is
  planned, and mark older per-distro simulation containers as legacy.
- `JonasVautherin/px4-gazebo-headless` is an Apache-2.0 unofficial PX4/Gazebo headless runtime image. Its README
  identifies PX4 `v1.17.0` as the latest supported release, shows `docker run --rm -it
  jonasvautherin/px4-gazebo-headless:1.17.0`, and documents host UDP `14550`/`14540` MAVLink routing.
- ArduPilot documents SITL as a way to run Plane, Copter, or Rover without hardware.

## Gates

### Parser Gate

Current command:

```bash
go test ./pkg/adapters/mavlink
```

Acceptance:

- Heartbeat, battery, global position, and attitude frames parse with expected system/component IDs and fields.
- Corrupted frames do not panic and do not publish governed graph state.
- Multiple messages in one buffer produce stable ordered packets.
- Battery status uses canonical MAVLink wire order, not the older self-consistent SemOps reference layout.
- COMMAND_LONG and COMMAND_ACK use canonical payload order and parse expected command/result fields.

### Projection Gate

Current command:

```bash
go test ./internal/projectors/mavlink
```

Acceptance:

- Heartbeat, global position, attitude, and battery packets create or update one current vehicle entity per MAVLink
  system.
- The source asset is born before the track writes a strict `cop.track.source` foreign edge.
- Raw frames remain in the bounded lane and current-state graph writes carry only `cop.provenance.source_ref`.
- Vehicle current state uses `indexing_profile=signal`.
- The graph writer targets the current SemStreams create/update-with-triples request subjects.
- A committed-but-degraded mutation response is treated as committed and not retried.
- NATS-backed graph mutation requests use SemStreams retry-aware mutation request handling.
- The in-process adapter harness exposes pollable health counters for raw ingest, projection, graph writes, and errors.
- Structural wiring can compose the NATS-backed writer path without launching the full stack.
- Commands, mission state, and battery alerts use `indexing_profile=control`.
- Replay/decode records use `indexing_profile=trace`.
- No graph entity is created per raw packet.

### Breaking-Tag Graph Gate

Current command:

```bash
SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL=<nats-url> go test ./internal/smoke/mavlink -v
```

This test skips unless `SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL` points at a live SemStreams graph stack.
When `SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL` is set, it also checks SemOps-specific graph-ingest counter deltas.

Latest evidence:

- 2026-06-17: passed against SemStreams `configs/graph-backend.json` and a JetStream NATS broker at
  `nats://127.0.0.1:55438`.
- 2026-06-17: passed again against a clean temporary NATS/SemStreams stack at `nats://127.0.0.1:4222` after
  registering SemOps COP ownership contracts and using registry/bind-result `OwnerToken` values.
- 2026-06-17: passed with `SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL=http://localhost:9090/metrics`, asserting zero
  SemOps-specific deltas for owner-token mismatch, foreign-edge, and indexing-profile-default counters.
- 2026-06-17: `cmd/semops` gained hosted composition-root wiring for COP ownership registration and MAVLink adapter
  construction; covered by `go test ./internal/app` and `go build ./cmd/semops`.
- 2026-06-17: `bash scripts/cop-stack-smoke.sh` built and launched the Docker Compose graph scaffold, polled
  SemStreams health and metrics, ran the MAVLink live graph smoke with `SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL`, and
  tore the stack down cleanly.
- 2026-06-17: after SemStreams exposed typed `ownership.OwnerToken`, SemOps migrated runtime/projector wiring away
  from local token suffix composition and reran `go test ./...`, `go build ./cmd/semops`, and
  `bash scripts/cop-stack-smoke.sh`.
- 2026-06-19: added opt-in hosted UDP datagram ingestion through `SEMOPS_MAVLINK_UDP_LISTEN_ADDR`, covered by
  `go test ./internal/adapters/mavlink ./internal/app`.
- SemStreams health remained green after the run via `/health` and the dedicated `/healthz` endpoint.
- SemStreams logged that `semops.feed.cap` has no enforceable owning or foreign-edge claim because CAP is currently
  append-evidence only; this is governance evidence, not write-fence protection.
- SemStreams accepted SemOps feedback to add typed, opaque owner-token minting on the registry/bind-result path and
  to split append-evidence declarations from enforceable ownership/write-fence claims.

Acceptance:

- Generated or replay heartbeat and position frames write through a live SemStreams graph path.
- The source asset is born before the track writes the strict `cop.track.source` edge.
- Known-track position updates do not rebirth the source asset and do not repeat the strict foreign edge.
- The run reports no `entity_not_found` mutation failures.
- The clean-stack run registers SemOps COP owners, enrolls them for heartbeat, and uses typed owner tokens minted by
  SemStreams registry/bind results.
- When metrics are enabled, the run asserts dropped foreign-edge, owner-token mismatch, and indexing-profile default
  counter deltas for SemOps message types.
- This gate is complete before PX4/SITL is treated as the next blocking MAVLink milestone.

### SITL Gate

Current skipped-by-default telemetry target:

```bash
SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL=http://127.0.0.1:8080/api/cop/snapshot \
SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID=c360.edge-compose.cop.mavlink.track.system-1 \
go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v
```

Stack target when an external simulator is already emitting MAVLink UDP to the SemOps host port:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=stack \
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL <version>" \
SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND="<simulator launch command>" \
bash scripts/mavlink-sitl-gate.sh
```

Preferred PX4/Gazebo headless Docker target:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
bash scripts/mavlink-sitl-gate.sh
```

If the image is not local and the operator deliberately accepts the large pull:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
SEMOPS_MAVLINK_SITL_DOCKER_PULL=true \
bash scripts/mavlink-sitl-gate.sh
```

Acceptance:

- The smoke skips cleanly unless `SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL` is set. [done]
- The smoke observes a simulator-owned MAVLink track through the hosted COP snapshot rather than sending generated
  frames from the test. [done in harness; not yet run against PX4/MAVSDK]
- The track uses `semops.feed.mavlink` provenance, carries a non-empty source reference, has non-zero position and
  velocity evidence, and appears while `feed.mavlink` is live. [done in harness; not yet run against PX4/MAVSDK]
- The smoke observes repeated simulator updates and can require actual position motion with
  `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true`. [done in harness; not yet run against PX4/MAVSDK]
- Local readiness preflight records whether PX4, MAVSDK, ArduPilot, or equivalent simulator tooling is actually
  available before attempting the stack gate. [done: 2026-06-23 no simulator runtime found]
- Focused and stack helpers require a named simulator source before running the evidence gate. [done]
- Preferred PX4/Gazebo headless Docker helper is wired, fail-closed on missing local image unless pull is explicitly
  enabled, and records simulator image/vehicle/world evidence. [done in helper; not yet run against pulled image]
- Against explicit ArduPilot SITL, the controller connects, reads status, and performs safe command smoke tests.
  [open]
- PX4 SITL or MAVSDK smoke evidence is recorded before calling MAVLink Phase 1 complete. [open]

### Replay Gate

Target artifact:

- A small captured MAVLink fixture with heartbeat, position, attitude, battery, low-battery, and stale-track cases.
- Current fixture format: JSON Lines `pkg/adapters/mavlink.RawFrameRecord` values, with frame bytes JSON-encoded as
  base64 by Go's standard encoder.

Acceptance:

- Replaying the fixture yields deterministic current-state graph output and feed-health/staleness transitions.

## Known Gaps

- The active module path, Go toolchain, and MAVLink parser/generator are modernized.
- The current-state projection planner, graph writer, COP ownership binding, and structural wiring now pass a
  generated-frame live graph smoke against SemStreams.
- The hosted runtime can opt into UDP datagram ingestion with `SEMOPS_MAVLINK_UDP_LISTEN_ADDR`. TCP, serial, and
  dedicated `semops-adapter-mavlink` process packaging remain open.
- Raw-lane capture and replay fixture storage are library boundaries; scenario-runner playback and stack retention
  policy are not implemented yet.
- Explicit SemOps COP owner registration and heartbeat coverage are wired into `cmd/semops`, and the graph scaffold can
  launch it with SemStreams. Scenario playback is not wired into the stack yet.
- The optional metrics smoke performs before/after counter deltas for SemOps message types; the hosted stack still
  needs to expand beyond the graph scaffold.
- Restart reconciliation now handles strict `entity_already_exists` create conflicts for known MAVLink asset/track
  births by marking the conflicted entity born and reprojecting the current packet. Durable checkpoint/read-back
  recovery and scenario-runner replay integration remain open.
- No live SITL controller remains; a modern harness must be rebuilt with explicit readiness and state polling before
  command/control demo claims.
- The external SITL telemetry smoke harness exists, but no local PX4, MAVSDK, or ArduPilot simulator run has been
  recorded yet. The 2026-06-23 laptop preflight found no local simulator runtime or image. The preferred PX4/Gazebo
  headless Docker path is now wired, but the image has not been pulled or run in SemOps evidence yet, so tasks `4.7`
  and `5.4` remain open.
- Old `RoboticsProcessor`, BaseMessage payload graphing, StreamKit, and ObjectStore paths have been removed from the
  active product path rather than preserved as migration targets.
- Command codec coverage is active for COMMAND_LONG and COMMAND_ACK, but live command/control is not.
- PX4 SITL/MAVSDK evidence is not yet in SemOps.

## Adversarial Feed-Entry Questions

- Are parser tests using real MAVLink bytes, or did the adapter start trusting structs?
- Is there exactly one current vehicle graph entity per vehicle, not one entity per packet?
- Does low-battery alert state belong to the feed owner or a derived rule owner?
- Does raw/replay detail stay out of semantic indexing?
- Does the demo claim PX4 coverage before a PX4 or MAVSDK smoke gate exists?
