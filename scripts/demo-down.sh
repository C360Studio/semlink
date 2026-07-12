#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_ROOT="$(cd "$SEMLINK_ROOT/.." && pwd)"
SEMCONNECT_ROOT="${SEMCONNECT_ROOT:-$DEMO_ROOT/semconnect}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-semlink-demo}"

export SEMLINK_ROOT

if [[ ! -f "$SEMCONNECT_ROOT/conformance/compose.yml" ]]; then
    echo "missing SemConnect conformance compose: $SEMCONNECT_ROOT/conformance/compose.yml" >&2
    exit 1
fi

docker compose -p "$COMPOSE_PROJECT_NAME" \
    -f "$SEMCONNECT_ROOT/conformance/compose.yml" \
    -f "$SEMLINK_ROOT/docs/semconnect-csapi-port.override.yml" \
    -f "$SEMLINK_ROOT/compose.semlink.yml" \
    down -v --remove-orphans
