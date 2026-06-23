#!/usr/bin/env bash
set -euo pipefail

url="${SEMOPS_CAP_CAPTURE_URL:-}"
user_agent="${SEMOPS_CAP_CAPTURE_USER_AGENT:-}"
out_dir="${SEMOPS_CAP_CAPTURE_OUT_DIR:-fixtures/cap/nws-samples}"
out_name="${SEMOPS_CAP_CAPTURE_OUT_NAME:-}"

if [[ -z "$url" ]]; then
  echo "SEMOPS_CAP_CAPTURE_URL is required, for example https://api.weather.gov/alerts/active?area=TX" >&2
  exit 2
fi

if [[ -z "$user_agent" ]]; then
  echo "SEMOPS_CAP_CAPTURE_USER_AGENT is required; use a project/contact identity accepted by the provider" >&2
  exit 2
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 2
fi

mkdir -p "$out_dir"

if [[ -z "$out_name" ]]; then
  out_name="nws-cap-$(date -u +%Y%m%dT%H%M%SZ).xml"
fi

out_path="$out_dir/$out_name"
meta_path="$out_path.meta"

curl \
  --fail \
  --show-error \
  --location \
  --header "Accept: application/cap+xml" \
  --header "User-Agent: $user_agent" \
  --output "$out_path" \
  "$url"

{
  echo "url=$url"
  echo "user_agent=$user_agent"
  echo "captured_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$out_path" | awk '{print "sha256="$1}'
  fi
} >"$meta_path"

echo "wrote $out_path"
echo "wrote $meta_path"
