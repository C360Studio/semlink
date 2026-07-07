# Mesh Watermarks And Diffs

Task 3.2 adds the first bounded delta-state synchronization primitive.

Implementation:

- `internal/mesh.SummaryIndex` stores the latest selected mesh summary per
  origin node, origin vehicle, predicate group, and entity ID.
- `internal/mesh.WatermarkSet` exposes compact watermarks by origin/state cell.
  Each watermark carries the highest origin sequence or HLC, operation ID,
  observed time, item count, and a deterministic cell hash.
- `SummaryIndex.Diff` compares local watermarks with a peer's watermarks and
  returns only missing or newer current summaries for normal catch-up.
- When a peer has the same high-water mark but a different cell count/hash, the
  diff falls back to a bounded cell repair for that origin/state cell.
- Expired items are omitted when the caller supplies a `Now` value.

Evidence boundary:

- This is still transport-neutral.
- The index is not an event ledger; replaced current summaries do not accumulate
  as replay history.
- Cell hashes repair skipped summaries without requiring a full local graph
  snapshot.
