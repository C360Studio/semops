# MAVLink ArduPilot Gazebo Image Review

Date: 2026-06-28
Scope: candidate Docker images for closing ArduPilot SITL telemetry parity task 5.95 through the managed
`ardupilot-stack` path.

## Verdict

Do not pin a default public ArduPilot/Gazebo image yet.

The current registry search found active official ArduPilot CI images and several third-party ArduPilot/Gazebo or
ArduPilot SITL images, but none is a reviewed, runnable, headless ArduPilot/Gazebo equivalent to the already-proven
PX4/Gazebo image. Keep `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE` explicit and leave 5.95 open until either a
SemOps-owned headless image or a deliberately reviewed external image passes `ardupilot-stack`.

## Findings

1. Official ArduPilot Docker Hub images are CI/build bases, not the demo runtime by themselves.

   `ardupilot/ardupilot-dev-base` is described as the ArduPilot base image for CI, and
   `ardupilot/ardupilot-dev-ros` is described as an ArduPilot CI image for ROS2 build. They are useful upstream bases,
   but the registry metadata does not make them a ready-made headless Gazebo telemetry source for SemOps.

2. Public ArduPilot/Gazebo images are not strong enough to become defaults.

   The search found third-party images such as
   `uobflightlabstarling/starling-sim-ardupilot-gazebo`,
   `robocin/ardupilot-sitl-gazebo`, and `zubeyirgenc/ardupilot`. Their metadata is low-signal for a SemOps default:
   stale tags, low pull counts, very large image sizes, empty descriptions, or unclear launch contracts. They may be
   worth a manual trial, but not an implicit pull in the product gate.

3. The safest next path is a SemOps-owned headless image recipe.

   Use the official ArduPilot SITL-with-Gazebo documentation and the `ArduPilot/ardupilot_gazebo` plugin as source
   material, then build a pinned SemOps image with a known ArduPilot commit, Gazebo/plugin version, launch command, and
   UDP route to `semops:14550`. That image should still prove itself by passing the existing observer-only
   `ardupilot-stack` smoke before any ArduPilot interoperability claim is closed.

## Candidate Snapshot

Live Docker Hub metadata checked on 2026-06-28:

- `ardupilot/ardupilot-dev-base`: official-looking ArduPilot namespace, CI base image, active 2026 tags, about
  5.9M pulls, linux/amd64.
- `ardupilot/ardupilot-dev-ros`: official-looking ArduPilot namespace, ROS2 build CI image, active 2026 tags, about
  298K pulls, linux/amd64 plus unknown manifest entries.
- `uobflightlabstarling/starling-sim-ardupilot-gazebo`: ArduPilot/Gazebo name match, last updated 2022, about
  1.9K pulls, linux/amd64.
- `robocin/ardupilot-sitl-gazebo`: ArduPilot/Gazebo name match, last updated 2026, about 400 pulls, roughly 5GB
  latest tag, linux/amd64 plus unknown manifest entries.
- `radarku/ardupilot-sitl`: SITL name match without Gazebo in the image name, mixed old/latest and SHA tags, about
  21.6K pulls, linux/amd64.
- `drnic/ardupilot-sitl`: SITL name match without Gazebo in the image name, last updated 2019, about 205K pulls,
  linux/amd64.
- `zubeyirgenc/ardupilot`: describes "Ardupilot-Mavproxy-Gazbo in one image", last updated 2022, about 100 pulls,
  linux/amd64.

## Sources Checked

- <https://hub.docker.com/r/ardupilot/ardupilot-dev-base>
- <https://hub.docker.com/r/ardupilot/ardupilot-dev-ros>
- <https://hub.docker.com/r/uobflightlabstarling/starling-sim-ardupilot-gazebo>
- <https://hub.docker.com/r/robocin/ardupilot-sitl-gazebo>
- <https://hub.docker.com/r/radarku/ardupilot-sitl>
- <https://hub.docker.com/r/drnic/ardupilot-sitl>
- <https://hub.docker.com/r/zubeyirgenc/ardupilot>
- <https://ardupilot.org/dev/docs/sitl-with-gazebo.html>
- <https://github.com/ArduPilot/ardupilot_gazebo>

## Recommendation

Next implementation slice: add a SemOps-owned Docker target for `semops-ardupilot-gazebo-headless`, or manually trial
one external candidate with an explicit tag, pinned launch command, and
`SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_PULL=true`. Do not make any external candidate the default unless it passes
`ardupilot-stack` and this review is updated with image digest, command, version output, UDP route, and evidence file.
