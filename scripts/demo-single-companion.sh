#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$SEMLINK_ROOT"

go run ./cmd/semlink-demo \
  -mode single \
  -vehicle-profile "${SEMLINK_DEMO_VEHICLE_PROFILE:-ardurover}" \
  -output "${SEMLINK_DEMO_REPORT:-.artifacts/semlink-demo-single/report.json}"
