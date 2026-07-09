## Why

SemOps now has a deterministic SemLink companion-fleet consumer proof: it can normalize SemLink single-node and
simple-mesh reports, expose a curated companion-fleet view model, and render at least three companion nodes in the COP.

The next demo step should prove the artifact handoff with real SemLink output instead of only committed fixtures. We
also want ArduPilot fidelity, but we should not require `n` ArduPilot SITL instances before the first live handoff
demo. SemLink already owns a no-Gazebo ArduRover SITL lane that feeds SemLink over MAVLink UDP. SemOps should consume
the generated evidence and show it clearly as SITL-backed SemLink evidence, while keeping deterministic mesh evidence
for the `n`-node fleet proof.

## What Changes

- Add a focused OpenSpec change for SemOps ingestion of SemLink-generated live/demo artifacts.
- Require a file/CLI or equivalent import path that accepts SemLink-produced report artifacts without SemOps starting
  or supervising SemLink.
- Require a mixed demo ladder:
  - one SemLink node backed by no-Gazebo ArduRover / ArduPilot SITL telemetry;
  - at least three SemLink companion nodes visible in SemOps through deterministic or generated mesh evidence;
  - explicit labels separating SITL-backed, deterministic, fixture, and hardware evidence.
- Keep `n` SITL instances out of the first acceptance path unless the demo claim says `n` live ArduPilot vehicles.
- Preserve no-transmit posture and keep command expansion out of scope.
- Add browser or stack smoke evidence that SemOps displays a SemLink-generated artifact, not only a checked-in fixture.

## Capabilities

### New Capabilities

- `semlink-live-companion-demo-ingest`: SemOps imports SemLink-generated companion demo/SITL artifacts and renders them
  as COP/GCS glass with explicit source-fidelity labels and no command-authority expansion.

### Related Capabilities

- `semlink-multi-companion-cop-demo`: Supplies the deterministic SemLink report parser, companion-fleet view model,
  and browser fleet-inspection surface.
- `semlink-semops-companion-contract`: Supplies the native SemLink readback contract and MAVLink/SemConnect standards
  boundary.
- `feed-validation-ladder`: Supplies the simulator-fidelity and command-safety ladder for MAVLink and ArduPilot SITL
  evidence.

## Impact

- New SemOps OpenSpec requirements and implementation tasks.
- Follow-up report-ingest path for SemLink-generated artifacts.
- Follow-up evidence metadata in the companion-fleet view model so the UI distinguishes deterministic, generated,
  SITL-backed, and future hardware evidence.
- Follow-up smoke that consumes a SemLink-produced artifact and verifies the COP surface.
