# MAVLink Command Baseline Telemetry Review

Date: 2026-06-27
Scope: `scripts/mavlink-sitl-gate.sh` `command-live-sim` safety posture after a local PX4/Gazebo command-gate attempt.

## Decision

Accept a pre-transmit baseline telemetry guard for `command-live-sim`.

The live command simulator gate must prove the named MAVLink track is visible through the hosted COP snapshot before it
runs any reviewed simulator transmitter command. If baseline telemetry is stale or missing, the gate writes blocked
evidence and exits before transmit.

## Evidence

- `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack bash scripts/mavlink-sitl-gate.sh` exited `2` with
  `result=blocked_no_local_simulator`; no `sim_vehicle.py` or ArduPilot/ArduCopter Docker image was present.
- `SEMOPS_MAVLINK_SITL_GATE_MODE=mavsdk-offboard-stack bash scripts/mavlink-sitl-gate.sh` exited `2` with
  `result=blocked_no_local_simulator`; no `mavsdk_server` or MAVSDK-family Docker image was present.
- `SEMOPS_MAVLINK_SITL_KEEP_SIMULATOR=true SEMOPS_COP_KEEP_STACK=true
  SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack bash scripts/mavlink-sitl-gate.sh` exited `1` with
  `result=failed` in `tmp/mavlink-sitl-evidence/2026-06-27T19-25-02Z-px4-headless-stack.env`; the local COP snapshot
  remained on scenario MAVLink `system-42` and never observed expected PX4 SITL track `system-1`.
- Post-patch `command-live-sim` with a two-second timeout and `SEMOPS_MAVLINK_COMMAND_TRANSMITTER="echo
  SHOULD_NOT_RUN"` exited `1` with `result=blocked_command_baseline_telemetry`. The dummy transmitter output was
  absent, proving baseline telemetry failure stops before transmit.

## Findings

1. ArduPilot and MAVSDK/offboard parity remain correctly blocked.
   The fail-closed family-specific lanes still prevent PX4 evidence from being reused as ArduPilot or MAVSDK parity.

2. The live command gate should not transmit into an unproven telemetry path.
   The prior `command-live-sim` flow validated safety attestations and transmitter review before transmit, then polled
   for ACK and post-command state. Today showed an earlier failure mode: the simulator/COP telemetry path itself can
   be stale or missing. Sending even an MVP read-side command before proving that path is unnecessary risk.

3. Baseline telemetry preflight is product safety, not command success evidence.
   The new preflight reuses `TestExternalSITLTelemetryCOPSnapshot` with the command post-state track ID and one required
   update. Passing that preflight only permits the transmitter step; task `5.97` remains open until `command-live-sim`
   observes graph-visible `COMMAND_ACK` task evidence and post-command track refresh after a real transmitter run.

## Required Follow-Up

- Rerun the PX4/Gazebo telemetry gate and fix the local host-route issue before trying the transmitter path again.
- Run `command-live-sim` with `cmd/semops-mavlink-command` only after baseline PX4 track `system-1` is live in the COP
  snapshot.
- Keep ArduPilot SITL and MAVSDK/offboard parity open until their family-specific sources pass.
