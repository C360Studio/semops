# MAVLink PX4 Compose Network Route Review

Date: 2026-06-27

Scope: `px4-headless-stack` route reliability for the hosted COP external SITL telemetry gate.

## Decision

Accept the managed Compose-network route as the default PX4/Gazebo headless Docker path for the SemOps stack gate.

The earlier host-targeted run used `host.docker.internal` for both PX4 QGC and API targets, but the hosted COP snapshot
continued to show only the deterministic system `42` fixture track. Starting the PX4 container on `semops-cop_default`
and targeting the Compose service alias `semops` delivered the external PX4 system `1` telemetry through the hosted
SemOps MAVLink UDP component.

Keep live command/control open. The same route hardening is sufficient for telemetry, but a follow-up
`command-live-sim` attempt did not produce native command readback evidence through the hosted MAVLink chain.

## Evidence

- `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack
  SEMOPS_MAVLINK_SITL_PX4_HOST_QGC=host.docker.internal
  SEMOPS_MAVLINK_SITL_PX4_HOST_API=host.docker.internal
  SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=60s
  SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true bash scripts/mavlink-sitl-gate.sh` failed. The snapshot endpoint was
  reachable during diagnosis, but it only exposed system `42`; the PX4 system `1` track did not arrive.
- Manually replacing PX4 with
  `docker run -d --rm --name semops-px4-gazebo-headless --network semops-cop_default
  jonasvautherin/px4-gazebo-headless:1.17.0 -v gz_x500 -w default semops semops` made
  `TestExternalSITLTelemetryCOPSnapshot` pass.
- The helper now starts PX4 through `SEMOPS_COP_SMOKE_BEFORE_MAVLINK_SITL_CMD`, after baseline hosted COP and live graph
  smokes. Starting PX4 earlier made the scenario-runner snapshot smoke time out under local load.
- The helper now stops PX4 through `SEMOPS_COP_SMOKE_BEFORE_CLEANUP_CMD`, before Compose removes
  `semops-cop_default`.
- Final verification:
  `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=90s
  SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true bash scripts/mavlink-sitl-gate.sh`.
- Result: `passed`; evidence file:
  `tmp/mavlink-sitl-evidence/2026-06-27T20-12-34Z-px4-headless-stack.env`.
- `TestExternalSITLTelemetryCOPSnapshot` passed in 0.52s after the PX4 boot wait, and the Compose network was removed
  cleanly after the before-cleanup hook stopped `semops-px4-gazebo-headless`.
- A kept-stack diagnostic run also confirmed the hosted SemOps container published both MAVLink UDP ports: primary
  `14550` and extra/offboard `14540`.

## Command Follow-Up

After the telemetry pass, `command-live-sim` was rerun against the same local PX4/Gazebo headless image. The command
helper was expanded with two diagnostic options: send a GCS heartbeat before the read-side `AUTOPILOT_VERSION` request,
and forward any simulator replies observed by the helper to a SemOps UDP listener.

Attempts covered host-published PX4 endpoints and an in-network helper route from the Compose network. Baseline PX4
telemetry stayed visible in the COP snapshot, but the helper reported `forwarded_replies=0`, the command-control
snapshot smoke did not find the expected `mavlink.command_ack` task, and a filtered decoded-stream subscriber observed
no command-related MAVLink frames (`COMMAND_LONG`, `COMMAND_ACK`, or `AUTOPILOT_VERSION`) entering the hosted MAVLink
component chain.

Latest failed command evidence:
`tmp/mavlink-sitl-evidence/2026-06-27T20-31-09Z-command-live-sim.env`, with
`result=failed_command_control_smoke`.

A follow-up transmitter-helper change added route learning from `semops.feed.mavlink.raw`. The helper can now subscribe
to NATS, filter raw MAVLink BaseMessages for the target system, and use the payload `remote_addr` as the simulator UDP
destination before sending the reviewed read-side command. This removed the remaining guessed-port variable from the
PX4 command attempt.

Learned-route command evidence:
`tmp/mavlink-sitl-evidence/2026-06-27T21-21-20Z-command-live-sim.env`, with
`result=failed_command_control_smoke`, and transmitter log
`tmp/mavlink-sitl-evidence/2026-06-27T21-21-20Z-command-live-sim-transmitter.log`. The transmitter output learned
`172.19.0.9:14580` for PX4 `system-1`, sent the heartbeat and `AUTOPILOT_VERSION` request to that route from inside
`semops-cop_default`, and still reported `forwarded_replies=0`. The COP snapshot after the run remained healthy but
still contained no `mavlink.command_ack` task.

A follow-up native command-session diagnostic kept the helper SDK-free while adding bounded retries, shared sender
sequence, direct reply parsing, and `COMMAND_LONG.confirmation` increments. It used the same learned route, target
component `0`, and three attempts.

Native retry command evidence:
`tmp/mavlink-sitl-evidence/2026-06-27T21-47-42Z-command-live-sim.env`, with
`result=failed_command_control_smoke`, and transmitter log
`tmp/mavlink-sitl-evidence/2026-06-27T21-47-42Z-command-live-sim-transmitter.log`. The transmitter output recorded
`confirmation=0`, `confirmation=1`, `confirmation=2`, then `command_attempts=3`, `direct_command_acks=0`,
`direct_autopilot_version_frames=0`, and `forwarded_replies=0`. A fresh snapshot after the failed smoke was healthy
and still had no `mavlink.command_ack` task.

## Boundary

This closes a route reliability gap for PX4 telemetry evidence. It does not close ArduPilot parity, MAVSDK/offboard
parity, live command/control, mission execution, serial/TCP transport, signed links, or hardware safety evidence. The
next command-control slice needs a simulator endpoint or command-driver path that actually emits native command reply
frames into the hosted MAVLink input -> decoder -> projector chain.
