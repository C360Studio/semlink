# SemLink

SemLink is a SemStreams-consuming MAVLink companion mesh service for
vehicle-local state, local rules, and intermittent peer replication. It grew out
of the implemented SemGCS ground-control demo; ADR 003 is the forward product
boundary, and ADR 001 records the original SemGCS demo boundary.

SemLink owns MAVLink decoding, simulator/replay adapters, companion runtime,
CLI/config shape, local status/evidence APIs, and robotics language. SemOps
owns broad GCS/COP glass. semstreams-ui can provide generic ops/debug views.
SemStreams owns the semantic substrate: NATS/JetStream, graph-ingest,
`ENTITY_STATES`, canonical mutation and authoritative-read APIs, projection
contracts, and indexing-profile policy. Projection contracts validate producer
intent and graph shape; they do not reserve predicates or authorize writes.

## Product Boundary

SemLink is BlueOS-compatible, not BlueOS-dependent. Treat BlueOS as the
vehicle appliance layer: useful for Navigator/Pi bring-up, firmware and
parameter workflows, networking, logs, MAVLink routing, camera/video services,
and extension lifecycle on Blue Robotics vehicles. SemLink does not try to
replace that full appliance surface for the MVP.

SemLink's lane is the vehicle-local semantic companion: native MAVLink ingest,
bounded raw-frame handling, current-state projection, local rule/command
evidence, optional selected-state mesh replication, and UI-consumable local
APIs. BlueOS registration and package lifecycle are deployment metadata.
MAVLink UDP, replay, and ArduPilot SITL remain the compatibility proof paths.

SemOps is the fleet COP/GCS glass and command-governance consumer. SemConnect
is the optional standards edge. Neither BlueOS REST/MAVLink2REST nor CS API is
part of the SemLink hot path for the companion MVP.

## Quick Start

Start with the repo-owned companion proofs. Beyond the Go toolchain, these
commands do not require Docker, BlueOS, Navigator hardware, Gazebo, SemOps,
SemConnect, semstreams-ui, SITL, or physical MAVLink devices:

```bash
go test ./internal/e2e -count=1
./scripts/demo-single-companion.sh
./scripts/demo-mesh-companions.sh
```

The demo scripts write JSON reports under `.artifacts/`. Add
`SEMLINK_DEMO_ARTIFACT=<path>` when you also want the downstream
`semlink-companion-demo-artifact-v0` envelope for SemOps ingestion:

```bash
SEMLINK_DEMO_ARTIFACT=.artifacts/semlink-demo-single/artifact.json \
  ./scripts/demo-single-companion.sh
```

Use the path that matches the claim you need to prove:

- Fast local confidence: `go test ./internal/e2e -count=1`
- Single-node evidence report: `./scripts/demo-single-companion.sh`
- Multi-companion mesh evidence report: `./scripts/demo-mesh-companions.sh`
- UI or downstream consumer contract:
  [`docs/evidence-api.md`](docs/evidence-api.md)
- SemStreams beta.160 schema, restart, and predicate migration:
  [`docs/semstreams-beta160-migration.md`](docs/semstreams-beta160-migration.md)
- ArduPilot SITL artifact proof:
  [`docs/sitl-ardurover.md`](docs/sitl-ardurover.md)
- BlueOS-style package lifecycle:
  [`docs/blueos-extension.md`](docs/blueos-extension.md)
- Navigator-class read-only hardware smoke:
  [`docs/navigator-readonly-smoke.md`](docs/navigator-readonly-smoke.md)
- Optional SemConnect / CS API bridge demo:
  [`docs/demo-script.md`](docs/demo-script.md)

## Optional CS API Bridge Demo

This retained bridge demo uses Docker Compose to prove the optional SemConnect / CS API standards
projection and the Svelte demo UI surface. It is not the SemLink quick-start path,
MVP companion hot path, or package-readiness dependency. It keeps two
NATS/SemStreams stacks:

- SemLink stack: raw MAVLink stream, current-state graph, alerts, command
  intent, local JSON/SSE APIs, and the Svelte demo UI.
- SemConnect stack: CS API Systems, Datastreams, Observations, SystemEvents,
  and Commands.
- HTTP bridge: curated, decimated standards projection from SemLink into
  SemConnect.

To run this optional bridge, clone `semconnect` beside `semlink`, then run from
the `semlink` checkout:

```bash
./scripts/demo-up.sh
```

This supported launcher performs a normal idempotent deployment. It preserves
the `semlink-nats` container and beta.160-specific named volume, so graceful or
unexpected restarts reopen exact stamped beta.160 state. An empty namespace is
required only for first initialization or an explicit incompatible-schema
reset; see the beta.160 migration guide above.

Then inspect:

- SemLink local API / Svelte demo UI: `http://127.0.0.1:8080`
- SemConnect CS API: `http://127.0.0.1:48080`

The local UI-consumer contract is documented in
[`docs/evidence-api.md`](docs/evidence-api.md):

```bash
curl -s http://127.0.0.1:8080/api/evidence
```

The evidence bundle includes downstream metadata for SemOps, semstreams-ui, and
the optional SemConnect CS API bridge.

The BlueOS-style companion handoff package and its release/tag checkpoint are
documented in [`docs/blueos-extension.md`](docs/blueos-extension.md). Start
there for `scripts/blueos-extension-smoke.sh`, required handoff profile inputs,
and the evidence gates for a package checkpoint.

If the script reports a missing SemConnect pinned vendor tree, stage it once:

```bash
cd ../semconnect
KEEP_STACK=0 ./conformance/run.sh
cd ../semlink
./scripts/demo-up.sh
```

If the sibling SemConnect checkout lives somewhere else, set `SEMCONNECT_ROOT`
before running the script.

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
current binary serves the Svelte demo UI, so build `ui/dist` when you want that
local dashboard. This explicit development command omits `StateDir`, so it uses
an owned temporary JetStream store:

```bash
npm --prefix ui install
npm --prefix ui run build
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20
```

Then use `http://127.0.0.1:8080` for the local API and Svelte demo UI. A shared
NATS topology is a later integration mode and should run one deliberate active
instance of each SemStreams graph processor. Set `SEMLINK_NATS_STATE_DIR` for
persistent embedded state; packaged profiles default to `/data/nats-beta160`.
External mode uses the same durable schema stamp. First initialization requires
an empty namespace, while later exact beta.160 starts validate and reuse it.

The demo uses a simulated MAVLink-like feed, but the frames are real unsigned
MAVLink 2 envelopes for the subset we support now: `HEARTBEAT`, `SYS_STATUS`,
and `GLOBAL_POSITION_INT`. It does not use MAVSDK.

## Spec Workflow

Product-boundary, mesh protocol, command-transmit, and SemStreams contract
changes use OpenSpec before implementation. The accepted companion-mesh
baseline is now tracked in main specs:

- `openspec/specs/companion-mesh-product/spec.md`
- `openspec/specs/mavlink-companion-runtime/spec.md`
- `openspec/specs/mesh-synchronization/spec.md`
- `openspec/specs/rules-and-command-safety/spec.md`
- `openspec/specs/companion-e2e-demo/spec.md`

The completed pivot change is archived under
`openspec/changes/archive/2026-07-07-pivot-companion-mesh/`.

```bash
openspec validate --all --strict
```

## Architecture

```text
simulator / replay / ArduPilot SITL / MAVLink UDP input
  -> internal/mavlink decoder
  -> SemStreams circular buffer
  -> MAVLINK_RAW JetStream stream
  -> internal/projector current-state projection
  -> SemStreams projection.MutationClient named-group reconcile
  -> strict zero-triple envelope create when the entity is absent
  -> SemStreams authoritative exact-entity read + same-entry KV revision
  -> local JSON/SSE status and evidence APIs
  -> optional Svelte demo UI
  -> optional SemConnect CS API bridge
```

High-volume telemetry is not modeled as one graph entity per raw frame. Raw
frames stay on a bounded stream lane, while current vehicle state is projected
into one signal-profiled graph entity per vehicle. Alerts and command intents
are control-profiled graph entities.

The optional CS API bridge publishes a curated, low-rate standards view:
Systems for vehicles, Datastreams for selected telemetry rollups, OMS
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

The local API and Svelte demo dashboard include a source-aware graph lens for the
selected vehicle or TAK COP entity: `SemLink Graph` shows the operational
SemStreams state, while `SemConnect Projection` shows the downstream CS API
materialization when `-csapi-url` is enabled.

## Roadmap

The ADR 003 companion-mesh pivot is now the accepted baseline: several
vehicle-local SemLink nodes, each with a local MAVLink feed and local SemStreams
state, exchanging selected current-state summaries over unreliable links.
SemLink exposes those facts through CLI/config and UI-consumable APIs for
SemOps or semstreams-ui, not through repo-owned GCS glass.

The next governed changes should stay narrow: hardware command authorization,
external consumer integration, stronger SITL/BlueOS fidelity, or a shared
SemOps/SemLink MAVLink package once duplication pressure is proven. The adapter
boundary remains `internal/mavlink.RawFrame`; no MAVSDK or equivalent vehicle
SDK is planned for this surface.
