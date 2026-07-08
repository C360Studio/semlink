#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_ROOT="$(cd "$SEMLINK_ROOT/.." && pwd)"
SEMSTREAMS_ROOT="${SEMSTREAMS_ROOT:-$DEMO_ROOT/semstreams}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-semlink-blueos-smoke}"
SEMLINK_HANDOFF_PROFILE_FILE="${SEMLINK_HANDOFF_PROFILE_FILE:-$SEMLINK_ROOT/configs/handoff/companion.env.example}"

if [[ ! -f "$SEMLINK_HANDOFF_PROFILE_FILE" ]]; then
  echo "missing SemLink handoff profile: $SEMLINK_HANDOFF_PROFILE_FILE" >&2
  echo "copy configs/handoff/companion.env.example or set SEMLINK_HANDOFF_PROFILE_FILE=/path/to/profile.env" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "$SEMLINK_HANDOFF_PROFILE_FILE"
set +a

SEMLINK_BLUEOS_HOST_PORT="${SEMLINK_BLUEOS_HOST_PORT:-8081}"

if [[ ! -f "$SEMSTREAMS_ROOT/go.mod" ]]; then
  echo "missing SemStreams checkout: $SEMSTREAMS_ROOT" >&2
  echo "clone semstreams beside semlink, or set SEMSTREAMS_ROOT=/path/to/semstreams" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for the BlueOS extension lifecycle smoke" >&2
  exit 127
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for the BlueOS extension lifecycle smoke" >&2
  exit 127
fi

export SEMLINK_ROOT
export SEMSTREAMS_ROOT
export SEMLINK_HANDOFF_PROFILE_FILE
export SEMLINK_BLUEOS_HOST_PORT

cleanup() {
  docker compose -p "$COMPOSE_PROJECT_NAME" -f "$SEMLINK_ROOT/compose.blueos.yml" down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -p "$COMPOSE_PROJECT_NAME" -f "$SEMLINK_ROOT/compose.blueos.yml" \
  up -d --build --wait semlink-blueos-extension

curl -fsS "http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT}/api/health" >/dev/null
curl -fsS "http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT}/register_service" >/dev/null

echo "BlueOS extension lifecycle smoke passed at http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT}"
