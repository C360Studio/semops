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
scenario frames. It also has JSON Lines replay fixture storage for raw-lane records. Ignored SITL
controller/scenario references were deleted after useful command encoding and ACK decoding moved into the active
adapter package.

## Development

Run the current active test gate:

```bash
go test ./...
```

The test suite validates the SemStreams contract gate, COP ownership model, and active MAVLink codec. SITL/PX4
simulator gates are still future evidence, not current product claims.

Run the hosted runtime against a live SemStreams/NATS stack:

```bash
SEMOPS_NATS_URL=nats://127.0.0.1:4222 go run ./cmd/semops
```

`cmd/semops` now connects to SemStreams, registers first-phase COP ownership, enrolls owners for heartbeat, and
composes the hosted MAVLink adapter with typed owner tokens minted by SemStreams registry/bind results.

Enable UDP MAVLink ingestion explicitly when you want the hosted runtime to listen for datagrams:

```bash
SEMOPS_NATS_URL=nats://127.0.0.1:4222 \
SEMOPS_MAVLINK_UDP_LISTEN_ADDR=:14550 \
go run ./cmd/semops
```

Run the current one-command graph smoke stack:

```bash
bash scripts/cop-stack-smoke.sh
```

This starts NATS, the SemStreams graph backend, and the SemOps runtime with Docker Compose, polls health and metrics,
then runs the MAVLink live graph smoke with both NATS and metrics URLs wired in. It is a substrate/runtime smoke, not
yet the full COP API, UI, scenario runner, or feed transport stack.
