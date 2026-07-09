## MODIFIED Requirements

### Requirement: Simulator Fidelity Ladder

SemLink SHALL prove companion behavior through deterministic simulation and SITL before hardware claims. The default
proof ladder MUST start with SemLink-owned e2e/demo lanes that do not require external glass, CS API, Gazebo, Navigator
hardware, or a full BlueOS stack.

#### Scenario: Fast companion e2e runs before external fidelity lanes

- **GIVEN** no Navigator hardware, SITL, BlueOS runtime, SemOps, SemConnect, or semstreams-ui is attached
- **WHEN** the fast companion e2e lane runs
- **THEN** deterministic companion harness nodes prove local state, evidence-compatible snapshots, command posture, and
  native downstream adapter semantics
- **AND** the proof is distinct from autopilot wire compatibility claims

#### Scenario: Single-node companion behavior is demonstrated

- **GIVEN** no external GCS glass is running
- **WHEN** the single-node companion demo runs
- **THEN** the runtime proves readiness, evidence API posture, command posture, and downstream optional dependency
  posture through a generated report

#### Scenario: Multi-node mesh behavior is tested

- **GIVEN** no Navigator hardware is attached
- **WHEN** the multi-node companion demo runs
- **THEN** deterministic simulated MAVLink vehicles can prove local state, mesh catch-up, rule traces, and
  consumer-readable evidence reports
- **AND** boat or ArduRover behavior is treated as the first demo profile rather than the runtime limit

#### Scenario: Autopilot wire behavior is tested

- **GIVEN** the simulator-only lane is stable
- **WHEN** SemLink claims autopilot MAVLink compatibility
- **THEN** ArduRover / ArduPilot SITL without Gazebo is the first fidelity gate
- **AND** PX4, Gazebo, and physical Navigator hardware remain separate evidence lanes
