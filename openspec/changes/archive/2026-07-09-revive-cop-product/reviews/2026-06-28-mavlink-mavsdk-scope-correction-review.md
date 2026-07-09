# MAVLink MAVSDK Scope Correction Review

Date: 2026-06-28
Scope: Task 5.96 and the boundary between read-side MAVLink telemetry parity and future MAVSDK/offboard authority.

## Verdict

Accept the scope correction: MAVSDK/offboard is not required to close read-side MAVLink telemetry parity.

SemOps has separate pass evidence for real PX4 MAVLink telemetry and real ArduPilot MAVLink telemetry through the
hosted UDP component and COP snapshot. MAVSDK can emit MAVLink, but it does not define a third telemetry wire protocol
for the feed-validation lane. Save MAVSDK/offboard for the full command/control lane, where mode, setpoint, local
override, ACK/task, and post-state semantics are the actual claim.

## Findings

1. MAVSDK should not be used as a synthetic telemetry family.

   Treating MAVSDK output as another telemetry parity blocker would duplicate the MAVLink wire evidence already covered
   by PX4 and ArduPilot. The useful MAVSDK proof is not "can SemOps parse MAVLink again"; it is "can SemOps safely
   reason about offboard/control behavior."

2. The fail-closed helper remains useful.

   `mavsdk-offboard-stack` still stamps `simulator_family=mavsdk` and blocks unless an explicit MAVSDK/offboard source
   is available. That preserves a guard for the future command/control lane without making it an immediate telemetry
   blocker.

3. Current telemetry parity remains narrow.

   This correction does not claim mission upload, mode change, arm/disarm, offboard setpoints, hardware behavior,
   serial/TCP transport, or signed-link behavior.

## Verification

- Documentation/spec scope only; no new telemetry run was needed or claimed.
- Existing pass evidence remains PX4/Gazebo telemetry, ArduPilot SITL telemetry, and PX4 read-side command readback.
