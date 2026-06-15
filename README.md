# SemLink

SemLink is a SemStreams-consuming ground-control demo. It owns MAVLink decoding, simulator/replay adapters,
operator UX, and robotics language. SemStreams owns the semantic substrate: NATS/JetStream, graph-ingest,
`ENTITY_STATES`, mutation/query subjects, projection contracts, and indexing-profile policy.

The first runnable target is `cmd/semgcs-demo`. The SemLink-only demo does not
need Docker Compose because it starts an embedded NATS JetStream server and the
SemStreams graph-ingest component in-process:

```bash
cd /Users/coby/Code/c360/semlink
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Then open `http://127.0.0.1:8080`.

For the full bridge demo, use Docker Compose for the SemConnect side and keep
SemLink as a local process. SemConnect's conformance Compose stack owns the
CS API gateway and its own NATS/SemStreams backend; the override below only
publishes `cs-api-server` to the host so SemLink can reach it:

```bash
cd /Users/coby/Code/c360/semconnect
docker compose -p semconnect-semlink-demo \
  -f conformance/compose.yml \
  -f /Users/coby/Code/c360/semlink/docs/semconnect-csapi-port.override.yml \
  up -d --build --wait nats semstreams-backend cs-api-server

curl -fsS http://127.0.0.1:48080/health
```

If that direct Compose command reports a missing `conformance/.vendor/semstreams`
build context, stage SemConnect's pinned vendors by running its conformance
harness once from `/Users/coby/Code/c360/semconnect`:

```bash
KEEP_STACK=0 ./conformance/run.sh
```

Then rerun the shorter Compose command above.

With SemConnect reachable on port `48080`, run SemLink with the standards
projection enabled:

```bash
cd /Users/coby/Code/c360/semlink
go run ./cmd/semgcs-demo \
  -embedded-nats=true \
  -vehicles=12 \
  -hz=20 \
  -csapi-url=http://127.0.0.1:48080
```

Tear the SemConnect demo stack down when done:

```bash
cd /Users/coby/Code/c360/semconnect
docker compose -p semconnect-semlink-demo \
  -f conformance/compose.yml \
  -f /Users/coby/Code/c360/semlink/docs/semconnect-csapi-port.override.yml \
  down -v --remove-orphans
```

For the first demo, keep SemLink and SemConnect on separate NATS/SemStreams
stacks and connect them only through the CS API HTTP bridge. That makes the
boundary obvious: SemLink owns MAVLink, operator state, raw telemetry streams,
and command intent; SemConnect owns the standards-facing CS API view. A shared
NATS topology is a later integration mode and should run one deliberate owner
for each SemStreams graph processor.

The demo uses a simulated MAVLink-like feed, but the frames are real unsigned MAVLink 2 envelopes for the subset we
support now: `HEARTBEAT`, `SYS_STATUS`, and `GLOBAL_POSITION_INT`. It does not use MAVSDK.

## Architecture

```text
sim/PX4 adapter
  -> internal/mavlink decoder
  -> SemStreams circular buffer
  -> MAVLINK_RAW JetStream stream
  -> internal/projector current-state projection
  -> SemStreams graph.mutation.entity.create_with_triples / update_with_triples
  -> SemStreams graph.ingest.query.entity
  -> Svelte GCS dashboard
  -> optional SemConnect CS API bridge
```

High-volume telemetry is not modeled as one graph entity per raw frame. Raw frames stay on a bounded stream lane,
while current vehicle state is projected into one signal-profiled graph entity per vehicle. Alerts and command intents
are control-profiled graph entities.

The optional CS API bridge publishes a curated, low-rate standards view:
Systems for UAVs, Datastreams for selected telemetry rollups, OMS
Observations, SystemEvents for alerts, and Command metadata for operator
intent. Raw MAVLink frames do not pass through CS API.

The dashboard includes a source-aware graph lens for the selected vehicle:
`SemLink Graph` shows the operational SemStreams state, while
`SemConnect Projection` shows the downstream CS API materialization when
`-csapi-url` is enabled.

## Roadmap

PX4 SITL is the next source adapter. It should feed UDP MAVLink packets into the same decoder and projector used by
the simulator. The adapter boundary is `internal/mavlink.RawFrame`; no MAVSDK or equivalent vehicle SDK is planned.
