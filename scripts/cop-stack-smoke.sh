#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${SEMOPS_COP_COMPOSE_FILE:-$ROOT/compose.cop.yml}"
PROJECT="${SEMOPS_COP_PROJECT:-semops-cop}"

NATS_HOST_PORT="${SEMOPS_NATS_HOST_PORT:-4222}"
HEALTH_HOST_PORT="${SEMOPS_SEMSTREAMS_HEALTH_HOST_PORT:-18080}"
METRICS_HOST_PORT="${SEMOPS_SEMSTREAMS_METRICS_HOST_PORT:-9090}"
API_HOST_PORT="${SEMOPS_API_HOST_PORT:-8088}"
CADDY_HOST_PORT="${SEMOPS_CADDY_HOST_PORT:-8080}"

NATS_URL="${SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL:-nats://127.0.0.1:${NATS_HOST_PORT}}"
HEALTH_URL="${SEMOPS_SEMSTREAMS_HEALTH_URL:-http://127.0.0.1:${HEALTH_HOST_PORT}/healthz}"
METRICS_URL="${SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL:-http://127.0.0.1:${METRICS_HOST_PORT}/metrics}"
API_HEALTH_URL="${SEMOPS_API_HEALTH_URL:-http://127.0.0.1:${API_HOST_PORT}/healthz}"
API_SNAPSHOT_URL="${SEMOPS_API_SNAPSHOT_URL:-http://127.0.0.1:${API_HOST_PORT}/api/cop/snapshot}"
COP_URL="${SEMOPS_COP_URL:-http://127.0.0.1:${CADDY_HOST_PORT}}"
COP_API_SNAPSHOT_URL="${SEMOPS_COP_API_SNAPSHOT_URL:-${COP_URL}/api/cop/snapshot}"

cleanup() {
  if [[ "${SEMOPS_COP_KEEP_STACK:-false}" == "true" ]]; then
    return
  fi
  docker compose -p "$PROJECT" -f "$COMPOSE_FILE" down -v --timeout 15
}
trap cleanup EXIT

wait_http() {
  local name="$1"
  local url="$2"
  local deadline="${3:-60}"
  local start
  start="$(date +%s)"

  while true; do
    if curl -fsS "$url" >/dev/null; then
      return
    fi
    if (( "$(date +%s)" - start >= deadline )); then
      echo "Timed out waiting for ${name}: ${url}" >&2
      return 1
    fi
    sleep 2
  done
}

docker compose -p "$PROJECT" -f "$COMPOSE_FILE" up -d --build --wait

wait_http "SemStreams health" "$HEALTH_URL" 90
wait_http "SemStreams metrics" "$METRICS_URL" 90
wait_http "SemOps API health" "$API_HEALTH_URL" 90
wait_http "SemOps COP snapshot" "$API_SNAPSHOT_URL" 90
wait_http "SemOps Caddy COP UI" "$COP_URL" 90
wait_http "SemOps Caddy COP snapshot" "$COP_API_SNAPSHOT_URL" 90

SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL="$NATS_URL" \
SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL="$METRICS_URL" \
  go test ./internal/smoke/mavlink -run TestLiveGraphMAVLinkBornFirstSmoke -count=1 -v
