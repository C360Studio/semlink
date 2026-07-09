## Why

SemLink now has clear companion, mesh, safety, and SemOps boundary specs, but it does not yet have a repo-owned
end-to-end demo shape that proves those contracts without requiring SemOps glass, SemConnect/CS API, BlueOS Navigator
hardware, Gazebo, or a full external stack. We need a lightweight proof ladder that maintainers and early adopters can
run locally to see single-node companion behavior, simple mesh catch-up, command-safety posture, and native SemOps
contract compatibility.

## What Changes

- Add a SemLink-owned companion e2e/demo capability with fast in-process tests, a single-node demo, and a simple
  multi-node mesh demo.
- Define generated evidence reports as the primary demo artifact instead of growing a repo-owned GCS UI.
- Require the demo to prove native SemOps readback contract behavior through a fake downstream receiver, with no CS API
  dependency in the hot path.
- Require mesh demo evidence for peer watermarks, bounded diffs, TTL/merge posture, and raw MAVLink exclusion.
- Require command-safety demo evidence that hardware transmit remains fail-closed while simulator-only command evidence
  is preserved.
- Keep BlueOS/SITL lanes as higher-fidelity optional proof, not the default e2e/demo dependency.

## Capabilities

### New Capabilities

- `companion-e2e-demo`: SemLink-owned local e2e and demo proof ladder for companion nodes, mesh catch-up, downstream
  adapter compatibility, and generated evidence reports.

### Modified Capabilities

- `mavlink-companion-runtime`: make the fast e2e, single-node demo, and simple mesh demo explicit layers in the simulator
  fidelity ladder.
- `mesh-synchronization`: require simple mesh demo evidence for watermark/diff catch-up and raw MAVLink exclusion.
- `rules-and-command-safety`: require demo evidence for fail-closed hardware command posture and preserved simulator-only
  command evidence.
- `companion-deployment-handoff`: include SemLink-owned e2e/demo proof in the evidence-based handoff release checkpoint.

## Impact

- Affected code areas: companion runtime harnesses, mesh HTTP transport/probes, evidence API assertions, SemOps readback
  adapter test surface, command-safety probes, demo scripts, and generated demo artifacts.
- Affected docs: companion handoff profile, evidence API/demo docs, BlueOS/package docs, and demo runbook.
- Runtime dependencies: no new required SemOps, SemConnect/CS API, semstreams-ui, Gazebo, Navigator, or hardware
  dependency for the default e2e/demo path.
- Validation impact: adds a fast local e2e gate plus optional single-node and simple mesh demo commands whose reports can
  be attached to release or handoff evidence.
