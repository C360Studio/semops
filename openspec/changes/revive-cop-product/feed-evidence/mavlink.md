# MAVLink Feed Evidence

Status: candidate Phase 1 feed, codec, projection planner, and SemStreams graph writer boundary extracted. Live feed
integration remains blocked by raw-lane, stack, and restart/replay work in `COP-004`.

## Decision

MAVLink should be the first feed because SemOps already contained parser, generator, payload, rule, and SITL material.
The active path now has a modern parser/generator package, current-state projection planner, and tested graph
request/reply writer boundary. Live feed work still needs raw-lane boundaries, container stack wiring, SITL/PX4
evidence, restart/replay reconciliation, and stack health checks.

## Local Evidence

- `pkg/adapters/mavlink/parser.go` parses MAVLink v1/v2 frames, validates checksums, handles stream buffering and
  resync, and registers the first COP message specs.
- `pkg/adapters/mavlink/generator.go` generates MAVLink v2 heartbeat, battery status, global position, attitude, and
  deterministic quadcopter scenario frames with CRC.
- `pkg/adapters/mavlink/parser_test.go` validates generator/parser compatibility, split buffers, noisy resync,
  checksum rejection, concurrent sequence generation, scenario frame generation, and canonical battery wire order.
- `internal/projectors/mavlink` maps decoded heartbeat, global position, attitude, and battery packets into ordered
  SemStreams graph mutation requests.
- `internal/projectors/mavlink/projector_test.go` proves source asset birth before strict `cop.track.source` edges,
  signal-profiled track current state, and update-only behavior after first birth.
- `internal/projectors/mavlink/writer.go` sends plans to SemStreams `graph.mutation.entity.create_with_triples` and
  `graph.mutation.entity.update_with_triples` request/reply subjects.
- `internal/projectors/mavlink/writer_test.go` proves write ordering, owner-token transit,
  committed-but-degraded response handling, cancellation, failure stops, and unsupported mutation rejection.
- `pkg/processors/mavlink/sitl` still contains ignored ArduPilot SITL controller/scenario reference files. These are
  retained only until command/control behavior is extracted or rejected.

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

### Projection Gate

Current command:

```bash
go test ./internal/projectors/mavlink
```

Acceptance:

- Heartbeat, global position, attitude, and battery packets create or update one current vehicle entity per MAVLink
  system.
- The source asset is born before the track writes a strict `cop.track.source` foreign edge.
- Raw frames remain on a bounded lane or source reference.
- Vehicle current state uses `indexing_profile=signal`.
- The graph writer targets the current SemStreams create/update-with-triples request subjects.
- A committed-but-degraded mutation response is treated as committed and not retried.
- Commands, mission state, and battery alerts use `indexing_profile=control`.
- Replay/decode records use `indexing_profile=trace`.
- No graph entity is created per raw packet.

### SITL Gate

Target command after SITL harness exists:

```bash
go test ./pkg/processors/mavlink/sitl
```

Acceptance:

- Tests skip cleanly when SITL is unavailable.
- Against explicit ArduPilot SITL, the controller connects, reads status, and performs safe command smoke tests.
- PX4 SITL or MAVSDK smoke evidence is added before calling MAVLink Phase 1 complete.

### Replay Gate

Target artifact:

- A small captured MAVLink fixture with heartbeat, position, attitude, battery, low-battery, and stale-track cases.

Acceptance:

- Replaying the fixture yields deterministic current-state graph output and feed-health/staleness transitions.

## Known Gaps

- The active module path, Go toolchain, and MAVLink parser/generator are modernized.
- The current-state projection planner and graph writer emit and send current SemStreams graph mutation shapes, but the
  writer is not yet wired into a live containerized stack.
- Raw-lane capture and replay fixture storage still need the containerized stack boundary.
- Restart/replay reconciliation is not implemented; a restarted adapter cannot yet prove whether entities are already
  born without a read-back or checkpoint path.
- SITL command/control reference files still need extraction behind modern SemOps package boundaries.
- Old `RoboticsProcessor`, BaseMessage payload graphing, StreamKit, and ObjectStore paths have been removed from the
  active product path rather than preserved as migration targets.
- Command coverage is not active until the SITL controller and COMMAND_LONG/COMMAND_ACK tests move into the adapter
  package.
- PX4 SITL/MAVSDK evidence is not yet in SemOps.

## Adversarial Feed-Entry Questions

- Are parser tests using real MAVLink bytes, or did the adapter start trusting structs?
- Is there exactly one current vehicle graph entity per vehicle, not one entity per packet?
- Does low-battery alert state belong to the feed owner or a derived rule owner?
- Does raw/replay detail stay out of semantic indexing?
- Does the demo claim PX4 coverage before a PX4 or MAVSDK smoke gate exists?
