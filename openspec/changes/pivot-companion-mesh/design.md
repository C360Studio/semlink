## Overview

This change turns the ADR 003 direction into an implementation-governing spec.
SemLink becomes the boat-local companion service: one node beside each MAVLink
vehicle, each node maintaining local semantic state and exchanging selected
current-state summaries with peers over unreliable links.

The goal is not a full COP. SemOps owns that. The goal is credible companion
behavior that a COP can consume later.

## Key Decisions

### Use Local SemStreams Per Node

Each SemLink node runs against local SemStreams state. This keeps link outages
from turning into global write outages and lets the mesh layer decide what is
worth sharing.

### Keep Mesh Causality Above SemStreams

SemStreams provides local `ENTITY_STATES`, timestamps, versions, and KV
revision/CAS behavior. The mesh protocol carries distributed metadata explicitly:
origin node, vehicle, entity, predicate group, origin sequence or hybrid logical
time, operation ID, observed time, expiry, confidence, merge policy, and payload
hash.

### Use Bounded Delta-State Synchronization

Peers exchange watermarks and send only missing or newer summaries. Full graph
snapshots waste RF bandwidth. Unbounded event ledgers create memory risk on
Raspberry Pi class devices.

### Reuse SemOps MAVLink Work Deliberately

SemOps has mature MAVLink surfaces that should be evaluated before SemLink
rewrites them: v1/v2 parser, UDP listener, bounded raw-frame lane, replay
fixtures, command helpers, COMMAND_ACK projection, and SITL gate discipline.
Reuse may be copy/port first and shared module later, depending on drift cost.

### Make Simulator Evidence First-Class

The first proof path uses deterministic multi-boat simulation and ArduPilot SITL
without Gazebo. Gazebo and physical Navigator hardware are later fidelity lanes,
not prerequisites for proving mesh behavior.

### Fail Closed On Commands

Rules are observe-only by default. Simulator command transmit requires explicit
safety profile, local override, allowlist, ACK observation, post-state polling,
and abort readiness. Hardware transmit remains out of scope until those gates
are real.

## Rollout

1. Initialize OpenSpec and accept this change.
2. Add simulator/harness tests for multiple local SemLink nodes.
3. Define mesh envelope, watermarks, TTLs, and merge classes.
4. Add one concrete transport for the unreliable-link harness.
5. Add local rule traces over mesh-visible state.
6. Add ArduPilot SITL lane without Gazebo.
7. Add simulator-only command gate.
8. Add BlueOS extension packaging and read-only hardware smoke.

## Open Questions

- Is the first transport explicit WebSocket federation, NATS leaf nodes, Zenoh,
  or a pluggable transport with one demo implementation?
- Should watermarks be per entity, per predicate group, per origin, or a
  two-level combination?
- Which rule language belongs in SemLink, and which rule substrate belongs
  upstream in SemStreams?
- Should SemOps MAVLink code move into a shared module after the first port, or
  stay repo-local until duplication pressure is proven?
- Which ArduPilot frame best models the first boat demo: Rover, skid-steer
  rover, sailboat, or a boat-specific fixture?
