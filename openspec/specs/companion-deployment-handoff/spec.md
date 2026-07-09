# companion-deployment-handoff Specification

## Purpose

Define the deployable early-adopter handoff for SemLink as a BlueOS-style
MAVLink companion package with local readiness/evidence APIs, no-Gazebo
SITL/UDP proof, downstream-only external consumers, and fail-closed hardware
command posture.
## Requirements
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

SemLink SHALL define release or tag readiness for the companion handoff through repeatable local evidence rather than
calendar or manual judgment alone. Release evidence MUST include the SemLink-owned e2e/demo proof ladder or explicit
recorded waivers for lanes that are not yet required.

#### Scenario: Handoff checkpoint is proposed

- **WHEN** maintainers propose tagging or publishing a companion handoff checkpoint
- **THEN** the documented Go tests, OpenSpec validation, package smoke, fast companion e2e, single-node demo, simple mesh
  demo, and SITL/UDP evidence lane have passing results or explicit recorded waivers
- **AND** the checkpoint states that hardware command transmit remains disabled

### Requirement: Handoff Does Not Require External Glass

The companion handoff MUST NOT require SemOps, semstreams-ui, or a SemLink-owned
GCS dashboard to prove local readiness. It also MUST NOT require CS API or
SemConnect as a companion package runtime dependency for the MVP handoff.

#### Scenario: Package runs standalone

- **WHEN** the handoff package starts with local SemStreams runtime enabled
- **THEN** local health, registration, and evidence APIs are sufficient to
  verify companion readiness
- **AND** SemOps, semstreams-ui, and SemConnect remain optional downstream
  consumers rather than startup, runtime, or readiness requirements

### Requirement: SemOps Readback Adapter Uses Native Draft Contract

SemLink SHALL keep any native adapter for the SemOps companion readback v0
contract inside the draft-gated SemOps/SemLink review boundary. For the MVP
companion handoff, SemLink SHALL keep this path MAVLink-native and MUST NOT
host, consume, or project CS API/SemConnect in order to submit readback intent.

#### Scenario: Companion submits SemOps readback intent

- **WHEN** SemLink submits MVP ArduPilot readback intent to SemOps
- **THEN** the request uses the
  `c360.semops.semlink.ardupilot.readback.v0` contract
- **AND** the request uses MAVLink-native `target_system_id`,
  `target_component_id`, `command_id`, and `requested_message_id` fields
- **AND** the request does not require SemLink to send `target_asset_id` or
  prove the SemOps born target
- **AND** the request does not require CS API/SemConnect in the hot path
- **AND** mirrored CS API projection fixtures are treated as downstream interop
  evidence only, not SemLink runtime inputs or outputs

#### Scenario: Companion adapter handles SemOps authority boundary

- **WHEN** SemLink builds or sends the draft readback request
- **THEN** the adapter does not mint trusted SemOps operator headers
- **AND** the adapter expects SemOps, a gateway, a sidecar, or a test harness to
  own trusted-header translation
- **AND** accepted or duplicate responses preserve no-native/no-companion
  transmit posture for the MVP path

#### Scenario: Standards projection remains downstream

- **WHEN** SemOps or SemConnect needs a CS API representation of accepted
  readback intent
- **THEN** that projection is owned at the SemOps/SemConnect standards edge
- **AND** SemLink does not add CS API client, server, or projection runtime
  coupling for the MVP companion handoff
- **AND** moving CS API runtime responsibility into SemLink requires a later
  accepted OpenSpec change

### Requirement: BlueOS Handoff Preserves Native Readback Contract

SemLink SHALL treat BlueOS, Navigator, and companion-Pi packaging as deployment
profiles for the handoff package, not as alternate SemOps readback protocols.

#### Scenario: Handoff submits readback from any deployment profile

- **WHEN** a handoff package submits the MVP SemOps ArduPilot readback intent
  from BlueOS, Navigator, a companion Pi, SITL, or a plain native process
- **THEN** it uses the same native
  `c360.semops.semlink.ardupilot.readback.v0` request shape
- **AND** it uses MAVLink-native command, message, target-system, and
  target-component fields
- **AND** it does not use BlueOS REST, MAVLink2REST, endpoint-manager state, CS
  API, or SemConnect as the SemOps hot-path protocol

#### Scenario: Deployment metadata is separated from compatibility evidence

- **WHEN** the handoff package reports BlueOS registration, package lifecycle,
  Navigator readiness, or companion-host metadata
- **THEN** that data is labeled as deployment evidence
- **AND** compatibility claims still require MAVLink/SITL/UDP evidence
- **AND** deployment evidence does not authorize hardware command transmit
