# ADR 001: SemGCS Product Boundary

## Status

Superseded by [ADR 003](003-companion-mesh-product-boundary.md).

This ADR remains the historical boundary for the implemented SemGCS telemetry
control-plane demo, including the SemConnect bridge and the "no MAVSDK for the
demo surface" dependency posture. It is no longer the forward product spec for
SemLink.

## Context

The demo needs to prove high-volume telemetry handling and semantic control-plane value without turning SemStreams into
a ground-control station.

## Decision

SemLink owns the robotics product layer:

- MAVLink frame support and source adapters
- simulator, replay, UDP, and future PX4 SITL input
- operator command vocabulary
- GCS dashboard and demo narrative

SemStreams owns the substrate:

- NATS and JetStream transport
- graph-ingest mutation and query subjects
- `ENTITY_STATES`
- projection ownership contracts
- indexing profiles for content, control, signal, and trace

SemLink writes current state through SemStreams graph mutation subjects. Raw telemetry is retained in a bounded stream
lane and summarized into signal-profiled current-state entities. Operator alerts and command intents are written as
control-profiled graph entities.

SemLink may optionally project a curated standards-facing view into SemConnect.
That bridge is downstream of SemLink's MAVLink decoder/projector and publishes
low-rate CS API Systems, Datastreams, Observations, SystemEvents, and Command
metadata for interoperability. It does not make CS API the internal robotics
model and does not route raw MAVLink through SemConnect.

The first demo topology keeps SemLink and SemConnect on separate
NATS/SemStreams stacks and connects them through HTTP. A one-NATS topology is a
valid later integration mode, but it must assign a single owner for each graph
processor to avoid duplicate request responders, stream/KV ownership ambiguity,
and unclear demo semantics.

## Consequences

This keeps high-rate telemetry from becoming an embedding workload while preserving graph visibility for governed state.
PX4 SITL should arrive as another source adapter that feeds the same MAVLink decoder and projector path. SemLink should
not depend on MAVSDK or an equivalent vehicle SDK for this demo surface.

The SemConnect projection gives standards-oriented sponsors an OGC API Connected
Systems entry point while preserving the product boundary: operators use the
GCS-native path, and standards consumers receive a decimated/enriched view.

Missing generic substrate primitives discovered by the demo should become SemStreams issues. MAVLink and GCS-specific
behavior stays here.
