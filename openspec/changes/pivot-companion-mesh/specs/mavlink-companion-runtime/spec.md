# MAVLink Companion Runtime Specification

## ADDED Requirements

### Requirement: Vehicle-Local Runtime

Each SemLink companion node SHALL maintain local runtime state for one MAVLink
vehicle context.

#### Scenario: Local node runs without mesh

- **GIVEN** a SemLink companion node has a MAVLink feed and local SemStreams
  runtime
- **WHEN** peer links are absent
- **THEN** the node continues to decode MAVLink, update current state, emit
  alerts, evaluate local rules, and serve local operator evidence

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
  rule traces, and UI evidence

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
