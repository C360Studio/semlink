## ADDED Requirements

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
