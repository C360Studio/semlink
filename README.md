# SemLink

SemLink is pivoting from the implemented SemGCS ground-control demo into a
SemStreams-consuming MAVLink companion mesh service for vehicle-local state,
local rules, and intermittent peer replication. ADR 003 is the forward product
boundary; ADR 001 remains the historical boundary for the implemented SemGCS
demo.

SemLink owns MAVLink decoding, simulator/replay adapters, companion runtime,
CLI/config shape, local status/evidence APIs, and robotics language. SemOps
owns broad GCS/COP glass. semstreams-ui can provide generic ops/debug views.
SemStreams owns the semantic substrate: NATS/JetStream, graph-ingest,
`ENTITY_STATES`, mutation/query subjects, projection contracts, and
indexing-profile policy.

## Run The Demo

The current legacy demo uses Docker Compose and keeps two NATS/SemStreams
stacks:

- SemLink stack: raw MAVLink stream, current-state graph, alerts, command
  intent, local JSON/SSE APIs, and the historical Svelte demo UI.
- SemConnect stack: CS API Systems, Datastreams, Observations, SystemEvents,
  and Commands.
- HTTP bridge: curated, decimated standards projection from SemLink into
  SemConnect.

For now, SemLink layers its services on top of SemConnect's conformance Compose
file and builds against a sibling SemStreams checkout. Clone `semconnect` and
`semstreams` beside `semlink`, then run from the `semlink` checkout:

```bash
./scripts/demo-up.sh
```

Then inspect:

- SemLink local API / historical UI: `http://127.0.0.1:8080`
- SemConnect CS API: `http://127.0.0.1:48080`

The local UI-consumer contract is documented in
[`docs/evidence-api.md`](docs/evidence-api.md):

```bash
curl -s http://127.0.0.1:8080/api/evidence
```

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

The Compose demo opens inbound TAK UDP on `:6970` by default and seeds five
sample CoT events so the map and sidebar show two operators, one marker, and
observed GeoChat messages:

```bash
./scripts/demo-up.sh
```

Set `SEMLINK_TAK_SEED=false` to skip the seeded sample dots. Set
`SEMLINK_TAK_INBOUND_UDP_LISTEN=` to disable the default inbound UDP listener.
Outbound multicast remains opt-in: `SEMLINK_TAK_ENABLED=true` enables UDP
multicast to `SEMLINK_TAK_MULTICAST_ADDR` (`239.2.3.1:6969` by default). Set
`SEMLINK_TAK_TCP_LISTEN=:6969` or `SEMLINK_TAK_INBOUND_TCP_LISTEN=:6971` to
open the corresponding TCP bridge path. The start script publishes only the
host ports for non-empty listen addresses; override them with the matching
`*_HOST_PORT` or `*_CONTAINER_PORT` variables.

For the first demo, keep SemLink and SemConnect on separate NATS/SemStreams
stacks and connect them only through the CS API HTTP bridge. That makes the
boundary obvious: SemLink owns MAVLink, companion state, raw telemetry streams,
rule/command evidence, and local API contracts; SemConnect owns the
standards-facing CS API view.

## Developer Mode

For quick SemLink-only work, run without Docker Compose. This starts embedded
NATS JetStream and the SemStreams graph-ingest component in-process. The
current binary still serves the historical Svelte UI, so build `ui/dist` until
that demo surface is retired:

```bash
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Then use `http://127.0.0.1:8080` for the local API and historical UI. A shared
NATS topology is a later integration mode and should run one deliberate owner
for each SemStreams graph processor.

The demo uses a simulated MAVLink-like feed, but the frames are real unsigned
MAVLink 2 envelopes for the subset we support now: `HEARTBEAT`, `SYS_STATUS`,
and `GLOBAL_POSITION_INT`. It does not use MAVSDK.

## Spec Workflow

Product-boundary, mesh protocol, command-transmit, and SemStreams contract
changes use OpenSpec before implementation. The active pivot is tracked under
`openspec/changes/pivot-companion-mesh/`.

```bash
openspec validate --all --strict
```

## Architecture

```text
sim/PX4 adapter
  -> internal/mavlink decoder
  -> SemStreams circular buffer
  -> MAVLINK_RAW JetStream stream
  -> internal/projector current-state projection
  -> SemStreams graph.mutation.entity.create_with_triples / update_with_triples
  -> SemStreams graph.ingest.query.entity
  -> local JSON/SSE status and evidence APIs
  -> optional historical Svelte demo UI
  -> optional SemConnect CS API bridge
```

High-volume telemetry is not modeled as one graph entity per raw frame. Raw
frames stay on a bounded stream lane, while current vehicle state is projected
into one signal-profiled graph entity per vehicle. Alerts and command intents
are control-profiled graph entities.

The optional CS API bridge publishes a curated, low-rate standards view:
Systems for UAVs, Datastreams for selected telemetry rollups, OMS
Observations, SystemEvents for alerts, and Command metadata for operator
intent. Raw MAVLink frames do not pass through CS API.

The optional TAK / CoT bridge can emit the simulated swarm to TAK clients and
ingest Tier 0 situational-awareness CoT events:

```bash
go run ./cmd/semgcs-demo -embedded-nats=true -tak=true
```

`-tak=true` enables outbound UDP multicast to `239.2.3.1:6969`. Use
`-tak-tcp=:6969` for outbound TCP streaming, `-tak-inbound-udp=:6970` for
inbound UDP CoT, and `-tak-inbound-tcp=:6971` for inbound TCP CoT. Inbound
operator positions, markers, and observed GeoChat land as governed `cop.*`
graph entities and are included in the CS API bridge when `-csapi-url` is
enabled.

The local API and historical dashboard include a source-aware graph lens for the
selected vehicle or TAK COP entity: `SemLink Graph` shows the operational
SemStreams state, while `SemConnect Projection` shows the downstream CS API
materialization when `-csapi-url` is enabled.

## Roadmap

The next product slice is the ADR 003 companion-mesh pivot: several boat-local
SemLink nodes, each with a local MAVLink feed and local SemStreams state,
exchanging selected current-state summaries over an unreliable mesh harness.
SemLink should expose those facts through CLI/config and UI-consumable APIs for
SemOps or semstreams-ui, not grow its own GCS glass. ArduRover / ArduPilot SITL
without Gazebo should be the first autopilot fidelity lane; PX4 and Gazebo
remain useful later lanes when the claim needs them. The adapter boundary is
`internal/mavlink.RawFrame`; no MAVSDK or equivalent vehicle SDK is planned for
this surface.
