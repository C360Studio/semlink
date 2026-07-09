## ADDED Requirements

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
