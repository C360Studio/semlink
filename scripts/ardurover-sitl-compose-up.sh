#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_ROOT="$(cd "$SEMLINK_ROOT/.." && pwd)"
SEMCONNECT_ROOT="${SEMCONNECT_ROOT:-$DEMO_ROOT/semconnect}"
SEMSTREAMS_ROOT="${SEMSTREAMS_ROOT:-$DEMO_ROOT/semstreams}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-semlink-demo}"
SEMLINK_HANDOFF_PROFILE_FILE="${SEMLINK_HANDOFF_PROFILE_FILE:-$SEMLINK_ROOT/configs/handoff/companion.env.example}"
ARDUPILOT_SITL_STANDARD_FILE="${ARDUPILOT_SITL_STANDARD_FILE:-$SEMLINK_ROOT/docker/ardupilot-sitl/standard.env}"

if [[ ! -f "$SEMLINK_HANDOFF_PROFILE_FILE" ]]; then
  echo "missing SemLink handoff profile: $SEMLINK_HANDOFF_PROFILE_FILE" >&2
  echo "copy configs/handoff/companion.env.example or set SEMLINK_HANDOFF_PROFILE_FILE=/path/to/profile.env" >&2
  exit 1
fi

if [[ ! -f "$ARDUPILOT_SITL_STANDARD_FILE" ]]; then
  echo "missing ArduPilot SITL standard env: $ARDUPILOT_SITL_STANDARD_FILE" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "$ARDUPILOT_SITL_STANDARD_FILE"
set +a

set -a
# shellcheck disable=SC1090
. "$SEMLINK_HANDOFF_PROFILE_FILE"
set +a

if [[ ! -f "$SEMCONNECT_ROOT/conformance/compose.yml" ]]; then
  echo "missing SemConnect conformance compose: $SEMCONNECT_ROOT/conformance/compose.yml" >&2
  echo "clone semconnect beside semlink, or set SEMCONNECT_ROOT=/path/to/semconnect" >&2
  exit 1
fi

if [[ ! -d "$SEMCONNECT_ROOT/conformance/.vendor/semstreams" ]]; then
  echo "missing SemConnect pinned semstreams vendor tree." >&2
  echo "stage it once with:" >&2
  echo "  cd \"$SEMCONNECT_ROOT\" && KEEP_STACK=0 ./conformance/run.sh" >&2
  exit 1
fi

if [[ ! -f "$SEMSTREAMS_ROOT/go.mod" ]]; then
  echo "missing SemStreams checkout: $SEMSTREAMS_ROOT" >&2
  echo "clone semstreams beside semlink, or set SEMSTREAMS_ROOT=/path/to/semstreams" >&2
  exit 1
fi

export CS_API_HOST_PORT="${CS_API_HOST_PORT:-48080}"
export SEMLINK_ROOT
export SEMSTREAMS_ROOT
export SEMLINK_HANDOFF_PROFILE_FILE
export SEMLINK_UI_HOST_PORT="${SEMLINK_UI_HOST_PORT:-8080}"
export NATS_HOST_PORT="${NATS_HOST_PORT:-14222}"
export NATS_MON_HOST_PORT="${NATS_MON_HOST_PORT:-18222}"
export SEMLINK_NATS_HOST_PORT="${SEMLINK_NATS_HOST_PORT:-14224}"
export SEMLINK_NATS_MON_HOST_PORT="${SEMLINK_NATS_MON_HOST_PORT:-18224}"
export SEMLINK_MAVLINK_UDP_PORT="${SEMLINK_MAVLINK_UDP_PORT:-14550}"
export SEMLINK_MAVLINK_UDP_LISTEN="${SEMLINK_MAVLINK_UDP_LISTEN:-:${SEMLINK_MAVLINK_UDP_PORT}}"
export SEMLINK_MAVLINK_UDP_CONTAINER_PORT="${SEMLINK_MAVLINK_UDP_CONTAINER_PORT:-$SEMLINK_MAVLINK_UDP_PORT}"
export SEMLINK_TAK_ENABLED="${SEMLINK_TAK_ENABLED:-false}"
export SEMLINK_TAK_MULTICAST_ADDR="${SEMLINK_TAK_MULTICAST_ADDR:-239.2.3.1:6969}"
export SEMLINK_TAK_TCP_LISTEN="${SEMLINK_TAK_TCP_LISTEN:-}"
export SEMLINK_TAK_INBOUND_UDP_LISTEN="${SEMLINK_TAK_INBOUND_UDP_LISTEN:-}"
export SEMLINK_TAK_INBOUND_TCP_LISTEN="${SEMLINK_TAK_INBOUND_TCP_LISTEN:-}"
export SEMLINK_TAK_INTERVAL="${SEMLINK_TAK_INTERVAL:-1s}"

docker compose -p "$COMPOSE_PROJECT_NAME" \
  -f "$SEMCONNECT_ROOT/conformance/compose.yml" \
  -f "$SEMLINK_ROOT/docs/semconnect-csapi-port.override.yml" \
  -f "$SEMLINK_ROOT/compose.semlink.yml" \
  -f "$SEMLINK_ROOT/compose.sitl.yml" \
  up -d --build --wait nats semstreams-backend cs-api-server semlink-nats semlink ardupilot-sitl

echo "SemLink UI: http://127.0.0.1:${SEMLINK_UI_HOST_PORT}"
echo "SemConnect CS API: http://127.0.0.1:${CS_API_HOST_PORT}"
echo "ArduPilot SITL image: ${ARDUPILOT_SITL_IMAGE}:${ARDUPILOT_SITL_TAG}"
echo "ArduPilot SITL ref: ${ARDUPILOT_REF}"
echo "ArduPilot SITL frame: ${ARDUPILOT_FRAME}"
echo "SemLink MAVLink UDP listen: ${SEMLINK_MAVLINK_UDP_LISTEN}"
