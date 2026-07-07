# HTTP Demo Transport

Task 3.4 adds one concrete demo transport for mesh summaries.

Implementation:

- `internal/mesh.HTTPTransport` exposes an explicit peer HTTP/JSON surface.
- `GET /mesh/v1/watermarks` returns compact local watermarks for inspection.
- `POST /mesh/v1/diff` accepts a peer watermark set and returns a bounded
  `Diff`.
- `PullFrom` lets a node request a peer diff and apply returned items to its
  local `SummaryIndex`.
- Tests use `httptest` to prove missing-summary pull, bounded catch-up rounds,
  watermarks inspection, bad-request handling, and absolute peer URL validation.

Transport choice:

- This slice chooses explicit HTTP peer pull for the first demo because it is
  inspectable, stdlib-only, and easy to fault-inject in tests.
- NATS leaf-node, Zenoh, and WebSocket federation remain future candidates.
- The transport carries selected mesh summaries only; raw MAVLink replication is
  still excluded and remains the next mesh slice proof.
