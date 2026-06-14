# ADR 001: SemGCS Product Boundary

## Status

Accepted.

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

## Consequences

This keeps high-rate telemetry from becoming an embedding workload while preserving graph visibility for governed state.
PX4 SITL should arrive as another source adapter that feeds the same MAVLink decoder and projector path. SemLink should
not depend on MAVSDK or an equivalent vehicle SDK for this demo surface.

Missing generic substrate primitives discovered by the demo should become SemStreams issues. MAVLink and GCS-specific
behavior stays here.
