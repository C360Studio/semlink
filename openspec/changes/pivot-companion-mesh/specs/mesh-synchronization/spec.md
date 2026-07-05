# Mesh Synchronization Specification

## ADDED Requirements

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

SemLink SHALL synchronize selected state through bounded deltas, not full graph
snapshots or unbounded ledgers.

#### Scenario: Peer reconnects after outage

- **GIVEN** two SemLink nodes have been disconnected
- **WHEN** they reconnect
- **THEN** they exchange compact watermarks by origin and state class
- **AND** each peer sends only missing or newer selected summaries
- **AND** the catch-up does not require blasting the full local graph
- **AND** the catch-up does not depend on an unbounded event ledger

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
