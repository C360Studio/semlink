## Why

SemLink has proved the first SemGCS demo shape: bounded raw MAVLink, current
state projection, alerts, command intent entities, a Svelte operator UI,
SemConnect egress, and TAK / CoT bridge work. SemOps has now been revived as the
complete COP and fusion product, so SemLink can become more focused.

The next credible demo is a MAVLink companion node for boat-class vehicles:
BlueOS on Raspberry Pi class hardware, Navigator-style I/O, ArduRover / boat
autopilots, local rules, and intermittent ad hoc peer state. We do not need
multiple Navigators to prove the software shape. We need deterministic multi-node
simulation, ArduPilot SITL fidelity, and a bounded mesh protocol before hardware
claims.

The SemStreams release pin was refreshed on 2026-07-06 to
`v1.0.0-beta.141`. Recent upstream work includes OpenSpec 1.5 migration,
`ENTITY_STATES` TTL correction, and graph-ingest observability/backpressure
work. SemStreams still exposes local metadata and CAS surfaces, not distributed
CRDT causality, so the SemLink mesh needs its own origin, ordering, expiry, and
merge-policy envelope.

## What Changes

- Initialize SemLink OpenSpec tracking.
- Promote ADR 003 from directional ADR into an active OpenSpec change.
- Define SemLink as a vehicle-local MAVLink companion and mesh node.
- Keep SemOps as the COP / fusion owner and SemConnect as optional standards
  egress.
- Require bounded delta-state mesh synchronization instead of full snapshots or
  unbounded ledgers.
- Require deterministic simulator/SITL gates before hardware claims.
- Require observe-only rules and simulator-only command gates before native
  transmit claims.
- Track SemOps MAVLink prior art as reuse or porting input, not as generic
  inspiration.

## Capabilities

### New Capabilities

- `companion-mesh-product`: Defines SemLink's product boundary as a MAVLink
  companion and mesh node.
- `mavlink-companion-runtime`: Defines local MAVLink ingress, projection,
  BlueOS-style packaging, simulator/SITL evidence, and SemOps prior-art reuse.
- `mesh-synchronization`: Defines bounded delta-state sync with explicit mesh
  causality metadata and merge classes.
- `rules-and-command-safety`: Defines local rule traces and safety-gated
  command transmit behavior.

### Modified Capabilities

- The old SemGCS demo becomes historical implementation baseline and prior art,
  not the forward product definition.
- TAK / CoT bridge work remains valid as an adapter, but no longer implies
  SemLink owns the broad COP identity.

## Impact

- `docs/adr/003-companion-mesh-product-boundary.md`
- `tickets/FEAT-002.yaml`
- future MAVLink parser/UDP/SITL code reuse from SemOps
- future simulator and mesh harness packages
- future Svelte operator UI language and workflows
- future SemStreams tag adoption as upstream substrate releases advance
