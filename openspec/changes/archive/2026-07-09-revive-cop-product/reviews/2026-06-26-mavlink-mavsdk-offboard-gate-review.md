# MAVLink MAVSDK/Offboard Gate Review

Date: 2026-06-26
Scope: `scripts/mavlink-sitl-gate.sh` MAVSDK/PX4 offboard parity gate before any MAVSDK command/control or offboard
interoperability claim.

## Verdict

Accept an explicit fail-closed `mavsdk-offboard-stack` mode as readiness evidence only. Do not close MAVSDK/offboard
parity.

The gate now makes MAVSDK/offboard evidence distinct from the already-passing PX4/Gazebo telemetry lane. The local run
blocked because this laptop has no `mavsdk_server` and no MAVSDK Docker image, even though the PX4 image exists.

## Findings

1. MAVSDK/offboard parity must not inherit PX4 telemetry evidence.

   `mavsdk-offboard-stack` stamps `simulator_family=mavsdk` and rejects contradictory family input. A raw PX4 track in
   the COP snapshot is still telemetry evidence, not offboard-control parity.

2. Motion should be required by default.

   The mode defaults `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true`. The first MAVSDK/offboard claim should prove
   observed movement through the hosted UDP component path, not just connection metadata.

3. This is not command authority.

   Even when the future gate passes, it should prove an offboard route plus COP telemetry readback. Native command
   transmit, local override, ACK/status correlation, mission execution, and CS API tasking remain under separate
   command-control gates.

## Required Before Claim

- Install or run a real MAVSDK/offboard route, with `mavsdk_server` or an explicitly reviewed MAVSDK-family container.
- Ensure the associated PX4 or simulator telemetry is routed to SemOps UDP `127.0.0.1:14550` while the offboard route
  is active.
- Run `SEMOPS_MAVLINK_SITL_GATE_MODE=mavsdk-offboard-stack bash scripts/mavlink-sitl-gate.sh`.
- Preserve the ignored evidence file path, simulator/offboard route label, version, command, MAVSDK route, SemOps
  commit, motion requirement, result, and unresolved limitations in the review/docs.
- Keep native command/control, hardware, serial/TCP transport, signed-link behavior, and formal conformance claims
  behind separate reviews.
