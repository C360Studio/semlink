## ADDED Requirements

### Requirement: SITL-Backed Demo Artifact Is Evidence-Derived

SemLink SHALL emit SITL-backed companion demo artifacts only from observed
local runtime evidence, not from deterministic demo reports or caller-supplied
source-fidelity labels alone.

#### Scenario: SITL-backed artifact is emitted from runtime evidence

- **WHEN** a running companion node exposes `/api/evidence` showing external
  MAVLink input, observed raw frames, decoded frames, at least one vehicle with
  a MAVLink system ID, raw MAVLink mesh exclusion, and blocked hardware
  transmit posture
- **THEN** SemLink writes a `semlink-companion-demo-artifact-v0` artifact that
  embeds an evidence-derived `single-node-companion-demo` report
- **AND** the artifact source records a real SemLink commit or version,
  `source_fidelity=sitl-backed`, no-transmit posture, generator metadata, and
  node metadata derived from the observed evidence
- **AND** the artifact does not require SemOps, SemConnect, CS API, BlueOS,
  Navigator hardware, Gazebo, or GCS glass to be running

#### Scenario: SITL-backed artifact fails closed without evidence

- **WHEN** external MAVLink input, observed frames, decoded vehicle state,
  MAVLink system ID, no-transmit posture, raw MAVLink mesh exclusion, or real
  SemLink commit/version evidence is missing
- **THEN** SemLink rejects SITL-backed artifact generation before writing the
  artifact
- **AND** deterministic demo artifact generation continues to emit
  `source_fidelity=deterministic` unless live-source metadata is fully
  evidenced

### Requirement: ArduPilot SITL E2E Is Env-Gated

SemLink SHALL include a named ArduPilot SITL e2e scenario in the test suite
without making ArduPilot, Gazebo, Navigator hardware, SemOps, SemConnect, CS
API, or GCS glass prerequisites for default local tests.

#### Scenario: Default test suite skips host SITL

- **WHEN** maintainers run the default Go test suite without enabling SITL e2e
- **THEN** SemLink skips the real `sim_vehicle.py` scenario
- **AND** deterministic, UDP-smoke, and artifact-builder tests still prove that
  SemLink does not overclaim SITL fidelity

#### Scenario: Env-gated SITL e2e produces artifact

- **WHEN** maintainers run the named e2e with SITL explicitly enabled and host
  `sim_vehicle.py` or the repo-owned SITL Docker image available
- **THEN** SemLink starts a local companion runtime with external MAVLink UDP
  input and launches ArduPilot Rover SITL without Gazebo
- **AND** Docker-backed runs may use MAVProxy only as a non-interactive MAVLink
  bridge, not as a GCS or runtime dependency
- **AND** the test waits until `/api/evidence` shows external MAVLink frames,
  decoded vehicle state, a MAVLink system ID, blocked hardware transmit
  posture, and raw MAVLink mesh exclusion
- **AND** the test emits and validates a `sitl-backed`
  `semlink-companion-demo-artifact-v0` artifact with real SemLink source
  metadata
