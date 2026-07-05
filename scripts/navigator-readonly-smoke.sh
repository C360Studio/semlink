#!/usr/bin/env bash
set -euo pipefail

BLUEOS_URL="${BLUEOS_URL:-http://blueos.local}"
ARDUPILOT_MANAGER_URL="${ARDUPILOT_MANAGER_URL:-http://blueos.local:8000}"
MAVLINK2REST_URL="${MAVLINK2REST_URL:-http://blueos.local:6040}"
SEMLINK_URL="${SEMLINK_URL:-}"
ARTIFACT_DIR="${ARTIFACT_DIR:-.artifacts/navigator-readonly-smoke}"
CURL_TIMEOUT="${CURL_TIMEOUT:-5}"

mkdir -p "$ARTIFACT_DIR"

required_get() {
  local label="$1"
  local url="$2"
  local output="$3"
  curl -fsS --max-time "$CURL_TIMEOUT" --request GET "$url" --output "$ARTIFACT_DIR/$output"
  echo "ok: $label <$url>"
}

optional_get() {
  local label="$1"
  local url="$2"
  local output="$3"
  if curl -fsS --max-time "$CURL_TIMEOUT" --request GET "$url" --output "$ARTIFACT_DIR/$output"; then
    echo "ok: $label <$url>"
  else
    echo "warn: $label unavailable <$url>" >&2
  fi
}

cat >"$ARTIFACT_DIR/README.txt" <<EOF
SemLink Navigator read-only smoke artifacts

Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
BLUEOS_URL=$BLUEOS_URL
ARDUPILOT_MANAGER_URL=$ARDUPILOT_MANAGER_URL
MAVLINK2REST_URL=$MAVLINK2REST_URL
SEMLINK_URL=$SEMLINK_URL

This smoke uses HTTP GET only. It must not change BlueOS config, autopilot
parameters, MAVLink endpoints, missions, modes, arming state, or command state.
EOF

required_get "BlueOS web root" "$BLUEOS_URL/" "blueos-root.html"
optional_get "ArduPilot Manager docs" "$ARDUPILOT_MANAGER_URL/v2.0/docs" "ardupilot-manager-docs.html"
optional_get "MAVLink2REST root" "$MAVLINK2REST_URL/" "mavlink2rest-root.txt"

if [[ -n "$SEMLINK_URL" ]]; then
  required_get "SemLink health" "$SEMLINK_URL/api/health" "semlink-health.json"
  required_get "SemLink BlueOS registration" "$SEMLINK_URL/register_service" "semlink-register-service.json"
fi

echo "Navigator read-only smoke complete. Artifacts: $ARTIFACT_DIR"
