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
