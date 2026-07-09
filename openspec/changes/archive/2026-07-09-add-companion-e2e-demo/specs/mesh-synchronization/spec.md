## MODIFIED Requirements

### Requirement: Bounded Delta-State Synchronization

SemLink SHALL synchronize selected state through bounded deltas, not full graph snapshots or unbounded ledgers. The
simple mesh demo MUST prove this behavior with local companion peers before SemLink claims broader ad hoc mesh behavior.

#### Scenario: Peer reconnects after outage

- **GIVEN** two SemLink nodes have been disconnected
- **WHEN** they reconnect
- **THEN** they exchange compact watermarks by origin and state class
- **AND** each peer sends only missing or newer selected summaries
- **AND** the catch-up does not require blasting the full local graph
- **AND** the catch-up does not depend on an unbounded event ledger

#### Scenario: Simple mesh demo catches up peers

- **GIVEN** two or more local SemLink companion nodes are started with static peer URLs
- **WHEN** the simple mesh demo mutates selected simulated vehicle state and triggers mesh synchronization
- **THEN** peers exchange compact watermarks and bounded diffs until expected selected summaries are visible
- **AND** each node's report records watermark posture, diff counts, peer counts, and assertion status

### Requirement: Deployment Mesh Configuration Is Evidence-Bearing

SemLink SHALL expose deployment mesh configuration and readiness as evidence without requiring raw MAVLink
replication. Demo evidence MUST state the mesh posture and raw MAVLink exclusion posture.

#### Scenario: Static peer configuration is supplied

- **WHEN** a handoff profile supplies static mesh peer URLs
- **THEN** local evidence reports the configured mesh posture and peer count
- **AND** mesh synchronization still uses selected summaries and watermarks

#### Scenario: Raw MAVLink exists in the deployment package

- **WHEN** the handoff package receives raw MAVLink frames locally
- **THEN** local evidence continues to state that raw MAVLink does not replicate over the mesh by default

#### Scenario: Simple mesh demo preserves raw MAVLink exclusion

- **WHEN** the simple mesh demo reports mesh synchronization evidence
- **THEN** the report states that raw MAVLink frames are local-only by default
- **AND** the report distinguishes selected semantic summaries from raw MAVLink frame replication
