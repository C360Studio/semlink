#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ARDUPILOT_SITL_STANDARD_FILE="${ARDUPILOT_SITL_STANDARD_FILE:-$SEMLINK_ROOT/docker/ardupilot-sitl/standard.env}"

if [[ ! -f "$ARDUPILOT_SITL_STANDARD_FILE" ]]; then
  echo "missing ArduPilot SITL standard env: $ARDUPILOT_SITL_STANDARD_FILE" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "$ARDUPILOT_SITL_STANDARD_FILE"
set +a

docker build \
  -f "$SEMLINK_ROOT/docker/ardupilot-sitl/Dockerfile" \
  --build-arg "ARDUPILOT_REF=$ARDUPILOT_REF" \
  -t "$ARDUPILOT_SITL_IMAGE:$ARDUPILOT_SITL_TAG" \
  "$SEMLINK_ROOT"

echo "Built ArduPilot SITL image: $ARDUPILOT_SITL_IMAGE:$ARDUPILOT_SITL_TAG"
echo "ArduPilot ref: $ARDUPILOT_REF"
