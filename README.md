# SemOps

SemOps is the SemStreams-backed data-fusion common operating picture product.

The current revival is intentionally greenfield. SemOps owns the operator COP, feed adapters, canonical COP model,
scenario runner, product governance, and UI. SemStreams owns the substrate: projection contracts, ownership rules,
graph mutation/query APIs, indexing profiles, NATS/JetStream runtime primitives, and reusable framework behavior.

## Current State

- OpenSpec change: `openspec/changes/revive-cop-product`
- Architecture baseline: `docs/cop-demo-revival-architecture.md`
- Feed evidence ladder: `docs/feed-validation-and-indexing-ladder.md`
- COP model baseline: `docs/cop-model-and-governance.md`
- MAVLink reference hold: `docs/legacy-quarantine.md`

The active Go path is modernized to `github.com/c360studio/semops` and current SemStreams module imports. Old
StreamKit, EntityStore, ObjectStore, and BaseProcessor product paths have been removed or are outside the active build.

## First Product Model

The first canonical entity set is:

- `track`
- `asset`
- `hazard_area`
- `sensor_footprint`
- `alert`
- `task`
- `advisory`

The initial ownership-contract matrix lives in `pkg/cop` and covers:

- MAVLink current track state as strict `signal`
- TAK/CoT current track state as strict `signal`
- CAP hazard/advisory evidence as append-only `content`
- deterministic fusion alerts as derived `control`

## Reference Material

Only MAVLink material with near-term extraction value remains under `pkg/processors/mavlink`, guarded by the `ignore`
build constraint:

- protocol constants
- binary parser and parser tests
- test frame generator
- ArduPilot SITL controller/scenario scaffolding

This is a temporary reference hold. Once the useful parser, generator, and SITL pieces are extracted into modern
SemOps package boundaries, the ignored reference files should be deleted.

## Development

Run the current active test gate:

```bash
go test ./...
```

The test suite validates the SemStreams contract gate and COP ownership model. Ignored MAVLink reference files are not
part of the active product build.
