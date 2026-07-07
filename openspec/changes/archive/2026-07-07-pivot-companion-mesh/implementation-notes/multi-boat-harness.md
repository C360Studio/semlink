# Multi-Boat Harness Slice

Task 2.2 adds the first deterministic companion runtime harness:

- `internal/mavlink.NewSimulatorWithStart` lets tests replay the same MAVLink
  frame stream from a fixed epoch while preserving the existing wall-clock
  `NewSimulator` behavior for the live demo.
- `internal/companion.NewHarness` creates N simulated companion nodes with one
  local MAVLink simulator, one projector, and one local graph island per node.
- Each node decodes MAVLink frames, applies the existing projector, and stores
  SemStreams-shaped graph projections in its own in-memory graph.
- `Harness.Step` and `Harness.StepAt` return node snapshots containing frame
  counts, projection counts, entity IDs, vehicles, and alerts for deterministic
  assertions and future UI/demo wiring.

This is intentionally not mesh federation and not a full SemStreams/NATS
runtime per simulated boat yet. The harness proves the local-node boundary and
gives the mesh slice a deterministic state source. Replacing `LocalGraph` with
a real per-node SemStreams runtime should be a narrow adapter change once the
mesh envelope and watermark contracts are defined.
