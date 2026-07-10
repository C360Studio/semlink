## Why

SemOps can now ingest a SemLink companion demo artifact envelope, but SemLink
still only produces raw demo reports. Before treating the envelope as accepted
producer contract, SemLink needs to own the fields it can generate, keep raw
reports available, and avoid claiming SITL or hardware-adjacent evidence without
concrete per-node source metadata.

## What Changes

- Add a SemLink-owned companion demo artifact envelope around existing
  single-node and simple-mesh reports.
- Preserve raw report output as the default/local evidence surface.
- Add CLI flags that let maintainers emit an artifact envelope with source
  fidelity, generator, SemLink version/commit, no-transmit posture, and optional
  per-node source metadata.
- Keep deterministic demos from claiming SITL-backed or hardware-adjacent
  fidelity unless SemLink has node source evidence such as MAVLink system IDs.
- Document the SemOps admission boundary: SemLink produces native demo
  artifacts; SemOps remains the COP/GCS consumer and standards projection owner.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `companion-e2e-demo`: demo reports gain an optional producer artifact
  envelope with provenance, source-fidelity, timestamp, no-transmit, and
  per-node metadata rules.

## Impact

- Affected code: `cmd/semlink-demo`, `internal/e2e`, demo tests, and demo docs.
- Affected artifacts: generated JSON under `.artifacts/semlink-demo-*` can now
  include raw reports and SemOps-ingestable artifact envelopes.
- Dependencies: none; the producer path remains native SemLink and does not add
  a SemOps, SemConnect, CS API, BlueOS, Navigator, SITL, or Gazebo runtime
  dependency.
