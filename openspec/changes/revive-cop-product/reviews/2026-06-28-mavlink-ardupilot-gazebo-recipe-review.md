# MAVLink ArduPilot Gazebo Recipe Review

Date: 2026-06-28
Scope: SemOps-owned `docker/ardupilot-gazebo-headless/` image recipe for the ArduPilot parity lane.

## Verdict

Accept the image recipe as opt-in setup infrastructure, but keep ArduPilot telemetry parity open.

The recipe gives SemOps a controlled alternative to low-signal public ArduPilot/Gazebo images. It pins the upstream
ArduPilot and `ardupilot_gazebo` refs checked during this slice, builds the official plugin, and launches the official
Iris Gazebo frame through headless `gz sim` plus `sim_vehicle.py`. It still must be built and pass `ardupilot-stack`
before task 5.95 can close.

## Findings

1. The recipe is deterministic enough to review.

   The Dockerfile defaults to `ardupilot/ardupilot-dev-base:v0.2.0`, ArduPilot
   `918718f6b063cca9a60de3921c3dcee2e8ca3524`, `ardupilot_gazebo`
   `082a0fe231f6e63bc8d1598f1cba461d9e2ea7f5`, and Gazebo Harmonic. Operators can override those build args, but the
   default path is no longer "latest at build time".

2. The launch command follows the official Gazebo plugin contract.

   The entrypoint starts `gz sim -s -r iris_runway.sdf`, exports the plugin and resource paths, then runs
   `sim_vehicle.py -v ArduCopter -f gazebo-iris --model JSON --out=udp:semops:14550` by default.

3. This does not weaken the evidence gate.

   `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE` is still required, and the docs require the wrapper command to be
   named explicitly. The existing `ardupilot-stack` smoke still has to observe real ArduPilot MAVLink telemetry
   through the hosted SemOps UDP component and COP snapshot.

## Verification

- `bash -n scripts/mavlink-sitl-gate.sh docker/ardupilot-gazebo-headless/entrypoint.sh`
- `openspec validate revive-cop-product --strict`

Attempted but blocked locally:

- `docker buildx build --check -f docker/ardupilot-gazebo-headless/Dockerfile docker/ardupilot-gazebo-headless`
  failed before parsing the build with Docker Desktop containerd snapshot storage reporting `no space left on device`.

The image was not built in this slice. A full build will download and compile ArduPilot, Gazebo dependencies, and the
plugin, so it should be run as the next explicit runtime evidence step after Docker storage is available.
