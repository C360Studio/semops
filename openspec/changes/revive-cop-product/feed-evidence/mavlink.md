# MAVLink Feed Evidence

Status: candidate Phase 1 feed with codec, bounded raw lane, projection planner, SemStreams graph writer boundary,
structural wiring, typed owner-token wiring, restart create-conflict reconciliation, opt-in UDP transport hosting,
multi-listener hosted UDP input, COP owner-registration smoke evidence, generated-frame live graph smoke evidence, a
skipped-by-default external PX4/MAVSDK/SITL telemetry smoke harness, passed PX4/Gazebo headless Docker telemetry smoke
through a managed Compose-network route, COMMAND_ACK control-task readback projection, and PX4 simulator read-side
command readback evidence. SemOps also has a product-owned command-intent graph contract for future desired tasking
state before native execution. Live feed integration remains blocked by durable replay playback, TCP/serial transport
work, ArduPilot parity, MAVSDK/offboard parity, and broader command authority/fidelity work in `COP-004`.

## Decision

MAVLink should be the first feed because SemOps already contained parser, generator, payload, rule, and SITL material.
The active path now has a modern parser/generator package, bounded in-memory raw lane, COMMAND_LONG/COMMAND_ACK
coverage, current-state projection planner, COMMAND_ACK control-task readback projection, tested graph request/reply
writer boundary, retry-aware SemStreams NATS requester boundary, a product-owned command-intent graph contract,
in-process adapter harness, hosted runtime wiring, opt-in UDP datagram ingestion, a one-command graph scaffold, and a
PX4 simulator read-side command readback pass. Live feed work still needs scenario-runner replay wiring, TCP/serial
transport, ArduPilot evidence, MAVSDK/offboard evidence, command authority/fidelity beyond the reviewed read-side
request, and full product-stack expansion.

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
  SemStreams graph mutation requests, and maps COMMAND_ACK packets into control-profiled command readback task state.
- `internal/projectors/mavlink/projector_test.go` proves source asset birth before strict `cop.track.source` edges,
  signal-profiled track current state, source-reference projection, pure projection before committed birth marking, and
  update-only behavior after first birth. It also proves COMMAND_ACK creates a control task with a strict born-first
  `cop.task.target` edge to the source asset and updates known command tasks without repeating that foreign edge.
- `internal/projectors/mavlink/writer.go` sends plans to SemStreams `graph.mutation.entity.create_with_triples` and
  `graph.mutation.entity.update_with_triples` request/reply subjects.
- `internal/projectors/mavlink/writer_test.go` proves write ordering, owner-token transit,
  committed-but-degraded response handling, cancellation, failure stops, and unsupported mutation rejection.
- `internal/graphrequest` adapts SemStreams `natsclient.Client.RequestWithRetryClassified` into the graph writer
  requester interface so mutation writers preserve retry behavior and ADR-060 classified graph mutation errors.
- `internal/adapters/mavlink` composes parser, raw lane, projector, graph plan writer, and pollable health counters
  for the future adapter service boundary.
- `internal/adapters/mavlink/adapter_test.go` proves valid telemetry writes graph plans, command ACK frames are
  captured and written as control-task readback state, corrupt frames stop before graph writes, writer failures are
  reflected in health, and typed `entity_already_exists` birth conflicts after restart are reconciled into update-only
  writes.
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
- `pkg/cop` defines `semops.command.intent.v1` as the product-owned desired command/tasking state contract, separate
  from MAVLink COMMAND_ACK readback evidence.
- `internal/projectors/command` validates desired command intent and produces command-intent create/update mutation
  plans without birthing target assets, bypassing born-first target checks, or transmitting native commands.
- `internal/projectors/command` constrains command-intent statuses to the current lifecycle vocabulary and provides a
  transition guard before CS API, UI, or native-driver status handlers exist.
- `internal/projectors/command` can build a pure cancellation intent that updates the existing command task to
  `cancel_requested` with request authority, idempotency, correlation, provenance, and desired cancel state.
- `internal/projectors/command` also includes a guarded admission path that rejects unresolved target assets, rejects
  expired intents against wall clock, and collapses duplicate idempotency keys before producing mutation plans.
- `internal/projectors/command` includes deterministic command-intent arbitration that selects at most one active
  command per target by local override, authority rank, priority, observation time, and native ID before any native
  execution candidate is exposed.
- `internal/projectors/command` includes a guarded batch projection path that admits commands, arbitrates admitted
  active intents, projects accepted/superseded command-intent status, and exposes only accepted commands as future
  native execution candidates.
- `internal/projectors/command` maps MAVLink COMMAND_ACK readback evidence into command-intent lifecycle status
  updates without rewriting desired state, authority, priority, or target edges. `accepted` remains native acceptance
  evidence, `in_progress` maps to `executing`, rejection-style MAVLink results map to `rejected`, and final mission or
  task success remains out of scope.
- `internal/app` and `cmd/semops` connect to SemStreams, register first-phase COP ownership, enroll heartbeat, and
  compose the hosted MAVLink adapter with registry-derived owner tokens. The hosted runtime can bind multiple MAVLink
  UDP listeners through `SEMOPS_MAVLINK_UDP_LISTEN_ADDR` and `SEMOPS_MAVLINK_UDP_EXTRA_LISTEN_ADDRS`.
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
  The managed stack path defaults to a Compose-network route: it starts PX4 after the baseline hosted COP/MAVLink graph
  smokes, attaches PX4 to `semops-cop_default`, points both PX4 target arguments at the `semops` service alias, and
  stops PX4 before Compose cleanup removes the network.
- 2026-06-23: `SEMOPS_MAVLINK_SITL_DOCKER_PULL=true SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack bash
  scripts/mavlink-sitl-gate.sh` passed on commit `767fef0` after pulling `jonasvautherin/px4-gazebo-headless:1.17.0`.
  The generated evidence recorded `result=passed`, PX4/Gazebo headless Docker simulator version `1.17.0`, vehicle
  `gz_x500`, world `default`, expected track `c360.edge-compose.cop.mavlink.track.system-1`, snapshot URL
  `http://127.0.0.1:8080/api/cop/snapshot`, `min_updates=2`, and `require_motion=false`. The full command also passed
  the hosted COP, MAVLink born-first, CoT born-first, CAP born-first, and SAPIENT preflight smokes before cleaning up
  SemOps compose resources and the PX4 simulator container.
- 2026-06-23: `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack
  SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=60s bash
  scripts/mavlink-sitl-gate.sh` passed on commit `0884cdd` with the image already local. The generated evidence
  recorded `result=passed`, `require_motion=true`, `timeout=60s`, `min_updates=2`, vehicle `gz_x500`, world `default`,
  and expected track `c360.edge-compose.cop.mavlink.track.system-1`. The external SITL snapshot smoke passed in 25.52s,
  proving the PX4/Gazebo headless route produced enough position delta for the motion-required gate.
- 2026-06-27: `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack
  SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=90s SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true bash
  scripts/mavlink-sitl-gate.sh` passed with `px4_headless_route_mode=compose-network`,
  `px4_headless_docker_network=semops-cop_default`, and `px4_headless_network_target=semops`. The external SITL smoke
  passed in 0.52s after the PX4 boot wait, and the before-cleanup hook stopped `semops-px4-gazebo-headless` before
  Compose removed the network. This is still PX4 telemetry evidence only; ArduPilot parity and MAVSDK/offboard parity
  remain separate gates, while PX4 command readback is covered by the later `command-live-sim` evidence.
- 2026-06-27: SemOps hosted runtime and Compose stack gained dual MAVLink UDP listener coverage. `compose.cop.yml`
  defaults to primary `:14550` plus extra `:14540`, and `go test ./cmd/semops-mavlink-command ./internal/app
  ./internal/smoke/mavlink` passed after adding the extra-listener config/runtime tests.
- 2026-06-27: command-gate diagnostics improved the MVP transmitter helper with optional MAVLink peer heartbeat-first
  and reply forwarding knobs, and focused command codec tests passed:
  `go test ./pkg/adapters/mavlink -run
  'TestGeneratedCommandLongUsesCanonicalWireOrderAndParses|TestGeneratedRequestMessageCommandIsReadSideMVPShape|TestGeneratedCommandAckParsesResult'
  -count=1 -v`.
- 2026-06-27: earlier `command-live-sim` attempts narrowed the PX4/Gazebo blocker from guessed host ports to command
  endpoint/readback routing. The helper learned PX4's offboard telemetry source `172.19.0.9:14580` and later retried
  with a heartbeat, target component `0`, and bounded `COMMAND_LONG.confirmation` values `0/1/2`, but that route
  produced no direct helper replies and no graph-visible command task.
- 2026-06-27: the helper gained raw-lane command observation diagnostics and a learned-route port override without
  adding MAVSDK as a product dependency. This made the PX4 route shape explicit: learn the PX4 host from raw telemetry,
  send the reviewed read-side command to the command peer UDP port `18570`, and observe ACKs on SemOps' raw lane
  because PX4 sends command replies to its configured SemOps partner rather than the helper socket. The transmitter
  therefore reports `direct_command_acks=0` while raw-lane counters report `raw_command_acks` and
  `raw_last_ack_result`.
- 2026-06-27: `command-live-sim` passed against the local PX4/Gazebo headless image after adding MAVLink task prefix
  discovery to the COP graph provider. The reviewed native transmitter ran from the SemOps network namespace, sent
  three bounded `MAV_CMD_REQUEST_MESSAGE` attempts for `AUTOPILOT_VERSION` using `-learn-route-port 18570`, and
  recorded `command_attempts=3`, `direct_command_acks=0`, `forwarded_replies=0`, `raw_command_acks=3`, and
  `raw_last_ack_result=accepted`. The command-control COP snapshot smoke found
  `c360.edge-compose.cop.mavlink.task.system-1-command-512-target-255-190` and a fresh post-command MAVLink track.
  Evidence file: `tmp/mavlink-sitl-evidence/2026-06-27T22-29-45Z-command-live-sim.env`.
- 2026-06-23: `go test ./pkg/cop ./internal/projectors/mavlink ./internal/adapters/mavlink
  ./internal/components/mavlink ./internal/copownership` passed after adding the MAVLink command-task ownership
  contract and COMMAND_ACK readback projection. This is evidence of governed command lifecycle readback only; live
  command transmit, safety interlocks, priority, TTL, and CS API command reconciliation remain open.
- 2026-06-23: `pkg/cop` gained the `semops.command.intent.v1` command-intent contract with authority, priority,
  expiry, correlation, idempotency, requested-by, desired-state, status, provenance, and strict target edge fields.
  This is a graph governance contract only; no CS API ingress, local operator UI, or native transmitter writes it yet.
- 2026-06-23: `go test ./internal/projectors/command` passed for the pure command-intent planner. The gate proves
  valid desired command state writes a control-profiled task with a strict target edge, known intents update without
  repeating that edge, malformed or expired intents fail closed, and the planner does not birth target assets.
- 2026-06-23: the same command projector package now includes admission tests proving unresolved targets, expired
  intents, and duplicate idempotency keys return no mutation plan before any writer or native transmitter exists.
- 2026-06-24: `SEMOPS_MAVLINK_SITL_GATE_MODE=command-preflight` with an explicit PX4 simulator family, command
  target, `hold_position` action, simulator-local safety profile, local override, ACK requirement, and post-command
  state-polling requirement exited with `result=blocked_no_native_command_transmitter`. This is fail-closed
  safety-posture evidence only, not command readback pass evidence.
- 2026-06-28T00:15:11Z: `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack bash scripts/mavlink-sitl-gate.sh`
  exited with `result=blocked_no_local_simulator`. The gate stamped `simulator_family=ardupilot`, defaulted to
  `ArduCopter`, defaulted to motion-required telemetry, found no `sim_vehicle.py`, and found no ArduPilot/ArduCopter
  Docker image even though the local PX4/Gazebo headless image was present. Evidence file:
  `tmp/mavlink-sitl-evidence/2026-06-28T00-15-11Z-ardupilot-stack.env`. This is readiness-gap evidence only; it does
  not close ArduPilot telemetry parity.
- 2026-06-28: ArduPilot/Gazebo Docker image review found no public image strong enough to pin as the default for
  task 5.95. Official ArduPilot namespace images are active CI/build bases, not ready-made SemOps headless Gazebo
  sources. Third-party ArduPilot/Gazebo images exist, but their Docker Hub metadata showed stale tags, low pull counts,
  very large image sizes, empty descriptions, or unclear launch contracts. Keep
  `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE` explicit until a SemOps-owned headless image or reviewed external
  image passes `ardupilot-stack`.
- 2026-06-28T00:15:13Z: `SEMOPS_MAVLINK_SITL_GATE_MODE=mavsdk-offboard-stack bash scripts/mavlink-sitl-gate.sh`
  exited with `result=blocked_no_local_simulator`. The gate stamped `simulator_family=mavsdk`, defaulted to
  `mavsdk_server udp://:14540`, defaulted to motion-required telemetry, found no `mavsdk_server`, and found no MAVSDK
  Docker image even though the local PX4/Gazebo headless image was present. Evidence file:
  `tmp/mavlink-sitl-evidence/2026-06-28T00-15-13Z-mavsdk-offboard-stack.env`. This is readiness-gap evidence only; it
  does not close MAVSDK/offboard parity.
- 2026-06-26T01:23:17Z: `SEMOPS_MAVLINK_SITL_GATE_MODE=command-live-sim bash scripts/mavlink-sitl-gate.sh`
  exited with `result=blocked_missing_command_transmitter`. The gate got through explicit PX4 simulator family,
  simulator-only safety profile, local override, abort readiness, ACK requirement, post-state requirement,
  reviewed-transmitter attestation, transmit enablement, expected ACK task, and expected post-state track guards, then
  stopped because no actual reviewed simulator transmitter command was provided. This is readiness-gap evidence only;
  the later 2026-06-27 `command-live-sim` pass supersedes it for the narrow read-side PX4 command-readback claim.
- 2026-06-26: `cmd/semops-mavlink-command` was added as the MVP simulator transmitter helper. It only allows
  `MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION`, requires simulator-only confirmation, and prints
  `expected_ack_task_suffix=system-1-command-512-target-255-190` in dry-run mode. This supports the read-side feed
  story and does not claim mission, mode, arm/disarm, offboard, or hardware command authority.
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
- NATS-backed graph mutation requests use SemStreams retry-aware classified mutation request handling.
- Graph mutation conflicts are reconciled only from ADR-060 typed classified errors, not text response bodies or
  retired `MutationResponse` failure fields.
- The in-process adapter harness exposes pollable health counters for raw ingest, projection, graph writes, and errors.
- Structural wiring can compose the NATS-backed writer path without launching the full stack.
- Commands, mission state, and battery alerts use `indexing_profile=control`.
- COMMAND_ACK packets project to `indexing_profile=control` command-task readback state with a strict
  `cop.task.target` edge to the born MAVLink source asset.
- Desired command intent uses `indexing_profile=control` under `semops.command.intent`; native feed ACK/status
  evidence remains separate.
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
  frames from the test. [done against PX4/Gazebo headless Docker on 2026-06-23]
- The track uses `semops.feed.mavlink` provenance, carries a non-empty source reference, has non-zero position and
  velocity evidence, and appears while `feed.mavlink` is live. [done against PX4/Gazebo headless Docker on 2026-06-23]
- The smoke observes repeated simulator updates and can require actual position motion with
  `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true`. [done against PX4/Gazebo headless Docker on 2026-06-23]
- Local readiness preflight records whether PX4, MAVSDK, ArduPilot, or equivalent simulator tooling is actually
  available before attempting the stack gate. [done: 2026-06-23 no simulator runtime found]
- Focused and stack helpers require a named simulator source before running the evidence gate. [done]
- Focused and stack helpers require an explicit simulator family before running the evidence gate, and helper evidence
  records `simulator_family` so PX4, ArduPilot, MAVSDK, hardware, and other MAVLink evidence cannot be collapsed into
  one generic pass. The helper also requires local tooling for the declared family, or an explicit remote-source
  override, before running the gate. [done]
- Preferred PX4/Gazebo headless Docker helper is wired, fail-closed on missing local image unless pull is explicitly
  enabled, and records simulator image/vehicle/world evidence. [done and passed against pulled image on 2026-06-23]
- Command-control preflight mode records the intended simulator family, target, action, safety profile, local override
  posture, ACK requirement, and post-command polling requirement, then exits blocked before native transmit because no
  transmitter belongs in preflight. [done as fail-closed evidence only]
- `command-live-sim` mode requires a simulator family other than hardware, simulator-only safety posture, abort
  readiness, explicit transmit enablement, a reviewed transmitter command, an expected MAVLink `COMMAND_ACK` task, and
  an expected post-command MAVLink track. It now first polls the COP snapshot for the named track as baseline
  live-telemetry evidence and blocks before transmit if that track is stale or missing; after transmit it polls for ACK
  task evidence and post-command track refresh before it can pass. [done as fail-closed readiness evidence only; open
  for real transmitter pass]
- `cmd/semops-mavlink-command` provides the reviewed MVP transmitter command shape: a simulator-only
  `MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION`, with dry-run metadata and an ACK task suffix. [done as dry-run
  evidence only; open for real simulator pass]
- Dedicated `ardupilot-stack` mode stamps the ArduPilot simulator family, defaults to `ArduCopter`, defaults to
  motion-required telemetry, and blocks unless a real ArduPilot source is available locally or explicitly routed in.
  [done as fail-closed readiness evidence only]
- The ArduPilot lane has an opt-in managed Docker path for reviewed ArduPilot-family images. It has no default image,
  can pull only when `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_PULL=true`, can run a custom
  `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_COMMAND`, and otherwise starts `sim_vehicle.py` on the SemOps Compose network
  with telemetry routed to `semops:14550`. [done as setup support; 5.95 remains open until a real pass]
- Dedicated `mavsdk-offboard-stack` mode stamps the MAVSDK simulator family, defaults to `mavsdk_server udp://:14540`,
  defaults to motion-required telemetry, and blocks unless a real MAVSDK/offboard source is available locally or
  explicitly routed in. [done as fail-closed readiness evidence only]
- Against explicit ArduPilot SITL, the controller connects, reads status, and performs telemetry parity checks before
  any ArduPilot interoperability claim. Live command smoke remains a separate reviewed gate.
  [open]
- PX4 SITL or MAVSDK smoke evidence is recorded before calling MAVLink Phase 1 complete. [done for PX4/Gazebo
  telemetry smoke and PX4 simulator command readback; MAVSDK/offboard parity remains open]

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
- Restart reconciliation now handles typed `entity_already_exists` create conflicts for known MAVLink asset/track
  births by marking the conflicted entity born and reprojecting the current packet. Durable checkpoint/read-back
  recovery and scenario-runner replay integration remain open.
- No live SITL controller remains; a modern harness must be rebuilt with explicit readiness and state polling before
  command/control demo claims.
- The external SITL telemetry smoke harness has passed against PX4/Gazebo headless Docker with and without
  motion-required assertions. The old bundled `5.4` gate is closed only for parser/generator and PX4 telemetry
  evidence; ArduPilot parity and MAVSDK/offboard parity remain separate open gates.
- The helper now fail-closes focused/stack evidence unless the run declares `SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY`.
  PX4 headless mode stamps `px4` automatically and refuses contradictory family values.
- The helper now has `command-preflight` mode for safety-posture evidence, but it always exits with blocked evidence
  because preflight is non-transmitting by design. Use `command-live-sim` for reviewed simulator transmit.
- Old `RoboticsProcessor`, BaseMessage payload graphing, StreamKit, and ObjectStore paths have been removed from the
  active product path rather than preserved as migration targets.
- Command codec coverage, COMMAND_ACK readback projection, and a PX4 simulator read-side command request are active,
  but command reconciliation, priority, TTL, mission/offboard authority, hardware authority, and safety interlocks are
  not.
- PX4/Gazebo headless telemetry, motion-required evidence, and simulator read-side command readback evidence are in
  SemOps; ArduPilot parity and MAVSDK/offboard parity remain open.

## Adversarial Feed-Entry Questions

- Are parser tests using real MAVLink bytes, or did the adapter start trusting structs?
- Is there exactly one current vehicle graph entity per vehicle, not one entity per packet?
- Does low-battery alert state belong to the feed owner or a derived rule owner?
- Does raw/replay detail stay out of semantic indexing?
- Does the demo claim all-family MAVLink simulator coverage from one PX4/Gazebo telemetry gate, or does the evidence
  name the simulator family that actually passed?
