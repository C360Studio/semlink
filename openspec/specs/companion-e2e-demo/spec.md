# companion-e2e-demo Specification

## Purpose
TBD - created by archiving change add-companion-e2e-demo. Update Purpose after archive.
## Requirements
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
- **AND** the artifact records `generated_at`, a real SemLink commit or version
  source reference, generator profile or command, source fidelity, and
  no-transmit posture
- **AND** the embedded report timestamp matches the artifact timestamp

#### Scenario: Simple-mesh artifact is emitted

- **WHEN** maintainers run the simple-mesh companion demo with artifact output
  enabled
- **THEN** SemLink writes a `semlink-companion-demo-artifact-v0` JSON artifact
  that embeds the `simple-mesh-companion-demo` report
- **AND** the artifact preserves raw MAVLink exclusion evidence from the
  embedded report
- **AND** SemLink rejects artifact output when it cannot resolve a real
  `semlink_commit` or `semlink_version`
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

### Requirement: One-Vehicle-Per-Companion Mesh Demo Produces Evidence Report

SemLink SHALL provide a deterministic local mesh demo that proves selected
state catch-up across multiple companion nodes, with exactly one simulated
MAVLink vehicle per companion node.

#### Scenario: Local mesh peers catch up one vehicle each

- **WHEN** maintainers run the default simple mesh companion demo
- **THEN** the demo uses `3` local companion nodes
- **AND** each node starts with `1` simulated MAVLink vehicle
- **AND** the generated report records `expected_summaries=3`
- **AND** each node starts with `initial_summary_count=1`
- **AND** each node reaches `final_summary_count=3` and `watermark_count=3`
- **AND** each node records bounded diff counts, peer count, TTL merge posture,
  visible origin vehicle references, and assertion status
- **AND** the proof remains deterministic and does not require SITL, BlueOS,
  Navigator hardware, SemOps, SemConnect, CS API, or GCS glass

#### Scenario: Multi-companion mesh artifact remains deterministic

- **WHEN** maintainers run the simple mesh companion demo with artifact output
- **THEN** SemLink writes a `semlink-companion-demo-artifact-v0` envelope that
  embeds the multi-companion mesh report
- **AND** the artifact source fidelity is `deterministic`
- **AND** the artifact generator metadata identifies the node-count proof shape

#### Scenario: Mesh perspective matrix runs in the deterministic e2e lane

- **WHEN** maintainers run the deterministic mesh e2e tests
- **THEN** SemLink exercises 3-node full mesh, line topology, partition-heal,
  and late-joiner selected-state scenarios
- **AND** each companion still owns exactly one simulated MAVLink vehicle
- **AND** each converged node reports all expected visible origin vehicle
  references
- **AND** partial stages report only the origin vehicles actually visible from
  that node's perspective
- **AND** the matrix does not require external GCS glass or standards bridge
  services to be running

