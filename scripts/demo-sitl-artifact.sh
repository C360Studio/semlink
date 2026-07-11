#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$SEMLINK_ROOT"

runtime_url="${SEMLINK_RUNTIME_URL:-http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}}"

args=(
  go run ./cmd/semlink-demo
  -mode sitl-artifact
  -runtime-url "$runtime_url"
  -vehicle-profile "${SEMLINK_DEMO_VEHICLE_PROFILE:-ardurover}"
  -output "${SEMLINK_SITL_DEMO_REPORT:-.artifacts/semlink-demo-sitl/report.json}"
  -artifact-output "${SEMLINK_SITL_DEMO_ARTIFACT:-.artifacts/semlink-demo-sitl/artifact.json}"
)

if [[ -n "${SEMLINK_DEMO_ARTIFACT_SEMLINK_VERSION:-}" ]]; then
  args+=(-artifact-semlink-version "$SEMLINK_DEMO_ARTIFACT_SEMLINK_VERSION")
fi
if [[ -n "${SEMLINK_DEMO_ARTIFACT_SEMLINK_COMMIT:-}" ]]; then
  args+=(-artifact-semlink-commit "$SEMLINK_DEMO_ARTIFACT_SEMLINK_COMMIT")
fi
if [[ -n "${SEMLINK_DEMO_ARTIFACT_GENERATOR_PROFILE:-}" ]]; then
  args+=(-artifact-generator-profile "$SEMLINK_DEMO_ARTIFACT_GENERATOR_PROFILE")
fi
if [[ -n "${SEMLINK_DEMO_ARTIFACT_GENERATOR_COMMAND:-}" ]]; then
  args+=(-artifact-generator-command "$SEMLINK_DEMO_ARTIFACT_GENERATOR_COMMAND")
fi
if [[ -n "${SEMLINK_DEMO_ARTIFACT_SIMULATOR_FAMILY:-}" ]]; then
  args+=(-artifact-simulator-family "$SEMLINK_DEMO_ARTIFACT_SIMULATOR_FAMILY")
fi
if [[ -n "${SEMLINK_DEMO_ARTIFACT_NO_TRANSMIT_POSTURE:-}" ]]; then
  args+=(-artifact-no-transmit-posture "$SEMLINK_DEMO_ARTIFACT_NO_TRANSMIT_POSTURE")
fi
if [[ -n "${SEMLINK_SITL_VEHICLE_SOURCE:-}" ]]; then
  args+=(-artifact-vehicle-source "$SEMLINK_SITL_VEHICLE_SOURCE")
fi
if [[ -n "${SEMLINK_SITL_ARTIFACT_ROUTE:-}" ]]; then
  args+=(-artifact-route "$SEMLINK_SITL_ARTIFACT_ROUTE")
fi

"${args[@]}"
