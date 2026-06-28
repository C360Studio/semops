# MAVLink External SITL Smoke

This smoke is the first SemOps simulator-fidelity gate for MAVLink telemetry. It proves an external PX4, MAVSDK,
ArduPilot SITL, or equivalent MAVLink source can feed the hosted SemOps UDP component and appear in the COP snapshot.

It does not prove command/control, mission handling, serial/TCP transport, signed links, hardware behavior, or PX4
conformance.

## Prerequisites

- A simulator or MAVSDK route that emits MAVLink heartbeat and `GLOBAL_POSITION_INT` telemetry to the hosted SemOps
  UDP component. The Compose default listens on `:14550` and also on `:14540` for PX4 offboard/MAVSDK-style return
  traffic.
- The simulator's first vehicle should use MAVLink system ID `1`, or `SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID`
  should be set to the expected graph track ID.
- Docker resources sufficient to run the COP stack.

## Preferred PX4 Docker Path

For the dev/demo lane, use `jonasvautherin/px4-gazebo-headless:1.17.0`. It is an Apache-2.0, unofficial
PX4/Gazebo headless image that packages a runnable simulator instead of only a PX4 build toolchain. The upstream
README says the current supported PX4 release is `v1.17.0`, the default vehicle is `gz_x500`, and the container can
send MAVLink to a named peer on UDP `14550` for QGroundControl and `14540` for offboard/MAVSDK-style clients.

The SemOps managed stack path defaults to `SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE=compose-network`. In that mode the
helper starts the COP stack first, starts PX4 on the Compose network `semops-cop_default`, points both PX4 target
arguments at the `semops` service alias, and stops PX4 before Compose teardown. This avoids relying on Docker Desktop
UDP host-port hairpin behavior for the simulator-to-SemOps path.

The hosted SemOps runtime can bind multiple MAVLink UDP listeners. The COP Compose stack defaults to
`SEMOPS_MAVLINK_UDP_LISTEN_ADDR=:14550` and `SEMOPS_MAVLINK_UDP_EXTRA_LISTEN_ADDRS=:14540`, publishing both UDP ports
to the host. That lets PX4's QGroundControl-style and offboard/MAVSDK-style telemetry return paths enter the same
MAVLink input -> decoder -> projector component chain.

The SemOps helper keeps the pull opt-in because the image is large:

```bash
docker pull jonasvautherin/px4-gazebo-headless:1.17.0
```

Then run the full SemOps stack smoke with the simulator container managed by the gate helper:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
bash scripts/mavlink-sitl-gate.sh
```

If the image is not local and you deliberately want the helper to pull it:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
SEMOPS_MAVLINK_SITL_DOCKER_PULL=true \
bash scripts/mavlink-sitl-gate.sh
```

Useful optional knobs:

- `SEMOPS_MAVLINK_SITL_DOCKER_IMAGE`: default `jonasvautherin/px4-gazebo-headless:1.17.0`.
- `SEMOPS_MAVLINK_SITL_DOCKER_CONTAINER`: default `semops-px4-gazebo-headless`.
- `SEMOPS_MAVLINK_SITL_PX4_VEHICLE`: default `gz_x500`.
- `SEMOPS_MAVLINK_SITL_PX4_WORLD`: default `default`.
- `SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE`: default `compose-network` unless explicit PX4 host targets are provided; use
  `host` for the older host-port route.
- `SEMOPS_MAVLINK_SITL_DOCKER_NETWORK`: default `semops-cop_default`.
- `SEMOPS_MAVLINK_SITL_PX4_NETWORK_TARGET`: default `semops`.
- `SEMOPS_MAVLINK_SITL_PX4_BOOT_WAIT`: default `20`, in seconds.
- `SEMOPS_MAVLINK_SITL_KEEP_SIMULATOR`: set `true` to leave the PX4 container running after the smoke. In
  `compose-network` mode this also keeps the COP stack alive so the Docker network remains available.
- `SEMOPS_MAVLINK_SITL_DOCKER_REPLACE`: set `true` to remove a stopped container with the configured name.
- `SEMOPS_MAVLINK_SITL_PX4_HOST_API`: optional explicit target IP for PX4 UDP `14540`.
- `SEMOPS_MAVLINK_SITL_PX4_HOST_QGC`: optional explicit target IP for PX4 UDP `14550`; requires
  `SEMOPS_MAVLINK_SITL_PX4_HOST_API`.
- `SEMOPS_MAVLINK_UDP_EXTRA_LISTEN_ADDRS`: extra hosted SemOps MAVLink UDP listeners; Compose defaults to `:14540`.
- `SEMOPS_MAVLINK_UDP_OFFBOARD_HOST_PORT`: host UDP port published for the default extra listener; default `14540`.

This is the fastest path to simulator-fidelity telemetry evidence. It is not official PX4 conformance and should not
be used for command authority claims without a separate reviewed command/ACK/state gate.

The PX4 helper stamps evidence with `simulator_family=px4`. That PX4 evidence must not be used to close ArduPilot,
MAVSDK/offboard, hardware, mission/offboard command authority, or broader command-control parity gates.

The official PX4 Docker docs remain useful as the upstream build/reference path. As of the 2026-06-23 check, PX4
documents `px4io/px4-dev:<version>` as the recommended build container and says a dedicated `px4-sim` image is planned.
Older `px4io/px4-dev-simulation-*` images still exist but are no longer the recommended path.

Latest local evidence:

- 2026-06-23: `SEMOPS_MAVLINK_SITL_DOCKER_PULL=true SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack bash
  scripts/mavlink-sitl-gate.sh` passed against `jonasvautherin/px4-gazebo-headless:1.17.0`, vehicle `gz_x500`, world
  `default`, with `TestExternalSITLTelemetryCOPSnapshot` observing the PX4-owned MAVLink track through
  `http://127.0.0.1:8080/api/cop/snapshot` in 23.52s. The smoke required two updates and did not require motion.
- 2026-06-23: `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack
  SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=60s bash
  scripts/mavlink-sitl-gate.sh` passed against the same local image and route, with
  `TestExternalSITLTelemetryCOPSnapshot` observing the required position delta in 25.52s.
- 2026-06-27: `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack
  SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=90s SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true bash
  scripts/mavlink-sitl-gate.sh` passed with the managed `compose-network` route. The helper started PX4 as
  `docker run -d --rm --name semops-px4-gazebo-headless --network semops-cop_default
  jonasvautherin/px4-gazebo-headless:1.17.0 -v gz_x500 -w default semops semops`, `TestExternalSITLTelemetryCOPSnapshot`
  passed in 0.52s after the boot wait, and the before-cleanup hook stopped PX4 before Compose removed the network.
- 2026-06-27: a kept-stack diagnostic pass confirmed the hosted SemOps container publishes both UDP `14550` and
  `14540`, and `TestExternalSITLTelemetryCOPSnapshot` still passed against the PX4 `system-1` track with the dual
  listener runtime enabled.

## Local Readiness Preflight

Before treating a laptop as ready for this gate, check for a real simulator path rather than relying on the SemOps
generated-frame smoke:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=preflight \
bash scripts/mavlink-sitl-gate.sh
```

The smoke should skip when `SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL` is unset. That skip proves the test is guarded; it
does not prove simulator fidelity. If no PX4, MAVSDK, ArduPilot, or equivalent simulator runtime is available on the
host or in local Docker images, do not run the stack gate and do not close the PX4/MAVSDK evidence task.

The helper writes a local ignored evidence file under `tmp/mavlink-sitl-evidence/`.

Focused and stack modes also require `SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY`. Use one of `px4`, `ardupilot`,
`mavsdk`, `hardware`, or `other`. The family is written to the evidence file so one simulator-family pass cannot be
reused as another family or as command-control authority.

## One-Command Stack Gate

Start the simulator first, or keep it ready to emit telemetry while the stack starts. Then run:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=stack \
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL <version>" \
SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4 \
SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND="<simulator launch command>" \
bash scripts/mavlink-sitl-gate.sh
```

The script still runs the deterministic generated-frame graph smoke for system `42`. The external SITL smoke observes
system `1` through `GET /api/cop/snapshot` and does not inject its own MAVLink frames.

For stricter motion evidence:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=stack \
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL <version>" \
SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4 \
SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND="<simulator launch command>" \
SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true \
bash scripts/mavlink-sitl-gate.sh
```

ArduPilot telemetry parity uses the same hosted SemOps path, but has a dedicated gate mode so the evidence file is
unambiguously stamped as ArduPilot. The mode defaults to vehicle `ArduCopter`, defaults to motion-required telemetry,
and requires `sim_vehicle.py`, an ArduPilot/ArduCopter Docker image, or an explicit remote-source override before it
will run the stack smoke:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack \
bash scripts/mavlink-sitl-gate.sh
```

Useful ArduPilot knobs:

- `SEMOPS_MAVLINK_SITL_ARDUPILOT_VEHICLE`: default `ArduCopter`.
- `SEMOPS_MAVLINK_SITL_SIMULATOR_NAME`: optional explicit simulator/version label.
- `SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND`: optional explicit launch command; default is
  `sim_vehicle.py -v <vehicle> --out=udp:127.0.0.1:14550`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE`: optional reviewed ArduPilot-family image to start as a managed
  Compose-network source. There is no default image.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_COMMAND`: optional command to run inside that image; default is
  `sim_vehicle.py -v <vehicle> --out=udp:semops:14550`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_PULL=true`: opt in only after reviewing the image. Without this, a missing
  configured image blocks before the stack smoke.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_BOOT_WAIT`: default `20`.
- `SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true`: allowed only when an ArduPilot source is already routing MAVLink to
  SemOps from outside the local PATH/Docker environment.

For a reviewed ArduPilot image that already contains `sim_vehicle.py`, the managed path is:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack \
SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE=<reviewed-ardupilot-image> \
bash scripts/mavlink-sitl-gate.sh
```

The helper starts that container on the SemOps Compose network before the external MAVLink SITL smoke, routes the
default `sim_vehicle.py` output to `semops:14550`, and stops the container during cleanup unless
`SEMOPS_MAVLINK_SITL_KEEP_SIMULATOR=true`.

Current local result: on 2026-06-28T00:15:11Z UTC, `ardupilot-stack` blocked with
`result=blocked_no_local_simulator`. The laptop had the PX4/Gazebo headless image, but no `sim_vehicle.py` and no
ArduPilot/ArduCopter Docker image. Evidence:
`tmp/mavlink-sitl-evidence/2026-06-28T00-15-11Z-ardupilot-stack.env`. That is readiness-gap evidence only; it does not
close ArduPilot parity.

MAVSDK/offboard parity is also separate from raw PX4 telemetry. Use the dedicated offboard lane so the evidence is
stamped as `mavsdk`, defaults to motion-required telemetry, and stays separate from the raw PX4/Gazebo telemetry
helper:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=mavsdk-offboard-stack \
bash scripts/mavlink-sitl-gate.sh
```

Useful MAVSDK/offboard knobs:

- `SEMOPS_MAVLINK_SITL_MAVSDK_OFFBOARD_ROUTE`: default `udp://:14540`.
- `SEMOPS_MAVLINK_SITL_SIMULATOR_NAME`: optional explicit MAVSDK/offboard route label.
- `SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND`: optional explicit launch command; default is
  `mavsdk_server udp://:14540`.
- `SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true`: allowed only when a MAVSDK/offboard route is already active and the
  associated MAVLink telemetry is already routing to SemOps.

Current local result: on 2026-06-28T00:15:13Z UTC, `mavsdk-offboard-stack` blocked with
`result=blocked_no_local_simulator`. The laptop had the PX4/Gazebo headless image, but no `mavsdk_server` and no
MAVSDK Docker image. Evidence:
`tmp/mavlink-sitl-evidence/2026-06-28T00-15-13Z-mavsdk-offboard-stack.env`. That is readiness-gap evidence only; it
does not close MAVSDK/offboard parity.

## Focused Smoke Against A Running Stack

If the COP stack is already running and a simulator is emitting telemetry:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=focused \
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL <version>" \
SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4 \
SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL=http://127.0.0.1:8080/api/cop/snapshot \
SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID=c360.edge-compose.cop.mavlink.track.system-1 \
bash scripts/mavlink-sitl-gate.sh
```

Useful optional knobs:

- `SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT`: default `2m`.
- `SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES`: default `2`.
- `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION`: default `false`.
- `SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE`: set to `true` only when the simulator or hardware-adjacent source runs
  outside the local PATH/Docker environment but is already routing MAVLink to SemOps.
- `SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack`: starts the preferred PX4/Gazebo headless container and then runs
  the full stack gate.

## Command-Control Gates

SemOps keeps command-control as a separate simulator evidence path. The non-transmitting `command-preflight` mode
records the minimum safety posture and exits blocked before any native transmit can happen:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=command-preflight \
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL command preflight" \
SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4 \
SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true \
SEMOPS_MAVLINK_COMMAND_TARGET_ID=c360.edge-compose.cop.mavlink.track.system-1 \
SEMOPS_MAVLINK_COMMAND_ACTION=hold_position \
SEMOPS_MAVLINK_COMMAND_SAFETY_PROFILE=simulator_local_operator \
SEMOPS_MAVLINK_COMMAND_LOCAL_OVERRIDE_CONFIRMED=true \
SEMOPS_MAVLINK_COMMAND_ACK_REQUIRED=true \
SEMOPS_MAVLINK_COMMAND_POST_STATE_POLL_REQUIRED=true \
bash scripts/mavlink-sitl-gate.sh
```

The expected result is an evidence file with `result=blocked_no_native_command_transmitter` and exit status `2`.
Missing simulator family, target, action, safety profile, local override, ACK requirement, or post-state polling
requirement also blocks before the gate can be cited. If `SEMOPS_MAVLINK_COMMAND_TRANSMITTER` is set or
`SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED=true`, this mode also blocks because preflight is non-transmitting by design.
Use `command-live-sim` for the reviewed simulator-transmit path.

The transmitting simulator gate is `command-live-sim`. It assumes the COP stack and simulator are already running,
first polls `GET /api/cop/snapshot` for the named MAVLink track as baseline live-telemetry evidence, runs an
explicitly reviewed simulator transmitter command only after that preflight passes, and then polls the same snapshot
until both the MAVLink `COMMAND_ACK` task and the post-command MAVLink track refresh are visible:

```bash
COMMAND_TX="go run ./cmd/semops-mavlink-command -confirm-simulator-only -send-heartbeat-first -route 127.0.0.1:<simulator-command-port>"

SEMOPS_MAVLINK_SITL_GATE_MODE=command-live-sim \
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL command smoke" \
SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4 \
SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true \
SEMOPS_MAVLINK_COMMAND_TARGET_ID=c360.edge-compose.cop.mavlink.track.system-1 \
SEMOPS_MAVLINK_COMMAND_ACTION=request_autopilot_version \
SEMOPS_MAVLINK_COMMAND_SAFETY_PROFILE=simulator_local_operator \
SEMOPS_MAVLINK_COMMAND_LOCAL_OVERRIDE_CONFIRMED=true \
SEMOPS_MAVLINK_COMMAND_ACK_REQUIRED=true \
SEMOPS_MAVLINK_COMMAND_POST_STATE_POLL_REQUIRED=true \
SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED=true \
SEMOPS_MAVLINK_COMMAND_ABORT_READY=true \
SEMOPS_MAVLINK_COMMAND_TRANSMITTER_REVIEWED=true \
SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED=true \
SEMOPS_MAVLINK_COMMAND_TRANSMITTER="$COMMAND_TX" \
SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_TASK_ID=c360.edge-compose.cop.mavlink.task.system-1-command-512-target-255-190 \
SEMOPS_MAVLINK_COMMAND_POST_STATE_TRACK_ID=c360.edge-compose.cop.mavlink.track.system-1 \
bash scripts/mavlink-sitl-gate.sh
```

`command-live-sim` refuses `simulator_family=hardware`, requires a simulator-scoped safety profile, and writes blocked
evidence unless the transmitter command is explicitly reviewed and transmit is explicitly enabled. Before transmit, it
reuses `TestExternalSITLTelemetryCOPSnapshot` with the post-state track ID and one required update so stale or missing
baseline telemetry blocks before any simulator command runs. After transmit, the acceptance test is
`TestCommandControlSimulatorGateCOPSnapshot`; it is skipped unless `SEMOPS_MAVLINK_COMMAND_SMOKE_SNAPSHOT_URL` is set
by the helper. The test requires the ACK task to be source `mavlink`, kind `mavlink.command_ack`, owned by
`semops.feed.mavlink`, non-stale relative to the command start timestamp, and in the expected status set (`accepted`
by default). It also requires the named MAVLink track to refresh after command start and can require motion with
`SEMOPS_MAVLINK_COMMAND_POST_STATE_REQUIRE_MOTION=true`.

Current pause, 2026-06-26: `command-live-sim` must not use scenario-runner direct graph state as its ACK or
post-command evidence. A passing command gate requires the simulator telemetry and `COMMAND_ACK` task to enter through
the hosted MAVLink input -> decoder -> projector component chain, with component health/flow and Prometheus evidence
available through the COP stack. Owner-token mismatch warnings between the hosted SemOps runtime and any scenario
runner or helper process fail the product command evidence even if the expected task or track eventually appears in
the graph. Direct graph smokes remain valid for MAVLink projection-contract coverage only.

For MVP, keep the allowlist narrow. The provided `semops-mavlink-command` helper only sends
`MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION`; it is a read-side command used to prove command ACK/readback through
the COP graph, not mission execution or vehicle control. The helper can optionally send a GCS heartbeat first and can
forward any simulator replies it receives to a SemOps UDP listener for diagnostics. It remains a native Go helper; it
does not embed MAVSDK. The useful pattern borrowed from mature MAVLink clients is command-session discipline: stable
sender identity, heartbeat, bounded retries, incrementing `COMMAND_LONG.confirmation`, and explicit reply observation:

```bash
go run ./cmd/semops-mavlink-command \
  -confirm-simulator-only \
  -send-heartbeat-first \
  -route 127.0.0.1:<simulator-command-port> \
  -forward-replies-to 127.0.0.1:14540 \
  -attempts 3 \
  -retry-interval 500ms \
  -reply-timeout 2s
```

The command `-route` is the simulator command endpoint, not the SemOps telemetry listener. Its dry-run should print:

```bash
go run ./cmd/semops-mavlink-command -confirm-simulator-only -dry-run -route udp://127.0.0.1:14540
```

Expected metadata includes `action=request_autopilot_version`, `command=512`, `request_message=148`, and
`expected_ack_task_suffix=system-1-command-512-target-255-190`. Live attempts print `command_attempts`,
`direct_command_acks`, `direct_autopilot_version_frames`, and `forwarded_replies` counters. PX4's configured partner
route may send ACKs to the SemOps UDP listener rather than the helper socket, so the helper also supports raw-lane
observation counters with `-observe-raw-nats-url`. In that PX4 topology, `raw_command_acks` and the graph-visible
`mavlink.command_ack` task are the acceptance signal; `direct_command_acks=0` is not by itself a failure.

The helper can also learn a simulator command route from live raw telemetry before it sends. This is intended for
Docker Compose-network PX4 runs where the correct UDP destination is the inbound `remote_addr` on
`semops.feed.mavlink.raw`, not a stable host-published port:

```bash
go run ./cmd/semops-mavlink-command \
  -confirm-simulator-only \
  -send-heartbeat-first \
  -learn-route-nats-url nats://127.0.0.1:4222 \
  -learn-route-timeout 10s \
  -learn-route-port 18570 \
  -observe-raw-nats-url nats://127.0.0.1:4222
```

When `command-live-sim` runs a transmitter, it writes transmitter stdout/stderr to the ignored evidence log named by
`command_transmitter_output_file` in the `.env` evidence file.

Current command-gate result, 2026-06-27: after rebuilding SemOps with MAVLink task prefix discovery, `command-live-sim`
passed against the local PX4/Gazebo headless image. The reviewed native transmitter ran from the SemOps network
namespace, learned the PX4 host from raw telemetry, overrode the destination to the PX4 GCS port `18570`, and observed
PX4 ACKs on the SemOps raw lane. The transmitter log recorded `command_attempts=3`, `direct_command_acks=0`,
`forwarded_replies=0`, `raw_command_acks=3`, and `raw_last_ack_result=accepted`; the command-control COP snapshot
smoke then found `c360.edge-compose.cop.mavlink.task.system-1-command-512-target-255-190` and a fresh post-command
MAVLink track. Evidence:
`tmp/mavlink-sitl-evidence/2026-06-27T22-29-45Z-command-live-sim.env`.

## Acceptance

The smoke requires:

- `feed.mavlink` is live.
- The expected MAVLink track is present with source `mavlink`.
- Position is non-zero.
- Velocity evidence is present.
- Provenance owner is `semops.feed.mavlink`.
- Source reference is non-empty.
- At least two simulator updates are observed.
- If motion is required, latitude or longitude changes between the first and last accepted observations.

## Claim Boundary

Passing the telemetry smoke is simulator telemetry evidence only. A passing `command-live-sim` run adds simulator
command-readback evidence for the reviewed read-side command, but it is still not mission execution authority or
hardware readiness.

For a future pass, record the simulator name and version, launch command, system ID, UDP route, SemOps commit,
simulator family, stack-smoke command, pass/fail result, whether motion was required, and any unresolved simulator
limitations.
