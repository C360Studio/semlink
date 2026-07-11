## Why

SemOps can already consume SemLink's deterministic demo artifact envelope, but
that proof only validates the handoff path. SemLink now needs a producer-owned
SITL artifact lane that proves native ArduPilot/MAVLink fidelity from observed
runtime evidence without asking SemOps to synthesize or overclaim SITL state.

## What Changes

- Add a SemLink producer path that reads local `/api/evidence` from a running
  companion node and emits a `semlink-companion-demo-artifact-v0` artifact only
  after external MAVLink/SITL evidence is present.
- Add an optional, env-gated ArduPilot SITL e2e scenario that launches SemLink
  and `sim_vehicle.py`, then produces the same SITL-backed artifact from the
  observed runtime evidence.
- Require SITL-backed artifact node metadata to be derived from observed
  evidence, including companion node ID, MAVLink system ID, simulator family,
  vehicle source, route/source detail, and no-transmit posture.
- Fail closed when the runtime has not observed external MAVLink frames,
  decoded vehicle state, or a real SemLink commit/version source reference.
- Keep SemOps, SemConnect, CS API, BlueOS, Navigator hardware, Gazebo, and GCS
  glass out of the artifact producer hot path.
- Preserve deterministic demo artifact behavior for non-SITL reports.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `companion-e2e-demo`: Add a SITL-backed artifact producer lane that embeds an
  evidence-derived single-node report and gates source-fidelity claims on
  observed `/api/evidence`.
- `mavlink-companion-runtime`: Clarify that native MAVLink/SITL evidence can
  satisfy the autopilot wire-fidelity proof and drives the artifact source
  metadata.

## Impact

- Affected code: e2e artifact builders, optional SITL e2e tests,
  `cmd/semlink-demo`, SITL/demo scripts, and focused unit tests around
  evidence-derived artifact generation.
- Affected docs/specs: companion demo artifact docs and ArduRover SITL demo
  guidance.
- Dependencies: no new SemOps, SemConnect, CS API, BlueOS, Gazebo, Navigator
  hardware, or network dependency in the producer path.
- Downstream systems: SemOps can continue consuming the existing envelope and
  add env-gated smoke assertions once SemLink provides a real SITL-backed
  artifact path.
