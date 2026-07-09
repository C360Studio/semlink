# mavlink-companion-runtime Specification

## Purpose

Define local MAVLink companion runtime expectations: bounded raw telemetry,
semantic current-state projection, deterministic simulator and SITL evidence,
BlueOS/Navigator safety boundaries, and deliberate reuse of SemOps MAVLink
prior art before duplicating parser or command behavior.

## Requirements
### Requirement: Vehicle-Local Runtime

Each SemLink companion node SHALL maintain local runtime state for one MAVLink
vehicle context.

#### Scenario: Local node runs without mesh

- **GIVEN** a SemLink companion node has a MAVLink feed and local SemStreams
  runtime
- **WHEN** peer links are absent
- **THEN** the node continues to decode MAVLink, update current state, emit
  alerts, evaluate local rules, and serve local status/evidence APIs

### Requirement: Bounded Raw MAVLink Lane

SemLink MUST keep raw MAVLink frames on a bounded lane and project semantic
current state separately.

#### Scenario: High-rate telemetry arrives

- **WHEN** MAVLink telemetry arrives faster than graph state should update
- **THEN** raw frames remain on a bounded raw lane
- **AND** current vehicle state is collapsed into signal-profiled graph entities
- **AND** alerts and command intents are control-profiled graph entities

### Requirement: Simulator Fidelity Ladder

SemLink SHALL prove companion behavior through deterministic simulation and SITL
before hardware claims.

#### Scenario: Multi-node mesh behavior is tested

- **GIVEN** no Navigator hardware is attached
- **WHEN** the multi-node companion demo runs
- **THEN** deterministic simulated boats can prove local state, mesh catch-up,
  rule traces, and UI-consumable evidence

#### Scenario: Autopilot wire behavior is tested

- **GIVEN** the simulator-only lane is stable
- **WHEN** SemLink claims autopilot MAVLink compatibility
- **THEN** ArduRover / ArduPilot SITL without Gazebo is the first fidelity gate
- **AND** PX4, Gazebo, and physical Navigator hardware remain separate evidence
  lanes

### Requirement: SemOps MAVLink Prior Art Is Evaluated First

SemLink MUST evaluate SemOps MAVLink code before implementing duplicate parser,
UDP, replay, or command-gate behavior.

#### Scenario: MAVLink runtime code is planned

- **WHEN** SemLink plans parser, UDP listener, replay, raw-lane, command helper,
  COMMAND_ACK, or SITL gate work
- **THEN** the implementation plan records whether SemOps code is reused,
  ported, shared, or deliberately not adopted

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

### Requirement: Compatibility Proof Remains MAVLink Native

SemLink SHALL use MAVLink-native evidence as the compatibility boundary for
autopilot behavior, even when the runtime is packaged for BlueOS, Navigator, or
a companion Pi.

#### Scenario: BlueOS is absent from compatibility proof

- **WHEN** SemLink claims ArduPilot or MAVLink wire compatibility
- **THEN** local UDP MAVLink, ArduPilot SITL, or another native MAVLink source
  can satisfy the compatibility proof without BlueOS-specific service state
- **AND** BlueOS service registration, Navigator readiness, and package
  lifecycle evidence are treated as deployment metadata only
- **AND** MAVLink command, message, target, ACK, and readback vocabulary remains
  the compatibility anchor

#### Scenario: BlueOS deployment uses the same runtime contract

- **WHEN** SemLink runs as a BlueOS extension, Navigator-attached service,
  companion-Pi service, SITL service, or plain native process
- **THEN** it exposes the same local evidence and MAVLink companion runtime
  contract for downstream consumers
- **AND** it does not require BlueOS REST, MAVLink2REST, endpoint-manager state,
  CS API, or SemConnect to prove runtime compatibility
