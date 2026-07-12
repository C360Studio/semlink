## Context

The simple mesh demo currently starts N deterministic companion nodes and has
each node pull selected summaries from every other node. The existing e2e test
proves count-based 3-node convergence, while lower-level mesh tests cover
bounded diffs, watermarks, duplicate delivery, TTL expiry, and raw MAVLink
exclusion.

That is enough for a first happy-path proof, but it leaves a documentation and
test gap: maintainers cannot see which origin node and vehicle references are
visible from each node's perspective, and the e2e lane does not exercise common
intermittent RF shapes.

## Goals / Non-Goals

**Goals:**

- Report per-node visible origin vehicle references in deterministic mesh
  evidence.
- Add table-driven e2e perspective tests for 3-node full mesh, line topology,
  partition-heal, and late-joiner cases.
- Preserve one simulated MAVLink vehicle per companion node.
- Keep deterministic mesh tests fast and local, with no SemOps, SemConnect, CS
  API, SITL, Gazebo, BlueOS, Navigator, or hardware dependency.

**Non-Goals:**

- No hardware transmit behavior.
- No raw MAVLink frame replication.
- No CS API-first runtime path.
- No probabilistic RF simulator or external network emulator in this slice.
- No SemOps artifact-consumption smoke in this repo-owned deterministic matrix.

## Decisions

1. **Use deterministic peer pull plans instead of a network emulator.**

   The matrix should prove SemLink's selected-state synchronization behavior
   and node perspectives. Explicit peer-index plans make topology intent easy
   to test, keep failures deterministic, and avoid turning the fast e2e lane
   into an RF simulator.

2. **Expose visible origin references from mesh watermarks.**

   Per-node visible origin vehicles are derived from the node's selected-state
   watermarks after sync and include both origin node and origin vehicle ID.
   This keeps the report aligned with the actual mesh catch-up surface instead
   of adding a separate bookkeeping path. It also avoids pretending MAVLink
   system IDs are globally unique across companion nodes.

3. **Keep production demo defaults full mesh.**

   The runnable simple mesh demo remains a clear N-node full-mesh proof. The
   scenario matrix lives in tests, reusing the same runtime/index/transport
   helpers so the evidence path and tests stay coupled.

## Risks / Trade-offs

- **Risk:** Deterministic topology plans can overstate real RF behavior.
  **Mitigation:** Name the matrix as deterministic selected-state proof and
  leave radio fidelity to future SITL/hardware-adjacent lanes.
- **Risk:** Count assertions can still pass if origin identity regresses.
  **Mitigation:** Tests assert exact visible origin vehicle references per
  node.
- **Risk:** Line topology convergence depends on repeated pulls.
  **Mitigation:** Tests model convergence in explicit rounds and keep bounded
  diff limits active.
