## ADDED Requirements

### Requirement: Deployable Companion Package

SemLink SHALL provide a deployable companion package that can be built and run
as a BlueOS-style Docker service.

#### Scenario: Package readiness smoke passes

- **WHEN** the companion package is launched through the handoff smoke
- **THEN** `/api/health` responds successfully
- **AND** `/register_service` responds with BlueOS service metadata
- **AND** `/api/evidence` responds with the companion evidence contract

### Requirement: Handoff Configuration Profile

SemLink SHALL define a documented handoff configuration profile for deployment
identity, MAVLink input, local SemStreams runtime, mesh peers, and downstream
consumer metadata.

#### Scenario: Operator configures a single companion node

- **WHEN** a user supplies the handoff profile for one companion node
- **THEN** the node can declare its node identity, MAVLink UDP listen address,
  SemStreams runtime mode, configured mesh peers, and downstream consumers
- **AND** those settings are visible through local evidence or readiness output

### Requirement: SITL Or UDP Handoff Evidence

SemLink SHALL prove the handoff package with a no-Gazebo SITL or UDP MAVLink
lane before making package-readiness claims.

#### Scenario: ArduRover or boat SITL feeds the package

- **WHEN** ArduRover or boat-class SITL sends MAVLink over UDP to the handoff
  package
- **THEN** SemLink buffers raw MAVLink frames on the bounded lane
- **AND** SemLink projects current vehicle state into local evidence
- **AND** the evidence can be read without a SemLink-owned dashboard

### Requirement: Release Checkpoint Is Evidence-Based

SemLink SHALL define release or tag readiness for the companion handoff through
repeatable local evidence rather than calendar or manual judgment alone.

#### Scenario: Handoff checkpoint is proposed

- **WHEN** maintainers propose tagging or publishing a companion handoff
  checkpoint
- **THEN** the documented Go tests, OpenSpec validation, package smoke, and
  SITL/UDP evidence lane have passing results or explicit recorded waivers
- **AND** the checkpoint states that hardware command transmit remains disabled

### Requirement: Handoff Does Not Require External Glass

The companion handoff MUST NOT require SemOps, semstreams-ui, or a SemLink-owned
GCS dashboard to prove local readiness.

#### Scenario: Package runs standalone

- **WHEN** the handoff package starts with local SemStreams runtime enabled
- **THEN** local health, registration, and evidence APIs are sufficient to
  verify companion readiness
- **AND** SemOps and semstreams-ui remain optional downstream consumers
