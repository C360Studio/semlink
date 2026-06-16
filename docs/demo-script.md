# SemGCS Demo Script

## Run

Run from the `semlink` checkout with `semconnect` and `semstreams` cloned
beside it:

```bash
./scripts/demo-up.sh
```

If the sibling checkouts live elsewhere, set `SEMCONNECT_ROOT` and
`SEMSTREAMS_ROOT` before running the script.

Open `http://127.0.0.1:8080`.

The full Compose start path opens inbound TAK UDP on `:6970` by default and
seeds sample CoT events for two operators, one marker, and observed GeoChat:

```bash
./scripts/demo-up.sh
```

Use `SEMLINK_TAK_SEED=false` to skip the sample dots, or set
`SEMLINK_TAK_INBOUND_UDP_LISTEN=` to disable the default inbound UDP listener.
Use `SEMLINK_TAK_TCP_LISTEN=:6969` for outbound TCP streaming and
`SEMLINK_TAK_INBOUND_TCP_LISTEN=:6971` for inbound TCP CoT. The start script
publishes host ports only for TAK listen addresses that are set.

If SemConnect's pinned semstreams vendor tree is missing, stage it once:

```bash
cd ../semconnect
KEEP_STACK=0 ./conformance/run.sh
cd ../semlink
./scripts/demo-up.sh
```

Use `./scripts/demo-down.sh` to tear down the stack.

## Topology

Use two NATS/SemStreams stacks for this first bridge demo:

- SemLink NATS/SemStreams stack: operator UI, raw MAVLink stream, current-state graph, alerts, commands.
- SemConnect NATS/SemStreams stack: CS API Systems, Datastreams, Observations, SystemEvents, Commands.
- HTTP bridge: decimated standards projection from SemLink into SemConnect.

Do not point SemConnect at SemLink's embedded NATS unless the SemStreams graph
backend ownership is planned explicitly. SemLink embeds `graph-ingest` for the
GCS demo; SemConnect's read endpoints expect the fuller graph backend/index
stack.

## Talk Track

1. Show the fleet map and telemetry counters.
2. Point to raw frames increasing faster than graph writes.
3. Select `UAV-001` and wait for the low-battery alert.
4. Open the `Graph` panel on `SemLink Graph` and show the selected vehicle, signal-profiled telemetry facts,
   control-profiled alert, graph revision, and indexing profile.
5. Send `Return`, `Hold`, or `Land` and show the command intent node/fact appear in the SemLink graph lens.
6. Select the TAK COP rows in the left sidebar and show that the map dots open the same SemStreams graph evidence.
7. Switch TAK COP graph views to `SemConnect Projection` and show the corresponding CS API System,
   Datastream, SamplingFeature, or GeoChat SystemEvent materialization.
8. If `-csapi-url` is enabled for a vehicle, switch the graph panel to `SemConnect Projection` and show the
   corresponding CS API System, Datastreams, Observation history, SystemEvent, ControlStream, and Command.
9. Wait for the final vehicle to enter its simulated link-loss window.
10. Use curl as backup evidence that SemConnect receives the curated standards projection:

```bash
curl -s http://127.0.0.1:48080/systems
curl -s http://127.0.0.1:48080/datastreams
curl -s http://127.0.0.1:48080/systemEvents
curl -s http://127.0.0.1:48080/commands
```

10. Query the graph lens directly for the selected vehicle:

```bash
curl -s 'http://127.0.0.1:8080/api/graph?vehicle_id=c360.semlink.robotics.fleet.drone.uav-001'
```

## Teardown

```bash
./scripts/demo-down.sh
```

## Claim

SemStreams is not the GCS. It is the semantic control plane underneath a GCS.

The demo shows raw telemetry flow, bounded projection into current state, control-plane alerts, and operator command
intent without indexing every raw frame.

With the optional SemConnect bridge, the demo also shows the standards boundary:
SemLink handles MAVLink and operator behavior, while SemConnect exposes the
selected state/events/commands as CS API resources for sponsors and integrators.
