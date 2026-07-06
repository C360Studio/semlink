# Mesh Envelope

Task 3.1 defines the transport-neutral mesh envelope before watermarks,
diffing, or transports are implemented.

Implementation:

- `internal/mesh.Envelope` carries origin node, origin vehicle, entity ID,
  predicate group, origin sequence or hybrid logical time, operation ID,
  observed time, expiry, source kind, confidence, merge policy, payload hash,
  and optional SemStreams evidence.
- `internal/mesh.Item` wraps an envelope with JSON payload bytes and verifies
  the payload hash.
- SemStreams metadata is explicitly evidence only. `KVRevision`,
  `EntityState.Version`, and timestamps do not satisfy distributed ordering.
- Merge/source enums are narrow and explicit so later diff and transport work
  cannot silently invent merge behavior.

Evidence boundary:

- No peer watermarks are generated yet.
- No mesh transport exists yet.
- Raw MAVLink frames remain outside the mesh item model by default.
