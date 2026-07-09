# MAVLink ArduPilot Gazebo Recipe Review

Date: 2026-06-28
Scope: SemOps-owned `docker/ardupilot-gazebo-headless/` image recipe for the ArduPilot parity lane.

## Verdict

Accept the image recipe as opt-in setup infrastructure for later Gazebo physics evidence.

The recipe gives SemOps a controlled alternative to low-signal public ArduPilot/Gazebo images. It pins the upstream
ArduPilot and `ardupilot_gazebo` refs checked during this slice, builds the official plugin, and launches the official
Iris Gazebo frame through headless `gz sim` plus `sim_vehicle.py`. The lighter `docker/ardupilot-sitl/` image closed
task 5.95 for hosted ArduPilot telemetry parity; this Gazebo recipe still needs its own build and `ardupilot-stack`
pass before SemOps can claim ArduPilot/Gazebo physics-backed evidence.

## Findings

1. The recipe is deterministic enough to review.

   The Dockerfile defaults to `ardupilot/ardupilot-dev-base:v0.2.0`, ArduPilot
   `918718f6b063cca9a60de3921c3dcee2e8ca3524`, `ardupilot_gazebo`
   `082a0fe231f6e63bc8d1598f1cba461d9e2ea7f5`, and Gazebo Harmonic. Operators can override those build args, but the
   default path is no longer "latest at build time". The recipe also declares `linux/amd64` because the inspected
   ArduPilot base-image tags are amd64-only.

2. The launch command follows the official Gazebo plugin contract.

   The entrypoint starts `gz sim -s -r iris_runway.sdf`, exports the plugin and resource paths, then runs
   `sim_vehicle.py -v ArduCopter -f gazebo-iris --model JSON --out=udp:semops:14550` by default.

3. This does not weaken the evidence gate.

   `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE` is still required, and the docs require the wrapper command to be
   named explicitly. The existing `ardupilot-stack` smoke still has to observe real ArduPilot MAVLink telemetry
   through the hosted SemOps UDP component and COP snapshot.

## Verification

- `bash -n scripts/mavlink-sitl-gate.sh docker/ardupilot-gazebo-headless/entrypoint.sh`
- `docker buildx build --check --platform linux/amd64 -f docker/ardupilot-gazebo-headless/Dockerfile
  docker/ardupilot-gazebo-headless`
- `openspec validate revive-cop-product --strict`

Attempted but intentionally stopped locally:

- `docker build --platform linux/amd64 --progress=plain -f docker/ardupilot-gazebo-headless/Dockerfile -t
  c360studio/semops-ardupilot-gazebo-headless:local docker/ardupilot-gazebo-headless`
  reached the Gazebo dependency install plan of 830 packages, about 439 MB of archives, and about 2 GB installed before
  the run was canceled in favor of the SITL-only ArduPilot lane.

The Gazebo image was not built in this slice. Keep it as the later physics-rich path after the lighter ArduPilot SITL
image proved hosted UDP/COP snapshot telemetry parity.
