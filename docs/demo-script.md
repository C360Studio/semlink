# SemGCS Demo Script

## Run

SemLink-only:

```bash
cd /Users/coby/Code/c360/semlink
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Open `http://127.0.0.1:8080`.

Full bridge demo:

```bash
cd /Users/coby/Code/c360/semconnect
docker compose -p semconnect-semlink-demo \
  -f conformance/compose.yml \
  -f /Users/coby/Code/c360/semlink/docs/semconnect-csapi-port.override.yml \
  up -d --build --wait nats semstreams-backend cs-api-server

curl -fsS http://127.0.0.1:48080/health
```

If `conformance/.vendor/semstreams` is missing, run the SemConnect conformance
harness once to stage its pinned vendor trees:

```bash
cd /Users/coby/Code/c360/semconnect
KEEP_STACK=0 ./conformance/run.sh
```

Then rerun the shorter Compose command above.

After SemConnect is healthy, enable the optional standards projection:

```bash
cd /Users/coby/Code/c360/semlink
npm --prefix ui run build
go run ./cmd/semgcs-demo \
  -embedded-nats=true \
  -vehicles=12 \
  -hz=20 \
  -csapi-url=http://127.0.0.1:48080
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
4. Open the `Graph` panel on `SemLink Graph` and show the selected vehicle, signal-profiled telemetry facts,
   control-profiled alert, graph revision, and indexing profile.
5. Send `Return`, `Hold`, or `Land` and show the command intent node/fact appear in the SemLink graph lens.
6. If `-csapi-url` is enabled, switch the graph panel to `SemConnect Projection` and show the corresponding
   CS API System, Datastreams, Observation history, SystemEvent, ControlStream, and Command.
7. Wait for the final vehicle to enter its simulated link-loss window.
8. Use curl as backup evidence that SemConnect receives the curated standards projection:

```bash
curl -s http://127.0.0.1:48080/systems
curl -s http://127.0.0.1:48080/datastreams
curl -s http://127.0.0.1:48080/systemEvents
curl -s http://127.0.0.1:48080/commands
```

9. Query the graph lens directly for the selected vehicle:

```bash
curl -s 'http://127.0.0.1:8080/api/graph?vehicle_id=c360.semlink.robotics.fleet.drone.uav-001'
```

## Teardown

```bash
cd /Users/coby/Code/c360/semconnect
docker compose -p semconnect-semlink-demo \
  -f conformance/compose.yml \
  -f /Users/coby/Code/c360/semlink/docs/semconnect-csapi-port.override.yml \
  down -v --remove-orphans
```

## Claim

SemStreams is not the GCS. It is the semantic control plane underneath a GCS.

The demo shows raw telemetry flow, bounded projection into current state, control-plane alerts, and operator command
intent without indexing every raw frame.

With the optional SemConnect bridge, the demo also shows the standards boundary:
SemLink handles MAVLink and operator behavior, while SemConnect exposes the
selected state/events/commands as CS API resources for sponsors and integrators.
