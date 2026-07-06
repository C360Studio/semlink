# ADR 003: MAVLink Companion Mesh Product Boundary

## Status

Proposed.

This ADR opens the new forward spec for SemLink and supersedes ADR 001 as the
product boundary. ADR 001 still describes the implemented SemGCS demo.

## Context

SemOps has been revived as the kitchen-sink COP and fusion project. That lets
SemLink become more focused: a MAVLink companion-computer service for vehicles
that need local state, local rules, and intermittent mesh replication.

The early-adopter shape is BlueRobotics-style hardware: BlueOS on a Raspberry Pi
class companion, Navigator as the vehicle I/O board, and ArduRover / boat-class
MAVLink vehicles. The realistic demo is not "one big GCS sees everything." It
is `N` boat-local SemLink nodes running beside the autopilot, exchanging selected
semantic current state across unreliable RF links, and giving operators or
SemOps a coherent downstream view when connectivity permits.

The SemStreams release pin was refreshed on 2026-07-06 to
`v1.0.0-beta.141`. Recent upstream work includes OpenSpec 1.5 migration,
`ENTITY_STATES` TTL correction, and graph-ingest observability/backpressure
work. The mutation path still exposes local bookkeeping and concurrency
surfaces:

- `EntityState.Version`
- `EntityState.UpdatedAt`
- `message.Triple.Timestamp`
- `UpdateEntityWithTriplesRequest.ExpectedRevision`
- mutation response `KVRevision`

Those are useful for local state and CAS, but they are not a distributed
causality or CRDT protocol. The SemLink mesh layer must therefore carry explicit
origin, ordering, expiry, and merge-policy metadata instead of treating the
six-part ID or `KVRevision` as enough.

SemOps has concrete MAVLink prior art that SemLink should reuse or port where
scoped: MAVLink v1/v2 parsing, UDP listener wiring, bounded raw-frame lane,
replay fixtures, COMMAND_LONG / COMMAND_ACK helpers, ArduPilot and PX4 SITL
gates, and command-safety evidence patterns. SemOps should keep the broad COP
and fusion UX; SemLink should keep the vehicle-local companion and mesh role.

## Decision

SemLink becomes a semantic companion node for MAVLink vehicles.

SemLink owns:

- BlueOS / companion-service packaging shape for a vehicle-local node
- MAVLink ingress from simulator, UDP, ArduPilot, PX4, and later hardware
- MAVLink egress only through safety-gated command intents
- source-local current-state projection into SemStreams
- bounded raw telemetry lanes
- local rule evaluation over vehicle state and mesh-visible peer state
- an intermittent mesh replication protocol for selected semantic state
- CLI/config shape plus local status/evidence APIs for node, vehicle, mesh,
  rule, and command evidence

SemOps owns:

- kitchen-sink COP / fusion UX
- GCS / operator glass
- cross-feed assimilation and richer operational dashboards
- multi-source correlation beyond the boat-local mesh
- campaign-scale or incident-scale workflows

semstreams-ui may consume SemLink evidence for generic ops/debug views, but
SemLink does not own a forward dashboard product.

SemStreams owns:

- NATS / JetStream substrate
- graph-ingest mutation and query subjects
- local `ENTITY_STATES`
- ownership and indexing-profile contracts
- local CAS and revision metadata
- generic substrate primitives discovered by the demo

SemConnect owns the standards-facing OGC API - Connected Systems path. SemLink
may still publish a curated, low-rate standards view, but CS API remains an
egress or interoperability surface, not the internal swarm protocol.

## Demo Shape

The next demo should show several SemLink nodes, one per simulated boat. Each
node has its own local SemStreams runtime and vehicle projection. Nodes exchange
mesh summaries over a deliberately unreliable link harness so reconnect,
partition, and duplicate-delivery behavior are visible.

The first fidelity ladder is:

1. Deterministic multi-boat SemLink simulator, no hardware.
2. ArduRover / ArduPilot SITL without Gazebo as the first autopilot parity lane.
3. Optional PX4 or Gazebo lanes only when the claim needs that simulator family.
4. BlueOS extension packaging and read-only hardware smoke.
5. Hardware command transmit only after simulator gates, explicit allowlists,
   and abort posture exist.

This avoids buying multiple Navigators before the software claim is crisp. A
mock Navigator is enough for packaging and service lifecycle tests; ArduPilot
SITL is enough for MAVLink wire behavior; physical Navigator hardware is for
the final read-only and command-gated smoke.

## Mesh Synchronization

The mesh protocol should use bounded delta-state synchronization, not full graph
snapshots and not an unbounded event ledger.

Peers exchange compact watermarks by origin and state class. A practical first
key is:

```text
origin_node_id / entity_id / predicate_group
```

Each transmitted mesh item carries an envelope such as:

```text
origin_node_id
origin_vehicle_id
entity_id
predicate_group
origin_sequence or hybrid_logical_time
operation_id
observed_at
expires_at
source_kind
confidence
merge_policy
payload_hash
```

Merge classes are explicit:

- Current telemetry is last-writer-wins only within the owning origin cell and
  always has a TTL.
- Peer health and link status are TTL-based liveness facts.
- Hazards, markers, and operator annotations use set-style semantics with
  bounded tombstones if deletes are needed.
- Rule traces are append-limited or sampled evidence, not unbounded history.
- Command intents never execute merely because they arrived over the mesh.
  Local authority, safety profile, simulator/hardware mode, ACK requirement,
  and post-state polling decide whether a command can transmit.

Raw MAVLink frames do not replicate over the mesh by default. SemLink replicates
current-state summaries, alerts, annotations, rule traces, and command evidence.

## Rules And Command Safety

Rules run locally first. A rule may observe local vehicle state, peer summaries,
mesh health, and mission annotations. The first demo should keep rules
deterministic and auditable: every rule firing should emit a trace entity that
explains input facts, decision, suggested action, and whether the action was
observe-only, simulated, or transmitted.

The default posture is observe-only. Simulator transmit is a separate gate and
should borrow SemOps command discipline:

- stable MAVLink sender identity
- heartbeat before command where needed
- narrow command allowlist
- bounded retries
- explicit `COMMAND_ACK` observation
- post-command state polling
- simulator-only confirmation
- abort-ready confirmation

Hardware transmit is out of scope until the simulator ladder proves the same
path and an operator authorization story exists.

## Transport Candidates

The first implementation can use a simple explicit peer transport, such as
WebSocket federation, because that keeps failure injection and demo inspection
straightforward.

NATS leaf nodes are attractive for edge deployments but should not be assumed to
provide automatic ad hoc mesh semantics; topology, ownership, and duplicate
responders still need deliberate design. Zenoh remains a serious candidate for
robotics-oriented peer/broker/hybrid data motion, especially if BlueOS extension
deployment makes it convenient. Transport choice should be decided after the
envelope and merge semantics are stable.

## Consequences

SemLink stops trying to become the COP. That work belongs in SemOps. SemLink can
be sharper and more credible as the boat-local companion service that turns
MAVLink telemetry, local rules, and intermittent peer state into governed
semantic state.

The old SemGCS demo remains useful. It already proves bounded raw telemetry,
current-state projection, alerts, command intent entities, a Svelte operator UI,
SemConnect egress, and TAK / CoT bridge work. The next work should refactor and
extend those pieces toward companion / mesh APIs, CLI/config, and external UI
consumption rather than grow a SemLink-owned GCS.

SemStreams index fixes should be adopted when the next tag lands, but they do
not remove the need for explicit mesh causality metadata in SemLink.

## Open Questions

- What is the smallest BlueOS extension manifest and service shape needed for a
  convincing companion-node install?
- Should the first transport be explicit WebSocket federation, NATS leaf nodes,
  Zenoh, or a pluggable transport interface with one concrete demo transport?
- Should mesh watermarks be per entity, per predicate group, per origin, or a
  two-level combination?
- Which rule language belongs here, and which rule substrate belongs upstream in
  SemStreams?
- Does SemOps MAVLink code get copied into SemLink, moved to a shared package,
  or consumed through a future shared module?
- Which ArduPilot SITL frame best represents the early adopter: Rover,
  skid-steer rover, sailboat, or a boat-specific fixture?
