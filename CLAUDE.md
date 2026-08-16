# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

SemLink is pivoting from an implemented **SemStreams-consuming ground-control
(GCS) demo** into a MAVLink companion mesh service. The current demo proves that
high-volume MAVLink telemetry can be handled with a semantic control plane
without making the substrate own robotics concerns. The current single binary
(`cmd/semgcs-demo`) still runs the simulator, decoder, projector, an in-memory
store, and an HTTP server that serves a JSON/SSE API plus the Svelte demo
dashboard.

The forward product boundary is ADR 003:
`docs/adr/003-companion-mesh-product-boundary.md`. ADR 001 records the original
SemGCS demo boundary.

- **SemLink owns** MAVLink decoding, source adapters, vehicle-local companion
  service behavior, local rules, mesh-visible current-state summaries,
  CLI/config shape, local status/evidence APIs, command vocabulary, and the
  robotics semantic predicates.
- **SemStreams owns** the substrate: NATS/JetStream, the `graph-ingest` processor,
  `ENTITY_STATES`, canonical mutation and authoritative-read APIs, projection contracts,
  and indexing profiles. Contracts validate producer intent and shape; they do not reserve
  predicates or authorize writes. SemLink only *writes through* the substrate.
- **SemOps** owns the kitchen-sink COP / fusion product surface and GCS glass.
- **semstreams-ui** can consume SemLink evidence for generic ops/debug views.
- **SemConnect** (optional, downstream) receives a curated, decimated OGC Connected
  Systems (CS API) view over HTTP. Raw MAVLink never flows through SemConnect.

## Dependencies

SemLink consumes SemStreams `v1.0.0-beta.160` at tag commit
`8403a2218000e45a31c5132fbfe01af42ed04f14`; keep normal development on that
pin and follow `docs/semstreams-beta160-migration.md` for state or predicate
cutovers. The optional SemConnect / CS API
bridge demo needs a sibling `../semconnect` checkout. Override that location
with `SEMCONNECT_ROOT`.

## Commands

```bash
# Build the Svelte demo UI when you want the Go server to serve ui/dist.
npm --prefix ui install
npm --prefix ui run build

# Run dev mode: embedded in-process NATS JetStream + graph-ingest, no Docker
go run ./cmd/semgcs-demo -embedded-nats=true -vehicles=12 -hz=20   # local API / Svelte UI at :8080
curl -s http://127.0.0.1:8080/api/evidence  # external UI evidence contract

# Go tests (pure unit tests with fakes; no Docker needed)
go test ./...
go test ./internal/projector -run TestProjectorCollapsesRawMessagesToCurrentVehicleEntity

# Svelte UI type/lint check, and a hot-reloading UI dev server (proxies /api to :8080)
npm --prefix ui run check
npm --prefix ui run dev   # Vite at :5173

# Full two-stack demo via Docker Compose (needs ../semconnect and Docker)
./scripts/demo-up.sh      # preserves beta.160 NATS state during normal deployment
./scripts/demo-down.sh

# OpenSpec governance for product-boundary or contract-sized changes
openspec validate --all --strict
openspec validate pivot-companion-mesh --strict
```

The Go unit tests use in-memory fakes (e.g. `fakeMutationClient` in
`internal/semstreams/client_test.go`); there is no testcontainers/Docker requirement for
`go test`.

## Data Flow (the core pipeline)

```
mavlink.Simulator (sim.go)            internal/gcs/demo.go owns the goroutines:
  -> mavlink.RawFrame                   runSimulator   -> circular buffer (DropOldest)
  -> circular buffer (bounded)          runProjector   -> drain batch, decode, project, write
  -> internal/mavlink.DecodeMessage     runLinkMonitor -> derive lost-link from silence
  -> internal/projector.Apply           +  publish raw bytes to MAVLINK_RAW JetStream lane
  -> projector.Projection (entity+triples+profile+contract+group)
  -> semstreams.GraphClient.UpsertProjection
       -> projection.MutationClient named-group reconcile
       -> zero-triple strict create only for a proven-absent entity
       -> authoritative exact read + same-entry KV revision
  -> gcs.Store (in-memory snapshot)  -> HTTP /api/snapshot + /api/events (SSE)
                                      -> Svelte demo UI / external consumers
  -> (optional) csapi.Bridge          -> decimated CS API POSTs to SemConnect
```

**Key modeling decision:** raw frames are *not* one graph entity per frame. High-rate
frames live on a bounded JetStream lane (`MAVLINK_RAW`, file storage, 5-min/250k cap);
current vehicle state is collapsed into **one signal-profiled graph entity per vehicle**;
alerts and command intents are **control-profiled** graph entities. This signal-vs-control
split is the whole point of the demo — preserve it.

## Package Map

- **`cmd/semgcs-demo`** — flags and wiring only. Starts the runtime, store, optional CS API
  bridge, demo goroutines, command service, and HTTP server.
- **`internal/mavlink`** — self-contained MAVLink 2 subset codec (unsigned frames only):
  `HEARTBEAT`, `SYS_STATUS`, `GLOBAL_POSITION_INT`, with X25 CRC (`crc.go`) and the
  deterministic `Simulator`. `RawFrame` is the **source-adapter boundary** — new inputs
  (PX4 SITL over UDP) should produce `RawFrame`s and feed the same decoder/projector. No
  MAVSDK.
- **`internal/projector`** — collapses decoded messages into per-vehicle current state and
  emits `Projection`s. This is where the **semantic vocabulary lives**: `predicates.go`
  (`robot.*` predicate IRIs + source constants), `contracts.go` (named reconcile groups +
  indexing profiles), `types.go` (payload → triples). `Apply` updates state and raises
  low-battery alerts; `CheckLinkTimeouts` derives lost-link alerts from silence.
- **`internal/semstreams`** — substrate bootstrap and client. `runtime.go` starts embedded
  NATS (ephemeral port `-1`), validates all contracts, ensures bounded consumer streams,
  and starts the `graph-ingest` processor in-process. `client.go` reconciles first, creates
  a zero-triple envelope for proven absence, resolves typed outcomes, and uses authoritative
  exact reads. Graph-ingest owns `ENTITY_SUFFIX_INDEX`; `COMPONENT_STATUS` is removed.
  Startup CAS-stamps an empty namespace at
  `SEMLINK_RUNTIME_META/state-schema-version=beta.160`, then later exact-version starts
  validate and reopen durable state. Missing or different stamps fail closed unchanged.
- **`internal/gcs`** — demo orchestration plus local
  status/evidence API. `demo.go` (the 3 goroutines above), `store.go`
  (thread-safe in-memory snapshot + metrics), `server.go` (HTTP:
  `/api/evidence`, `/api/snapshot`, `/api/events` SSE, `/api/graph`,
  `/api/commands`, static Svelte UI), `evidence.go` (versioned bundle for
  SemOps/semstreams-ui consumers), `commands.go` (operator intent →
  control-profiled graph write), `graph_view.go` (the **source-aware graph
  lens**: builds a SemLink operational lens from the graph + snapshot, and a
  SemConnect lens by querying the live CS API).
- **`internal/csapi`** — optional downstream `Bridge`. Polls the store snapshot and POSTs a
  decimated standards view to SemConnect: Systems, Datastreams, Observations (OM-JSON),
  SystemEvents, ControlStreams, Commands. Tracks what it has already posted to stay
  idempotent.
- **`ui/`** — Svelte 5 (runes: `$state`/`$derived`/`$effect`) + Vite
  + TypeScript dashboard. It consumes the JSON/SSE API; built to `ui/dist` and
  served statically by the Go server until the demo UI is retired.

## Conventions & Gotchas

- **OpenSpec:** large product-boundary changes, mesh protocol changes, command-transmit
  changes, and SemStreams contract migrations should start under `openspec/changes/`.
  The active substrate migration is `openspec/changes/migrate-semstreams-beta-160/`.
- **Entity IDs** are dotted, hierarchical, and parsed by convention:
  `c360.semlink.robotics.fleet.drone.uav-NNN`, `...fleet.alert.<kind>-uav-NNN`,
  `...fleet.command.<verb>-uav-NNN-<unixmilli>`. `systemIDFromEntity` extracts the `uav-NNN`
  number, so keep the `uav-NNN` token intact when adding ID forms.
- **Indexing profiles** are deliberate: telemetry current state = `signal`, alerts/commands
  = `control`. Declare any new graph footprint in `projector.Contracts()` and register its
  vocabulary and payload before startup. The complete contract set is validated with
  `projection.ValidateContracts`; every current group has a stable name and
  `projection.ModeReconcile`.
- **The simulator is scripted for the demo narrative**: the last vehicle periodically drops
  all frames (exercises lost-link detection via `runLinkMonitor`), and vehicle 1's battery
  is forced low after ~18s (exercises the low-battery alert). Don't "fix" these as bugs.
- **`go.mod` targets Go 1.26.3.** Embedded NATS binds an ephemeral port and writes to a temp
  `StateDir` only when the caller omits the setting for explicit development/tests.
  Packaged and BlueOS profiles use `SEMLINK_NATS_STATE_DIR=/data/nats-beta160`; configured
  state is never deleted by `Stop`. Compose uses the persistent
  `semlink-nats-beta160-data` volume. Empty state is required only for first beta.160
  initialization; there is no alpha/beta transform, copy, adoption, upgrade, or downgrade.
- **Dependency posture:** prefer hand-rolled, scoped implementations over libraries unless a dep is mature and
  scope-aligned. Precedent: ADR 001 (no MAVSDK; hand-wrote the MAVLink subset) and
  `docs/adr/002-tak-cot-bridge.md` (hand-roll the CoT codec; `cotlib` is optional reference only; `kdudkov/goatak`
  is AGPL-3.0 and is clean-room reference only — never imported or copied).
- Use **conventional commits** (`feat(scope):`, `fix(scope):`, `docs(scope):`).
