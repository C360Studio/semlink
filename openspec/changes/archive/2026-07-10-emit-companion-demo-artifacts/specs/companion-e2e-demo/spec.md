## ADDED Requirements

### Requirement: Demo Artifact Envelope Is Producer-Owned

SemLink SHALL be able to emit a native companion demo artifact envelope for
single-node and simple-mesh demo reports without requiring SemOps, SemConnect,
CS API, BlueOS, Navigator hardware, Gazebo, ArduPilot SITL, or physical MAVLink
devices to be running.

#### Scenario: Single-node artifact is emitted

- **WHEN** maintainers run the single-node companion demo with artifact output
  enabled
- **THEN** SemLink writes a `semlink-companion-demo-artifact-v0` JSON artifact
  that embeds the `single-node-companion-demo` report
- **AND** the artifact records `generated_at`, SemLink source reference,
  generator profile or command, source fidelity, and no-transmit posture
- **AND** the embedded report timestamp matches the artifact timestamp

#### Scenario: Simple-mesh artifact is emitted

- **WHEN** maintainers run the simple-mesh companion demo with artifact output
  enabled
- **THEN** SemLink writes a `semlink-companion-demo-artifact-v0` JSON artifact
  that embeds the `simple-mesh-companion-demo` report
- **AND** the artifact preserves raw MAVLink exclusion evidence from the
  embedded report
- **AND** the demo does not require downstream SemOps COP/GCS glass to pass

### Requirement: Demo Artifact Source Fidelity Is Evidence-Gated

SemLink SHALL only emit source-fidelity claims that are supported by producer
evidence available to the demo command.

#### Scenario: Deterministic demos do not overclaim live source fidelity

- **WHEN** the current deterministic single-node or simple-mesh demo emits an
  artifact without explicit live source metadata
- **THEN** the artifact source fidelity is `deterministic`
- **AND** the artifact does not claim `sitl-backed` or `hardware-adjacent`
  fidelity

#### Scenario: Live-source claims require per-node metadata

- **WHEN** a future demo or caller requests `sitl-backed` or
  `hardware-adjacent` fidelity
- **THEN** SemLink requires each claimed live-source node to include concrete
  source metadata such as node ID, vehicle source, and MAVLink system ID
- **AND** SemLink rejects unsupported or under-evidenced source-fidelity claims
  before writing the artifact
