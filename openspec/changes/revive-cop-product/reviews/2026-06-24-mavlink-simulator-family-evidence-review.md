# MAVLink Simulator Family Evidence Review

Date: 2026-06-24
Scope: `scripts/mavlink-sitl-gate.sh` evidence stamping for PX4, ArduPilot, MAVSDK, hardware, and other MAVLink
simulator-family runs.

## Verdict

Accept simulator-family evidence stamping as a necessary guard before ArduPilot, MAVSDK/offboard, hardware, or
command-control parity claims.

This does not close the ArduPilot SITL, MAVSDK/offboard, or live command-control evidence gates. It prevents the
current PX4/Gazebo telemetry pass from being reused as if those gates were satisfied.

## Findings

1. Name-only simulator attestation is too weak.
   A free-form simulator name can describe a run, but it cannot reliably drive parity decisions later. Focused and
   stack modes now require `SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY` with one of `px4`, `ardupilot`, `mavsdk`,
   `hardware`, or `other`.

2. Tooling readiness must match the declared family.
   An ArduPilot-stamped run should not pass the local readiness gate just because a PX4 image exists on the laptop.
   The helper now checks local tooling by declared family unless the run explicitly opts into a remote-source path.

3. PX4 headless mode must stamp its own lane.
   `px4-headless-stack` now writes `simulator_family=px4` automatically and rejects contradictory family values. That
   keeps the helper convenient while keeping the evidence precise.

4. Family stamping is not command authority.
   The stamp says which family produced telemetry observed through the hosted SemOps component path. It does not prove
   offboard behavior, mode semantics, ACK reconciliation, mission success, safety interlocks, or native command
   transmit.

## Boundaries

- PX4 telemetry evidence remains PX4 telemetry evidence only.
- ArduPilot parity still requires an ArduPilot SITL run feeding the hosted UDP component path.
- MAVSDK/offboard parity still requires a MAVSDK/offboard run and separate command-control review.
- Hardware-adjacent evidence must use `hardware` and still needs safety and operations review before product claims.

## Follow-Ups

- Run the ArduPilot SITL telemetry gate and record family, vehicle type, version, launch command, UDP route, and motion
  requirement before closing task 5.95.
- Run a MAVSDK/offboard parity gate before closing task 5.96.
- Design the safe live command/control gate with ACK and post-command state polling before closing task 5.97.
