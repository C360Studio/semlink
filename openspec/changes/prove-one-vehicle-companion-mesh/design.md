## Context

The accepted mesh requirements already say the simple mesh demo must prove
bounded selected-state synchronization with local companion peers. The product
boundary says a SemLink companion node is vehicle-local: one companion runtime
beside one MAVLink autopilot/vehicle. A mesh demo control for multiple local
autopilots makes the deterministic proof look like one companion can own more
than one vehicle, which is the wrong deployment claim for the MVP.

This change turns the guarded proof into a simple `N` companion, `N` vehicle
shape so future docs, artifacts, and release evidence cannot accidentally imply
multi-autopilot companion ownership.

## Decisions

- Use `3` nodes and `1` vehicle per node as the default mesh proof. It is still
  fast, deterministic, and local, and it matches the intended hardware model.
- Remove the mesh demo CLI/script multi-autopilot control.
- Keep the internal companion harness one-vehicle-per-node so tests and future
  demos fail early if they try to model more than one vehicle per companion.
- Keep `source_fidelity=deterministic`. This is a mesh proof, not a SITL proof.
- Keep node and vehicle counts in reports so downstream review can see the
  one-vehicle-per-companion proof shape without opening the embedded graph
  evidence first.

## Non-Goals

- Do not run one ArduPilot SITL per mesh node.
- Do not claim `sitl-backed` fidelity for deterministic mesh artifacts.
- Do not replicate raw MAVLink frames across the mesh.
- Do not move SemConnect or CS API into the companion mesh hot path.
- Do not support multiple MAVLink vehicles behind one SemLink companion runtime
  in the MVP proof.

## Follow-Ups

- Add a mixed-fidelity proof where one SemLink node is SITL-backed and the
  remaining mesh nodes stay deterministic.
- Add an `N` SITL container lane when we need to claim `N` live ArduPilot
  vehicles, still with one SITL/autopilot per companion.
