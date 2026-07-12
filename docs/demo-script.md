# SemLink Companion Evidence Demo Script

This document covers the SemLink-owned companion evidence demos and the optional
SemGCS UI-backed bridge demo. The forward SemLink product shape is a
companion service with CLI/config and UI-consumable local APIs; SemOps owns
GCS/COP glass, and semstreams-ui can provide generic ops/debug views.

## Lightweight Companion Demos

Run the repo-owned single-node companion demo without Docker, SemOps,
SemConnect, semstreams-ui, Gazebo, BlueOS, Navigator hardware, or physical
MAVLink devices:

```bash
./scripts/demo-single-companion.sh
```

The command writes `.artifacts/semlink-demo-single/report.json` by default. Set
`SEMLINK_DEMO_REPORT=/path/to/report.json` to choose a different artifact path
or `SEMLINK_DEMO_VEHICLE_PROFILE=<profile>` to label the simulated MAVLink
vehicle profile. The first supported profile is `ardurover`, but the report
shape is N-vehicle/companion-profile oriented rather than boat-only.

Set `SEMLINK_DEMO_ARTIFACT=.artifacts/semlink-demo-single/artifact.json` to
also emit a `semlink-companion-demo-artifact-v0` envelope for downstream
SemOps ingestion. The raw report remains the local proof artifact; the envelope
adds producer provenance, source fidelity, generator metadata, timestamp
coherence, and no-transmit posture around the same report.

The single-node report includes:

- deterministic companion node ID, vehicle profile, vehicle count, and graph
  entity count;
- `/api/health`, `/register_service`, and `/api/evidence` probe status;
- the local evidence bundle, including downstream optional posture;
- hardware command-transmit block evidence and separate simulator-only command
  evidence; and
- fake SemOps native readback contract evidence without a CS API hot path.

Treat a missing demo report or requested artifact as a release blocker unless a
run-specific waiver explains the environment failure.

Run the simple local mesh demo. By default it starts three companion nodes with
one simulated MAVLink vehicle per node:

```bash
./scripts/demo-mesh-companions.sh
```

The command writes `.artifacts/semlink-demo-mesh/report.json` by default. Set
`SEMLINK_DEMO_NODES=<n>`, `SEMLINK_DEMO_REPORT=/path/to/report.json`, or
`SEMLINK_DEMO_VEHICLE_PROFILE=<profile>` to adjust the run.

Set `SEMLINK_DEMO_ARTIFACT=.artifacts/semlink-demo-mesh/artifact.json` to also
emit a SemOps-ingestable artifact envelope for the mesh report. The default
artifact source fidelity is `deterministic`; SemLink will not claim
`sitl-backed` or `hardware-adjacent` fidelity unless the caller provides
per-node source metadata.

The simple mesh report includes:

- deterministic companion node IDs, vehicle counts, and static local peer URLs;
- initial and final selected-summary counts for each node;
- peer count, applied diff count, diff item count, and watermark count;
- TTL/merge posture for selected state catch-up; and
- raw MAVLink exclusion evidence that distinguishes selected summaries from
  raw frame replication.

## Artifact Envelope Fields

The optional artifact envelope has this top-level shape:

```json
{
  "artifact_kind": "semlink-companion-demo-artifact-v0",
  "generated_at": "2026-07-09T17:00:00Z",
  "source": {
    "source_fidelity": "deterministic",
    "semlink_commit": "0123456789abcdef0123456789abcdef01234567",
    "generator_command": "semlink-demo -mode mesh -vehicle-profile ardurover",
    "generator_profile": "mesh-deterministic",
    "no_transmit_posture": "SemLink demo; no SemOps or hardware transmit authority"
  },
  "report": {}
}
```

Use these environment variables to override producer metadata when invoking the
wrapper scripts:

- `SEMLINK_DEMO_ARTIFACT_SOURCE_FIDELITY`: `deterministic`,
  `sitl-backed`, or `hardware-adjacent`.
- `SEMLINK_DEMO_ARTIFACT_SEMLINK_VERSION` and
  `SEMLINK_DEMO_ARTIFACT_SEMLINK_COMMIT`: source reference fields.
  When neither is set, `cmd/semlink-demo` records the Go build's VCS revision
  or the checkout `HEAD` as `semlink_commit`. Artifact output fails if no real
  source reference is available.
- `SEMLINK_DEMO_ARTIFACT_GENERATOR_PROFILE` and
  `SEMLINK_DEMO_ARTIFACT_GENERATOR_COMMAND`: stable generator identity.
- `SEMLINK_DEMO_ARTIFACT_SIMULATOR_FAMILY`: optional simulator family label.
- `SEMLINK_DEMO_ARTIFACT_NO_TRANSMIT_POSTURE`: override no-transmit text while
  preserving the structured command-safety evidence in the embedded report.
- `SEMLINK_DEMO_ARTIFACT_NODE_SOURCES`: semicolon-separated per-node
  `key=value` sets for live-source claims, using keys `node_id`,
  `source_fidelity`, `vehicle_source`, `mavlink_system_id`,
  `simulator_family`, and `route`.

Example node-source metadata:

```bash
node_source='node_id=vehicle-node-1,source_fidelity=sitl-backed,vehicle_source=sitl'
node_source="$node_source"',mavlink_system_id=42,simulator_family=ardupilot'
SEMLINK_DEMO_ARTIFACT_SOURCE_FIDELITY=sitl-backed \
SEMLINK_DEMO_ARTIFACT_NODE_SOURCES="$node_source" \
SEMLINK_DEMO_ARTIFACT=.artifacts/semlink-demo-mesh/artifact.json \
./scripts/demo-mesh-companions.sh
```

The current lightweight demos are deterministic local harnesses. Live-source
claims are accepted only when each claimed node includes matching report node
identity and enough source metadata for SemOps to understand the evidence lane.

For the ArduPilot SITL lane, prefer the evidence-derived producer instead of
hand-authoring node metadata:

```bash
scripts/demo-sitl-artifact.sh
```

That wrapper fetches the running companion node's `/api/evidence` and emits
`.artifacts/semlink-demo-sitl/artifact.json` with
`source_fidelity=sitl-backed` only after external MAVLink/SITL evidence is
observed. It does not require SemOps, SemConnect, CS API, BlueOS, Navigator
hardware, Gazebo, or GCS glass in the producer path.

## Optional CS API Bridge Demo

This retained bridge demo proves the optional SemConnect / CS API standards
projection and the Svelte demo UI surface. It is not required for the companion
quick start, MVP hot path, or package readiness. Run from the `semlink` checkout
with `semconnect` cloned beside it:

```bash
./scripts/demo-up.sh
```

If the sibling SemConnect checkout lives elsewhere, set `SEMCONNECT_ROOT` before
running the script.

Open `http://127.0.0.1:8080` for the local API and Svelte demo UI.

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

- SemLink NATS/SemStreams stack: local API, Svelte demo UI, raw MAVLink stream,
  current-state graph, alerts, commands.
- SemConnect NATS/SemStreams stack: CS API Systems, Datastreams, Observations,
  SystemEvents, Commands.
- HTTP bridge: decimated standards projection from SemLink into SemConnect.

Do not point SemConnect at SemLink's embedded NATS unless the SemStreams graph
backend ownership is planned explicitly. SemLink embeds `graph-ingest` for the
demo runtime; SemConnect's read endpoints expect the fuller graph backend/index
stack.

## Talk Track

1. Show the fleet map and telemetry counters.
2. Point to raw frames increasing faster than graph writes.
3. Select `UAV-001` and wait for the low-battery alert.
4. Open the `Graph` panel on `SemLink Graph` and show the selected vehicle,
   signal-profiled telemetry facts, control-profiled alert, graph revision, and
   indexing profile.
5. Send `Return`, `Hold`, or `Land` and show the command intent node/fact appear
   in the SemLink graph lens.
6. Select the TAK COP rows in the left sidebar and show that the map dots open
   the same SemStreams graph evidence.
7. Switch TAK COP graph views to `SemConnect Projection` and show the
   corresponding CS API System, Datastream, SamplingFeature, or GeoChat
   SystemEvent materialization.
8. If `-csapi-url` is enabled for a vehicle, switch the graph panel to
   `SemConnect Projection` and show the corresponding CS API System,
   Datastreams, Observation history, SystemEvent, ControlStream, and Command.
9. Wait for the final vehicle to enter its simulated link-loss window.
10. Use curl as backup evidence that SemConnect receives the curated standards projection:

```bash
curl -s http://127.0.0.1:48080/systems
curl -s http://127.0.0.1:48080/datastreams
curl -s http://127.0.0.1:48080/systemEvents
curl -s http://127.0.0.1:48080/commands
```

11. Query the graph lens directly for the selected vehicle:

```bash
curl -s 'http://127.0.0.1:8080/api/graph?vehicle_id=c360.semlink.robotics.fleet.drone.uav-001'
```

12. Query the external UI evidence contract:

```bash
curl -s http://127.0.0.1:8080/api/evidence
```

Confirm that the `downstream` section marks SemOps and semstreams-ui as optional
pull consumers, SemConnect as optional standards egress, and all three as
`dependency_mode=optional-downstream`,
`required_for_readiness=false`, and `runtime_dependency=false`.

## Validation Backstop

Before presenting the companion-mesh slice as current, run the local checks:

```bash
go test ./...
go build ./...
openspec validate --all --strict
git diff --check
```

The heavier fidelity lanes stay operator-invoked because they require Docker,
SITL, companion package context, sibling checkouts, or hardware:

```bash
scripts/ardurover-sitl-lane.sh
scripts/blueos-extension-smoke.sh
scripts/navigator-readonly-smoke.sh
```

The real ArduPilot SITL e2e is also in the Go test suite, skipped unless
explicitly enabled:

```bash
SEMLINK_E2E_SITL=1 go test ./internal/e2e -run TestArduPilotSITLArtifactE2E -count=1 -v
```

Use this lane when SemLink needs to prove a real SITL-backed artifact rather
than the default deterministic or UDP-smoke evidence.

## Teardown

```bash
./scripts/demo-down.sh
```

## Claim

SemLink is not the GCS glass. It is the companion/service layer that turns
MAVLink, local rules, and selected peer state into governed evidence a GCS can
consume.

The demo shows raw telemetry flow, bounded projection into current state,
control-plane alerts, and operator command intent without indexing every raw
frame.

With the optional SemConnect bridge, the demo also shows the standards boundary:
SemLink handles MAVLink and local evidence, while SemConnect exposes the
selected state/events/commands as CS API resources for sponsors and integrators.
