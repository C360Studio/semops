# MAVLink Command-Control Preflight Review

Date: 2026-06-24
Scope: `scripts/mavlink-sitl-gate.sh` command-control preflight before any live MAVLink native transmitter exists.

## Verdict

Accept a fail-closed command-control preflight as safety-posture evidence only.

This does not close the live MAVLink command/control gate. It records the simulator family, target, intended command
action, safety profile, local override posture, ACK requirement, and post-command state-polling requirement, then
blocks before native transmit.

## Findings

1. A preflight that exits cleanly would be too easy to overclaim.
   The helper exits with blocked evidence because SemOps has command-intent and COMMAND_ACK readback seams, but no
   reviewed native transmitter path. That keeps task 5.97 open.

2. Command evidence needs stricter inputs than telemetry evidence.
   The preflight requires an explicit born-first target, command action, safety profile, local override attestation,
   ACK requirement, and post-command state-polling requirement before it writes command-control evidence.

3. Accidental transmit must fail before evidence can be cited.
   If `SEMOPS_MAVLINK_COMMAND_TRANSMITTER` is set or `SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED=true`, the preflight
   blocks as an unreviewed transmitter rather than treating the configured path as readiness.

4. Simulator-family boundaries still apply.
   The command preflight reuses the simulator attestation gate, so PX4, ArduPilot, MAVSDK/offboard, hardware, and
   other evidence cannot be collapsed into one generic command-control claim.

## Boundaries

- This is not live command transmission.
- This is not mission success, mode-change success, offboard-control parity, or hardware-adjacent authority.
- This does not prove ACK correlation or post-command state polling; it only requires those as future gate
  requirements.
- Native command transmit remains blocked until safety interlocks, stale-command rejection, cancellation,
  supersession, local override, ACK correlation, and post-command state polling are reviewed against a real simulator.

## Follow-Ups

- Build the native MAVLink transmitter behind command-intent arbitration, not as a direct NATS subject shortcut.
- Add a simulator-only command smoke with ACK correlation and post-command state polling before closing task 5.97.
- Keep ArduPilot and MAVSDK/offboard parity separate from any PX4 command smoke.
