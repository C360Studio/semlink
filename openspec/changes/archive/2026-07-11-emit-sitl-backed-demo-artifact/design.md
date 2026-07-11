## Context

SemLink already has three relevant pieces: an ArduRover SITL lane that can feed
local UDP MAVLink into the companion runtime, `/api/evidence` that records
external MAVLink input and projected vehicle state, and a
`semlink-companion-demo-artifact-v0` envelope accepted by SemOps for
deterministic demo evidence. The missing bridge is an honest producer lane that
turns observed SITL evidence into the same artifact envelope without letting a
deterministic report claim live source fidelity.

The downstream split is fixed for this slice. SemLink owns running the runtime,
MAVLink/SITL wiring, companion-node evidence, source metadata, and artifact
emission. SemOps owns ingestion, malformed-artifact rejection, COP/GCS
projection, no-transmit/readback preservation, and downstream smoke assertions.

## Goals / Non-Goals

**Goals:**

- Produce a SemOps-consumable artifact from a running SemLink companion node
  whose `/api/evidence` proves external MAVLink/SITL input.
- Provide an optional e2e suite lane that proves the same artifact path against
  real ArduPilot SITL when host `sim_vehicle.py` or the repo-owned SITL Docker
  image is available.
- Derive SITL-backed node metadata from evidence instead of accepting a naked
  command-line source-fidelity claim.
- Fail closed before writing an artifact when external MAVLink frames, decoded
  vehicles, per-node metadata, no-transmit posture, or SemLink source refs are
  missing.
- Keep deterministic demo artifacts valid and deterministic by default.

**Non-Goals:**

- Do not start hardware transmit, COMMAND_LONG execution, or trusted operator
  authority.
- Do not make CS API, SemConnect, SemOps, BlueOS, Navigator hardware, Gazebo,
  or GCS glass a runtime dependency for the producer path.
- Do not make SemOps synthesize SITL fidelity from deterministic SemLink
  artifacts.

## Decisions

- Add an evidence-derived report builder rather than mutating deterministic demo
  reports. This keeps the current single-node and mesh demos stable while
  creating a distinct path for live-source claims.
- Read `/api/evidence` over the local runtime API for the producer CLI. This
  matches what downstream debuggers and SemOps can inspect, and it keeps source
  fidelity tied to the same local evidence contract.
- Make real SITL e2e env-gated, not default. The default `go test ./...` lane
  remains hermetic and fast; `SEMLINK_E2E_SITL=1` enables the host-sensitive
  `sim_vehicle.py` lane for manual, nightly, or dedicated CI runs.
- Treat SITL-backed artifact generation as a fail-closed operation. The builder
  requires external MAVLink input posture, observed raw and decoded frames, at
  least one MAVLink vehicle system ID, raw MAVLink mesh exclusion, blocked
  hardware transmit posture, and a real SemLink commit or version.
- Keep source metadata SemLink-native. The artifact records simulator family,
  vehicle source, MAVLink system ID, source route, and no-transmit posture
  without introducing CS API or BlueOS-specific fields into the hot path.

## Risks / Trade-offs

- Evidence may be delayed while SITL starts -> the producer should support
  retry/wait behavior or produce a clear failure that tells the operator which
  evidence gate is missing.
- A synthetic test can prove the builder but not ArduPilot fidelity -> the repo
  should keep a unit-level fail-closed test and a documented operator lane for
  running real `sim_vehicle.py` when available.
- Host SITL can be slow or missing -> skip by default, support a repo-owned
  Docker image path with a prebuilt ArduRover binary, run Docker MAVProxy as a
  non-interactive output-only bridge, and fail clearly when explicitly enabled
  but neither launcher nor required runtime evidence is available.
- Multiple vehicles may appear in evidence -> this slice can emit one node with
  the first observed vehicle system ID and leave N-node SITL mesh artifact
  generation as a follow-up.
- Source routes can vary across local and Docker runs -> derive defaults from
  `/api/evidence` when present and allow explicit route override for scripts.
