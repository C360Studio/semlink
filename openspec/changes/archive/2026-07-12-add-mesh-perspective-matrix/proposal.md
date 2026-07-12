## Why

The current simple mesh proof shows three local companion nodes converging by
count, but it does not make each node's view of the mesh explicit across common
RF shapes such as line topologies, healed partitions, or late joiners. We need
that perspective matrix before claiming that the demo covers a 3-vehicle mesh
from every vehicle's companion point of view.

## What Changes

- Add deterministic mesh perspective scenarios that exercise full mesh, line,
  partition-heal, and late-joiner flows.
- Extend mesh evidence so node reports can list the origin node and vehicle
  references visible from that node's perspective.
- Keep the MVP deployment model at exactly one simulated MAVLink vehicle per
  companion node.
- Preserve raw MAVLink local-only posture and bounded selected-state diffs.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mesh-synchronization`: require topology-aware perspective proof for selected
  state catch-up without raw MAVLink replication.
- `companion-e2e-demo`: require the deterministic demo test lane to exercise a
  mesh perspective matrix and report visible origin vehicle references per
  node.

## Impact

- `internal/e2e` mesh demo helpers and tests gain topology-aware scenarios.
- Generated simple mesh reports gain per-node visible origin vehicle
  references.
- OpenSpec mesh and companion-e2e-demo specs document the stronger proof path.
- No new external dependencies, no SemOps/SemConnect runtime dependency, and no
  hardware/SITL requirement for the deterministic matrix.
