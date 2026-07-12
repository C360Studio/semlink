## Why

SemLink can already demonstrate a local mesh of companion nodes, but the
active proof briefly drifted toward a multi-autopilot companion shape. That does
not match the companion-computer deployment model: one SemLink companion runtime
sits beside one autopilot/vehicle. The next product claim should be stronger in
the right direction: multiple companion nodes, each with one local MAVLink
vehicle, synchronize selected summaries without promoting raw MAVLink
replication or requiring SITL, BlueOS, SemOps, SemConnect, or GCS glass.

## What Changes

- Make the repo-owned simple mesh demo's default proof shape `3` companion
  nodes with `1` simulated MAVLink vehicle per node.
- Require the generated mesh report to prove `expected_summaries = nodes`.
- Remove the public multi-autopilot demo knob so artifacts cannot overclaim a
  companion topology with more than one vehicle behind a node.
- Keep source fidelity `deterministic` for this proof.
- Preserve the existing raw MAVLink exclusion evidence: selected summaries
  replicate; raw frames stay local by default.
- Keep the real ArduPilot SITL lane separate as the next fidelity layer, not a
  prerequisite for the multi-companion mesh proof.

## Impact

- Affected code/tests: `cmd/semlink-demo`, `scripts/demo-mesh-companions.sh`,
  and `internal/e2e` mesh tests.
- Affected docs: root quick start and companion demo documentation.
- Downstream systems: SemOps can consume a deterministic multi-companion mesh
  artifact now, while still treating SITL-backed, hardware-adjacent, or
  multi-autopilot claims as separate evidence-gated lanes.
