#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEMLINK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_ROOT="$(cd "$SEMLINK_ROOT/.." && pwd)"
SEMCONNECT_ROOT="${SEMCONNECT_ROOT:-$DEMO_ROOT/semconnect}"
SEMSTREAMS_ROOT="${SEMSTREAMS_ROOT:-$DEMO_ROOT/semstreams}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-semlink-demo}"

listen_port() {
    local listen="$1"
    local fallback="$2"
    if [[ "$listen" =~ :([0-9]+)$ ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo "$fallback"
    fi
}

is_falsey() {
    case "$1" in
        0 | false | FALSE | no | NO | off | OFF) return 0 ;;
        *) return 1 ;;
    esac
}

send_udp_cot() {
    local payload="$1"
    printf '%s' "$payload" | nc -u -w1 127.0.0.1 "$SEMLINK_TAK_INBOUND_UDP_HOST_PORT"
}

seed_tak_samples() {
    local now
    now="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    send_udp_cot '<event version="2.0" uid="ANDROID-ALPHA" type="a-f-G-U-C" how="m-g" time="'"$now"'"><point lat="38.8920" lon="-77.0350" hae="24" ce="5" le="5"/><detail><contact callsign="ALPHA"/></detail></event>'
    send_udp_cot '<event version="2.0" uid="ANDROID-BRAVO" type="a-f-G-U-C" how="m-g" time="'"$now"'"><point lat="38.8910" lon="-77.0410" hae="22" ce="5" le="5"/><detail><contact callsign="BRAVO"/></detail></event>'
    send_udp_cot '<event version="2.0" uid="MARKER-NORTH-GATE" type="u-d-p" how="m-g" time="'"$now"'"><point lat="38.8940" lon="-77.0380" hae="0" ce="5" le="5"/><detail><contact callsign="North Gate"/><remarks>checkpoint</remarks></detail></event>'
    send_udp_cot '<event version="2.0" uid="CHAT-ALPHA-1" type="b-t-f" how="h-g-i-g-o" time="'"$now"'"><point lat="38.8920" lon="-77.0350" hae="24" ce="5" le="5"/><detail><contact callsign="ALPHA"/><remarks>hold at checkpoint</remarks><__chat senderUid="ANDROID-ALPHA" message="hold at checkpoint"/></detail></event>'
    send_udp_cot '<event version="2.0" uid="CHAT-BRAVO-1" type="b-t-f" how="h-g-i-g-o" time="'"$now"'"><point lat="38.8910" lon="-77.0410" hae="22" ce="5" le="5"/><detail><contact callsign="BRAVO"/><remarks>copy, holding west approach</remarks><__chat senderUid="ANDROID-BRAVO" message="copy, holding west approach"/></detail></event>'
}

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
export SEMLINK_UI_HOST_PORT="${SEMLINK_UI_HOST_PORT:-8080}"
export NATS_HOST_PORT="${NATS_HOST_PORT:-14222}"
export NATS_MON_HOST_PORT="${NATS_MON_HOST_PORT:-18222}"
export SEMLINK_NATS_HOST_PORT="${SEMLINK_NATS_HOST_PORT:-14224}"
export SEMLINK_NATS_MON_HOST_PORT="${SEMLINK_NATS_MON_HOST_PORT:-18224}"
export SEMLINK_TAK_ENABLED="${SEMLINK_TAK_ENABLED:-false}"
export SEMLINK_TAK_MULTICAST_ADDR="${SEMLINK_TAK_MULTICAST_ADDR:-239.2.3.1:6969}"
export SEMLINK_TAK_TCP_LISTEN="${SEMLINK_TAK_TCP_LISTEN:-}"
export SEMLINK_TAK_INBOUND_UDP_LISTEN="${SEMLINK_TAK_INBOUND_UDP_LISTEN-:6970}"
export SEMLINK_TAK_INBOUND_TCP_LISTEN="${SEMLINK_TAK_INBOUND_TCP_LISTEN:-}"
export SEMLINK_TAK_INTERVAL="${SEMLINK_TAK_INTERVAL:-1s}"
export SEMLINK_TAK_SEED="${SEMLINK_TAK_SEED:-true}"

compose_files=(
    -f "$SEMCONNECT_ROOT/conformance/compose.yml"
    -f "$SEMLINK_ROOT/docs/semconnect-csapi-port.override.yml"
    -f "$SEMLINK_ROOT/compose.semlink.yml"
)

tak_ports=()
if [[ -n "$SEMLINK_TAK_TCP_LISTEN" ]]; then
    SEMLINK_TAK_TCP_CONTAINER_PORT="${SEMLINK_TAK_TCP_CONTAINER_PORT:-$(listen_port "$SEMLINK_TAK_TCP_LISTEN" 6969)}"
    SEMLINK_TAK_TCP_HOST_PORT="${SEMLINK_TAK_TCP_HOST_PORT:-$SEMLINK_TAK_TCP_CONTAINER_PORT}"
    export SEMLINK_TAK_TCP_CONTAINER_PORT
    export SEMLINK_TAK_TCP_HOST_PORT
    tak_ports+=("      - \"${SEMLINK_TAK_TCP_HOST_PORT}:${SEMLINK_TAK_TCP_CONTAINER_PORT}/tcp\"")
fi
if [[ -n "$SEMLINK_TAK_INBOUND_UDP_LISTEN" ]]; then
    SEMLINK_TAK_INBOUND_UDP_CONTAINER_PORT="${SEMLINK_TAK_INBOUND_UDP_CONTAINER_PORT:-$(listen_port "$SEMLINK_TAK_INBOUND_UDP_LISTEN" 6970)}"
    SEMLINK_TAK_INBOUND_UDP_HOST_PORT="${SEMLINK_TAK_INBOUND_UDP_HOST_PORT:-$SEMLINK_TAK_INBOUND_UDP_CONTAINER_PORT}"
    export SEMLINK_TAK_INBOUND_UDP_CONTAINER_PORT
    export SEMLINK_TAK_INBOUND_UDP_HOST_PORT
    tak_ports+=("      - \"${SEMLINK_TAK_INBOUND_UDP_HOST_PORT}:${SEMLINK_TAK_INBOUND_UDP_CONTAINER_PORT}/udp\"")
fi
if [[ -n "$SEMLINK_TAK_INBOUND_TCP_LISTEN" ]]; then
    SEMLINK_TAK_INBOUND_TCP_CONTAINER_PORT="${SEMLINK_TAK_INBOUND_TCP_CONTAINER_PORT:-$(listen_port "$SEMLINK_TAK_INBOUND_TCP_LISTEN" 6971)}"
    SEMLINK_TAK_INBOUND_TCP_HOST_PORT="${SEMLINK_TAK_INBOUND_TCP_HOST_PORT:-$SEMLINK_TAK_INBOUND_TCP_CONTAINER_PORT}"
    export SEMLINK_TAK_INBOUND_TCP_CONTAINER_PORT
    export SEMLINK_TAK_INBOUND_TCP_HOST_PORT
    tak_ports+=("      - \"${SEMLINK_TAK_INBOUND_TCP_HOST_PORT}:${SEMLINK_TAK_INBOUND_TCP_CONTAINER_PORT}/tcp\"")
fi

tak_ports_override=""
if ((${#tak_ports[@]} > 0)); then
    tak_ports_override="$(mktemp "${TMPDIR:-/tmp}/semlink-tak-compose.XXXXXX")"
    trap '[[ -n "${tak_ports_override:-}" ]] && rm -f "$tak_ports_override"' EXIT
    {
        echo "services:"
        echo "  semlink:"
        echo "    ports:"
        for port in "${tak_ports[@]}"; do
            echo "$port"
        done
    } > "$tak_ports_override"
    compose_files+=(-f "$tak_ports_override")
fi

docker compose -p "$COMPOSE_PROJECT_NAME" \
    "${compose_files[@]}" \
    up -d --build --wait nats semstreams-backend cs-api-server semlink-nats semlink

echo "SemLink UI: http://127.0.0.1:${SEMLINK_UI_HOST_PORT}"
echo "SemConnect CS API: http://127.0.0.1:${CS_API_HOST_PORT}"
if ! is_falsey "$SEMLINK_TAK_ENABLED"; then
    echo "TAK multicast: ${SEMLINK_TAK_MULTICAST_ADDR}"
fi
if [[ -n "${SEMLINK_TAK_TCP_HOST_PORT:-}" ]]; then
    echo "TAK outbound TCP: 127.0.0.1:${SEMLINK_TAK_TCP_HOST_PORT}"
fi
if [[ -n "${SEMLINK_TAK_INBOUND_UDP_HOST_PORT:-}" ]]; then
    echo "TAK inbound UDP: 127.0.0.1:${SEMLINK_TAK_INBOUND_UDP_HOST_PORT}"
fi
if [[ -n "${SEMLINK_TAK_INBOUND_TCP_HOST_PORT:-}" ]]; then
    echo "TAK inbound TCP: 127.0.0.1:${SEMLINK_TAK_INBOUND_TCP_HOST_PORT}"
fi
if [[ -n "${SEMLINK_TAK_INBOUND_UDP_HOST_PORT:-}" ]] && ! is_falsey "$SEMLINK_TAK_SEED"; then
    if command -v nc >/dev/null 2>&1; then
        if seed_tak_samples; then
            echo "TAK sample dots seeded: ALPHA/BRAVO operators, North Gate marker, observed GeoChat"
        else
            echo "warning: failed to seed TAK sample dots on UDP ${SEMLINK_TAK_INBOUND_UDP_HOST_PORT}" >&2
        fi
    else
        echo "warning: nc not found; skip TAK sample dot seeding" >&2
    fi
fi
