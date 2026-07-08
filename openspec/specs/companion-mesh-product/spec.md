# companion-mesh-product Specification

## Purpose

Capture SemLink's forward product boundary as a vehicle-local MAVLink companion
and mesh node. SemLink exposes APIs, traces, and evidence for external GCS/COP
surfaces while preserving SemOps COP ownership and SemConnect standards-egress
ownership.

## Requirements
### Requirement: SemLink Owns The Companion Mesh Product

SemLink SHALL be the product repo for MAVLink vehicle-local companion and mesh
behavior.

#### Scenario: Product boundary is explicit

- **WHEN** architecture, tickets, specs, or implementation decide where
  vehicle-local MAVLink companion behavior belongs
- **THEN** SemLink owns companion runtime behavior, MAVLink ingress, local state,
  local rules, mesh summaries, command-intent evidence, CLI/config surfaces, and
  UI-consumable local status/evidence APIs
- **AND** SemOps owns broad COP and fusion product behavior
- **AND** SemConnect owns standards-facing CS API bridge and conformance claims

#### Scenario: SemGCS remains historical baseline

- **WHEN** future work references ADR 001 or the SemGCS demo
- **THEN** it treats that work as implemented prior art
- **AND** it treats ADR 003 and this OpenSpec change as the forward product
  boundary

### Requirement: SemOps COP Boundary Remains Preserved

SemLink MUST NOT absorb the broad COP / fusion product surface.

#### Scenario: COP behavior is requested

- **WHEN** a feature requires cross-feed assimilation, incident-scale dashboard
  behavior, scenario orchestration, or multi-source operational fusion
- **THEN** the work is routed to SemOps unless a new OpenSpec change explicitly
  reassigns that product boundary

### Requirement: SemLink Provides UI-Consumable APIs, Not Owned Glass

SemLink SHALL expose companion, mesh, rule, and command evidence for external UI
consumers without growing a repo-owned GCS or dashboard as the forward product.

#### Scenario: External glass needs companion state

- **WHEN** SemOps, semstreams-ui, or another external operator surface needs
  SemLink state
- **THEN** SemLink provides local API, trace, semantic evidence, and CLI/config
  contracts for node status, vehicle status, mesh peers, rule decisions, command
  gate outcomes, and health
- **AND** SemLink does not require those consumers to embed or depend on a
  SemLink-owned Svelte application

#### Scenario: New dashboard behavior is proposed

- **WHEN** a feature would add SemLink-owned GCS screens, dashboards, or broad
  operator workflows
- **THEN** the work is routed to SemOps or a later OpenSpec change unless it is
  only a temporary demo/debug view with explicit retirement criteria

### Requirement: SemStreams Remains The Substrate

SemLink MUST consume SemStreams substrate contracts rather than reimplementing
framework-owned graph behavior locally.

#### Scenario: Companion state enters the graph

- **WHEN** SemLink writes vehicle state, alerts, rule traces, or command intent
- **THEN** the write uses SemStreams graph mutation, projection ownership, and
  indexing-profile contracts
- **AND** raw high-rate telemetry remains on bounded lanes rather than becoming
  one graph entity per frame

### Requirement: CS API Is Egress, Not Swarm Protocol

SemLink SHALL treat OGC API - Connected Systems as optional downstream egress.

#### Scenario: Standards consumer needs vehicle state

- **WHEN** SemLink publishes standards-facing Systems, Datastreams,
  Observations, SystemEvents, or Commands
- **THEN** it emits a curated, low-rate projection to SemConnect
- **AND** it does not route raw MAVLink or internal mesh replication through
  CS API

### Requirement: Deployable Handoffs Preserve Product Boundary

SemLink deployable handoffs SHALL expose local APIs, configuration, and
evidence for external operator surfaces without adding repo-owned GCS glass.

#### Scenario: External operator surface consumes handoff evidence

- **WHEN** SemOps, semstreams-ui, or another external operator surface needs
  companion deployment state
- **THEN** the handoff exposes local readiness, configuration, and evidence API
  contracts
- **AND** the external surface does not need to embed or depend on a SemLink UI

#### Scenario: Handoff documentation names downstream consumers

- **WHEN** handoff docs or evidence describe SemOps, semstreams-ui, or
  SemConnect
- **THEN** they describe those systems as optional downstream consumers
- **AND** they do not describe them as required runtime dependencies for
  companion package readiness
- **AND** they do not describe CS API or SemConnect as SemLink MVP runtime
  responsibilities

### Requirement: Historical SemGCS Runtime Surface Is Migrated

SemLink SHALL migrate the historical SemGCS-named runtime and package surfaces
toward a companion-service identity without removing working companion runtime
capability before equivalent handoff evidence exists.

#### Scenario: Historical runtime names are referenced

- **WHEN** handoff specs, docs, Docker metadata, or code reference
  `cmd/semgcs-demo`, `internal/gcs`, or the historical Svelte UI
- **THEN** they treat those surfaces as migration candidates toward a
  companion-service runtime such as `cmd/semlink-companion` and
  companion/evidence/API packages
- **AND** they do not treat the SemGCS name or repo-owned dashboard as the
  forward product surface

#### Scenario: Historical UI or static serving is retired

- **WHEN** a future slice removes the historical SemGCS UI or static serving
  from the deployable handoff
- **THEN** `/api/health`, `/register_service`, and `/api/evidence` still prove
  package readiness without a SemLink-owned dashboard
- **AND** MAVLink ingest, local evidence, optional CS API egress, and optional
  TAK bridge behavior remain available or are explicitly rehomed before removal

### Requirement: MVP Handoff Keeps Standards Projection Downstream

For the MVP companion handoff, SemLink SHALL keep CS API and SemConnect outside
the companion runtime path. Standards projection remains a downstream
SemOps/SemConnect edge unless a later accepted OpenSpec change explicitly moves
that responsibility.

#### Scenario: CS API projection is requested during MVP handoff work

- **WHEN** SemLink handoff work needs to describe CS API, SemConnect, or
  standards-facing readback projection
- **THEN** the work treats that projection as downstream SemOps/SemConnect
  ownership
- **AND** SemLink remains responsible for native MAVLink companion behavior,
  local evidence APIs, configuration, and native SemOps readback intent
- **AND** SemLink does not add CS API runtime coupling as part of the MVP
  companion handoff
