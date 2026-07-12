# Design

## Boundary

The MVP deployment model is:

- one vehicle has one nearby SemLink companion boundary;
- that companion reports local MAVLink evidence and safe readback intent;
- SemOps consumes many companion outputs and renders the fleet COP.

SemOps may receive those outputs directly, through an authenticated gateway, from generated artifacts, or through a
future standards projection. In all cases, SemOps owns the fleet-level merge and the authority posture.

## Multi-Vehicle Inputs

Multiple MAVLink vehicles behind one network edge are still possible, but they should not become the default SemLink
runtime obligation. If a gateway or aggregator multiplexes several companions, the SemOps ingress boundary must retain
per-companion provenance and target identity before writing COP state.

If a SemLink report has `vehicle_count > 1`, SemOps can preserve that evidence, but it should label the proof as an
aggregator or stress profile unless the artifact also carries a reviewed profile that says multi-vehicle SemLink is the
intended runtime mode.

## Target Identity

SemLink does not send `target_asset_id` in the v0 readback request. SemOps derives canonical targets from the
authenticated companion boundary, MAVLink target system/component IDs, configured source mapping, and born-target graph
state. This keeps SemLink from owning COP asset proof and keeps SemOps from conflating equal MAVLink system IDs that
arrive from different companion boundaries.

## Non-Goals

- Requiring SemLink to support multiple external MAVLink ports per runtime.
- Promoting CS API into the SemLink-to-SemOps hot path.
- Claiming radio, BlueOS, Navigator, or live hardware reliability from deterministic mesh evidence.
