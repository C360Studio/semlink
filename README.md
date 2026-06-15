# SemLink

SemLink is a SemStreams-consuming ground-control demo. It owns MAVLink decoding, simulator/replay adapters,
operator UX, and robotics language. SemStreams owns the semantic substrate: NATS/JetStream, graph-ingest,
`ENTITY_STATES`, mutation/query subjects, projection contracts, and indexing-profile policy.

## Run The Demo

The full demo uses Docker Compose and keeps two NATS/SemStreams stacks:

- SemLink stack: raw MAVLink stream, current-state graph, alerts, command intent, and operator UI.
- SemConnect stack: CS API Systems, Datastreams, Observations, SystemEvents, and Commands.
- HTTP bridge: curated, decimated standards projection from SemLink into SemConnect.

For now, SemLink layers its services on top of SemConnect's conformance Compose
file and builds against a sibling SemStreams checkout. Clone `semconnect` and
`semstreams` beside `semlink`, then run from the `semlink` checkout:

```bash
./scripts/demo-up.sh
```

Then open:

- SemLink UI: `http://127.0.0.1:8080`
- SemConnect CS API: `http://127.0.0.1:48080`

If the script reports a missing SemConnect pinned vendor tree, stage it once:

```bash
cd ../semconnect
KEEP_STACK=0 ./conformance/run.sh
cd ../semlink
./scripts/demo-up.sh
```

If your sibling checkouts live somewhere else, set `SEMCONNECT_ROOT` and
`SEMSTREAMS_ROOT` before running the script.

Tear the stack down with:

```bash
./scripts/demo-down.sh
```

The helper scripts set safe default host ports:

- `SEMLINK_UI_HOST_PORT=8080`
- `CS_API_HOST_PORT=48080`
- `NATS_HOST_PORT=14222` for SemConnect NATS debug access
- `SEMLINK_NATS_HOST_PORT=14224` for SemLink NATS debug access

For the first demo, keep SemLink and SemConnect on separate NATS/SemStreams
stacks and connect them only through the CS API HTTP bridge. That makes the
boundary obvious: SemLink owns MAVLink, operator state, raw telemetry streams,
and command intent; SemConnect owns the standards-facing CS API view.

## Developer Mode

For quick SemLink-only work, run without Docker Compose. This starts embedded
NATS JetStream and the SemStreams graph-ingest component in-process:

```bash
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Then open `http://127.0.0.1:8080`. A shared NATS topology is a later
integration mode and should run one deliberate owner for each SemStreams graph
processor.

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
