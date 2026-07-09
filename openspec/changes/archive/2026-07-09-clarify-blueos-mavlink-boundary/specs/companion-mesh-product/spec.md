## ADDED Requirements

### Requirement: BlueOS Packaging Is Optional Product Lane

SemLink SHALL support BlueOS/Navigator packaging as an optional deployment lane
without making it the product, mesh, or SemOps protocol boundary.

#### Scenario: Product boundary is evaluated

- **WHEN** SemLink work chooses between native, SITL, companion-Pi, Navigator,
  or BlueOS deployment targets
- **THEN** the work keeps SemLink's product boundary on vehicle-local MAVLink
  companion behavior, local evidence, and mesh-node state
- **AND** SemOps continues to consume SemLink through evidence and native
  readback contracts rather than BlueOS-specific APIs
- **AND** SemConnect and CS API remain downstream standards projection edges
  unless a later accepted OpenSpec change moves that responsibility

#### Scenario: BlueOS-first work is proposed

- **WHEN** a slice starts with BlueOS extension or Navigator-readiness work
- **THEN** it records the equivalent MAVLink-native or SITL compatibility proof
  needed for non-BlueOS deployments
- **AND** it does not reject plain native companion deployments merely because
  BlueOS service metadata is unavailable
