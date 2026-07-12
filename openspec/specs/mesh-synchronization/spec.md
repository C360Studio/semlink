# mesh-synchronization Specification

## Purpose

Define bounded delta-state synchronization for selected companion state across
unreliable peer links. Mesh causality, expiry, confidence, merge behavior, and
raw-MAVLink exclusion are explicit SemLink protocol responsibilities.
## Requirements
### Requirement: Mesh Envelope Carries Distributed Metadata

Every mesh item SHALL carry explicit distributed synchronization metadata.

#### Scenario: State summary is transmitted

- **WHEN** a SemLink node sends a mesh state summary
- **THEN** the item includes origin node, origin vehicle, entity ID, predicate
  group, origin sequence or hybrid logical time, operation ID, observed time,
  expiry, source kind, confidence, merge policy, and payload hash

#### Scenario: SemStreams metadata is present

- **WHEN** a mesh item was derived from SemStreams state
- **THEN** local `KVRevision`, `EntityState.Version`, or timestamps may be
  included as evidence
- **AND** they are not treated as distributed causality metadata

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

### Requirement: Merge Classes Are Explicit

SemLink SHALL declare merge behavior by state class.

#### Scenario: Telemetry summary is merged

- **WHEN** a node receives current telemetry for a peer vehicle
- **THEN** it applies last-writer-wins only within the owning origin cell
- **AND** the fact expires when its TTL passes

#### Scenario: Durable annotation is merged

- **WHEN** a node receives a hazard, marker, or operator annotation
- **THEN** the merge uses set-style semantics
- **AND** deletes use bounded tombstones if delete behavior exists

#### Scenario: Raw MAVLink is present locally

- **WHEN** raw MAVLink frames are available on a node
- **THEN** they do not replicate over the mesh by default

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

### Requirement: Mesh Catch-Up Preserves One Vehicle Per Companion

SemLink SHALL prove that selected-state mesh catch-up is keyed by origin node
and vehicle identity while preserving the MVP deployment model of one vehicle
per companion node.

#### Scenario: Multiple companion nodes synchronize one local vehicle each

- **GIVEN** each SemLink node owns exactly one simulated MAVLink vehicle
- **WHEN** peers exchange compact watermarks and selected diffs
- **THEN** every peer receives one selected summary per origin vehicle
- **AND** expected summary counts equal `node_count`
- **AND** raw MAVLink frames remain excluded from mesh replication by default
