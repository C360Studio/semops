# MAVLink Reference Hold

The SemOps revival is greenfield. Legacy product paths should not stay in the repository unless they have concrete
reference value for the next extraction slice.

## Retained Reference Files

Only these MAVLink references remain under `pkg/processors/mavlink`, guarded by the `ignore` build constraint:

- `sitl/*`

They are not part of the active product compile path. They are retained because SemOps appears to have useful ArduPilot
SITL control scenarios that still need extraction behind the active adapter boundary.

## Extracted Reference Behavior

These MAVLink references have moved into the active product path:

- `pkg/adapters/mavlink/parser.go` parses MAVLink v1/v2 frames, validates checksums, handles stream buffering and
  resync, and decodes the first COP telemetry messages.
- `pkg/adapters/mavlink/generator.go` generates MAVLink v2 heartbeat, global position, attitude, battery, and
  deterministic quadcopter scenario frames.
- `pkg/adapters/mavlink/parser_test.go` proves those frames with real binary decode tests, including split buffers,
  noisy resync, checksum rejection, and canonical battery wire order.

## Removed Legacy Paths

These old paths were removed because they carried stale architecture rather than useful reference value:

- `pkg/entities`
- old ignored MAVLink constants, parser, generator, and parser/generator tests
- `pkg/processors/mavlink/payloads`
- `pkg/processors/mavlink/rules`
- `pkg/processors/mavlink/vocabulary`
- old MAVLink BaseProcessor, metrics, error, compliance, and routing tests
- `test/migrated_tests`

The removed code depended on old SemStreams or StreamKit surfaces such as EntityStore conversion helpers,
ObjectStore migration tests, BaseProcessor lifecycle assumptions, and old payload graphing.

## Re-entry Rule

Reference code can re-enter the product path only after it is extracted behind a modern SemOps package boundary and
tested against current SemStreams contracts. The current accepted patterns are:

- SemStreams contract gate: `internal/contracts/semstreams_contract_test.go`
- COP ownership contracts: `pkg/cop/contracts.go`
- COP contract tests: `pkg/cop/contracts_test.go`

## Deletion Rule

Delete retained reference files as soon as their useful SITL behavior has either been extracted or deliberately
rejected.
