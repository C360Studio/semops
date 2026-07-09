# MAVLink PX4 Headless Docker Review

Date: 2026-06-23

Scope: selection and helper wiring for a PX4/Gazebo headless Docker SITL lane for the SemOps MAVLink external
simulator smoke.

## Finding

The `jonasvautherin/px4-gazebo-headless:1.17.0` image is a better dev/demo lane than the official PX4 build container
for the immediate SemOps smoke because it is a runnable PX4/Gazebo headless simulator. It also aligns with the hosted
SemOps UDP listener: the upstream README documents host MAVLink targets on UDP `14550` and `14540`, and the entrypoint
uses `HEADLESS=1` with default vehicle `gz_x500`.

## Guardrails

- The image is unofficial. Passing this gate is simulator-fidelity telemetry evidence, not official PX4 conformance.
- The image is large, so `scripts/mavlink-sitl-gate.sh` must not pull it unless the operator sets
  `SEMOPS_MAVLINK_SITL_DOCKER_PULL=true`.
- The helper should stop only the simulator container it started and should leave already-running simulator containers
  alone unless explicitly told to replace or keep them.
- `4.7` and `5.4` remain open until a pulled image run produces SemOps evidence through the hosted COP snapshot.

## Outcome

Accept the third-party headless image as the preferred MVP PX4 SITL path and keep the official PX4 Docker docs as the
upstream build/reference path. Revisit if PX4 publishes the planned official `px4-sim` image or if the third-party image
fails on SemOps' macOS Docker Desktop path.
