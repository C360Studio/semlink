## ADDED Requirements

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
