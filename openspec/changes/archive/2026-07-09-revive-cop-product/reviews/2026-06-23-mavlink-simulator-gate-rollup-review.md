# MAVLink Simulator Gate Rollup Review

Date: 2026-06-23
Scope: splitting the stale bundled MAVLink parser/generator/ArduPilot/PX4 evidence gate after PX4/Gazebo telemetry
passes

## Decision

Accept the MAVLink parser/generator and PX4/Gazebo telemetry evidence as closed for the MVP feed ladder. Keep
ArduPilot parity, MAVSDK/offboard parity, and live command/control as explicit open gates.

The old `5.4` task bundled unlike risks: codec correctness, simulator telemetry, simulator-family parity, and command
authority. That made it too easy to either keep proven work looking open or, worse, let one PX4 telemetry smoke imply
more than it proved. The new task split preserves the proof while keeping the remaining risks visible.

## Red-Team Findings

1. PX4 telemetry is not all MAVLink interoperability.

   The PX4/Gazebo headless smoke proves the hosted SemOps UDP component can ingest simulator-produced telemetry and
   surface it through the COP snapshot. It does not prove ArduPilot-specific modes, MAVSDK offboard behavior, serial or
   TCP links, signed links, or hardware-adjacent behavior.

2. Motion evidence is still telemetry evidence.

   The motion-required pass is stronger than a static heartbeat/position smoke, but it still observes state. It does
   not arm, upload missions, command modes, validate actuator behavior, or reconcile desired-vs-actual task success.

3. Command ACK readback is not command authority.

   COMMAND_ACK projection and command-intent lifecycle fixtures are valuable governed-control evidence. They remain
   readback and desired-state evidence until a reviewed native transmitter, safety interlocks, cancellation behavior,
   local override, and post-command state polling exist.

## Accepted Evidence

- `go test ./pkg/adapters/mavlink` covers parser/generator behavior against real bytes and command codec frames.
- `scripts/mavlink-sitl-gate.sh` provides guarded preflight, focused, stack, and PX4/Gazebo headless stack modes.
- The PX4/Gazebo headless Docker lane passed the one-command stack smoke on 2026-06-23.
- The same lane passed with `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true` on 2026-06-23.

## Follow-Ups

- Run explicit ArduPilot SITL telemetry parity before claiming ArduPilot simulator interoperability.
- Run MAVSDK/PX4 offboard parity before claiming MAVSDK command/control or offboard interoperability.
- Add a safe live MAVLink command/control simulator gate with ACK and post-command state polling before native command
  authority claims.
