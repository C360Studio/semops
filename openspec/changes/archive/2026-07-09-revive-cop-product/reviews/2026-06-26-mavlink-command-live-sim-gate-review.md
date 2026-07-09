# MAVLink Command Live Simulator Gate Review

Scope: `scripts/mavlink-sitl-gate.sh` command-control simulator mode and
`TestCommandControlSimulatorGateCOPSnapshot`.

## Decision

Accept the `command-live-sim` gate as fail-closed simulator command-control readiness evidence.

Do not close task 5.97 or claim live command authority until a real reviewed transmitter command runs against a named
simulator and the ACK/post-command COP snapshot smoke passes.

## Findings

1. The live simulator gate is still external-transmitter based.
   The gate can run a reviewed transmitter command, but SemOps does not yet own a native MAVLink command transmitter.
   This is the right boundary for the current slice because it keeps command write authority outside the adapter until
   safety interlocks and operator workflows are reviewed.

2. The gate refuses to pass on private transmitter success.
   `TestCommandControlSimulatorGateCOPSnapshot` polls `GET /api/cop/snapshot` and requires a MAVLink-owned
   `mavlink.command_ack` task plus post-command MAVLink track refresh after the command start timestamp. This keeps
   acceptance tied to operator-visible graph state rather than shell output.

3. Hardware claims remain blocked.
   `command-live-sim` rejects `simulator_family=hardware`, requires a simulator-scoped safety profile, and requires
   local override, abort readiness, ACK requirement, and post-state polling attestations before it can execute a
   transmitter command.

4. The local verification is readiness-gap evidence only.
   The 2026-06-26 local run provided simulator, safety, ACK, post-state, reviewed-transmitter, and transmit-enabled
   attestations, then blocked with `result=blocked_missing_command_transmitter`. That proves the missing-transmitter
   guard, not command execution.

## Evidence

- `bash -n scripts/mavlink-sitl-gate.sh`
- `go test ./internal/smoke/mavlink`
- `SEMOPS_MAVLINK_SITL_GATE_MODE=command-live-sim ... bash scripts/mavlink-sitl-gate.sh` exited `2` with
  `result=blocked_missing_command_transmitter`

## Required Follow-Up

- Provide a reviewed simulator transmitter command for a safe MAVLink action such as hold/loiter or an intentionally
  disallowed arm command.
- Run `command-live-sim` against a running simulator and COP stack, preserving the evidence file.
- Keep ArduPilot, MAVSDK/offboard, mission upload, hardware, serial/TCP transport, signed-link security, and command
  priority/TTL reconciliation as separate gates.
