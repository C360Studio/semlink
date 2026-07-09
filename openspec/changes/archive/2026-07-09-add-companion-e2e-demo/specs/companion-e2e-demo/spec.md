## ADDED Requirements

### Requirement: Fast Companion E2E Is Repo-Owned

SemLink SHALL provide a fast companion e2e lane that proves core companion behavior without requiring Docker, SemOps,
SemConnect, semstreams-ui, BlueOS, Navigator hardware, Gazebo, ArduPilot SITL, or physical MAVLink devices.

#### Scenario: In-process e2e runs locally

- **WHEN** maintainers run the fast companion e2e command
- **THEN** SemLink starts deterministic companion nodes through repo-owned harnesses
- **AND** the test verifies local state projection, evidence-compatible state, command posture, and native readback
  adapter semantics
- **AND** the test does not require external GCS glass or CS API

### Requirement: Single-Node Demo Produces Evidence Report

SemLink SHALL provide a runnable single-node demo that launches one companion runtime and emits a structured evidence
report.

#### Scenario: Single companion node is demonstrated

- **WHEN** maintainers run the single-node companion demo
- **THEN** the demo verifies runtime readiness, local health, `/register_service`, `/api/evidence`, and companion
  profile evidence
- **AND** the demo records command-safety posture and downstream dependency posture
- **AND** the demo writes a machine-readable report under a deterministic artifact directory

### Requirement: Simple Mesh Demo Produces Evidence Report

SemLink SHALL provide a runnable simple mesh demo that launches N local MAVLink companion runtimes and proves selected
vehicle-state catch-up through bounded mesh synchronization. Boat/ArduRover SHALL be the first supported demo profile,
not the architectural limit.

#### Scenario: Local mesh peers catch up selected state

- **WHEN** maintainers run the simple mesh demo
- **THEN** each node starts with deterministic identity, vehicle profile, static peer configuration, and simulated
  vehicle state
- **AND** peers exchange compact watermarks and selected diffs until the expected summaries are visible
- **AND** the generated report records peer counts, watermark posture, diff counts, TTL/merge posture, and assertion
  status for each node

### Requirement: Demo Reports Are Consumer-Oriented Evidence

SemLink SHALL treat generated demo reports and API evidence as the demo surface instead of growing a repo-owned GCS UI.

#### Scenario: Demo output is consumed downstream

- **WHEN** a downstream consumer such as SemOps, semstreams-ui, SemConnect, or release notes needs demo evidence
- **THEN** SemLink provides structured report artifacts and local API evidence
- **AND** SemLink does not require the downstream consumer to be running for the demo to pass

### Requirement: SemOps Contract Is Proven With Fake Downstream

SemLink SHALL prove native SemOps companion contract compatibility through a local fake receiver in the default e2e/demo
path.

#### Scenario: Native readback adapter posts to fake SemOps

- **WHEN** the demo exercises the SemOps readback adapter
- **THEN** SemLink posts MAVLink-native target, command, message, request time, TTL, correlation, idempotency, and
  companion node fields to the fake receiver
- **AND** the fake receiver validates the contract fixture without requiring CS API, SemConnect, or SemOps trusted
  operator headers
- **AND** the evidence report records the adapter result as downstream optional compatibility evidence
