#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$SEMLINK_ROOT"

args=(
  go run ./cmd/semlink-demo
  -mode mesh
  -nodes "${SEMLINK_DEMO_NODES:-3}"
  -vehicle-profile "${SEMLINK_DEMO_VEHICLE_PROFILE:-ardurover}"
  -output "${SEMLINK_DEMO_REPORT:-.artifacts/semlink-demo-mesh/report.json}"
)

if [[ -n "${SEMLINK_DEMO_ARTIFACT:-}" ]]; then
  args+=(-artifact-output "$SEMLINK_DEMO_ARTIFACT")
fi
if [[ -n "${SEMLINK_DEMO_ARTIFACT_SOURCE_FIDELITY:-}" ]]; then
  args+=(-artifact-source-fidelity "$SEMLINK_DEMO_ARTIFACT_SOURCE_FIDELITY")
fi
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
if [[ -n "${SEMLINK_DEMO_ARTIFACT_NODE_SOURCES:-}" ]]; then
  IFS=';' read -r -a node_sources <<< "$SEMLINK_DEMO_ARTIFACT_NODE_SOURCES"
  for node_source in "${node_sources[@]}"; do
    [[ -n "$node_source" ]] && args+=(-artifact-node-source "$node_source")
  done
fi

"${args[@]}"
