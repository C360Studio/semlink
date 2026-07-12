## MODIFIED Requirements

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

#### Scenario: Mesh perspective matrix proves visible origin vehicles

- **GIVEN** three SemLink companion nodes each own exactly one simulated MAVLink
  vehicle
- **WHEN** deterministic full-mesh, line-topology, partition-heal, and
  late-joiner selected-state scenarios run
- **THEN** each node's evidence records the exact origin node and vehicle ID
  references visible from that node's perspective
- **AND** converged scenarios show all three origin vehicles on every node
- **AND** partial stages such as an isolated partition or not-yet-joined node do
  not overclaim visibility
- **AND** compact watermarks and bounded diffs remain the synchronization
  mechanism
- **AND** raw MAVLink frames remain local-only by default
