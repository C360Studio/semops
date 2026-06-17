# MAVLink Feed Evidence

Status: candidate Phase 1 feed, codec extraction started. Projection remains blocked by `COP-004`.

## Decision

MAVLink should be the first feed because SemOps already contained parser, generator, payload, rule, and SITL material.
The active path now has a modern parser/generator package. Projection work still needs current-state SemStreams writes,
raw-lane boundaries, born-first source/track creation, and SITL/PX4 evidence.

## Local Evidence

- `pkg/adapters/mavlink/parser.go` parses MAVLink v1/v2 frames, validates checksums, handles stream buffering and
  resync, and registers the first COP message specs.
- `pkg/adapters/mavlink/generator.go` generates MAVLink v2 heartbeat, battery status, global position, attitude, and
  deterministic quadcopter scenario frames with CRC.
- `pkg/adapters/mavlink/parser_test.go` validates generator/parser compatibility, split buffers, noisy resync,
  checksum rejection, concurrent sequence generation, scenario frame generation, and canonical battery wire order.
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

Target test before replacing old processor wiring:

```bash
go test ./internal/projectors/mavlink
```

Acceptance:

- A heartbeat plus position plus battery sequence creates or updates one current vehicle entity.
- Raw frames remain on a bounded lane or source reference.
- Vehicle current state uses `indexing_profile=signal`.
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
