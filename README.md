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
- Active MAVLink codec boundary: `pkg/adapters/mavlink`

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

## MAVLink Salvage

The active codec now lives in `pkg/adapters/mavlink` with parser/generator tests for heartbeat, global position,
attitude, battery status, COMMAND_LONG, COMMAND_ACK, split buffers, resync, checksum rejection, raw-lane bounds, and
scenario frames. Ignored SITL controller/scenario references were deleted after useful command encoding and ACK
decoding moved into the active adapter package.

## Development

Run the current active test gate:

```bash
go test ./...
```

The test suite validates the SemStreams contract gate, COP ownership model, and active MAVLink codec. SITL/PX4
simulator gates are still future evidence, not current product claims.
