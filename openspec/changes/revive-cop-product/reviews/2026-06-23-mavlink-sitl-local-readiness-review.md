# MAVLink SITL Local Readiness Review

Date: 2026-06-23

## Decision

Do not close the MAVLink PX4/MAVSDK/SITL evidence gates from the current laptop state.

The external SITL smoke harness exists and is correctly guarded, but no local PX4, MAVSDK, ArduPilot, or equivalent
simulator runtime was available to feed the hosted SemOps UDP component. The current evidence remains parser,
projector, component, live generated-frame graph, and observer-harness readiness evidence only.

## Evidence Checked

- `px4` was not present on PATH.
- `mavsdk_server` was not present on PATH.
- `sim_vehicle.py` was not present on PATH.
- Docker was available, but local Docker images did not include PX4, MAVSDK, ArduPilot, or ArduCopter simulator
  images.
- `go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v` skipped because
  `SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL` was unset.
- `go test ./pkg/adapters/mavlink ./internal/projectors/mavlink ./internal/adapters/mavlink ./internal/components/mavlink ./internal/stack`
  passed.

## Red-Team Findings

1. A skipped smoke is not simulator evidence.

   The skip proves the test cannot accidentally pass without a configured COP snapshot. It does not prove PX4,
   MAVSDK, ArduPilot, or hardware-adjacent behavior.

2. Local Docker availability is not enough.

   Docker is present, but no simulator image is available locally. Pulling/building PX4 or ArduPilot would be a
   separate environment setup task and should not happen silently in the feed evidence gate.

3. Generated MAVLink remains useful but separate.

   SemOps generated-frame graph evidence is still the right deterministic regression path for SemStreams governance,
   owner tokens, and born-first graph writes. It must not be relabeled as simulator fidelity.

## Future Pass Requirements

Record the following before closing tasks `4.7` or `5.4`:

- Simulator family and version: PX4 SITL, MAVSDK route, ArduPilot SITL, or named equivalent.
- Launch command and UDP route to the SemOps host port.
- MAVLink system ID and expected COP track ID.
- SemOps commit and stack-smoke command.
- Whether `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true` was used.
- Pass/fail result and any simulator limitations.
- Explicit statement that command/control, mission state, safe ACK handling, serial/TCP transports, signed links, and
  hardware behavior remain separate gates unless tested.

## Verification

- `go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v`
- `go test ./pkg/adapters/mavlink ./internal/projectors/mavlink ./internal/adapters/mavlink ./internal/components/mavlink ./internal/stack`
- `openspec validate revive-cop-product --strict`
- `git diff --check`
