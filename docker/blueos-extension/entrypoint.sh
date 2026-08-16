#!/usr/bin/env sh
set -eu

profile_path="${SEMLINK_HANDOFF_PROFILE:-}"
if [ -z "$profile_path" ] && [ -f /data/companion.env ]; then
  profile_path="/data/companion.env"
fi
if [ -n "$profile_path" ]; then
  if [ ! -f "$profile_path" ]; then
    echo "missing SemLink handoff profile: $profile_path" >&2
    exit 64
  fi
  set -a
  . "$profile_path"
  set +a
fi

exec /app/semgcs-demo \
  "-listen=${SEMLINK_HTTP_LISTEN:-:80}" \
  "-static=${SEMLINK_STATIC_DIR:-}" \
  "-embedded-nats=${SEMLINK_EMBEDDED_NATS:-true}" \
  "-state-dir=${SEMLINK_NATS_STATE_DIR:-/data/nats-beta160}" \
  "-nats-url=${NATS_URL:-nats://127.0.0.1:4222}" \
  "-vehicles=${SEMLINK_VEHICLES:-1}" \
  "-hz=${SEMLINK_HZ:-5}" \
  "-buffer=${SEMLINK_BUFFER:-10000}" \
  "-mavlink-udp=${SEMLINK_MAVLINK_UDP_LISTEN:-}" \
  "-csapi-url=${CS_API_URL:-}" \
  "-csapi-interval=${SEMLINK_CSAPI_INTERVAL:-2s}" \
  "-csapi-observation-interval=${SEMLINK_CSAPI_OBSERVATION_INTERVAL:-5s}" \
  "-tak=${SEMLINK_TAK_ENABLED:-false}" \
  "-tak-multicast=${SEMLINK_TAK_MULTICAST_ADDR:-239.2.3.1:6969}" \
  "-tak-tcp=${SEMLINK_TAK_TCP_LISTEN:-}" \
  "-tak-inbound-udp=${SEMLINK_TAK_INBOUND_UDP_LISTEN:-}" \
  "-tak-inbound-tcp=${SEMLINK_TAK_INBOUND_TCP_LISTEN:-}" \
  "-tak-interval=${SEMLINK_TAK_INTERVAL:-1s}"
