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
MAVLINK_UDP_HOST_PORT="${SEMOPS_MAVLINK_UDP_HOST_PORT:-14550}"
COT_UDP_HOST_PORT="${SEMOPS_COT_UDP_HOST_PORT:-18090}"

NATS_URL="${SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL:-nats://127.0.0.1:${NATS_HOST_PORT}}"
COT_NATS_URL="${SEMOPS_COT_LIVE_GRAPH_NATS_URL:-$NATS_URL}"
HEALTH_URL="${SEMOPS_SEMSTREAMS_HEALTH_URL:-http://127.0.0.1:${HEALTH_HOST_PORT}/healthz}"
METRICS_URL="${SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL:-http://127.0.0.1:${METRICS_HOST_PORT}/metrics}"
API_HEALTH_URL="${SEMOPS_API_HEALTH_URL:-http://127.0.0.1:${API_HOST_PORT}/healthz}"
API_SNAPSHOT_URL="${SEMOPS_API_SNAPSHOT_URL:-http://127.0.0.1:${API_HOST_PORT}/api/cop/snapshot}"
COP_URL="${SEMOPS_COP_URL:-http://127.0.0.1:${CADDY_HOST_PORT}}"
COP_API_SNAPSHOT_URL="${SEMOPS_COP_API_SNAPSHOT_URL:-${COP_URL}/api/cop/snapshot}"
MAVLINK_UDP_ADDR="${SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR:-127.0.0.1:${MAVLINK_UDP_HOST_PORT}}"
COT_UDP_ADDR="${SEMOPS_COP_SMOKE_COT_UDP_ADDR:-127.0.0.1:${COT_UDP_HOST_PORT}}"

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

wait_svelte_assets() {
  local name="$1"
  local url="$2"
  local deadline="${3:-60}"
  local start
  start="$(date +%s)"

  while true; do
    local html
    html="$(curl -fsS "$url")"
    local css_asset
    css_asset="$(printf '%s' "$html" | grep -Eo '(\./)?_app/immutable/assets/[^"]+\.css' | head -n 1 || true)"
    local js_asset
    js_asset="$(printf '%s' "$html" | grep -Eo '(\./)?_app/immutable/entry/[^"]+\.js' | head -n 1 || true)"
    if [[ -n "$css_asset" && -n "$js_asset" ]]; then
      css_asset="${css_asset#./}"
      js_asset="${js_asset#./}"
      if curl -fsS -I "${url%/}/$css_asset" | grep -qi 'cache-control: .*immutable' &&
        curl -fsS -I "${url%/}/$js_asset" | grep -qi 'cache-control: .*immutable'; then
        return
      fi
    fi
    if (( "$(date +%s)" - start >= deadline )); then
      echo "Timed out waiting for ${name} Svelte assets via ${url}" >&2
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
wait_svelte_assets "SemOps Caddy COP UI" "$COP_URL" 90
wait_http "SemOps Caddy COP snapshot" "$COP_API_SNAPSHOT_URL" 90

SEMOPS_COP_SMOKE_SNAPSHOT_URL="$COP_API_SNAPSHOT_URL" \
SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR="$MAVLINK_UDP_ADDR" \
SEMOPS_COP_SMOKE_COT_UDP_ADDR="$COT_UDP_ADDR" \
  go test ./internal/smoke/cop -run 'TestHostedCOPSnapshotReflects(MAVLink|CoT)UDP' -count=1 -v

SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL="$NATS_URL" \
SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL="$METRICS_URL" \
  go test ./internal/smoke/mavlink -run TestLiveGraphMAVLinkBornFirstSmoke -count=1 -v

SEMOPS_COT_LIVE_GRAPH_NATS_URL="$COT_NATS_URL" \
  go test ./internal/smoke/cot -run TestLiveGraphCoTBornFirstSmoke -count=1 -v
