# MAVLink Reference Hold

The SemOps revival is greenfield. Legacy product paths should not stay in the repository unless they have concrete
reference value for the next extraction slice.

## Retained Reference Files

None.

All retained MAVLink reference files have either been extracted into the active adapter package or deliberately
rejected. The ignored SITL controller/scenario tree was deleted after COMMAND_LONG, COMMAND_ACK, ArduCopter mode
mapping, and MAV_RESULT naming moved into `pkg/adapters/mavlink`.

## Extracted Reference Behavior

These MAVLink references have moved into the active product path:

- `pkg/adapters/mavlink/parser.go` parses MAVLink v1/v2 frames, validates checksums, handles stream buffering and
  resync, and decodes the first COP telemetry messages.
- `pkg/adapters/mavlink/generator.go` generates MAVLink v2 heartbeat, global position, attitude, battery, and
  deterministic quadcopter scenario frames.
- `pkg/adapters/mavlink/commands.go` provides command result naming and ArduCopter mode mapping.
- `pkg/adapters/mavlink/commands_test.go` proves COMMAND_LONG and COMMAND_ACK frame generation/parsing from real
  MAVLink bytes.
- `pkg/adapters/mavlink/raw_lane.go` keeps copied MAVLink frames under record and byte caps without creating graph
  entities per packet.
- `pkg/adapters/mavlink/parser_test.go` proves those frames with real binary decode tests, including split buffers,
  noisy resync, checksum rejection, and canonical battery wire order.

## Removed Legacy Paths

These old paths were removed because they carried stale architecture rather than useful reference value:

- `pkg/entities`
- old ignored MAVLink constants, parser, generator, and parser/generator tests
- ignored MAVLink SITL controller, command wrapper, status, handler, scenario, example, and integration-test files
- `pkg/processors/mavlink/payloads`
- `pkg/processors/mavlink/rules`
- `pkg/processors/mavlink/vocabulary`
- old MAVLink BaseProcessor, metrics, error, compliance, and routing tests
- `test/migrated_tests`
- `configs/robotics-flow.json`

The removed code depended on old SemStreams or StreamKit surfaces such as EntityStore conversion helpers,
ObjectStore migration tests, BaseProcessor lifecycle assumptions, and old payload graphing.
The removed flow config also depended on raw subject topology that no longer describes the active SemStreams
component lifecycle, flowgraph, payload-registry, port, config-schema, projection, and graph mutation surfaces.

## Re-entry Rule

Reference code can re-enter the product path only after it is extracted behind a modern SemOps package boundary and
tested against current SemStreams contracts. The current accepted patterns are:

- SemStreams contract gate: `internal/contracts/semstreams_contract_test.go`
- COP ownership contracts: `pkg/cop/contracts.go`
- COP contract tests: `pkg/cop/contracts_test.go`
- Hosted feed service gates must use SemStreams input/processor component lifecycle, flowgraph, registered
  `message.BaseMessage` payloads, ports, config schema, health, and flow metrics.
- Transport listeners are input components; parser, decoder, projector, and fusion behavior are processor components
  connected through declared ports.

## Deletion Rule

Delete retained reference files as soon as their useful behavior has either been extracted or deliberately rejected.
