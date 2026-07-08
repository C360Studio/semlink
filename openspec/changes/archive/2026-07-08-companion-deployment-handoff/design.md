## Context

The archived companion-mesh pivot defines SemLink as the boat-local MAVLink
companion and mesh product. The repo already has useful proof pieces: a
BlueOS-style Docker image skeleton, `/register_service`, `/api/health`,
`/api/evidence`, an ArduRover SITL lane without Gazebo, and read-only Navigator
smoke docs.

Those pieces are not yet packaged as a coherent early-adopter handoff. The next
useful milestone is a deployable companion service profile that can be run with
simulated/SITL MAVLink or placed near a BlueOS/Navigator-class stack for
read-only inspection, while preserving the hardware transmit block.

## Goals / Non-Goals

**Goals:**

- Define a deployable BlueOS-style companion handoff package.
- Make node identity, MAVLink UDP input, local SemStreams, mesh peers, and
  downstream consumers configurable through a documented profile.
- Prove package readiness with `/api/health`, `/register_service`, and
  `/api/evidence`.
- Keep the first fidelity lane hardware-free: ArduRover/boat SITL or UDP
  MAVLink without Gazebo.
- Produce release/tag readiness criteria for a future package checkpoint.

**Non-Goals:**

- No hardware MAVLink command transmit.
- No BlueOS Bazaar publication or external registry release in this change.
- No SemLink-owned GCS/dashboard expansion.
- No SemOps or semstreams-ui runtime dependency.
- No CS API or SemConnect runtime dependency for the SemLink MVP companion
  handoff; standards projection stays downstream with SemOps/SemConnect unless
  a later accepted OpenSpec change moves that boundary.
- No MAVSDK adoption.

## Decisions

### Package Profile Is BlueOS-Style Docker First

The first handoff target is the existing BlueOS-style Docker image plus
metadata and local Compose smoke. This aligns with BlueOS extension mechanics
and Raspberry Pi class deployment without requiring a Navigator on every
developer desk.

Alternative considered: create a generic Linux service package first. That is
useful later, but it does less for the early adopter's BlueRobotics/Navigator
shape and would delay the first tag-worthy artifact.

### Configuration Is Explicit And File/Env Friendly

The handoff should document a concrete profile for node ID, optional vehicle ID,
MAVLink UDP listen address, embedded/local SemStreams mode, mesh peer URLs, and
downstream consumer metadata. Environment variables remain the Docker/BlueOS
entrypoint boundary; a sample config file can provide copyable operator shape.

Alternative considered: discover everything dynamically. That is attractive for
mesh demos, but the first deployable artifact needs predictable startup and
debuggable evidence.

### Readiness Means Local Evidence, Not UI Ownership

The readiness contract is local and machine-readable: `/api/health`,
`/register_service`, and `/api/evidence`. SemOps and semstreams-ui can consume
the evidence bundle, but this repo does not grow new GCS glass.

Alternative considered: rebuild a focused SemLink dashboard for the package.
That would undermine the archived product boundary and compete with SemOps and
semstreams-ui.

### SemOps Readback Adapter Is Draft-Contract Gated

SemOps owns the GCS/COP command-intent admission boundary. SemLink may carry a
small native POST adapter for the SemOps
`c360.semops.semlink.ardupilot.readback.v0` draft contract so the companion
runtime can submit `MAV_CMD_REQUEST_MESSAGE` / `AUTOPILOT_VERSION` readback
intent without CS API in the hot path.

This adapter is not final contract acceptance, runtime enablement, or trusted
SemOps auth ownership. SemLink mirrors the v0 fixtures locally for CI and keeps
the adapter behind package code until the SemOps contract branch is visible and
final hold-out review is complete.

### Historical SemGCS Names Are Migration Debt

The current `cmd/semgcs-demo` binary and `internal/gcs` package still carry the
main companion runtime, store, local API, evidence bundle, optional adapters,
and legacy static UI serving. Those pieces should be renamed or split toward a
companion-service shape rather than deleted while they remain the handoff
runtime. The historical Svelte UI is the strongest retirement candidate, but
removal should be gated by package smoke and evidence checks that do not depend
on repo-owned glass.

### SITL/UDP Is The Release Fidelity Lane

The package should prove MAVLink wire behavior through UDP and ArduRover/boat
SITL without Gazebo. Dockerized SITL may remain a heavier lane, but the core
handoff should be understandable with simple UDP input and local evidence reads.

Alternative considered: make physical Navigator smoke the primary release gate.
That would make the release depend on hardware we do not have in quantity and
would blur the safety boundary.

### Hardware Transmit Remains Fail-Closed

The handoff may run on or near BlueOS hardware, but hardware command transmit
stays disabled until a separate OpenSpec change defines authorization and
safety evidence. Read-only smoke remains useful and safe.

Alternative considered: include a hidden or disabled-by-default hardware
transmit path. That increases safety risk before we have the authorization
model, operator workflow, and recovery evidence.

## Risks / Trade-offs

- BlueOS extension conventions can drift -> keep Docker labels, metadata, and
  registration tests small and easy to update.
- Docker/SITL smokes are heavier than unit tests -> separate quick local tests
  from operator-invoked fidelity lanes.
- Environment configuration can sprawl -> keep one documented handoff profile
  and validate the fields that affect safety/evidence.
- Early adopters may expect command control -> make the hardware transmit block
  visible in API evidence and docs.
- SemStreams runtime pins may move -> keep package tests on the repo pin and
  leave upstream bumps as separate dependency slices.

## Migration Plan

1. Add or tighten the handoff configuration profile and docs.
2. Extend the BlueOS-style entrypoint/Compose smoke to load that profile.
3. Add readiness/evidence tests for package metadata and local APIs.
4. Add the draft SemOps readback adapter against the v0 fixtures without
   enabling hardware transmit or CS API-first runtime behavior.
5. Connect the SITL/UDP handoff script to the package profile.
6. Update demo/readme docs with the release-check command set.
7. Validate with Go tests, OpenSpec, and the local package smoke when Docker is
   available.

Rollback is simple: do not use the new handoff profile or image tag. Existing
developer/demo commands remain valid.

## Open Questions

- What should the first image/tag naming convention be: repo SHA,
  `companion-handoff-*`, or a SemLink semantic version?
- Should node ID default to a configured value, hostname-derived value, or
  deterministic profile value for demos?
- Should mesh peers be static URLs for the first handoff, or should discovery
  be a later change?
