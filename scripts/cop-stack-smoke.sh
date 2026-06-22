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
SCENARIO_HOST_PORT="${SEMOPS_SCENARIO_HOST_PORT:-8090}"
FEED_FIXTURES_HOST_PORT="${SEMOPS_FEED_FIXTURES_HOST_PORT:-8091}"
MAVLINK_UDP_HOST_PORT="${SEMOPS_MAVLINK_UDP_HOST_PORT:-14550}"
COT_UDP_HOST_PORT="${SEMOPS_COT_UDP_HOST_PORT:-18090}"

NATS_URL="${SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL:-nats://127.0.0.1:${NATS_HOST_PORT}}"
COT_NATS_URL="${SEMOPS_COT_LIVE_GRAPH_NATS_URL:-$NATS_URL}"
CAP_NATS_URL="${SEMOPS_CAP_LIVE_GRAPH_NATS_URL:-$NATS_URL}"
HEALTH_URL="${SEMOPS_SEMSTREAMS_HEALTH_URL:-http://127.0.0.1:${HEALTH_HOST_PORT}/healthz}"
METRICS_URL="${SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL:-http://127.0.0.1:${METRICS_HOST_PORT}/metrics}"
API_HEALTH_URL="${SEMOPS_API_HEALTH_URL:-http://127.0.0.1:${API_HOST_PORT}/healthz}"
API_SNAPSHOT_URL="${SEMOPS_API_SNAPSHOT_URL:-http://127.0.0.1:${API_HOST_PORT}/api/cop/snapshot}"
API_RUNTIME_URL="${SEMOPS_API_RUNTIME_URL:-http://127.0.0.1:${API_HOST_PORT}/api/cop/runtime}"
API_METRICS_URL="${SEMOPS_API_METRICS_URL:-http://127.0.0.1:${API_HOST_PORT}/metrics}"
COP_URL="${SEMOPS_COP_URL:-http://127.0.0.1:${CADDY_HOST_PORT}}"
COP_API_SNAPSHOT_URL="${SEMOPS_COP_API_SNAPSHOT_URL:-${COP_URL}/api/cop/snapshot}"
COP_API_RUNTIME_URL="${SEMOPS_COP_API_RUNTIME_URL:-${COP_URL}/api/cop/runtime}"
COP_METRICS_URL="${SEMOPS_COP_METRICS_URL:-${COP_URL}/metrics}"
SCENARIO_STATUS_URL="${SEMOPS_SCENARIO_STATUS_URL:-http://127.0.0.1:${SCENARIO_HOST_PORT}/scenario/status}"
SCENARIO_STATUS_DEADLINE="${SEMOPS_SCENARIO_STATUS_DEADLINE:-120}"
SCENARIO_STATUS_STALE_AFTER="${SEMOPS_SCENARIO_STATUS_STALE_AFTER:-30}"
FEED_FIXTURES_HEALTH_URL="${SEMOPS_FEED_FIXTURES_HEALTH_URL:-http://127.0.0.1:${FEED_FIXTURES_HOST_PORT}/healthz}"
MAVLINK_UDP_ADDR="${SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR:-127.0.0.1:${MAVLINK_UDP_HOST_PORT}}"
COT_UDP_ADDR="${SEMOPS_COP_SMOKE_COT_UDP_ADDR:-127.0.0.1:${COT_UDP_HOST_PORT}}"

export SEMOPS_ADSB_ENABLED="${SEMOPS_ADSB_ENABLED:-true}"
export SEMOPS_ADSB_HTTP_URL="${SEMOPS_ADSB_HTTP_URL:-http://semops-feed-fixtures:8091/adsb/states}"
export SEMOPS_ADSB_HTTP_POLL_INTERVAL="${SEMOPS_ADSB_HTTP_POLL_INTERVAL:-1s}"
export SEMOPS_ADSB_HTTP_STALE_AFTER="${SEMOPS_ADSB_HTTP_STALE_AFTER:-10s}"
export SEMOPS_SAPIENT_ENABLED="${SEMOPS_SAPIENT_ENABLED:-true}"
export SEMOPS_SAPIENT_HTTP_URL="${SEMOPS_SAPIENT_HTTP_URL:-http://semops-feed-fixtures:8091/sapient/messages}"
export SEMOPS_SAPIENT_HTTP_POLL_INTERVAL="${SEMOPS_SAPIENT_HTTP_POLL_INTERVAL:-1s}"
export SEMOPS_SAPIENT_HTTP_STALE_AFTER="${SEMOPS_SAPIENT_HTTP_STALE_AFTER:-10s}"
export SEMOPS_SAPIENT_HTTP_ENCODING="${SEMOPS_SAPIENT_HTTP_ENCODING:-json}"

cleanup() {
  if [[ "${SEMOPS_COP_KEEP_STACK:-false}" == "true" ]]; then
    return
  fi
  docker compose -p "$PROJECT" -f "$COMPOSE_FILE" down -v --timeout 15
}
trap cleanup EXIT

print_compose_failure_diagnostics() {
  echo "Compose stack failed to become healthy; recent SemOps COP infrastructure state follows." >&2
  docker compose -p "$PROJECT" -f "$COMPOSE_FILE" ps >&2 || true
  docker compose -p "$PROJECT" -f "$COMPOSE_FILE" logs --no-color --tail=120 semops-nats semstreams >&2 || true
}

print_runtime_failure_diagnostics() {
  local reason="$1"
  local body="${2:-}"

  echo "$reason" >&2
  if [[ -n "$body" ]]; then
    echo "Last scenario status body:" >&2
    printf '%s\n' "$body" >&2
  fi
  docker compose -p "$PROJECT" -f "$COMPOSE_FILE" ps >&2 || true
  docker compose -p "$PROJECT" -f "$COMPOSE_FILE" logs --no-color --tail=120 \
    semops semops-scenario-runner semops-feed-fixtures caddy >&2 || true
}

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

json_string_field() {
  local body="$1"
  local field="$2"
  printf '%s' "$body" | sed -nE 's/.*"'"$field"'":"([^"]*)".*/\1/p'
}

json_number_field() {
  local body="$1"
  local field="$2"
  printf '%s' "$body" | sed -nE 's/.*"'"$field"'":([0-9]+).*/\1/p'
}

wait_scenario_succeeded() {
  local name="$1"
  local url="$2"
  local deadline="${3:-120}"
  local stale_after="${4:-30}"
  local start
  local last_progress_at
  local last_progress_key=""
  local last_body=""

  start="$(date +%s)"
  last_progress_at="$start"

  while true; do
    local now
    now="$(date +%s)"

    local body
    if body="$(curl -fsS "$url" 2>/dev/null)"; then
      last_body="$body"

      local state completed failed step updated_at last_error progress_key
      state="$(json_string_field "$body" state)"
      completed="$(json_number_field "$body" completed_steps)"
      failed="$(json_number_field "$body" failed_steps)"
      step="$(json_string_field "$body" current_step)"
      updated_at="$(json_string_field "$body" updated_at)"
      last_error="$(json_string_field "$body" last_error)"
      progress_key="${state}|${completed}|${failed}|${step}|${updated_at}|${last_error}"

      if [[ "$progress_key" != "$last_progress_key" ]]; then
        echo "${name} status: state=${state:-unknown} completed=${completed:-0} failed=${failed:-0} step=${step:-none} updated_at=${updated_at:-unknown}"
        last_progress_key="$progress_key"
        last_progress_at="$now"
      fi

      case "$state" in
        succeeded)
          return 0
          ;;
        failed)
          print_runtime_failure_diagnostics "${name} failed before smoke verification." "$body"
          return 1
          ;;
      esac
    else
      if [[ "$last_progress_key" != "unreachable" ]]; then
        echo "${name} status endpoint is not reachable yet: ${url}"
        last_progress_key="unreachable"
        last_progress_at="$now"
      fi
    fi

    if (( now - start >= deadline )); then
      print_runtime_failure_diagnostics "Timed out waiting for ${name}: ${url}" "$last_body"
      return 1
    fi
    if (( now - last_progress_at >= stale_after )); then
      print_runtime_failure_diagnostics "${name} made no status progress for ${stale_after}s." "$last_body"
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

if ! docker compose -p "$PROJECT" -f "$COMPOSE_FILE" up -d --build --wait; then
  print_compose_failure_diagnostics
  exit 1
fi

wait_http "SemStreams health" "$HEALTH_URL" 90
wait_http "SemStreams metrics" "$METRICS_URL" 90
wait_http "SemOps API health" "$API_HEALTH_URL" 90
wait_http "SemOps COP snapshot" "$API_SNAPSHOT_URL" 90
wait_http "SemOps COP runtime" "$API_RUNTIME_URL" 90
wait_http "SemOps component metrics" "$API_METRICS_URL" 90
wait_http "SemOps Caddy COP UI" "$COP_URL" 90
wait_svelte_assets "SemOps Caddy COP UI" "$COP_URL" 90
wait_http "SemOps Caddy COP snapshot" "$COP_API_SNAPSHOT_URL" 90
wait_http "SemOps Caddy COP runtime" "$COP_API_RUNTIME_URL" 90
wait_http "SemOps Caddy component metrics" "$COP_METRICS_URL" 90
wait_http "SemOps feed fixtures" "$FEED_FIXTURES_HEALTH_URL" 90
wait_scenario_succeeded "SemOps scenario runner" "$SCENARIO_STATUS_URL" "$SCENARIO_STATUS_DEADLINE" "$SCENARIO_STATUS_STALE_AFTER"

SEMOPS_COP_SMOKE_SNAPSHOT_URL="$COP_API_SNAPSHOT_URL" \
SEMOPS_COP_SMOKE_RUNTIME_URL="$COP_API_RUNTIME_URL" \
SEMOPS_COP_SMOKE_SCENARIO_STATUS_URL="$SCENARIO_STATUS_URL" \
SEMOPS_COP_SMOKE_COMPONENT_METRICS_URL="$COP_METRICS_URL" \
SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR="$MAVLINK_UDP_ADDR" \
SEMOPS_COP_SMOKE_COT_UDP_ADDR="$COT_UDP_ADDR" \
SEMOPS_COP_SMOKE_ADSB_HTTP_ENABLED="$SEMOPS_ADSB_ENABLED" \
SEMOPS_COP_SMOKE_SAPIENT_HTTP_ENABLED="$SEMOPS_SAPIENT_ENABLED" \
  go test ./internal/smoke/cop -run 'TestHostedCOP(SnapshotReflects(MAVLinkUDP|CoTUDP|ScenarioRunner|ADSBHTTPProvider|HADRSharedAirspace)|ComponentPrometheusMetricsReflectFeedFlow|RuntimeReflectsFeedFlow)' -count=1 -v

SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL="$NATS_URL" \
SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL="$METRICS_URL" \
  go test ./internal/smoke/mavlink -run TestLiveGraphMAVLinkBornFirstSmoke -count=1 -v

SEMOPS_COT_LIVE_GRAPH_NATS_URL="$COT_NATS_URL" \
  go test ./internal/smoke/cot -run TestLiveGraphCoTBornFirstSmoke -count=1 -v

SEMOPS_CAP_LIVE_GRAPH_NATS_URL="$CAP_NATS_URL" \
  go test ./internal/smoke/cap -run TestLiveGraphCAPBornFirstSmoke -count=1 -v

SEMOPS_SAPIENT_LIVE_PREFLIGHT_NATS_URL="$NATS_URL" \
  go test ./internal/smoke/sapient -run TestLiveSAPIENTPreflightDecodedSmoke -count=1 -v
