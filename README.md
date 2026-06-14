# SemLink

SemLink is a SemStreams-consuming ground-control demo. It owns MAVLink decoding, simulator/replay adapters,
operator UX, and robotics language. SemStreams owns the semantic substrate: NATS/JetStream, graph-ingest,
`ENTITY_STATES`, mutation/query subjects, projection contracts, and indexing-profile policy.

The first runnable target is `cmd/semgcs-demo`:

```bash
cd /Users/coby/Code/c360/semlink
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Then open `http://127.0.0.1:8080`.

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
```

High-volume telemetry is not modeled as one graph entity per raw frame. Raw frames stay on a bounded stream lane,
while current vehicle state is projected into one signal-profiled graph entity per vehicle. Alerts and command intents
are control-profiled graph entities.

## Roadmap

PX4 SITL is the next source adapter. It should feed UDP MAVLink packets into the same decoder and projector used by
the simulator. The adapter boundary is `internal/mavlink.RawFrame`; no MAVSDK or equivalent vehicle SDK is planned.
