# MAVLink Feed Evidence

Status: candidate Phase 1 feed, codec, bounded raw lane, projection planner, and SemStreams graph writer boundary
extracted. Live feed integration remains blocked by stack, durable replay, and restart/replay work in `COP-004`.

## Decision

MAVLink should be the first feed because SemOps already contained parser, generator, payload, rule, and SITL material.
The active path now has a modern parser/generator package, bounded in-memory raw lane, COMMAND_LONG/COMMAND_ACK
coverage, current-state projection planner, tested graph request/reply writer boundary, retry-aware SemStreams NATS
requester boundary, and in-process adapter harness. Live feed work still needs scenario-runner replay wiring,
container stack wiring, SITL/PX4 evidence, restart/replay reconciliation, and stack health checks.

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
  signal-profiled track current state, source-reference projection, and update-only behavior after first birth.
- `internal/projectors/mavlink/writer.go` sends plans to SemStreams `graph.mutation.entity.create_with_triples` and
  `graph.mutation.entity.update_with_triples` request/reply subjects.
- `internal/projectors/mavlink/writer_test.go` proves write ordering, owner-token transit,
  committed-but-degraded response handling, cancellation, failure stops, and unsupported mutation rejection.
- `internal/graphrequest` adapts SemStreams `natsclient.Client.RequestWithRetry` into the graph writer requester
  interface so mutation writers do not use bare query-style request calls.
- `internal/adapters/mavlink` composes parser, raw lane, projector, graph plan writer, and pollable health counters
  for the future adapter service boundary.
- `internal/adapters/mavlink/adapter_test.go` proves valid telemetry writes graph plans, command ACK frames are
  captured without graph writes, corrupt frames stop before graph writes, and writer failures are reflected in health.
- Ignored ArduPilot SITL controller/scenario reference files were deleted after command encoding and ACK parsing moved
  into the active adapter and the live controller was rejected as legacy scaffolding.

## External Evidence

- MAVLink developer guide documents MAVLink as a lightweight messaging protocol for drones and onboard components.
- PX4 documents a simulator path and a Simulator MAVLink API for exchanging simulated sensor and actuator data.
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
- Commands, mission state, and battery alerts use `indexing_profile=control`.
- Replay/decode records use `indexing_profile=trace`.
- No graph entity is created per raw packet.

### SITL Gate

Target command after SITL harness exists:

```bash
go test ./internal/adapters/mavlink/sitl
```

Acceptance:

- Tests skip cleanly when SITL is unavailable.
- Against explicit ArduPilot SITL, the controller connects, reads status, and performs safe command smoke tests.
- PX4 SITL or MAVSDK smoke evidence is added before calling MAVLink Phase 1 complete.

### Replay Gate

Target artifact:

- A small captured MAVLink fixture with heartbeat, position, attitude, battery, low-battery, and stale-track cases.
- Current fixture format: JSON Lines `pkg/adapters/mavlink.RawFrameRecord` values, with frame bytes JSON-encoded as
  base64 by Go's standard encoder.

Acceptance:

- Replaying the fixture yields deterministic current-state graph output and feed-health/staleness transitions.

## Known Gaps

- The active module path, Go toolchain, and MAVLink parser/generator are modernized.
- The current-state projection planner and graph writer emit and send current SemStreams graph mutation shapes, but the
  writer is not yet wired into a live containerized stack.
- The in-process adapter harness is not a UDP/TCP listener and is not yet hosted as `semops-adapter-mavlink`.
- Raw-lane capture and replay fixture storage are library boundaries; scenario-runner playback and stack retention
  policy are not implemented yet.
- Restart/replay reconciliation is not implemented; a restarted adapter cannot yet prove whether entities are already
  born without a read-back or checkpoint path.
- No live SITL controller remains; a modern harness must be rebuilt with explicit readiness and state polling before
  command/control demo claims.
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
