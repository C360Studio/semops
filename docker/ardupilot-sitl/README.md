# SemOps ArduPilot SITL Image

This directory contains the preferred first SemOps-owned image recipe for ArduPilot parity. It packages the official
ArduCopter Linux SITL binary and MAVProxy inside a Linux container and runs them without Gazebo, so the first proof
stays focused on the service boundary that matters for SemOps: ArduPilot MAVLink telemetry entering the hosted UDP
component and appearing in the COP snapshot.

Use `docker/ardupilot-gazebo-headless/` later when the claim needs Gazebo physics, the official `ardupilot_gazebo`
plugin, or source compilation evidence. Do not require Gazebo just to prove ArduPilot-family telemetry parity.

Current default refs:

- ArduPilot: `918718f6b063cca9a60de3921c3dcee2e8ca3524`
- Firmware version: `ArduCopter V4.8.0-dev`
- Binary URL: `https://firmware.ardupilot.org/Copter/latest/SITL_x86_64_linux_gnu/arducopter`
- Binary SHA-256: `1646845efcf96b196ad2f39f186a5b9dd325877b938245e7374d2dbc8a29c9bd`
- Base image: `ubuntu:22.04`
- Base image platform: `linux/amd64`

Build the image explicitly:

```bash
docker build \
  --platform linux/amd64 \
  -f docker/ardupilot-sitl/Dockerfile \
  -t c360studio/semops-ardupilot-sitl:local \
  docker/ardupilot-sitl
```

Run the SemOps stack gate with the image:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack \
SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE=c360studio/semops-ardupilot-sitl:local \
SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_COMMAND=/usr/local/bin/semops-ardupilot-sitl \
SEMOPS_MAVLINK_SITL_ARDUPILOT_BOOT_WAIT=45 \
bash scripts/mavlink-sitl-gate.sh
```

Useful runtime knobs:

- `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_ROUTE`: default `semops:14550`; passed to MAVProxy as `--out`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_PLATFORM`: default `linux/amd64` in the SemOps gate helper.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_VEHICLE`: default `ArduCopter`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_MODEL`: default `quad`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_HOME`: default `-35.363261,149.165230,584,353`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_SPEEDUP`: default `1`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_SYSID`: default `1`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_EXTRA_ARGS`: optional extra `arducopter` args.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_SIM_VEHICLE_EXTRA_ARGS`: backwards-compatible alias for extra args.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_MASTER`: default `tcp:127.0.0.1:5760`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_SITL`: default `127.0.0.1:5501`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_EXTRA_ARGS`: optional extra `mavproxy.py` args; the wrapper already adds
  `--non-interactive --nowait` for detached container runs.

Local 2026-06-28 evidence closed the first ArduPilot telemetry parity gate with this image:
`tmp/mavlink-sitl-evidence/2026-06-28T12-48-33Z-ardupilot-stack.env` recorded `result=passed`,
`simulator_family=ardupilot`, `require_motion=true`, Docker platform `linux/amd64`, route `semops:14550`, and the
hosted COP snapshot URL.
