#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$SEMLINK_ROOT"

go run ./cmd/semlink-demo \
  -mode mesh \
  -nodes "${SEMLINK_DEMO_NODES:-3}" \
  -vehicles-per-node "${SEMLINK_DEMO_VEHICLES_PER_NODE:-1}" \
  -vehicle-profile "${SEMLINK_DEMO_VEHICLE_PROFILE:-ardurover}" \
  -output "${SEMLINK_DEMO_REPORT:-.artifacts/semlink-demo-mesh/report.json}"
