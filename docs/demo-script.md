# SemGCS Demo Script

## Run

```bash
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Open `http://127.0.0.1:8080`.

## Talk Track

1. Show the fleet map and telemetry counters.
2. Point to raw frames increasing faster than graph writes.
3. Select `UAV-001` and wait for the low-battery alert.
4. Wait for the final vehicle to enter its simulated link-loss window.
5. Send `Return`, `Hold`, or `Land` and show the command intent revision.
6. Query SemStreams directly for the selected vehicle:

```bash
curl -s http://127.0.0.1:8080/api/snapshot
```

## Claim

SemStreams is not the GCS. It is the semantic control plane underneath a GCS.

The demo shows raw telemetry flow, bounded projection into current state, control-plane alerts, and operator command
intent without indexing every raw frame.
