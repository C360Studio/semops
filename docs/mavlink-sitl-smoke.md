# MAVLink External SITL Smoke

This smoke is the first SemOps simulator-fidelity gate for MAVLink telemetry. It proves an external PX4, MAVSDK,
ArduPilot SITL, or equivalent MAVLink source can feed the hosted SemOps UDP component and appear in the COP snapshot.

It does not prove command/control, mission handling, serial/TCP transport, signed links, hardware behavior, or PX4
conformance.

## Prerequisites

- A simulator or MAVSDK route that emits MAVLink heartbeat and `GLOBAL_POSITION_INT` telemetry to UDP
  `127.0.0.1:14550`.
- The simulator's first vehicle should use MAVLink system ID `1`, or `SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID`
  should be set to the expected graph track ID.
- Docker resources sufficient to run the COP stack.

## Preferred PX4 Docker Path

For the dev/demo lane, use `jonasvautherin/px4-gazebo-headless:1.17.0`. It is an Apache-2.0, unofficial
PX4/Gazebo headless image that packages a runnable simulator instead of only a PX4 build toolchain. The upstream
README says the current supported PX4 release is `v1.17.0`, the default vehicle is `gz_x500`, and the container sends
MAVLink to the host on UDP `14550` for QGroundControl and `14540` for offboard/MAVSDK-style clients.

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
- `SEMOPS_MAVLINK_SITL_PX4_BOOT_WAIT`: default `20`, in seconds.
- `SEMOPS_MAVLINK_SITL_KEEP_SIMULATOR`: set `true` to leave the PX4 container running after the smoke.
- `SEMOPS_MAVLINK_SITL_DOCKER_REPLACE`: set `true` to remove a stopped container with the configured name.
- `SEMOPS_MAVLINK_SITL_PX4_HOST_API`: optional explicit target IP for PX4 UDP `14540`.
- `SEMOPS_MAVLINK_SITL_PX4_HOST_QGC`: optional explicit target IP for PX4 UDP `14550`; requires
  `SEMOPS_MAVLINK_SITL_PX4_HOST_API`.

This is the fastest path to simulator-fidelity telemetry evidence. It is not official PX4 conformance and should not
be used for command authority claims without a separate reviewed command/ACK/state gate.

The PX4 helper stamps evidence with `simulator_family=px4`. That PX4 evidence must not be used to close ArduPilot,
MAVSDK/offboard, hardware, or command-control parity gates.

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
- `SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true`: allowed only when an ArduPilot source is already routing MAVLink to
  SemOps from outside the local PATH/Docker environment.

Current local result: on 2026-06-26T00:44:17Z UTC, `ardupilot-stack` blocked with
`result=blocked_no_local_simulator`. The laptop had the PX4/Gazebo headless image, but no `sim_vehicle.py` and no
ArduPilot/ArduCopter Docker image. That is readiness-gap evidence only; it does not close ArduPilot parity.

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

Current local result: on 2026-06-26T01:08:56Z UTC, `mavsdk-offboard-stack` blocked with
`result=blocked_no_local_simulator`. The laptop had the PX4/Gazebo headless image, but no `mavsdk_server` and no
MAVSDK Docker image. That is readiness-gap evidence only; it does not close MAVSDK/offboard parity or command/control.

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

## Command-Control Preflight

SemOps intentionally has no live MAVLink command transmitter in this gate yet. The `command-preflight` mode records the
minimum safety posture for a future command-control smoke and exits blocked before any native transmit can happen:

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
`SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED=true`, this mode also blocks because a reviewed native transmitter gate does
not exist yet.

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

Passing this smoke is simulator telemetry evidence only. Command authority and mission semantics need a separate
reviewed gate with safe commands, ACK handling, post-command state polling, and simulator-specific readiness checks.

For a future pass, record the simulator name and version, launch command, system ID, UDP route, SemOps commit,
simulator family, stack-smoke command, pass/fail result, whether motion was required, and any unresolved simulator
limitations.
