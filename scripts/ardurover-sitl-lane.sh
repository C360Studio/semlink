#!/usr/bin/env bash
set -euo pipefail

SIM_VEHICLE="${SIM_VEHICLE:-sim_vehicle.py}"
ARDUPILOT_FRAME="${ARDUPILOT_FRAME:-rover}"
ARDUPILOT_AIRCRAFT="${ARDUPILOT_AIRCRAFT:-semlink-ardurover}"
ARDUPILOT_SPEEDUP="${ARDUPILOT_SPEEDUP:-1}"
SEMLINK_MAVLINK_UDP_HOST="${SEMLINK_MAVLINK_UDP_HOST:-127.0.0.1}"
SEMLINK_MAVLINK_UDP_PORT="${SEMLINK_MAVLINK_UDP_PORT:-14550}"

if ! command -v "${SIM_VEHICLE}" >/dev/null 2>&1; then
  echo "sim_vehicle.py not found. Set SIM_VEHICLE=/path/to/sim_vehicle.py or add ArduPilot Tools/autotest to PATH." >&2
  exit 127
fi

exec "${SIM_VEHICLE}" \
  -v Rover \
  -f "${ARDUPILOT_FRAME}" \
  --no-mavproxy \
  "--speedup=${ARDUPILOT_SPEEDUP}" \
  --aircraft "${ARDUPILOT_AIRCRAFT}" \
  -A "--serial0=udpclient:${SEMLINK_MAVLINK_UDP_HOST}:${SEMLINK_MAVLINK_UDP_PORT}" \
  -w
