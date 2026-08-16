# ADR 002: TAK / CoT Interoperability Bridge

## Status

Accepted for the SemLink TAK / CoT bridge. Broad SemLink-as-COP product
positioning is superseded by [ADR 003](003-companion-mesh-product-boundary.md).

Outbound first (Phase 0) remains decided. The `cop.*` entity/domain naming below
still applies to the TAK / CoT bridge code paths that live in SemLink, but it no
longer implies that SemLink should become the kitchen-sink COP. SemOps now owns
that COP / fusion product surface.

## Context

SemLink already projects per-vehicle position, heading, speed, battery, link status, alerts, and command intent into a
governed in-memory store (`internal/gcs.Store`) and through SemStreams graph mutations. That is precisely the data a TAK
Common Operating Picture (COP) consumes. TAK (Team Awareness Kit: ATAK/iTAK/WinTAK) is the de-facto tactical edge client,
free and widely deployed, but the TAK situational-awareness bus is flat and ephemeral: no provenance, no history API, no
semantic typing, no standards query. SemConnect's OGC API - Connected Systems (CS API) is the opposite — governed,
queryable, standards-facing — but it is not an operator UX.

Adding TAK support lets SemLink bridge three surfaces: MAVLink (machine telemetry), TAK (humans in the field), and
CS API (standards / enterprise), with the SemStreams semantic knowledge graph (SKG) as the broker in the middle.

We evaluated `kdudkov/goatak`, a mature Go TAK server/client. It reaches a high tier (v1 XML + v2 protobuf, certificate
enrollment, mission packages, video, user management, visibility scopes) and proves the full ladder is tractable in Go.
It is licensed **AGPL-3.0** (strong copyleft with the network-server clause). Importing it would subject SemLink to AGPL,
including source disclosure to network users; copying its code would create derivative-work exposure.

## Decision

### Scope: Tier 0 situational-awareness relay only

SemLink adds a TAK/CoT interoperability surface limited to **Tier 0**: emit and ingest Cursor-on-Target (CoT) events over
plain transports. We explicitly do **not** build a full TAK Server — no mission / data-sync (Marti), federation, video,
or user management. If TLS and certificate enrollment are needed later (Tier 1), we delegate the certificate authority to
**step-ca (Smallstep)** and wrap enrollment rather than reimplementing PKI.

**Tasking is out of scope.** No phase implements a CoT tasking round-trip (CoT tasking → `…fleet.command.*`, or operator
command intent → outbound CoT). Bridging SemLink's existing command intent to TAK tasking is deferred because it needs
its own decisions — command authority / authorization, loop prevention (do not re-emit a command just ingested), and
status / ack semantics — that Tier 0 SA relay does not. The outbound bridge emits only vehicle tracks and alerts; the
inbound path ingests only operator position, markers, and geochat.

### Clean-room with respect to goatak

`goatak` is AGPL-3.0. We do not import it, copy from it, or study its source or package structure as a design input.
The only use we make of it is the high-level confirmation, from its feature list, that the full tier ladder is feasible
in Go. The clean-room source of truth is public artifacts only:

- the MITRE Cursor-on-Target base XML schema (the `event` / `point` / `detail` structure);
- the public DoD `takproto` `.proto` definitions for the v1 protobuf payload (deferred to Phase 2); and
- CoT wire captures / test vectors recorded from a real ATAK / iTAK client.

Any study of `goatak` beyond confirming feasibility requires legal review first.

### Dependency posture: hand-roll the CoT subset

Consistent with ADR 001's "no MAVSDK" decision — where we hand-wrote a MAVLink v2 subset decoder rather than depend on a
vehicle SDK — we hand-roll a minimal CoT codec in `internal/cot` covering only the event shapes the demo emits and
ingests (track, alert / emergency, marker, geochat). We treat `cotlib` (MIT) as an optional reference for the
MIL-STD-2525 type catalog, not a default dependency, and reconsider adopting it only if our CoT surface outgrows the demo
subset. Protobuf v1 (`TakMessage`) is deferred to Phase 2 and, when added, is generated from the public `.proto` using
`google.golang.org/protobuf`, which is already an indirect dependency. Transport (`internal/tak`) uses the standard
library `net` package. We do not adopt goatak's Fiber / GORM / Koanf stack; the new packages stay as dependency-light as
the rest of the codebase.

### Package map

The inbound path is real work, not a bucket. It decomposes into single-responsibility packages, each mirroring the
existing MAVLink structure, introduced per phase rather than all at once. We deliberately reject a single
`internal/field` "non-MAVLink inputs" package — it is named by what it is not and hides the actual work.

| Package | Responsibility | MAVLink parallel | Introduced |
| --- | --- | --- | --- |
| `internal/cot` | CoT codec + MIL-STD-2525 type mapping (bytes ↔ event model) | `internal/mavlink` | Phase 0 |
| `internal/tak` | TAK transport + bridge: UDP multicast / TCP streaming, client lifecycle, outbound fan-out (reads `gcs.Store`), inbound listen | (new) | Phase 0 out, Phase 1 in |
| `internal/graphprojection` | neutral, source-agnostic SemStreams projection primitives, extracted from `projector` | (extraction) | Phase 1 prerequisite |
| `internal/cop` | COP domain: `cop.*` payloads, predicates, contracts, and the `uid`-keyed CoT → projection translator | `internal/projector` | Phase 1 |

`internal/graphprojection` is named to avoid colliding with the SemStreams `pkg/projection` already imported by
`internal/projector/contracts.go`.

### Why `graphprojection` must be extracted (not deferred)

The reuse the inbound path needs is real but not yet accessible. `Projection` is exported, but `triple` and
`projectionFromPayload` are **unexported** in `internal/projector/types.go`, and the `projector` package imports
`internal/mavlink` (in `projector.go`) and owns the accumulator. So `internal/cop` cannot reuse those helpers without
duplicating them or dragging in MAVLink. Extracting an exported, MAVLink-free `internal/graphprojection` (`Projection`, a
triple builder, `ProjectionFromPayload`, the payload interface) is therefore a **prerequisite of Phase 1**, with
`internal/projector` refactored to import it. This corrects this ADR's earlier "optional, can wait" framing.

### Inbound CoT projection path

Field clients' position reports, markers, and geochat are ingested via `internal/cot` decode → `internal/cop` translate →
`semstreams.GraphClient`, landing as governed SKG entities under six-part `c360.semlink.cop.*` IDs (operator,
marker, message).
Because a CoT event is a self-contained snapshot keyed by string `uid` (identity, position, and course in one message)
rather than fragments to accumulate, the `cop` translator is a thin `uid`-keyed mapper plus a last-seen / stale tracker —
much lighter than `projector`'s `vehicleAccumulator`. The MAVLink accumulator is untouched.

### `cop` contracts and registration

`internal/cop` mirrors `internal/projector`'s graph-footprint surface, not just its payloads. It declares
`cop.Contracts()` with stable named reconcile groups and indexing profiles, registers its canonical vocabulary, and
implements `cop.RegisterPayloads`. Beta.160 removed semantic ownership from projection contracts. The runtime
registers COP vocabulary and payloads before validating the complete contract set and starting graph-ingest.

Indexing profiles are explicit per kind, following the SemStreams profile semantics — `signal` is telemetry / readings,
`content` is the retrieval corpus for prose-bearing / domain context, and `control` is durable, low-cardinality
lifecycle / control machinery:

- **operator** → `signal` (high-rate current position, like vehicle telemetry);
- **marker** → `content` (a placed, descriptive point of interest — searchable, not high-rate);
- **message** (geochat) → `content` (human free-text communication, indexed for retrieval).

If a marker or message carries actual tasking semantics, the class profile does **not** silently flip to `control`.
Either profile that specific subtype as `control`, or — cleaner — derive a separate control / task entity from the
content-bearing original, which preserves the searchable POI / transcript while giving directives their own control-plane
shape. This is deferred with tasking (see the out-of-scope note under Scope); the default for generic markers and
messages stays `content`.

### Identity and dedup

Two cases, decided differently:

- **`cop.*` entities (operators, markers, messages)** get SemLink-minted six-part IDs derived deterministically from the CoT
  `uid`. Determinism means no durable mapping store is needed — a restart re-derives the same IDs — so the in-memory,
  derive-on-demand pattern the `csapi` bridge uses is sufficient. The derivation must be **collision-safe**, however: CoT
  UIDs are arbitrary strings, and the repo's existing `safeToken` helpers (e.g. `internal/csapi/bridge.go`) are *lossy*
  normalizers that collapse runs of non-alphanumerics, so two distinct UIDs can map to one token. The `cop` ID therefore
  uses a collision-safe encoding of the raw UID — base32url (no padding), or a human-readable slug with a hash suffix for
  legibility — never bare `safeToken`; the current six-part shapes are `c360.semlink.cop.operator.position.<uid-token>`,
  `c360.semlink.cop.marker.poi.<uid-token>`, and `c360.semlink.cop.message.geochat.<uid-token>`. The raw CoT UID is always
  preserved as a predicate (`cop.identity.cot-uid`) for audit and debugging. `safeToken` may still be used for display
  labels, never for identity. The canonical audit predicate is `cop.identity.cot-uid`.
- **Cross-source UAV reconciliation** (a TAK client reporting a UAV that MAVLink also produces as `uav-NNN`) is a durable
  equivalence / identity-resolution problem (a sameAs policy) and is **out of scope for the demo**. In the demo, TAK is a
  *consumer* of UAV tracks (outbound), never a second *producer*; inbound TAK only creates `cop.*` entities, which never
  collide with `uav-NNN`. Naming this boundary is what keeps Phase 1 small. Durable identity resolution, when eventually
  needed, is a likely SemStreams substrate primitive (per ADR 001), not a SemLink-local workaround.

### SemConnect / CS API egress is not automatic

The CS API bridge (`internal/csapi/bridge.go`) syncs only `Vehicles`, `Alerts`, and `Commands` from `gcs.Snapshot`, and
it reads the **in-memory store, not the SKG**. Inbound `cop.*` entities therefore do **not** egress automatically.
Phase 1 must explicitly:

1. give `cop.*` entities a representation the bridge can see — extend `gcs.Store` / `gcs.Snapshot`, or give the bridge an
   SKG read path; and
2. add CS API resource mappers for the new kinds in `internal/csapi`.

Whether this is SemLink-only work depends on SemConnect: if its CS API already serves Sampling Features (markers),
System Events (messages), and mobile Systems (operators), the work is SemLink-side only; if not, the gap is a SemConnect
issue, per ADR 001's upstream-primitive principle. **This must be verified before Phase 1 commits.**

### Outbound bridge (Phase 0)

`internal/tak` polls `gcs.Store.Snapshot()` and encodes each vehicle as a CoT track and each alert as a CoT event,
fanning out to the default multicast group `udp://239.2.3.1:6969` and to subscribed TCP streaming clients. It is wired in
`main.go` behind a flag, like `-csapi-url`. Emit-only; reuses existing store state; needs neither `cop` nor
`graphprojection`. **First acceptance test: multicast** — an ATAK (or a CoT listener) on the same LAN renders the
simulated swarm with near-zero client configuration. TCP streaming follows in the same phase.

### SemConnect / CS API leverage (the value)

The SKG is the broker; CS API is the governed standards egress. With the Phase 1 store/bridge extensions above, the
mapping is:

| TAK / CoT | SemStreams SKG entity | CS API (OGC Connected Systems) |
| --- | --- | --- |
| UAV track (`a-f-A-*`) | `…robotics.fleet.drone.uav-NNN` (signal) | System + Datastreams + Observations |
| Operator position (`a-f-G-U-C`) | `…cop.operator.*` (signal) | System (mobile platform) + position Datastream |
| Marker / point of interest | `…cop.marker.*` (content) | Sampling Feature / Feature of Interest |
| Alert (low-battery / lost-link) | `…robotics.fleet.alert.*` (control) | System Event |
| GeoChat | `…cop.message.*` (content) | System Event / notification Datastream |

Tasking is intentionally absent from this table — see the out-of-scope note under Scope. This buys, via CS API, what TAK
cannot do itself: persistent queryable history, provenance and governance from the SKG, and a non-TAK
standards-consumer path.

### Phasing

- **Phase 0 — outbound (decided first).** `internal/cot` encode + `internal/tak` transport, reading the existing store.
  Acceptance: ATAK shows the swarm over multicast, then TCP. Needs no domain or projection extraction.
- **Phase 1 — inbound + egress.** Extract `internal/graphprojection` (prerequisite); add the `internal/cop` projection
  path with `cop.Contracts()` / `cop.RegisterPayloads`, and register it in
  `internal/semstreams/runtime.go` alongside projector payloads; ingest `cop.*` into the SKG; extend
  `gcs.Store` + `internal/csapi` for `cop.*` egress; verify SemConnect resource support first. Identity is
  collision-safe deterministic `cop.*` IDs (raw UID kept as `cop.identity.cot-uid`); cross-source UAV reconciliation and
  tasking stay out of scope.
- **Phase 2 (optional).** protobuf v1 negotiation (generated from the public `.proto`), then TLS / enrollment via
  step-ca.

## Consequences

Phase 0 is cheap and independent: it reuses the store and the `csapi` bridge pattern and needs no new domain or
projection code, so outbound-first carries the least risk and the highest demo value. It preserves the
adapter/substrate/standards split from ADR 001, now carried forward by ADR 003: SemLink owns protocol adapters and
local evidence APIs, SemStreams owns the substrate, SemConnect owns the standards view, and SemOps owns broad operator
glass. TAK becomes just another adapter family.

Phase 1 carries named, non-optional prerequisites that earlier drafts understated: the `graphprojection` extraction
(because the reusable helpers are unexported and MAVLink-entangled today), COP contracts/profiles plus
registering `cop.RegisterPayloads` in the runtime, explicit
`gcs.Store` and `internal/csapi` extensions for `cop.*` egress (because the bridge syncs only `Vehicles` / `Alerts` /
`Commands` from the in-memory store), and a SemConnect resource-support check. Scoping cross-source UAV reconciliation
and tasking out of the demo is what keeps Phase 1 tractable; durable identity resolution and the tasking round-trip,
when needed, are candidate follow-ons (the former a likely SemStreams primitive).

Hand-rolling the CoT codec means more code we own, but the subset is small and this avoids dependency philosophy and
velocity mismatch — the trade ADR 001 already accepted for MAVLink. The clean-room posture costs us copy-paste
convenience and forbids studying `goatak`'s structure, but is required by its AGPL license.

The former UI rebrand question is closed by ADR 003: SemLink language should move toward companion / mesh operations,
CLI/config, and external UI consumption, not a broad COP identity. The `cop.*` domain naming is already adopted above
for TAK / CoT entities.

## Open questions

- **CoT source of truth:** confirm the exact public schema/spec revision and capture a baseline set of ATAK-generated
  wire vectors to test against (resolves the clean-room source question concretely).
- **SemConnect resource support:** verify CS API serving of Sampling Features, System Events, and mobile Systems before
  Phase 1 (decides SemLink-only vs SemConnect issue).
- **UI rebrand:** superseded by ADR 003. Future SemLink operator-surface
  language should move toward companion / mesh APIs and external UI consumption,
  while SemOps owns the broader COP identity.
