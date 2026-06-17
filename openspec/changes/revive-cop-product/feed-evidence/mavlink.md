# MAVLink Feed Evidence

Status: candidate Phase 1 feed, blocked from implementation by `COP-001`, `COP-002`, `COP-007`, and `COP-008`.

## Decision

MAVLink should be the first feed because SemOps already contains parser, generator, payload, rule, and SITL material.
The salvage path is credible, but the current package still uses the old SemStreams module path and old processor
shape, so the first implementation work must be modernization and projection contract tests.

## Local Evidence

- `pkg/processors/mavlink/parser/mavlink_parser.go` parses MAVLink v1/v2 frames, validates checksums, handles buffer
  resync, and registers standard message specs.
- `pkg/processors/mavlink/testing/mavlink/generator.go` generates MAVLink v2 heartbeat, battery, global position, and
  attitude frames with CRC.
- `pkg/processors/mavlink/testing/mavlink/generator_test.go` validates generator structure, thread safety, realistic
  message sequences, and parser compatibility.
- `pkg/processors/mavlink/parser/mavlink_parser_integration_test.go` exercises generated frames through the parser,
  multiple messages in one buffer, corrupted messages, and field extraction.
- `pkg/processors/mavlink/sitl` contains an ArduPilot SITL controller, command helpers, status handling, and skip-clean
  integration tests when SITL is unavailable.

## External Evidence

- MAVLink developer guide documents MAVLink as a lightweight messaging protocol for drones and onboard components.
- PX4 documents a simulator path and a Simulator MAVLink API for exchanging simulated sensor and actuator data.
- ArduPilot documents SITL as a way to run Plane, Copter, or Rover without hardware.

## Gates

### Parser Gate

Target command after `COP-001`:

```bash
go test ./pkg/processors/mavlink/parser ./pkg/processors/mavlink/testing/mavlink
```

Acceptance:

- Heartbeat, battery, global position, and attitude frames parse with expected system/component IDs and fields.
- Corrupted frames do not panic and do not publish governed graph state.
- Multiple messages in one buffer produce stable ordered packets.

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

- `go.mod` still uses `github.com/c360/semstreams` and Go `1.25.3`; current SemStreams uses
  `github.com/c360studio/semstreams` and a newer toolchain.
- Old `RoboticsProcessor` wiring still publishes old BaseMessage-shaped semantics rather than modern SemStreams
  projection writes.
- Processor comments still contain simplified parsing paths that should not survive the salvage.
- PX4 SITL/MAVSDK evidence is not yet in SemOps.

## Adversarial Feed-Entry Questions

- Are parser tests using real MAVLink bytes, or did the adapter start trusting structs?
- Is there exactly one current vehicle graph entity per vehicle, not one entity per packet?
- Does low-battery alert state belong to the feed owner or a derived rule owner?
- Does raw/replay detail stay out of semantic indexing?
- Does the demo claim PX4 coverage before a PX4 or MAVSDK smoke gate exists?
