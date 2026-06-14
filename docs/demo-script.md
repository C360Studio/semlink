# SemGCS Demo Script

## Run

```bash
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Open `http://127.0.0.1:8080`.

When SemConnect is available, enable the optional standards projection:

```bash
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20 -csapi-url=http://127.0.0.1:8081
```

Use two NATS/SemStreams stacks for this first bridge demo:

- SemLink embedded NATS/SemStreams: operator UI, raw MAVLink stream, current-state graph, alerts, commands.
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
4. Wait for the final vehicle to enter its simulated link-loss window.
5. Send `Return`, `Hold`, or `Land` and show the command intent revision.
6. If `-csapi-url` is enabled, show that SemConnect receives the curated standards projection:

```bash
curl -s http://127.0.0.1:8081/systems
curl -s http://127.0.0.1:8081/datastreams
curl -s http://127.0.0.1:8081/systemEvents
curl -s http://127.0.0.1:8081/commands
```

7. Query SemStreams directly for the selected vehicle:

```bash
curl -s http://127.0.0.1:8080/api/snapshot
```

## Claim

SemStreams is not the GCS. It is the semantic control plane underneath a GCS.

The demo shows raw telemetry flow, bounded projection into current state, control-plane alerts, and operator command
intent without indexing every raw frame.

With the optional SemConnect bridge, the demo also shows the standards boundary:
SemLink handles MAVLink and operator behavior, while SemConnect exposes the
selected state/events/commands as CS API resources for sponsors and integrators.
