## Context

SemLink has converged on a boat-local MAVLink companion role: it owns MAVLink
ingress, bounded raw lanes, local evidence, command-intent evidence, and mesh
node behavior. SemOps owns GCS/COP glass and the standards-facing projection
edge with SemConnect. BlueOS/Navigator remains important for the early adopter
deployment lane, but it should not become the contract SemOps depends on.

## Goals / Non-Goals

**Goals:**

- Keep SemLink compatible with native MAVLink UDP and ArduPilot SITL deployments
  without requiring BlueOS.
- Keep BlueOS/Navigator as a valuable package and hardware-readiness lane.
- Keep SemOps readback intent MAVLink-native and independent of BlueOS REST,
  MAVLink2REST, endpoint-manager state, CS API, or SemConnect.

**Non-Goals:**

- Add BlueOS runtime integration beyond the existing package metadata and smoke
  evidence.
- Add hardware command transmit or Navigator hardware control.
- Move CS API/SemConnect into the SemLink-to-SemOps hot path.
- Change existing request fixtures or payload field names.

## Decisions

- Native MAVLink/SITL evidence remains the compatibility anchor. This preserves
  interoperability with non-BlueOS companion deployments and keeps ArduPilot
  proof focused on MAVLink command, target, ACK, and readback vocabulary.
- BlueOS/Navigator evidence is deployment metadata. Package lifecycle,
  `/register_service`, and Navigator readiness are useful for installability and
  operations, but they do not prove readback compatibility.
- The SemOps adapter does not branch on deployment profile. A BlueOS extension,
  companion Pi, SITL service, or native process sends the same SemOps readback
  request shape.

## Risks / Trade-offs

- **Risk:** BlueOS-first implementation work may drift into BlueOS-only
  assumptions. **Mitigation:** Keep compatibility gates satisfiable with native
  UDP/SITL evidence and document BlueOS metadata as deployment-only.
- **Risk:** Downstream consumers may mistake package readiness for MAVLink wire
  proof. **Mitigation:** Require SITL/UDP evidence for compatibility claims and
  keep package lifecycle evidence labeled as deployment metadata.
