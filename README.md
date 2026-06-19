# SemOps

SemOps is the SemStreams-backed data-fusion common operating picture product.

The current revival is intentionally greenfield. SemOps owns the operator COP, feed adapters, canonical COP model,
scenario runner, product governance, and UI. SemStreams owns the substrate: projection contracts, ownership rules,
graph mutation/query APIs, indexing profiles, NATS/JetStream runtime primitives, and reusable framework behavior.

## Current State

- OpenSpec change: `openspec/changes/revive-cop-product`
- Architecture baseline: `docs/cop-demo-revival-architecture.md`
- Feed evidence ladder: `docs/feed-validation-and-indexing-ladder.md`
- Feed product roadmap: `docs/feed-product-roadmap.md`
- COP model baseline: `docs/cop-model-and-governance.md`
- COP UI baseline: `docs/cop-ui-stack.md`
- Active MAVLink codec boundary: `pkg/adapters/mavlink`
- Active TAK/CoT codec, replay, projection, and graph-wiring boundary: `pkg/adapters/cot`,
  `internal/adapters/cot`, `internal/projectors/cot`

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
- TAK/CoT marker/task control state as strict `control`
- TAK/CoT GeoChat/advisory text as strict `content`
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

The test suite validates the SemStreams contract gate, COP ownership model, active MAVLink codec, and TAK/CoT
parser/replay/projection/graph-wiring gates. SITL/PX4 simulator gates are still future evidence, not current product
claims.

Run the hosted runtime against a live SemStreams/NATS stack:

```bash
SEMOPS_NATS_URL=nats://127.0.0.1:4222 go run ./cmd/semops
```

`cmd/semops` now connects to SemStreams, registers first-phase COP ownership, enrolls owners for heartbeat, and
composes hosted MAVLink and opt-in TAK/CoT adapters with typed owner tokens minted by SemStreams registry/bind results.

Enable UDP MAVLink ingestion explicitly when you want the hosted runtime to listen for datagrams:

```bash
SEMOPS_NATS_URL=nats://127.0.0.1:4222 \
SEMOPS_MAVLINK_UDP_LISTEN_ADDR=:14550 \
go run ./cmd/semops
```

Enable TAK/CoT ingestion explicitly when you want the hosted runtime to listen for CoT XML. This writes governed graph
state but is not yet surfaced through the COP snapshot API/UI:

```bash
SEMOPS_NATS_URL=nats://127.0.0.1:4222 \
SEMOPS_COT_ENABLED=true \
SEMOPS_COT_UDP_LISTEN_ADDR=:8087 \
go run ./cmd/semops
```

Run the current one-command graph smoke stack:

```bash
bash scripts/cop-stack-smoke.sh
```

This starts NATS, the SemStreams graph backend, the SemOps runtime/API, the Svelte COP UI, and Caddy with Docker
Compose. The smoke polls SemStreams health and metrics, the direct SemOps API, and the Caddy-routed browser path before
running the MAVLink live graph smoke with both NATS and metrics URLs wired in.

The local browser entrypoint is:

```text
http://localhost:8080
```

Caddy proxies `/api/*` and `/healthz` to SemOps API so the UI uses the same-origin path operators will expect from
real infrastructure. The direct API remains available on `http://localhost:8088` for diagnostics. The current UI/API
snapshot has a fixture fallback plus MAVLink graph-backed readback; CoT graph state, stale policy, and the scenario
runner are still being built.
