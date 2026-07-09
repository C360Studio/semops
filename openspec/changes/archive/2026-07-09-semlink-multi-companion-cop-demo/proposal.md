## Why

SemLink has completed its companion e2e/demo surface: single-node reports, simple mesh reports, local API evidence,
and a fake SemOps readback receiver. SemOps now needs the matching product proof: GCS/COP glass that can consume that
evidence for `n` companion nodes without becoming the companion runtime, mesh transport, BlueOS package, or raw
MAVLink relay.

The existing SemLink companion contract proves the MVP readback hot path. This change adds the demo acceptance layer
above it: SemOps can ingest SemLink demo evidence, expose a curated cross-node view model, and render/verify operator
glass for boats, drones, or rovers while preserving the native SemLink hot path and the SemConnect/CS API interop edge.

## What Changes

- Add a focused OpenSpec change for the SemLink multi-companion COP demo.
- Define SemOps as the consumer of SemLink single-node and simple-mesh demo reports.
- Require a SemOps fixture/report adapter before live SemLink/SemOps compose wiring.
- Require a curated COP API/view model for companion nodes, vehicles, mesh-readiness evidence, readback status, and
  no-transmit posture.
- Require browser/smoke coverage that proves `n` companion nodes are visible and inspectable in the operator surface.
- Keep SemLink mesh mechanics, BlueOS/Navigator/Pi packaging, MAVLink telemetry ports, and local command-safety rules
  owned by SemLink.
- Keep SemConnect/CS API as the standards projection edge, not a required SemLink-to-SemOps hot-path dependency.

## Capabilities

### New Capabilities

- `semlink-multi-companion-cop-demo`: SemOps consumes SemLink e2e/demo evidence and renders cross-node companion
  COP/GCS glass with explicit ownership and no-transmit boundaries.

### Related Capabilities

- `semlink-semops-companion-contract`: Supplies the native readback intent/result contract that the demo evidence
  must preserve.
- `cop-ui-experience`: Supplies the map-first, API-curated browser surface that will display the companion fleet.
- `cop-product-ownership`: Keeps SemOps responsible for COP/GCS glass and SemLink responsible for companion/mesh
  runtime.

## Impact

- New SemOps OpenSpec requirements and implementation tasks.
- Follow-up SemOps fixture/testdata for SemLink report consumption.
- Follow-up COP API/view-model fields for companion nodes and SemLink readback posture.
- Follow-up browser or stack smoke proving `n` SemLink companion nodes without requiring live BlueOS or hardware.
