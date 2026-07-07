## ADDED Requirements

### Requirement: Deployment Mesh Configuration Is Evidence-Bearing

SemLink SHALL expose deployment mesh configuration and readiness as evidence
without requiring raw MAVLink replication.

#### Scenario: Static peer configuration is supplied

- **WHEN** a handoff profile supplies static mesh peer URLs
- **THEN** local evidence reports the configured mesh posture and peer count
- **AND** mesh synchronization still uses selected summaries and watermarks

#### Scenario: Raw MAVLink exists in the deployment package

- **WHEN** the handoff package receives raw MAVLink frames locally
- **THEN** local evidence continues to state that raw MAVLink does not
  replicate over the mesh by default
- **AND** peer synchronization does not depend on raw frame transfer
