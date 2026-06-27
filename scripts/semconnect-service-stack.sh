#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ACTION="${1:-up}"
SEMCONNECT_DIR="${SEMOPS_SEMCONNECT_DIR:-$ROOT/../semconnect}"
SEMSTREAMS_DIR="${SEMOPS_SEMSTREAMS_DIR:-$ROOT/../semstreams}"
RUN_DIR="${SEMOPS_SEMCONNECT_SERVICE_DIR:-$ROOT/tmp/semconnect-service}"
PROJECT="${SEMOPS_SEMCONNECT_PROJECT:-semops-semconnect-service}"
PROFILE="${SEMOPS_SEMCONNECT_PROFILE:-statistical}"
CS_API_HOST_PORT="${SEMOPS_SEMCONNECT_CS_API_PORT:-48080}"
CS_API_BASE_URL="${SEMOPS_SEMCONNECT_CS_API_BASE_URL:-http://127.0.0.1:${CS_API_HOST_PORT}}"

usage() {
  cat <<'USAGE'
Usage: scripts/semconnect-service-stack.sh [up|down|status|logs|help]

Starts SemConnect as a service for SemOps bridge smokes:
  - NATS with JetStream from the SemStreams tiered stack
  - SemStreams backend for graph/query subjects
  - SemConnect cs-api-server exposed on localhost

This does not start TeamEngine. Use SemConnect's conformance/run.sh only when
you need the OGC ETS conformance gate.

Environment:
  SEMOPS_SEMCONNECT_DIR            SemConnect checkout, default ../semconnect
  SEMOPS_SEMSTREAMS_DIR            SemStreams checkout, default ../semstreams
  SEMOPS_SEMCONNECT_PROFILE        statistical|semantic|structural, default statistical
  SEMOPS_SEMCONNECT_CS_API_PORT    Host port for cs-api-server, default 48080
  SEMOPS_SEMCONNECT_PROJECT        Compose project, default semops-semconnect-service
  SEMOPS_SEMCONNECT_SERVICE_DIR    Generated config dir, default tmp/semconnect-service

After up:
  go run ./cmd/semops-semconnect-fixture -base-url http://127.0.0.1:48080
USAGE
}

require_file() {
  local path="$1"
  local label="$2"
  if [[ ! -f "$path" ]]; then
    echo "Missing ${label}: ${path}" >&2
    exit 1
  fi
}

backend_service_for_profile() {
  case "$PROFILE" in
    statistical)
      printf 'semstreams'
      ;;
    semantic)
      printf 'semstreams-ml'
      ;;
    structural)
      printf 'semstreams-structural'
      ;;
    *)
      echo "SEMOPS_SEMCONNECT_PROFILE must be statistical, semantic, or structural; got ${PROFILE}" >&2
      exit 1
      ;;
  esac
}

ready_url_for_profile() {
  case "$PROFILE" in
    semantic)
      printf 'http://127.0.0.1:38180/readyz'
      ;;
    statistical|structural)
      printf 'http://127.0.0.1:38080/readyz'
      ;;
  esac
}

wait_http() {
  local name="$1"
  local url="$2"
  local deadline="${3:-120}"
  local start
  start="$(date +%s)"

  while true; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return
    fi
    if (( "$(date +%s)" - start >= deadline )); then
      echo "Timed out waiting for ${name}: ${url}" >&2
      return 1
    fi
    sleep 2
  done
}

write_run_files() {
  local backend_service="$1"
  mkdir -p "$RUN_DIR"

  cat >"$RUN_DIR/cs-api.config.json" <<'JSON'
{
  "nats_url": "nats://nats:4222",
  "bind_address": ":8080",
  "log_level": "info",
  "system_id_prefix": "c360.semconnect.systems.csapi.system",
  "datastream_id_prefix": "c360.semconnect.systems.csapi.datastream",
  "deployment_id_prefix": "c360.semconnect.systems.csapi.deployment",
  "system_event_id_prefix": "c360.semconnect.systems.csapi.systemevent"
}
JSON

  cat >"$RUN_DIR/semconnect.cs-api.override.yml" <<YAML
services:
  cs-api-server:
    build:
      context: "${SEMCONNECT_DIR}"
      dockerfile: Dockerfile
    image: semconnect/cs-api-server:semops-service
    command: ["-config", "/etc/cs-api-server/config.json"]
    depends_on:
      nats:
        condition: service_healthy
      ${backend_service}:
        condition: service_healthy
    volumes:
      - "${RUN_DIR}/cs-api.config.json:/etc/cs-api-server/config.json:ro"
    ports:
      - "${CS_API_HOST_PORT}:8080"
    networks:
      - semstreams-tiered-net
YAML
}

compose() {
  docker compose \
    -p "$PROJECT" \
    -f "$SEMSTREAMS_DIR/docker/compose/tiered.yml" \
    -f "$RUN_DIR/semconnect.cs-api.override.yml" \
    --profile "$PROFILE" \
    "$@"
}

case "$ACTION" in
  help|-h|--help)
    usage
    ;;
  up)
    require_file "$SEMCONNECT_DIR/Dockerfile" "SemConnect Dockerfile"
    require_file "$SEMSTREAMS_DIR/docker/compose/tiered.yml" "SemStreams tiered compose"
    backend_service="$(backend_service_for_profile)"
    write_run_files "$backend_service"
    compose up -d --build --wait
    wait_http "SemStreams ${PROFILE} readyz" "$(ready_url_for_profile)" 180
    wait_http "SemConnect CS API health" "${CS_API_BASE_URL}/health" 90
    echo "SemConnect CS API: ${CS_API_BASE_URL}"
    echo "Fixture smoke: go run ./cmd/semops-semconnect-fixture -base-url ${CS_API_BASE_URL}"
    echo "Generated config: ${RUN_DIR}"
    ;;
  down)
    if [[ ! -f "$RUN_DIR/semconnect.cs-api.override.yml" ]]; then
      backend_service="$(backend_service_for_profile)"
      write_run_files "$backend_service"
    fi
    compose down -v --remove-orphans
    ;;
  status)
    if [[ ! -f "$RUN_DIR/semconnect.cs-api.override.yml" ]]; then
      backend_service="$(backend_service_for_profile)"
      write_run_files "$backend_service"
    fi
    compose ps
    ;;
  logs)
    if [[ ! -f "$RUN_DIR/semconnect.cs-api.override.yml" ]]; then
      backend_service="$(backend_service_for_profile)"
      write_run_files "$backend_service"
    fi
    compose logs --no-color --tail=200 nats cs-api-server "$(backend_service_for_profile)"
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
