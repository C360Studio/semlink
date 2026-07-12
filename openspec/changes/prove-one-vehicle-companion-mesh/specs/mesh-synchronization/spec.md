## ADDED Requirements

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
