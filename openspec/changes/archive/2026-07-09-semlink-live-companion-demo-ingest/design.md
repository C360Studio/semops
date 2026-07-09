## Context

SemLink has an ArduRover SITL lane that runs without Gazebo and routes MAVLink UDP into SemLink. SemOps has already
archived a deterministic companion-fleet consumer proof. The gap is the handoff seam between those two facts:

- SemLink should run companion/SITL/demo workflows and emit report artifacts.
- SemOps should ingest those artifacts and render COP/GCS glass.
- The first live proof should add one SITL-backed companion node, not force an `n`-SITL fleet before the demo has a
  stable handoff.

## Goals / Non-Goals

**Goals**

- Prove SemOps can ingest SemLink-generated report artifacts, not only committed SemOps fixtures.
- Support a mixed demo where at least one SemLink node is backed by ArduPilot SITL telemetry and at least three
  SemLink nodes remain visible/inspectable in the COP.
- Preserve source fidelity labels: deterministic, generated, SITL-backed, fixture, and future hardware evidence must
  not collapse into one vague "live" state.
- Keep command/readback posture visible while preserving no native transmit and no companion hardware transmit
  authority.
- Record when the demo claim would require `n` SITL instances.

**Non-Goals**

- SemOps does not start, stop, or supervise SemLink, ArduPilot SITL, BlueOS, Navigator, or mesh peers.
- SemOps does not require SemConnect or CS API in the SemLink-to-SemOps hot path.
- SemOps does not expose mission upload, mode change, arm/disarm, offboard control, native transmit, peer management,
  BlueOS controls, or mesh topology editing.
- The first live handoff demo does not claim `n` live ArduPilot vehicles unless there are `n` distinct SITL/hardware
  vehicle sources with distinct MAVLink identities and routes.

## Decisions

### 1. Artifact ingest before runtime orchestration

The SemOps-owned slice should accept a SemLink-generated report artifact from a file, CLI input, or equivalent local
handoff. This keeps the first proof deterministic enough for CI/browser smoke while still using SemLink's output shape.
Runtime orchestration, compose wiring, and SITL startup stay SemLink-owned until a later cross-repo demo harness needs
to coordinate both products.

### 2. One SITL-backed node is enough for the first ArduPilot fidelity proof

The first SemOps proof should require exactly one SITL-backed SemLink node in the mixed demo. That proves the important
compatibility boundary: ArduPilot SITL emits MAVLink, SemLink consumes/project evidence, and SemOps renders the
result. Requiring `n` SITLs would add port, identity, CPU, and orchestration complexity before the demo needs it.

### 3. N-node fleet proof can remain mixed

The SemOps fleet proof remains `n >= 3` companion nodes. Those nodes may be deterministic/generated mesh evidence, and
one node may be marked SITL-backed. The UI must make that split visible. The demo may say "three SemLink companion
nodes, one SITL-backed ArduPilot lane" but must not say "three live ArduPilot vehicles" unless each node is backed by
its own live vehicle source.

### 4. N SITLs require explicit identities and routes

If a later demo requires `n` live ArduPilot vehicles, each SemLink node should have a paired SITL or hardware-adjacent
source with distinct MAVLink system IDs and UDP routes. In Compose terms, the intended topology is private-network
pairs such as `ardupilot-sitl-alpha -> semlink-alpha`, `ardupilot-sitl-bravo -> semlink-bravo`, and so on. SemOps
should consume the resulting artifacts; it should not infer per-node live fidelity from mesh summaries alone.

### 5. Command scope remains observe/readback only

SITL does not change command authority. The SemOps view may show readback adapter status, `COMMAND_ACK`, and
`AUTOPILOT_VERSION` result evidence when SemLink emits them. It must still state that native execution and companion
hardware transmit are not authorized by the COP demo.

## Risks / Trade-offs

- A generated report path can drift from SemLink unless the smoke consumes actual SemLink output or a fixture copied
  directly from SemLink's current e2e shape. The mitigation is to label fixtures and generated artifacts separately and
  add a small import validation layer.
- SITL startup can be slow or host-dependent. The mitigation is to keep the first SemOps smoke artifact-based and let
  SemLink own the SITL readiness lane.
- A mixed demo can be oversold. The mitigation is explicit UI/source labels and a spec rule that `n` live ArduPilot
  vehicles require `n` live vehicle sources.

## Implementation Order

1. Add SemOps import metadata and validation for SemLink-generated companion demo artifacts.
2. Extend the companion-fleet view model with source-fidelity labels and per-node live-source posture.
3. Add a generated-artifact smoke fixture copied from or produced by SemLink's e2e/SITL lane.
4. Add browser smoke proving the SemOps COP displays generated SemLink artifact evidence, including one SITL-backed
   node and at least three total companion nodes.
5. Add review notes that the first proof does not claim `n` live ArduPilot vehicles or command authority.
