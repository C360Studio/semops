#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${SEMOPS_MAVLINK_SITL_GATE_MODE:-preflight}"
EVIDENCE_DIR="${SEMOPS_MAVLINK_SITL_EVIDENCE_DIR:-$ROOT/tmp/mavlink-sitl-evidence}"
EVIDENCE_STAMP="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
EVIDENCE_FILE="${SEMOPS_MAVLINK_SITL_EVIDENCE_FILE:-$EVIDENCE_DIR/${EVIDENCE_STAMP}-${MODE}.env}"

SIMULATOR_NAME="${SEMOPS_MAVLINK_SITL_SIMULATOR_NAME:-}"
SIMULATOR_VERSION="${SEMOPS_MAVLINK_SITL_SIMULATOR_VERSION:-}"
SIMULATOR_COMMAND="${SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND:-}"
SIMULATOR_ROUTE="${SEMOPS_MAVLINK_SITL_SIMULATOR_UDP_ROUTE:-127.0.0.1:${SEMOPS_MAVLINK_UDP_HOST_PORT:-14550}}"
ALLOW_REMOTE_SOURCE="${SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE:-false}"

SNAPSHOT_URL="${SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL:-http://127.0.0.1:${SEMOPS_CADDY_HOST_PORT:-8080}/api/cop/snapshot}"
EXPECTED_TRACK_ID="${SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID:-c360.edge-compose.cop.mavlink.track.system-1}"
TIMEOUT="${SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT:-2m}"
MIN_UPDATES="${SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES:-2}"
REQUIRE_MOTION="${SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION:-false}"

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

have_simulator_image() {
  if ! have_command docker; then
    return 1
  fi
  docker image ls --format '{{.Repository}}:{{.Tag}}' 2>/dev/null |
    grep -Eiq 'px4|mavsdk|ardupilot|arducopter'
}

have_local_simulator_tooling() {
  have_command px4 || have_command mavsdk_server || have_command sim_vehicle.py || have_simulator_image
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
    echo "simulator_version=$SIMULATOR_VERSION"
    echo "simulator_command=$SIMULATOR_COMMAND"
    echo "simulator_udp_route=$SIMULATOR_ROUTE"
    echo "expected_track_id=$EXPECTED_TRACK_ID"
    echo "snapshot_url=$SNAPSHOT_URL"
    echo "timeout=$TIMEOUT"
    echo "min_updates=$MIN_UPDATES"
    echo "require_motion=$REQUIRE_MOTION"
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
  echo
  echo "Local simulator commands:"
  echo "  px4: $(command -v px4 2>/dev/null || echo missing)"
  echo "  mavsdk_server: $(command -v mavsdk_server 2>/dev/null || echo missing)"
  echo "  sim_vehicle.py: $(command -v sim_vehicle.py 2>/dev/null || echo missing)"
  if have_command docker; then
    echo
    echo "Local simulator-ish Docker images:"
    docker image ls --format '  {{.Repository}}:{{.Tag}}' 2>/dev/null |
      grep -Ei 'px4|mavsdk|ardupilot|arducopter' || echo "  none"
  fi
}

require_simulator_attestation() {
  if [[ -z "$SIMULATOR_NAME" ]]; then
    cat >&2 <<'EOF'
SEMOPS_MAVLINK_SITL_SIMULATOR_NAME is required for focused or stack mode.

Name the external simulator source explicitly, for example:
  SEMOPS_MAVLINK_SITL_SIMULATOR_NAME="PX4 SITL 1.15"
  SEMOPS_MAVLINK_SITL_SIMULATOR_COMMAND="make px4_sitl gz_x500"

This guard prevents generated-frame evidence from being mistaken for PX4/MAVSDK/SITL fidelity.
EOF
    write_evidence "blocked_missing_simulator_name" 2
    exit 2
  fi

  if ! have_local_simulator_tooling && ! bool_is_true "$ALLOW_REMOTE_SOURCE"; then
    cat >&2 <<'EOF'
No local PX4, MAVSDK, ArduPilot command, or local simulator Docker image was found.

If the simulator is running remotely or on hardware-adjacent infrastructure and is already sending MAVLink to the
SemOps UDP route, set:
  SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true

Otherwise install/start the simulator first. This gate must observe an external source, not generated test frames.
EOF
    write_evidence "blocked_no_local_simulator" 2
    exit 2
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
  (
    cd "$ROOT"
    SEMOPS_COP_MAVLINK_SYSTEM_IDS="${SEMOPS_COP_MAVLINK_SYSTEM_IDS:-1,42}" \
    SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true \
    SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID="$EXPECTED_TRACK_ID" \
    SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT="$TIMEOUT" \
    SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES="$MIN_UPDATES" \
    SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION="$REQUIRE_MOTION" \
      bash scripts/cop-stack-smoke.sh
  )
}

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
  *)
    echo "Unsupported SEMOPS_MAVLINK_SITL_GATE_MODE=$MODE; expected preflight, focused, or stack." >&2
    write_evidence "blocked_bad_mode" 2
    exit 2
    ;;
esac
