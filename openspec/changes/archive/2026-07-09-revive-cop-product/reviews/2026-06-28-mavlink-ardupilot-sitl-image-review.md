# MAVLink ArduPilot SITL Image Review

Date: 2026-06-28
Scope: SemOps-owned `docker/ardupilot-sitl/` image recipe for the first ArduPilot parity lane.

## Verdict

Accept the SITL-only Linux image as the first task 5.95 ArduPilot telemetry parity proof, while keeping
ArduPilot/Gazebo as later physics-rich evidence.

The recipe keeps the first proof focused on SemOps' real boundary: ArduPilot code emits MAVLink into the hosted UDP
component, and the COP snapshot observer sees family-stamped ArduPilot telemetry. Gazebo can add physical scene
credibility later, but it should not be required before the ArduPilot-family telemetry lane passes the hosted stack.

## Findings

1. The image avoids unnecessary first-proof weight.

   The canceled Gazebo build had already resolved an install plan of 830 packages, about 439 MB of archives, and about
   2 GB installed. That cost may be justified for physics evidence, but it is not the narrow blocker for 5.95.

2. The recipe is still reviewed and pinned.

   The Dockerfile packages the official ArduCopter `V4.8.0-dev` Linux SITL binary for ArduPilot
   `918718f6b063cca9a60de3921c3dcee2e8ca3524`, declares the amd64 runtime platform explicitly, verifies SHA-256
   `1646845efcf96b196ad2f39f186a5b9dd325877b938245e7374d2dbc8a29c9bd`, and pins MAVProxy `1.8.74`.

3. The evidence gate remained unchanged and passed.

   `ardupilot-stack` starts the reviewed image, routes ArduPilot MAVLink to `semops:14550` through MAVProxy, observes
   `simulator_family=ardupilot`, requires motion, and passes the hosted COP snapshot smoke before ArduPilot telemetry
   parity closes.

## Verification

- Passed: `bash -n scripts/mavlink-sitl-gate.sh docker/ardupilot-sitl/entrypoint.sh
  docker/ardupilot-gazebo-headless/entrypoint.sh`
- Passed: `docker buildx build --check --platform linux/amd64 -f docker/ardupilot-sitl/Dockerfile
  docker/ardupilot-sitl`
- Passed: `docker build --platform linux/amd64 --progress=plain -f docker/ardupilot-sitl/Dockerfile -t
  c360studio/semops-ardupilot-sitl:local docker/ardupilot-sitl`
- Passed: `openspec validate revive-cop-product --strict`
- Passed: `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack
  SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE=c360studio/semops-ardupilot-sitl:local
  SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_COMMAND=/usr/local/bin/semops-ardupilot-sitl
  SEMOPS_MAVLINK_SITL_ARDUPILOT_BOOT_WAIT=45 SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=90s
  bash scripts/mavlink-sitl-gate.sh`

Recorded evidence: `tmp/mavlink-sitl-evidence/2026-06-28T12-48-33Z-ardupilot-stack.env` with `result=passed`,
`simulator_family=ardupilot`, `ardupilot_docker_platform=linux/amd64`, and `require_motion=true`.
