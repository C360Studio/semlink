## ADDED Requirements

### Requirement: Package Runtime Profiles Produce Evidence

SemLink SHALL provide package runtime profiles that prove MAVLink companion
behavior through local evidence.

#### Scenario: Package runs with internal simulator disabled

- **WHEN** the handoff package is configured with an external MAVLink UDP input
- **THEN** the package listens for MAVLink without requiring the internal
  simulator
- **AND** current vehicle state appears in `/api/evidence` after telemetry is
  received

#### Scenario: Package runs without external MAVLink

- **WHEN** the handoff package starts without an external MAVLink input
- **THEN** it still reports local health, registration, and configuration
  evidence
- **AND** it does not claim autopilot wire compatibility from that run alone

### Requirement: BlueOS-Style Package Metadata Is Testable

SemLink SHALL keep BlueOS-style package metadata testable without requiring
physical Navigator hardware.

#### Scenario: Package metadata is inspected locally

- **WHEN** the BlueOS-style package metadata or `/register_service` response is
  tested locally
- **THEN** it declares SemLink as a MAVLink companion and mesh node
- **AND** it does not declare hardware command transmit capability
