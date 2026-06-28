#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -gt 0 ]]; then
  exec "$@"
fi

vehicle="${SEMOPS_MAVLINK_SITL_ARDUPILOT_VEHICLE:-${ARDUPILOT_VEHICLE:-ArduCopter}}"
model="${SEMOPS_MAVLINK_SITL_ARDUPILOT_MODEL:-${ARDUPILOT_MODEL:-quad}}"
route="${SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_ROUTE:-${ARDUPILOT_ROUTE:-semops:14550}}"
home="${SEMOPS_MAVLINK_SITL_ARDUPILOT_HOME:-${ARDUPILOT_HOME:--35.363261,149.165230,584,353}}"
speedup="${SEMOPS_MAVLINK_SITL_ARDUPILOT_SPEEDUP:-${ARDUPILOT_SPEEDUP:-1}}"
sysid="${SEMOPS_MAVLINK_SITL_ARDUPILOT_SYSID:-${ARDUPILOT_SYSID:-1}}"
extra_args="${SEMOPS_MAVLINK_SITL_ARDUPILOT_EXTRA_ARGS:-${SEMOPS_MAVLINK_SITL_ARDUPILOT_SIM_VEHICLE_EXTRA_ARGS:-${ARDUPILOT_SIM_VEHICLE_EXTRA_ARGS:-}}}"
mavproxy_master="${SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_MASTER:-${ARDUPILOT_MAVPROXY_MASTER:-tcp:127.0.0.1:5760}}"
mavproxy_sitl="${SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_SITL:-${ARDUPILOT_MAVPROXY_SITL:-127.0.0.1:5501}}"
mavproxy_extra_args="${SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_EXTRA_ARGS:-${ARDUPILOT_MAVPROXY_EXTRA_ARGS:-}}"
serial0="${SEMOPS_MAVLINK_SITL_ARDUPILOT_SERIAL0:-${ARDUPILOT_SERIAL0:-}}"

if [[ "$vehicle" != "ArduCopter" ]]; then
  echo "This image packages the official ArduCopter SITL binary; got vehicle=${vehicle}." >&2
  exit 2
fi

if [[ "$route" == udpclient:* ]]; then
  out_arg="${route#udpclient:}"
elif [[ "$route" == udp:* ]]; then
  out_arg="${route#udp:}"
else
  out_arg="${route}"
fi

vehicle_args=(--model "$model" --home "$home" --speedup "$speedup" --sysid "$sysid")
if [[ -n "$serial0" ]]; then
  vehicle_args+=(--serial0 "$serial0")
fi
if [[ -n "$extra_args" ]]; then
  # shellcheck disable=SC2206
  vehicle_args+=($extra_args)
fi

mavproxy_args=(--master "$mavproxy_master" --sitl "$mavproxy_sitl" --out "$out_arg" --non-interactive --nowait)
if [[ -n "$mavproxy_extra_args" ]]; then
  # shellcheck disable=SC2206
  mavproxy_args+=($mavproxy_extra_args)
fi

vehicle_pid=""
mavproxy_pid=""

cleanup() {
  if [[ -n "${mavproxy_pid}" ]] && kill -0 "$mavproxy_pid" >/dev/null 2>&1; then
    kill "$mavproxy_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "${vehicle_pid}" ]] && kill -0 "$vehicle_pid" >/dev/null 2>&1; then
    kill "$vehicle_pid" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

echo "Starting ArduPilot SITL: vehicle=${vehicle} model=${model} mavproxy_out=${out_arg}"
/usr/local/bin/arducopter "${vehicle_args[@]}" &
vehicle_pid="$!"

sleep "${SEMOPS_MAVLINK_SITL_ARDUPILOT_MAVPROXY_BOOT_WAIT:-${ARDUPILOT_MAVPROXY_BOOT_WAIT:-2}}"

echo "Starting MAVProxy: master=${mavproxy_master} sitl=${mavproxy_sitl} out=${out_arg}"
mavproxy.py "${mavproxy_args[@]}" &
mavproxy_pid="$!"

wait "$mavproxy_pid"
