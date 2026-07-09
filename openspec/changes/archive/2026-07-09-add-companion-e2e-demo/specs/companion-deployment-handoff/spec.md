## MODIFIED Requirements

### Requirement: Release Checkpoint Is Evidence-Based

SemLink SHALL define release or tag readiness for the companion handoff through repeatable local evidence rather than
calendar or manual judgment alone. Release evidence MUST include the SemLink-owned e2e/demo proof ladder or explicit
recorded waivers for lanes that are not yet required.

#### Scenario: Handoff checkpoint is proposed

- **WHEN** maintainers propose tagging or publishing a companion handoff checkpoint
- **THEN** the documented Go tests, OpenSpec validation, package smoke, fast companion e2e, single-node demo, simple mesh
  demo, and SITL/UDP evidence lane have passing results or explicit recorded waivers
- **AND** the checkpoint states that hardware command transmit remains disabled
