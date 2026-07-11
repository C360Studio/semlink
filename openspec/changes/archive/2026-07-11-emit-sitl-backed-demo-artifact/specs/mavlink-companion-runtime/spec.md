## ADDED Requirements

### Requirement: SITL Artifact Metadata Uses Native MAVLink Evidence

SemLink SHALL derive SITL artifact source metadata from native companion runtime
evidence.

#### Scenario: Native evidence populates SITL source metadata

- **WHEN** SemLink produces a SITL-backed demo artifact
- **THEN** the artifact node metadata identifies the companion node, MAVLink
  system ID, simulator family, vehicle source, source route or listen address,
  and no-transmit posture from SemLink runtime evidence or explicit
  producer-owned script configuration
- **AND** the metadata does not require CS API, SemConnect, BlueOS REST,
  MAVLink2REST, endpoint-manager state, SemOps headers, or downstream COP/GCS
  projection

#### Scenario: Deterministic and SITL fidelity remain distinct

- **WHEN** SemLink has deterministic companion evidence but no observed external
  MAVLink/SITL evidence
- **THEN** SemLink does not claim `sitl-backed` source fidelity for that run
- **AND** any downstream consumer can distinguish deterministic proof from
  native SITL/autopilot wire proof by inspecting the artifact source metadata
