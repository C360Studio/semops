# Design

## Demo Shape

The first SemOps mesh demo should use a SemLink-generated deterministic artifact
from the one-vehicle-per-companion mesh proof:

- `artifact_kind = semlink-companion-demo-artifact-v0`
- `report.kind = simple-mesh-companion-demo`
- `report.expected_summaries = 3`
- `report.nodes[].vehicle_count = 1`
- `source.source_fidelity = deterministic`

SemOps should ingest the artifact through the existing
`SEMOPS_COP_SEMLINK_ARTIFACT_PATH` path or an equivalent smoke harness and then
render a companion fleet in the COP snapshot/browser surface.

## Evidence Boundaries

The demo may claim:

- SemOps can display a SemLink-produced companion mesh artifact as GCS/COP
  glass.
- The artifact represents at least three vehicle-local SemLink companions.
- Selected summaries synchronize across the SemLink mesh while raw MAVLink
  frames remain local-only by default.

The demo must not claim:

- live BlueOS, Navigator, hardware, radio, or SITL fidelity;
- one SemLink runtime multiplexing multiple vehicles;
- native MAVLink transmit, companion hardware transmit, mission upload, mode
  change, arm/disarm, or offboard control authority;
- CS API or SemConnect in the SemLink hot path.

## Proof Strategy

Use two layers:

1. A fast SemOps parser/provider/view-model smoke over a SemLink-generated
   artifact fixture.
2. A browser or component smoke that proves the COP surface exposes the mesh
   posture and no-transmit/source-fidelity labels an operator needs.

The generated artifact used for release evidence should carry a real SemLink
commit or version and generator command. Static fixtures may exist for tests,
but shared demo evidence should be generated from the SemLink repo.
