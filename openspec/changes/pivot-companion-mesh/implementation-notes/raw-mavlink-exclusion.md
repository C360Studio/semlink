# Raw MAVLink Mesh Exclusion

Task 3.5 proves that raw MAVLink frames remain local by default.

Implementation:

- `internal/mesh.SourceKindRawMAVLink` reserves the raw-frame source kind so
  tests can exercise the boundary explicitly.
- `SourceKind.ReplicatesOverMeshByDefault` only allows projected MAVLink
  summaries, peer summaries, operator annotations, and rule evidence into the
  default summary replication path.
- `SummaryIndex.Upsert` rejects raw MAVLink items before they can contribute
  watermarks or diffs.
- `HTTPTransport.PullFrom` applies peer diffs through `SummaryIndex.Upsert`, so
  a peer cannot accidentally inject raw frames into local mesh state.

Evidence:

- `TestSummaryIndexRejectsRawMAVLinkFramesByDefault` verifies raw frames do not
  enter the local summary index or watermarks.
- `TestHTTPTransportRejectsRawMAVLinkDiffItemsByDefault` verifies the concrete
  HTTP demo transport refuses a raw-frame diff item.
