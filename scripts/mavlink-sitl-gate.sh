#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${SEMOPS_MAVLINK_SITL_GATE_MODE:-preflight}"
EVIDENCE_DIR="${SEMOPS_MAVLINK_SITL_EVIDENCE_DIR:-$ROOT/tmp/mavlink-sitl-evidence}"
EVIDENCE_STAMP="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
EVIDENCE_FILE="${SEMOPS_MAVLINK_SITL_EVIDENCE_FILE:-$EVIDENCE_DIR/${EVIDENCE_STAMP}-${MODE}.env}"

SIMULATOR_NAME="${SEMOPS_MAVLINK_SITL_SIMULATOR_NAME:-}"
SIMULATOR_FAMILY="${SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY:-}"
SIMULATOR_VERSION="${SEMOPS_MAVLINK_SITL_SIMULATOR_VERSION:-}"
SIMULATOR_COMMAND="${SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND:-}"
SIMULATOR_ROUTE="${SEMOPS_MAVLINK_SITL_SIMULATOR_UDP_ROUTE:-127.0.0.1:${SEMOPS_MAVLINK_UDP_HOST_PORT:-14550}}"
ALLOW_REMOTE_SOURCE="${SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE:-false}"

PX4_HEADLESS_IMAGE="${SEMOPS_MAVLINK_SITL_DOCKER_IMAGE:-jonasvautherin/px4-gazebo-headless:1.17.0}"
PX4_HEADLESS_CONTAINER="${SEMOPS_MAVLINK_SITL_DOCKER_CONTAINER:-semops-px4-gazebo-headless}"
PX4_HEADLESS_VEHICLE="${SEMOPS_MAVLINK_SITL_PX4_VEHICLE:-gz_x500}"
PX4_HEADLESS_WORLD="${SEMOPS_MAVLINK_SITL_PX4_WORLD:-default}"
PX4_HEADLESS_HOST_QGC="${SEMOPS_MAVLINK_SITL_PX4_HOST_QGC:-}"
PX4_HEADLESS_HOST_API="${SEMOPS_MAVLINK_SITL_PX4_HOST_API:-}"
if [[ -n "${SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE:-}" ]]; then
  PX4_HEADLESS_ROUTE_MODE="$SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE"
elif [[ -n "$PX4_HEADLESS_HOST_QGC" || -n "$PX4_HEADLESS_HOST_API" ]]; then
  PX4_HEADLESS_ROUTE_MODE="host"
else
  PX4_HEADLESS_ROUTE_MODE="compose-network"
fi
PX4_HEADLESS_DOCKER_NETWORK="${SEMOPS_MAVLINK_SITL_DOCKER_NETWORK:-${SEMOPS_COP_PROJECT:-semops-cop}_default}"
PX4_HEADLESS_NETWORK_TARGET="${SEMOPS_MAVLINK_SITL_PX4_NETWORK_TARGET:-semops}"
PX4_HEADLESS_BOOT_WAIT="${SEMOPS_MAVLINK_SITL_PX4_BOOT_WAIT:-20}"
PX4_HEADLESS_PULL="${SEMOPS_MAVLINK_SITL_DOCKER_PULL:-false}"
PX4_HEADLESS_REPLACE="${SEMOPS_MAVLINK_SITL_DOCKER_REPLACE:-false}"
PX4_HEADLESS_KEEP="${SEMOPS_MAVLINK_SITL_KEEP_SIMULATOR:-false}"
PX4_HEADLESS_STARTED=false

ARDUPILOT_VEHICLE="${SEMOPS_MAVLINK_SITL_ARDUPILOT_VEHICLE:-ArduCopter}"
MAVSDK_OFFBOARD_ROUTE="${SEMOPS_MAVLINK_SITL_MAVSDK_OFFBOARD_ROUTE:-udp://:14540}"

DEFAULT_SNAPSHOT_URL="http://127.0.0.1:${SEMOPS_CADDY_HOST_PORT:-8080}/api/cop/snapshot"
SNAPSHOT_URL="${SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL:-$DEFAULT_SNAPSHOT_URL}"
EXPECTED_TRACK_ID="${SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID:-c360.edge-compose.cop.mavlink.track.system-1}"
TIMEOUT="${SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT:-2m}"
MIN_UPDATES="${SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES:-2}"
REQUIRE_MOTION="${SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION:-false}"

COMMAND_TARGET_ID="${SEMOPS_MAVLINK_COMMAND_TARGET_ID:-}"
COMMAND_ACTION="${SEMOPS_MAVLINK_COMMAND_ACTION:-}"
COMMAND_SAFETY_PROFILE="${SEMOPS_MAVLINK_COMMAND_SAFETY_PROFILE:-}"
COMMAND_LOCAL_OVERRIDE_CONFIRMED="${SEMOPS_MAVLINK_COMMAND_LOCAL_OVERRIDE_CONFIRMED:-false}"
COMMAND_ACK_REQUIRED="${SEMOPS_MAVLINK_COMMAND_ACK_REQUIRED:-false}"
COMMAND_POST_STATE_POLL_REQUIRED="${SEMOPS_MAVLINK_COMMAND_POST_STATE_POLL_REQUIRED:-false}"
COMMAND_TRANSMITTER="${SEMOPS_MAVLINK_COMMAND_TRANSMITTER:-}"
COMMAND_TRANSMIT_ENABLED="${SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED:-false}"
COMMAND_TRANSMITTER_REVIEWED="${SEMOPS_MAVLINK_COMMAND_TRANSMITTER_REVIEWED:-false}"
COMMAND_SIMULATOR_ONLY_CONFIRMED="${SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED:-false}"
COMMAND_ABORT_READY="${SEMOPS_MAVLINK_COMMAND_ABORT_READY:-false}"
COMMAND_EXPECTED_ACK_TASK_ID="${SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_TASK_ID:-}"
COMMAND_EXPECTED_ACK_STATUS="${SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_STATUS:-accepted}"
COMMAND_EXPECTED_TASK_TARGET_ID="${SEMOPS_MAVLINK_COMMAND_EXPECTED_TASK_TARGET_ID:-}"
COMMAND_POST_STATE_TRACK_ID="${SEMOPS_MAVLINK_COMMAND_POST_STATE_TRACK_ID:-}"
COMMAND_SMOKE_TIMEOUT="${SEMOPS_MAVLINK_COMMAND_SMOKE_TIMEOUT:-$TIMEOUT}"
COMMAND_POST_STATE_MIN_UPDATES="${SEMOPS_MAVLINK_COMMAND_POST_STATE_MIN_UPDATES:-$MIN_UPDATES}"
COMMAND_POST_STATE_REQUIRE_MOTION="${SEMOPS_MAVLINK_COMMAND_POST_STATE_REQUIRE_MOTION:-$REQUIRE_MOTION}"

bool_is_true() {
  case "${1:-}" in
    1|true|TRUE|True|yes|YES|Yes|on|ON|On)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

have_command() {
  command -v "$1" >/dev/null 2>&1
}

have_px4_headless_image() {
  have_command docker && docker image inspect "$PX4_HEADLESS_IMAGE" >/dev/null 2>&1
}

docker_image_matches() {
  local pattern="$1"
  have_command docker && docker image ls --format '{{.Repository}}:{{.Tag}}' 2>/dev/null |
    grep -Eiq "$pattern"
}

have_simulator_image() {
  if ! have_command docker; then
    return 1
  fi
  have_px4_headless_image || docker_image_matches 'px4|mavsdk|ardupilot|arducopter'
}

have_local_simulator_tooling() {
  have_command px4 || have_command mavsdk_server || have_command sim_vehicle.py || have_simulator_image
}

have_family_simulator_tooling() {
  case "$SIMULATOR_FAMILY" in
    px4)
      have_command px4 || have_px4_headless_image || docker_image_matches 'px4'
      ;;
    ardupilot)
      have_command sim_vehicle.py || docker_image_matches 'ardupilot|arducopter'
      ;;
    mavsdk)
      have_command mavsdk_server || docker_image_matches 'mavsdk'
      ;;
    hardware|other)
      have_local_simulator_tooling
      ;;
    *)
      return 1
      ;;
  esac
}

write_evidence() {
  local result="$1"
  local status="$2"

  mkdir -p "$EVIDENCE_DIR"
  {
    echo "created_at=$EVIDENCE_STAMP"
    echo "mode=$MODE"
    echo "result=$result"
    echo "exit_status=$status"
    echo "semops_commit=$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || true)"
    echo "simulator_name=$SIMULATOR_NAME"
    echo "simulator_family=$SIMULATOR_FAMILY"
    echo "simulator_version=$SIMULATOR_VERSION"
    echo "simulator_command=$SIMULATOR_COMMAND"
    echo "simulator_udp_route=$SIMULATOR_ROUTE"
    echo "px4_headless_image=$PX4_HEADLESS_IMAGE"
    echo "px4_headless_image_present=$(if have_px4_headless_image; then echo true; else echo false; fi)"
    echo "px4_headless_container=$PX4_HEADLESS_CONTAINER"
    echo "px4_headless_vehicle=$PX4_HEADLESS_VEHICLE"
    echo "px4_headless_world=$PX4_HEADLESS_WORLD"
    echo "px4_headless_host_qgc=$PX4_HEADLESS_HOST_QGC"
    echo "px4_headless_host_api=$PX4_HEADLESS_HOST_API"
    echo "px4_headless_route_mode=$PX4_HEADLESS_ROUTE_MODE"
    echo "px4_headless_docker_network=$PX4_HEADLESS_DOCKER_NETWORK"
    echo "px4_headless_network_target=$PX4_HEADLESS_NETWORK_TARGET"
    echo "px4_headless_boot_wait=$PX4_HEADLESS_BOOT_WAIT"
    echo "px4_headless_pull_allowed=$PX4_HEADLESS_PULL"
    echo "ardupilot_vehicle=$ARDUPILOT_VEHICLE"
    echo "mavsdk_offboard_route=$MAVSDK_OFFBOARD_ROUTE"
    echo "expected_track_id=$EXPECTED_TRACK_ID"
    echo "snapshot_url=$SNAPSHOT_URL"
    echo "timeout=$TIMEOUT"
    echo "min_updates=$MIN_UPDATES"
    echo "require_motion=$REQUIRE_MOTION"
    echo "command_target_id=$COMMAND_TARGET_ID"
    echo "command_action=$COMMAND_ACTION"
    echo "command_safety_profile=$COMMAND_SAFETY_PROFILE"
    echo "command_local_override_confirmed=$COMMAND_LOCAL_OVERRIDE_CONFIRMED"
    echo "command_ack_required=$COMMAND_ACK_REQUIRED"
    echo "command_post_state_poll_required=$COMMAND_POST_STATE_POLL_REQUIRED"
    echo "command_transmitter=$COMMAND_TRANSMITTER"
    echo "command_transmit_enabled=$COMMAND_TRANSMIT_ENABLED"
    echo "command_transmitter_reviewed=$COMMAND_TRANSMITTER_REVIEWED"
    echo "command_simulator_only_confirmed=$COMMAND_SIMULATOR_ONLY_CONFIRMED"
    echo "command_abort_ready=$COMMAND_ABORT_READY"
    echo "command_expected_ack_task_id=$COMMAND_EXPECTED_ACK_TASK_ID"
    echo "command_expected_ack_status=$COMMAND_EXPECTED_ACK_STATUS"
    echo "command_expected_task_target_id=$COMMAND_EXPECTED_TASK_TARGET_ID"
    echo "command_post_state_track_id=$COMMAND_POST_STATE_TRACK_ID"
    echo "command_smoke_timeout=$COMMAND_SMOKE_TIMEOUT"
    echo "command_post_state_min_updates=$COMMAND_POST_STATE_MIN_UPDATES"
    echo "command_post_state_require_motion=$COMMAND_POST_STATE_REQUIRE_MOTION"
    echo "px4_path=$(command -v px4 2>/dev/null || true)"
    echo "mavsdk_server_path=$(command -v mavsdk_server 2>/dev/null || true)"
    echo "sim_vehicle_path=$(command -v sim_vehicle.py 2>/dev/null || true)"
    echo "docker_available=$(if have_command docker; then echo true; else echo false; fi)"
    echo "remote_source_allowed=$ALLOW_REMOTE_SOURCE"
  } > "$EVIDENCE_FILE"
  echo "MAVLink SITL gate evidence: $EVIDENCE_FILE"
}

print_preflight() {
  echo "MAVLink SITL gate mode: $MODE"
  echo "Expected simulator UDP route: $SIMULATOR_ROUTE"
  echo "Expected COP track: $EXPECTED_TRACK_ID"
  echo "Simulator family: ${SIMULATOR_FAMILY:-missing}"
  if [[ "$MODE" == "command-preflight" || "$MODE" == "command-live-sim" ]]; then
    echo "Command target: ${COMMAND_TARGET_ID:-missing}"
    echo "Command action: ${COMMAND_ACTION:-missing}"
    echo "Command safety profile: ${COMMAND_SAFETY_PROFILE:-missing}"
    echo "Command local override confirmed: $COMMAND_LOCAL_OVERRIDE_CONFIRMED"
    echo "Command ACK required: $COMMAND_ACK_REQUIRED"
    echo "Command post-state poll required: $COMMAND_POST_STATE_POLL_REQUIRED"
    echo "Command transmitter: ${COMMAND_TRANSMITTER:-missing}"
    echo "Command transmit enabled: $COMMAND_TRANSMIT_ENABLED"
    echo "Command transmitter reviewed: $COMMAND_TRANSMITTER_REVIEWED"
    echo "Command simulator only confirmed: $COMMAND_SIMULATOR_ONLY_CONFIRMED"
    echo "Command abort ready: $COMMAND_ABORT_READY"
    echo "Command expected ACK task: ${COMMAND_EXPECTED_ACK_TASK_ID:-missing}"
    echo "Command expected ACK status: $COMMAND_EXPECTED_ACK_STATUS"
    echo "Command expected task target: ${COMMAND_EXPECTED_TASK_TARGET_ID:-optional}"
    echo "Command post-state track: ${COMMAND_POST_STATE_TRACK_ID:-missing}"
  fi
  echo
  echo "Local simulator commands:"
  echo "  px4: $(command -v px4 2>/dev/null || echo missing)"
  echo "  mavsdk_server: $(command -v mavsdk_server 2>/dev/null || echo missing)"
  echo "  sim_vehicle.py: $(command -v sim_vehicle.py 2>/dev/null || echo missing)"
  if have_command docker; then
    echo
    echo "Preferred PX4 headless Docker lane:"
    echo "  image: $PX4_HEADLESS_IMAGE"
    echo "  image present: $(if have_px4_headless_image; then echo yes; else echo no; fi)"
    echo "  container: $PX4_HEADLESS_CONTAINER"
    echo "  vehicle/world: $PX4_HEADLESS_VEHICLE / $PX4_HEADLESS_WORLD"
    echo "  route mode: $PX4_HEADLESS_ROUTE_MODE"
    if [[ "$PX4_HEADLESS_ROUTE_MODE" == "compose-network" ]]; then
      echo "  docker network: $PX4_HEADLESS_DOCKER_NETWORK"
      echo "  network target: $PX4_HEADLESS_NETWORK_TARGET"
    fi
    echo "  pull allowed: $PX4_HEADLESS_PULL"
    echo
    echo "Local simulator-ish Docker images:"
    docker image ls --format '  {{.Repository}}:{{.Tag}}' 2>/dev/null |
      grep -Ei 'px4|mavsdk|ardupilot|arducopter' || echo "  none"
  fi
  if [[ "$MODE" == "ardupilot-stack" ]]; then
    echo
    echo "ArduPilot parity lane:"
    echo "  vehicle: $ARDUPILOT_VEHICLE"
    echo "  default command: $(ardupilot_command_string)"
    echo "  motion required by default: true"
  fi
  if [[ "$MODE" == "mavsdk-offboard-stack" ]]; then
    echo
    echo "MAVSDK/offboard parity lane:"
    echo "  route: $MAVSDK_OFFBOARD_ROUTE"
    echo "  default command: $(mavsdk_offboard_command_string)"
    echo "  motion required by default: true"
  fi
}

require_simulator_family() {
  case "$SIMULATOR_FAMILY" in
    px4|ardupilot|mavsdk|hardware|other)
      ;;
    "")
      cat >&2 <<'EOF'
SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY is required for focused, stack, or command-control mode.

Use one of:
  px4
  ardupilot
  mavsdk
  hardware
  other

This guard prevents one simulator-family pass from being reused as ArduPilot, MAVSDK/offboard, or hardware parity.
EOF
      write_evidence "blocked_missing_simulator_family" 2
      exit 2
      ;;
    *)
      cat >&2 <<EOF
Unsupported SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=$SIMULATOR_FAMILY

Expected px4, ardupilot, mavsdk, hardware, or other.
EOF
      write_evidence "blocked_bad_simulator_family" 2
      exit 2
      ;;
  esac
}

px4_headless_args() {
  local host_qgc="$PX4_HEADLESS_HOST_QGC"
  local host_api="$PX4_HEADLESS_HOST_API"
  if [[ "$PX4_HEADLESS_ROUTE_MODE" == "compose-network" ]]; then
    host_qgc="${host_qgc:-$PX4_HEADLESS_NETWORK_TARGET}"
    host_api="${host_api:-$PX4_HEADLESS_NETWORK_TARGET}"
  fi

  PX4_HEADLESS_ARGS=(-v "$PX4_HEADLESS_VEHICLE" -w "$PX4_HEADLESS_WORLD")
  if [[ -n "$host_qgc" && -n "$host_api" ]]; then
    PX4_HEADLESS_ARGS+=("$host_qgc" "$host_api")
  elif [[ -n "$host_api" ]]; then
    PX4_HEADLESS_ARGS+=("$host_api")
  elif [[ -n "$host_qgc" ]]; then
    cat >&2 <<'EOF'
SEMOPS_MAVLINK_SITL_PX4_HOST_QGC requires SEMOPS_MAVLINK_SITL_PX4_HOST_API.

The headless PX4 entrypoint accepts either:
  [HOST_API]
  [HOST_QGC HOST_API]
EOF
    write_evidence "blocked_bad_px4_headless_hosts" 2
    exit 2
  fi
}

px4_headless_docker_args() {
  PX4_HEADLESS_DOCKER_ARGS=(-d --rm --name "$PX4_HEADLESS_CONTAINER")
  if [[ "$PX4_HEADLESS_ROUTE_MODE" == "compose-network" ]]; then
    PX4_HEADLESS_DOCKER_ARGS+=(--network "$PX4_HEADLESS_DOCKER_NETWORK")
  fi
  PX4_HEADLESS_DOCKER_ARGS+=("$PX4_HEADLESS_IMAGE")
}

px4_headless_command_string() {
  px4_headless_args
  px4_headless_docker_args
  printf 'docker run'
  printf ' %q' "${PX4_HEADLESS_DOCKER_ARGS[@]}"
  printf ' %q' "${PX4_HEADLESS_ARGS[@]}"
  printf '\n'
}

px4_headless_start_hook_command_string() {
  printf 'cd %q && ' "$ROOT"
  printf 'SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-start'
  printf ' SEMOPS_MAVLINK_SITL_DOCKER_IMAGE=%q' "$PX4_HEADLESS_IMAGE"
  printf ' SEMOPS_MAVLINK_SITL_DOCKER_CONTAINER=%q' "$PX4_HEADLESS_CONTAINER"
  printf ' SEMOPS_MAVLINK_SITL_PX4_VEHICLE=%q' "$PX4_HEADLESS_VEHICLE"
  printf ' SEMOPS_MAVLINK_SITL_PX4_WORLD=%q' "$PX4_HEADLESS_WORLD"
  printf ' SEMOPS_MAVLINK_SITL_PX4_HOST_QGC=%q' "$PX4_HEADLESS_HOST_QGC"
  printf ' SEMOPS_MAVLINK_SITL_PX4_HOST_API=%q' "$PX4_HEADLESS_HOST_API"
  printf ' SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE=%q' "$PX4_HEADLESS_ROUTE_MODE"
  printf ' SEMOPS_MAVLINK_SITL_DOCKER_NETWORK=%q' "$PX4_HEADLESS_DOCKER_NETWORK"
  printf ' SEMOPS_MAVLINK_SITL_PX4_NETWORK_TARGET=%q' "$PX4_HEADLESS_NETWORK_TARGET"
  printf ' SEMOPS_MAVLINK_SITL_PX4_BOOT_WAIT=%q' "$PX4_HEADLESS_BOOT_WAIT"
  printf ' SEMOPS_MAVLINK_SITL_DOCKER_PULL=%q' "$PX4_HEADLESS_PULL"
  printf ' SEMOPS_MAVLINK_SITL_DOCKER_REPLACE=%q' "$PX4_HEADLESS_REPLACE"
  printf ' bash scripts/mavlink-sitl-gate.sh'
}

px4_headless_stop_hook_command_string() {
  printf 'docker stop %q' "$PX4_HEADLESS_CONTAINER"
}

ardupilot_command_string() {
  printf 'sim_vehicle.py -v %q --out=udp:%q\n' "$ARDUPILOT_VEHICLE" "$SIMULATOR_ROUTE"
}

mavsdk_offboard_command_string() {
  printf 'mavsdk_server %q\n' "$MAVSDK_OFFBOARD_ROUTE"
}

ensure_px4_headless_image() {
  if have_px4_headless_image; then
    return
  fi
  if bool_is_true "$PX4_HEADLESS_PULL"; then
    docker pull "$PX4_HEADLESS_IMAGE"
    return
  fi
  cat >&2 <<EOF
PX4 headless Docker image is not local: $PX4_HEADLESS_IMAGE

Pull it explicitly, or let this helper pull it:
  docker pull $PX4_HEADLESS_IMAGE
EOF
  cat >&2 <<'EOF'
  SEMOPS_MAVLINK_SITL_DOCKER_PULL=true \
    SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
    bash scripts/mavlink-sitl-gate.sh

The pull is intentionally opt-in because the image is large.
EOF
  write_evidence "blocked_missing_px4_headless_image" 2
  exit 2
}

container_exists() {
  docker ps -a --format '{{.Names}}' | grep -Fxq "$PX4_HEADLESS_CONTAINER"
}

container_running() {
  docker ps --format '{{.Names}}' | grep -Fxq "$PX4_HEADLESS_CONTAINER"
}

container_on_px4_network() {
  docker inspect -f '{{range $name, $_ := .NetworkSettings.Networks}}{{println $name}}{{end}}' "$PX4_HEADLESS_CONTAINER" 2>/dev/null |
    grep -Fxq "$PX4_HEADLESS_DOCKER_NETWORK"
}

start_px4_headless_container() {
  if ! have_command docker; then
    echo "Docker is required for px4-headless-stack mode." >&2
    write_evidence "blocked_missing_docker" 2
    exit 2
  fi
  case "$PX4_HEADLESS_ROUTE_MODE" in
    host|compose-network)
      ;;
    *)
      cat >&2 <<EOF
Unsupported SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE=$PX4_HEADLESS_ROUTE_MODE

Expected host or compose-network.
EOF
      write_evidence "blocked_bad_px4_route_mode" 2
      exit 2
      ;;
  esac
  ensure_px4_headless_image
  if [[ "$PX4_HEADLESS_ROUTE_MODE" == "compose-network" ]] &&
    ! docker network inspect "$PX4_HEADLESS_DOCKER_NETWORK" >/dev/null 2>&1; then
    cat >&2 <<EOF
PX4 compose-network route requires an existing Docker network: $PX4_HEADLESS_DOCKER_NETWORK

The px4-headless-stack mode creates this by starting the COP stack first. For a standalone PX4 container, either create
the network yourself or set SEMOPS_MAVLINK_SITL_PX4_ROUTE_MODE=host.
EOF
    write_evidence "blocked_missing_px4_docker_network" 2
    exit 2
  fi
  if container_running; then
    if [[ "$PX4_HEADLESS_ROUTE_MODE" == "compose-network" ]] && ! container_on_px4_network; then
      if bool_is_true "$PX4_HEADLESS_REPLACE"; then
        docker rm -f "$PX4_HEADLESS_CONTAINER" >/dev/null
      else
        cat >&2 <<EOF
PX4 headless container is already running but is not attached to $PX4_HEADLESS_DOCKER_NETWORK: $PX4_HEADLESS_CONTAINER

Stop it yourself or set:
  SEMOPS_MAVLINK_SITL_DOCKER_REPLACE=true
EOF
        write_evidence "blocked_px4_container_wrong_network" 2
        exit 2
      fi
    else
      echo "Reusing running PX4 headless container: $PX4_HEADLESS_CONTAINER"
      return
    fi
  fi
  if container_running; then
    echo "Reusing running PX4 headless container: $PX4_HEADLESS_CONTAINER"
    return
  fi
  if container_exists; then
    if bool_is_true "$PX4_HEADLESS_REPLACE"; then
      docker rm "$PX4_HEADLESS_CONTAINER" >/dev/null
    else
      cat >&2 <<EOF
PX4 headless container name already exists but is not running: $PX4_HEADLESS_CONTAINER

Remove it yourself or set:
  SEMOPS_MAVLINK_SITL_DOCKER_REPLACE=true
EOF
      write_evidence "blocked_existing_px4_headless_container" 2
      exit 2
    fi
  fi
  px4_headless_args
  px4_headless_docker_args
  docker run "${PX4_HEADLESS_DOCKER_ARGS[@]}" "${PX4_HEADLESS_ARGS[@]}" >/dev/null
  PX4_HEADLESS_STARTED=true
  echo "Started PX4 headless container: $PX4_HEADLESS_CONTAINER"
  echo "Waiting $PX4_HEADLESS_BOOT_WAIT for PX4/Gazebo boot..."
  sleep "$PX4_HEADLESS_BOOT_WAIT"
}

cleanup_px4_headless_container() {
  if [[ "$PX4_HEADLESS_STARTED" != "true" ]] || bool_is_true "$PX4_HEADLESS_KEEP"; then
    return
  fi
  docker stop "$PX4_HEADLESS_CONTAINER" >/dev/null 2>&1 || true
}

require_simulator_attestation() {
  if [[ -z "$SIMULATOR_NAME" ]]; then
    cat >&2 <<'EOF'
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME is required for focused, stack, or command-control mode.

Name the external simulator source explicitly, for example:
  SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL 1.15"
  SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND="make px4_sitl gz_x500"

This guard prevents generated-frame evidence from being mistaken for PX4/MAVSDK/SITL fidelity.
EOF
    write_evidence "blocked_missing_simulator_name" 2
    exit 2
  fi
  require_simulator_family

  if ! have_family_simulator_tooling && ! bool_is_true "$ALLOW_REMOTE_SOURCE"; then
    cat >&2 <<EOF
No local simulator command or Docker image was found for family: $SIMULATOR_FAMILY

If the simulator is running remotely or on hardware-adjacent infrastructure and is already sending MAVLink to the
SemOps UDP route, set:
  SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true

Otherwise install/start the simulator first. This gate must observe an external source, not generated test frames.
EOF
    write_evidence "blocked_no_local_simulator" 2
    exit 2
  fi
}

require_command_input() {
  local name="$1"
  local value="$2"
  local result="$3"
  local hint="${4:-}"

  if [[ -n "$value" ]]; then
    return
  fi

  cat >&2 <<EOF
$name is required for command-control mode.

$hint
EOF
  write_evidence "$result" 2
  exit 2
}

require_command_bool() {
  local name="$1"
  local value="$2"
  local result="$3"

  if bool_is_true "$value"; then
    return
  fi

  local hint="${4:-}"

  cat >&2 <<EOF
$name must be true for command-control mode.

$hint
EOF
  write_evidence "$result" 2
  exit 2
}

require_command_simulator_family() {
  if [[ "$SIMULATOR_FAMILY" == "hardware" ]]; then
    cat >&2 <<'EOF'
command-live-sim mode refuses simulator_family=hardware.

Run this gate against PX4, ArduPilot, MAVSDK/offboard, or another explicitly named simulator source before any
hardware-adjacent command authority claim.
EOF
    write_evidence "blocked_hardware_command_control" 2
    exit 2
  fi
}

require_command_simulator_safety_profile() {
  case "$COMMAND_SAFETY_PROFILE" in
    simulator|simulator_*|*_simulator|*_simulator_*|*simulator*)
      return
      ;;
  esac

  cat >&2 <<'EOF'
SEMOPS_MAVLINK_COMMAND_SAFETY_PROFILE must be simulator-scoped for command-live-sim mode.

Use a reviewed simulator-only profile name such as simulator_local_operator. Do not use production or hardware command
profiles in this gate.
EOF
  write_evidence "blocked_non_simulator_command_safety_profile" 2
  exit 2
}

run_command_control_smoke() {
  local started_at="$1"

  (
    cd "$ROOT"
    SEMOPS_MAVLINK_COMMAND_SMOKE_SNAPSHOT_URL="$SNAPSHOT_URL" \
    SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_TASK_ID="$COMMAND_EXPECTED_ACK_TASK_ID" \
    SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_STATUS="$COMMAND_EXPECTED_ACK_STATUS" \
    SEMOPS_MAVLINK_COMMAND_EXPECTED_TASK_TARGET_ID="$COMMAND_EXPECTED_TASK_TARGET_ID" \
    SEMOPS_MAVLINK_COMMAND_POST_STATE_TRACK_ID="$COMMAND_POST_STATE_TRACK_ID" \
    SEMOPS_MAVLINK_COMMAND_STARTED_AT="$started_at" \
    SEMOPS_MAVLINK_COMMAND_SMOKE_TIMEOUT="$COMMAND_SMOKE_TIMEOUT" \
    SEMOPS_MAVLINK_COMMAND_POST_STATE_MIN_UPDATES="$COMMAND_POST_STATE_MIN_UPDATES" \
    SEMOPS_MAVLINK_COMMAND_POST_STATE_REQUIRE_MOTION="$COMMAND_POST_STATE_REQUIRE_MOTION" \
      go test ./internal/smoke/mavlink -run TestCommandControlSimulatorGateCOPSnapshot -count=1 -v
  )
}

run_command_telemetry_preflight() {
  (
    cd "$ROOT"
    SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL="$SNAPSHOT_URL" \
    SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID="$COMMAND_POST_STATE_TRACK_ID" \
    SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT="$COMMAND_SMOKE_TIMEOUT" \
    SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES=1 \
    SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=false \
      go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v
  )
}

run_command_preflight() {
  require_simulator_attestation
  require_command_simulator_family
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_TARGET_ID" \
    "$COMMAND_TARGET_ID" \
    "blocked_missing_command_target" \
    "Name the born-first graph target that a future native transmitter would command."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_ACTION" \
    "$COMMAND_ACTION" \
    "blocked_missing_command_action" \
    "Name the safe simulator action under review, for example hold_position, loiter, or arm_disallowed."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_SAFETY_PROFILE" \
    "$COMMAND_SAFETY_PROFILE" \
    "blocked_missing_command_safety_profile" \
    "Name the reviewed command safety profile. Do not use a production or hardware profile for this preflight."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_LOCAL_OVERRIDE_CONFIRMED" \
    "$COMMAND_LOCAL_OVERRIDE_CONFIRMED" \
    "blocked_missing_command_local_override" \
    "Command evidence must say local operator override and abort authority are understood before any transmit gate."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_ACK_REQUIRED" \
    "$COMMAND_ACK_REQUIRED" \
    "blocked_missing_command_ack_requirement" \
    "The future live gate must correlate COMMAND_ACK or equivalent native readback before claiming command acceptance."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_POST_STATE_POLL_REQUIRED" \
    "$COMMAND_POST_STATE_POLL_REQUIRED" \
    "The future live gate must poll post-command state before claiming execution or effect."

  if [[ -n "$COMMAND_TRANSMITTER" ]] || bool_is_true "$COMMAND_TRANSMIT_ENABLED"; then
    cat >&2 <<'EOF'
command-preflight mode is non-transmitting by design.

Remove SEMOPS_MAVLINK_COMMAND_TRANSMITTER and keep SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED=false for preflight.
Use command-live-sim for the reviewed simulator-transmit path.
EOF
    write_evidence "blocked_unreviewed_command_transmitter" 2
    exit 2
  fi

  cat >&2 <<'EOF'
MAVLink command-preflight remains blocked by design: SemOps has command intent, arbitration, and COMMAND_ACK readback
plumbing, but preflight does not transmit.

This preflight wrote evidence for the intended safety posture and stopped before native transmit. Use
command-live-sim for the reviewed simulator-transmit gate.
EOF
  write_evidence "blocked_no_native_command_transmitter" 2
  exit 2
}

run_command_live_simulator_gate() {
  require_simulator_attestation
  require_command_simulator_family
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_TARGET_ID" \
    "$COMMAND_TARGET_ID" \
    "blocked_missing_command_target" \
    "Name the born-first graph target that the simulator command will affect."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_ACTION" \
    "$COMMAND_ACTION" \
    "blocked_missing_command_action" \
    "Name the safe simulator action under review, for example hold_position, loiter, or arm_disallowed."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_SAFETY_PROFILE" \
    "$COMMAND_SAFETY_PROFILE" \
    "blocked_missing_command_safety_profile" \
    "Name the reviewed simulator-only command safety profile."
  require_command_simulator_safety_profile
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_LOCAL_OVERRIDE_CONFIRMED" \
    "$COMMAND_LOCAL_OVERRIDE_CONFIRMED" \
    "blocked_missing_command_local_override" \
    "Command evidence must say local operator override and abort authority are understood before simulator transmit."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_ACK_REQUIRED" \
    "$COMMAND_ACK_REQUIRED" \
    "blocked_missing_command_ack_requirement" \
    "The gate must correlate COMMAND_ACK or equivalent native readback before claiming command acceptance."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_POST_STATE_POLL_REQUIRED" \
    "$COMMAND_POST_STATE_POLL_REQUIRED" \
    "blocked_missing_command_post_state_poll_requirement" \
    "The gate must poll post-command state before claiming execution or effect."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_SIMULATOR_ONLY_CONFIRMED" \
    "$COMMAND_SIMULATOR_ONLY_CONFIRMED" \
    "blocked_missing_command_simulator_only_confirmation" \
    "This mode is simulator-only and refuses hardware-adjacent command claims."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_ABORT_READY" \
    "$COMMAND_ABORT_READY" \
    "blocked_missing_command_abort_ready" \
    "A local abort or override path must be identified before running a live simulator command."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_TRANSMITTER_REVIEWED" \
    "$COMMAND_TRANSMITTER_REVIEWED" \
    "blocked_unreviewed_command_transmitter" \
    "The transmitter command must be reviewed for simulator-only use before this gate can run it."
  require_command_bool \
    "SEMOPS_MAVLINK_COMMAND_TRANSMIT_ENABLED" \
    "$COMMAND_TRANSMIT_ENABLED" \
    "blocked_missing_command_transmit_enabled" \
    "Set this true only for command-live-sim after the simulator and transmitter command are reviewed."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_TRANSMITTER" \
    "$COMMAND_TRANSMITTER" \
    "blocked_missing_command_transmitter" \
    "Provide the reviewed simulator transmitter command to run after the COP stack and simulator are already live."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_TASK_ID" \
    "$COMMAND_EXPECTED_ACK_TASK_ID" \
    "blocked_missing_command_ack_task" \
    "Name the expected COP task ID created from MAVLink COMMAND_ACK readback."
  require_command_input \
    "SEMOPS_MAVLINK_COMMAND_POST_STATE_TRACK_ID" \
    "$COMMAND_POST_STATE_TRACK_ID" \
    "blocked_missing_command_post_state_track" \
    "Name the MAVLink track that must refresh after the command starts."

  if run_command_telemetry_preflight; then
    :
  else
    status=$?
    write_evidence "blocked_command_baseline_telemetry" "$status"
    exit "$status"
  fi

  local started_at
  started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "Running reviewed MAVLink simulator transmitter at $started_at"
  if bash -lc "$COMMAND_TRANSMITTER"; then
    :
  else
    status=$?
    write_evidence "failed_command_transmitter" "$status"
    exit "$status"
  fi
  if run_command_control_smoke "$started_at"; then
    write_evidence "passed" 0
  else
    status=$?
    write_evidence "failed_command_control_smoke" "$status"
    exit "$status"
  fi
}

run_guarded_skip_check() {
  (
    cd "$ROOT"
    go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v
  )
}

run_focused_smoke() {
  (
    cd "$ROOT"
    SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL="$SNAPSHOT_URL" \
    SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID="$EXPECTED_TRACK_ID" \
    SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT="$TIMEOUT" \
    SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES="$MIN_UPDATES" \
    SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION="$REQUIRE_MOTION" \
      go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v
  )
}

run_stack_smoke() {
  local after_stack_ready_cmd="${1:-${SEMOPS_COP_SMOKE_AFTER_STACK_READY_CMD:-}}"
  local before_mavlink_sitl_cmd="${2:-${SEMOPS_COP_SMOKE_BEFORE_MAVLINK_SITL_CMD:-}}"
  local before_cleanup_cmd="${3:-${SEMOPS_COP_SMOKE_BEFORE_CLEANUP_CMD:-}}"
  local keep_stack="${4:-${SEMOPS_COP_KEEP_STACK:-}}"
  (
    cd "$ROOT"
    SEMOPS_COP_MAVLINK_SYSTEM_IDS="${SEMOPS_COP_MAVLINK_SYSTEM_IDS:-1,42}" \
    SEMOPS_COP_KEEP_STACK="$keep_stack" \
    SEMOPS_COP_SMOKE_AFTER_STACK_READY_CMD="$after_stack_ready_cmd" \
    SEMOPS_COP_SMOKE_BEFORE_MAVLINK_SITL_CMD="$before_mavlink_sitl_cmd" \
    SEMOPS_COP_SMOKE_BEFORE_CLEANUP_CMD="$before_cleanup_cmd" \
    SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true \
    SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID="$EXPECTED_TRACK_ID" \
    SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT="$TIMEOUT" \
    SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES="$MIN_UPDATES" \
    SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION="$REQUIRE_MOTION" \
      bash scripts/cop-stack-smoke.sh
  )
}

run_px4_headless_start_only() {
  if [[ -n "$SIMULATOR_FAMILY" && "$SIMULATOR_FAMILY" != "px4" ]]; then
    cat >&2 <<EOF
px4-headless-start mode requires SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4, got $SIMULATOR_FAMILY
EOF
    write_evidence "blocked_bad_px4_headless_family" 2
    exit 2
  fi
  SIMULATOR_FAMILY="px4"
  start_px4_headless_container
}

run_px4_headless_stack_smoke() {
  if [[ -n "$SIMULATOR_FAMILY" && "$SIMULATOR_FAMILY" != "px4" ]]; then
    cat >&2 <<EOF
px4-headless-stack mode requires SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=px4, got $SIMULATOR_FAMILY
EOF
    write_evidence "blocked_bad_px4_headless_family" 2
    exit 2
  fi
  SIMULATOR_FAMILY="px4"
  if [[ -z "$SIMULATOR_NAME" ]]; then
    SIMULATOR_NAME="PX4 Gazebo headless Docker"
  fi
  if [[ -z "$SIMULATOR_VERSION" ]]; then
    SIMULATOR_VERSION="${PX4_HEADLESS_IMAGE##*:}"
  fi
  if [[ -z "$SIMULATOR_COMMAND" ]]; then
    SIMULATOR_COMMAND="$(px4_headless_command_string)"
  fi
  trap cleanup_px4_headless_container EXIT
  require_simulator_attestation
  if [[ "$PX4_HEADLESS_ROUTE_MODE" == "compose-network" ]]; then
    PX4_HEADLESS_STARTED=true
    local keep_stack="${SEMOPS_COP_KEEP_STACK:-}"
    if bool_is_true "$PX4_HEADLESS_KEEP"; then
      keep_stack="true"
    fi
    if run_stack_smoke "" "$(px4_headless_start_hook_command_string)" "$(px4_headless_stop_hook_command_string)" "$keep_stack"; then
      write_evidence "passed" 0
    else
      status=$?
      write_evidence "failed" "$status"
      exit "$status"
    fi
    return
  fi
  start_px4_headless_container
  if run_stack_smoke; then
    write_evidence "passed" 0
  else
    status=$?
    write_evidence "failed" "$status"
    exit "$status"
  fi
}

run_ardupilot_stack_smoke() {
  if [[ -n "$SIMULATOR_FAMILY" && "$SIMULATOR_FAMILY" != "ardupilot" ]]; then
    cat >&2 <<EOF
ardupilot-stack mode requires SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=ardupilot, got $SIMULATOR_FAMILY
EOF
    write_evidence "blocked_bad_ardupilot_family" 2
    exit 2
  fi
  SIMULATOR_FAMILY="ardupilot"
  if [[ -z "$SIMULATOR_NAME" ]]; then
    SIMULATOR_NAME="ArduPilot SITL $ARDUPILOT_VEHICLE"
  fi
  if [[ -z "$SIMULATOR_COMMAND" ]]; then
    SIMULATOR_COMMAND="$(ardupilot_command_string)"
  fi
  if [[ -z "${SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION:-}" ]]; then
    REQUIRE_MOTION=true
  fi
  require_simulator_attestation
  if run_stack_smoke; then
    write_evidence "passed" 0
  else
    status=$?
    write_evidence "failed" "$status"
    exit "$status"
  fi
}

run_mavsdk_offboard_stack_smoke() {
  if [[ -n "$SIMULATOR_FAMILY" && "$SIMULATOR_FAMILY" != "mavsdk" ]]; then
    cat >&2 <<EOF
mavsdk-offboard-stack mode requires SEMOPS_MAVLINK_SITL_SIMULATOR_FAMILY=mavsdk, got $SIMULATOR_FAMILY
EOF
    write_evidence "blocked_bad_mavsdk_family" 2
    exit 2
  fi
  SIMULATOR_FAMILY="mavsdk"
  if [[ -z "$SIMULATOR_NAME" ]]; then
    SIMULATOR_NAME="MAVSDK/PX4 offboard route"
  fi
  if [[ -z "$SIMULATOR_COMMAND" ]]; then
    SIMULATOR_COMMAND="$(mavsdk_offboard_command_string)"
  fi
  if [[ -z "${SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION:-}" ]]; then
    REQUIRE_MOTION=true
  fi
  require_simulator_attestation
  if run_stack_smoke; then
    write_evidence "passed" 0
  else
    status=$?
    write_evidence "failed" "$status"
    exit "$status"
  fi
}

case "$MODE" in
  px4-headless-stack|px4-headless-start)
    if [[ -z "$SIMULATOR_FAMILY" ]]; then
      SIMULATOR_FAMILY="px4"
    fi
    ;;
  ardupilot-stack)
    if [[ -z "$SIMULATOR_FAMILY" ]]; then
      SIMULATOR_FAMILY="ardupilot"
    fi
    if [[ -z "$SIMULATOR_NAME" ]]; then
      SIMULATOR_NAME="ArduPilot SITL $ARDUPILOT_VEHICLE"
    fi
    if [[ -z "$SIMULATOR_COMMAND" ]]; then
      SIMULATOR_COMMAND="$(ardupilot_command_string)"
    fi
    if [[ -z "${SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION:-}" ]]; then
      REQUIRE_MOTION=true
    fi
    ;;
  mavsdk-offboard-stack)
    if [[ -z "$SIMULATOR_FAMILY" ]]; then
      SIMULATOR_FAMILY="mavsdk"
    fi
    if [[ -z "$SIMULATOR_NAME" ]]; then
      SIMULATOR_NAME="MAVSDK/PX4 offboard route"
    fi
    if [[ -z "$SIMULATOR_COMMAND" ]]; then
      SIMULATOR_COMMAND="$(mavsdk_offboard_command_string)"
    fi
    if [[ -z "${SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION:-}" ]]; then
      REQUIRE_MOTION=true
    fi
    ;;
esac

print_preflight

case "$MODE" in
  preflight)
    run_guarded_skip_check
    write_evidence "preflight_only" 0
    ;;
  focused)
    require_simulator_attestation
    if run_focused_smoke; then
      write_evidence "passed" 0
    else
      status=$?
      write_evidence "failed" "$status"
      exit "$status"
    fi
    ;;
  stack)
    require_simulator_attestation
    if run_stack_smoke; then
      write_evidence "passed" 0
    else
      status=$?
      write_evidence "failed" "$status"
      exit "$status"
    fi
    ;;
  px4-headless-stack)
    run_px4_headless_stack_smoke
    ;;
  px4-headless-start)
    run_px4_headless_start_only
    ;;
  ardupilot-stack)
    run_ardupilot_stack_smoke
    ;;
  mavsdk-offboard-stack)
    run_mavsdk_offboard_stack_smoke
    ;;
  command-preflight)
    run_command_preflight
    ;;
  command-live-sim)
    run_command_live_simulator_gate
    ;;
  *)
    echo "Unsupported SEMOPS_MAVLINK_SITL_GATE_MODE=$MODE" >&2
    echo "Expected preflight, focused, stack, px4-headless-stack, px4-headless-start, ardupilot-stack, mavsdk-offboard-stack, command-preflight, or command-live-sim." >&2
    write_evidence "blocked_bad_mode" 2
    exit 2
    ;;
esac
