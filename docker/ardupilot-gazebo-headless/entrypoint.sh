#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -gt 0 ]]; then
  exec "$@"
fi

export GZ_SIM_SYSTEM_PLUGIN_PATH="/opt/ardupilot_gazebo/build:${GZ_SIM_SYSTEM_PLUGIN_PATH:-}"
export GZ_SIM_RESOURCE_PATH="/opt/ardupilot_gazebo/models:/opt/ardupilot_gazebo/worlds:${GZ_SIM_RESOURCE_PATH:-}"
export LIBGL_ALWAYS_SOFTWARE="${LIBGL_ALWAYS_SOFTWARE:-1}"
export QT_QPA_PLATFORM="${QT_QPA_PLATFORM:-offscreen}"

vehicle="${SEMOPS_MAVLINK_SITL_ARDUPILOT_VEHICLE:-${ARDUPILOT_VEHICLE:-ArduCopter}}"
frame="${SEMOPS_MAVLINK_SITL_ARDUPILOT_FRAME:-${ARDUPILOT_FRAME:-gazebo-iris}}"
model="${SEMOPS_MAVLINK_SITL_ARDUPILOT_MODEL:-${ARDUPILOT_MODEL:-JSON}}"
route="${SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_ROUTE:-${ARDUPILOT_ROUTE:-semops:14550}}"
world="${SEMOPS_MAVLINK_SITL_ARDUPILOT_GAZEBO_WORLD:-${ARDUPILOT_GAZEBO_WORLD:-iris_runway.sdf}}"
verbosity="${SEMOPS_MAVLINK_SITL_ARDUPILOT_GAZEBO_VERBOSITY:-${ARDUPILOT_GAZEBO_VERBOSITY:-3}}"
gazebo_boot_wait="${SEMOPS_MAVLINK_SITL_ARDUPILOT_GAZEBO_BOOT_WAIT:-${ARDUPILOT_GAZEBO_BOOT_WAIT:-8}}"
extra_args="${SEMOPS_MAVLINK_SITL_ARDUPILOT_SIM_VEHICLE_EXTRA_ARGS:-${ARDUPILOT_SIM_VEHICLE_EXTRA_ARGS:-}}"

if [[ "$route" == udp:* ]]; then
  out_arg="--out=${route}"
else
  out_arg="--out=udp:${route}"
fi

cleanup() {
  if [[ -n "${gazebo_pid:-}" ]] && kill -0 "$gazebo_pid" >/dev/null 2>&1; then
    kill "$gazebo_pid" >/dev/null 2>&1 || true
    wait "$gazebo_pid" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

echo "Starting Gazebo headless server: world=${world}"
gz sim -s -r -v "$verbosity" "$world" &
gazebo_pid="$!"

sleep "$gazebo_boot_wait"
if ! kill -0 "$gazebo_pid" >/dev/null 2>&1; then
  echo "Gazebo exited before ArduPilot SITL could start." >&2
  wait "$gazebo_pid"
fi

sim_args=(-v "$vehicle" -f "$frame" --model "$model" "$out_arg")
if [[ -n "$extra_args" ]]; then
  # shellcheck disable=SC2206
  sim_args+=($extra_args)
fi

echo "Starting ArduPilot SITL: vehicle=${vehicle} frame=${frame} route=${out_arg}"
cd /opt/ardupilot
exec sim_vehicle.py "${sim_args[@]}"
