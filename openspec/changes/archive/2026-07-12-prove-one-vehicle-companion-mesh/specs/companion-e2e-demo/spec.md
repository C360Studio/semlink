## ADDED Requirements

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
  and assertion status
- **AND** the proof remains deterministic and does not require SITL, BlueOS,
  Navigator hardware, SemOps, SemConnect, CS API, or GCS glass

#### Scenario: Multi-companion mesh artifact remains deterministic

- **WHEN** maintainers run the simple mesh companion demo with artifact output
- **THEN** SemLink writes a `semlink-companion-demo-artifact-v0` envelope that
  embeds the multi-companion mesh report
- **AND** the artifact source fidelity is `deterministic`
- **AND** the artifact generator metadata identifies the node-count proof shape
