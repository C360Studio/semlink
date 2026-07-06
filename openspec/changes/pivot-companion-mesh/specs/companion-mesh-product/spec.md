# Companion Mesh Product Specification

## ADDED Requirements

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
