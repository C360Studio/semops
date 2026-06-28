# SemOps ArduPilot/Gazebo Headless Image

This directory contains the SemOps-owned image recipe for the ArduPilot parity lane. It is intentionally opt-in: the
MAVLink SITL gate still requires `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE`, and task 5.95 stays open until this
image or another reviewed image passes `ardupilot-stack`.

The recipe follows the official `ArduPilot/ardupilot_gazebo` shape:

- Ubuntu 22.04 plus Gazebo Harmonic.
- Pinned ArduPilot checkout.
- Pinned official `ardupilot_gazebo` plugin checkout.
- Headless Gazebo server for `iris_runway.sdf`.
- `sim_vehicle.py -v ArduCopter -f gazebo-iris --model JSON --out=udp:semops:14550`.

Current default refs:

- ArduPilot: `918718f6b063cca9a60de3921c3dcee2e8ca3524`
- `ardupilot_gazebo`: `082a0fe231f6e63bc8d1598f1cba461d9e2ea7f5`
- Base image: `ardupilot/ardupilot-dev-base:v0.2.0`
- Gazebo package set: `gz-harmonic`

Build the image explicitly:

```bash
docker build \
  -f docker/ardupilot-gazebo-headless/Dockerfile \
  -t c360studio/semops-ardupilot-gazebo-headless:local \
  docker/ardupilot-gazebo-headless
```

Run the SemOps stack gate with the image:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack \
SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE=c360studio/semops-ardupilot-gazebo-headless:local \
SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_COMMAND=/usr/local/bin/semops-ardupilot-gazebo-headless \
SEMOPS_MAVLINK_SITL_ARDUPILOT_BOOT_WAIT=45 \
bash scripts/mavlink-sitl-gate.sh
```

Useful runtime knobs:

- `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_ROUTE`: default `semops:14550`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_VEHICLE`: default `ArduCopter`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_FRAME`: default `gazebo-iris`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_MODEL`: default `JSON`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_GAZEBO_WORLD`: default `iris_runway.sdf`.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_GAZEBO_BOOT_WAIT`: default `8`, in seconds.
- `SEMOPS_MAVLINK_SITL_ARDUPILOT_SIM_VEHICLE_EXTRA_ARGS`: optional extra `sim_vehicle.py` args.

Do not publish or promote this image from a local build alone. First capture the image digest, launch command, route,
version evidence, and passing `tmp/mavlink-sitl-evidence/*-ardupilot-stack.env` file.
