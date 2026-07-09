# MAVLink MVP Command Scope Review

Scope: `cmd/semops-mavlink-command`, MAVLink command constants, and the command-live-sim MVP transmitter shape.

## Decision

Accept a single read-side simulator command as the MVP command-control proof target.

The MVP helper may send `MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION` only. It exists to prove a reviewed command
frame can be sent to a simulator and correlated back through MAVLink `COMMAND_ACK` graph state. It is not a ground
control station, mission uploader, mode controller, or vehicle authority layer.

## Findings

1. Read-side priority is preserved.
   The helper requests `AUTOPILOT_VERSION` rather than arming, changing modes, uploading missions, or driving offboard
   control. That makes the command path a validation hook for ACK/readback while the demo remains feed-ingest first.

2. Simulator-only confirmation is enforced at the helper boundary.
   `semops-mavlink-command` refuses to build or send a frame unless `-confirm-simulator-only` or
   `SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED=true` is present.

3. The helper does not close live command/control by itself.
   Dry-run proves the frame shape and expected ACK suffix only. Task 5.97 remains open until `command-live-sim` runs
   against a named simulator and COP stack, then observes the ACK task and post-command track refresh.

4. Route semantics are intentionally destination-only.
   The helper rejects listen-style routes such as `udp://:14540` and requires a destination `host:port`, which avoids
   confusing MAVSDK server bind syntax with a transmitter target.

## Evidence

- `go test ./cmd/semops-mavlink-command ./pkg/adapters/mavlink`
- `go run ./cmd/semops-mavlink-command -confirm-simulator-only -dry-run -route udp://127.0.0.1:14540`
- Dry-run metadata: `action=request_autopilot_version`, `command=512`, `request_message=148`,
  `expected_ack_task_suffix=system-1-command-512-target-255-190`

## Required Follow-Up

- Run `command-live-sim` against a live PX4/Gazebo stack using the helper as `SEMOPS_MAVLINK_COMMAND_TRANSMITTER`.
- Preserve the evidence file and COP snapshot result before closing task 5.97.
- Keep mode control, arm/disarm, mission upload, offboard control, command priority, TTL, and authority arbitration as
  post-MVP gates.
