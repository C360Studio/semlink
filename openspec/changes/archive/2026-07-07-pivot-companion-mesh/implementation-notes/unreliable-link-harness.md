# Unreliable Link Harness Tests

Task 3.3 adds transport-neutral tests for unreliable mesh links before choosing
the demo transport.

Implementation:

- `internal/mesh/unreliable_link_test.go` models sender and receiver nodes with
  local `SummaryIndex` instances and explicit diff delivery.
- Reconnect catch-up proves only missing or newer current summaries are sent,
  not the sender's whole current graph.
- Duplicate delivery is idempotent because repeated items do not change receiver
  watermarks or increase summary count.
- Expired telemetry is omitted from reconnect watermarks and diffs when `Now`
  is supplied.
- Bounded catch-up with `MaxItems` progresses across repeated reconnect rounds
  until the receiver reaches the sender's compact watermark.

Evidence boundary:

- No concrete transport is introduced here.
- The harness applies already-built diffs directly to the receiver index.
- Raw MAVLink frame exclusion remains for the next mesh slice.
