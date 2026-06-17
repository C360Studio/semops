# Legacy Quarantine

The SemOps revival is greenfield. Legacy code is preserved for salvage, but it is no longer part of the default
product compile path.

## Quarantined Trees

These trees now use the `ignore` build constraint:

- `pkg/entities`
- `pkg/processors/mavlink`
- `test/migrated_tests`

The quarantine is deliberate. These packages still depend on old SemStreams and StreamKit surfaces such as
`github.com/c360/semstreams/pkg/interfaces/store`, `github.com/c360/streamkit`, old BaseProcessor wiring, and migrated
ObjectStore assumptions. Current SemStreams exposes projection contracts, ownership registration, graph mutation
requests, component lifecycle interfaces, and message triples instead.

The `ignore` constraint is intentional. `go mod tidy` evaluates ordinary build tags broadly, so a plain `legacy` tag
would still force stale private modules into dependency resolution. Re-entry should move salvageable code behind a
modern package boundary instead of compiling these files in place.

## Salvage Policy

Do salvage:

- MAVLink frame parsing, message specs, test generators, and SITL scenario/controller ideas.
- Real-frame tests that prove protocol decoding.
- Domain vocabulary that still fits the COP entity model.

Do not salvage by default:

- StreamKit flow-runtime wiring.
- BaseProcessor lifecycle assumptions.
- ObjectStore migration tests as product contracts.
- EntityStore conversion helpers tied to removed SemStreams interfaces.

## Re-entry Rule

Legacy code can re-enter the product path only after it is moved behind a modern SemOps package boundary and tested
against current SemStreams contracts. The first accepted pattern is the contract test in
`internal/contracts/semstreams_contract_test.go`.
