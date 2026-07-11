## 1. Evidence-Derived Artifact Builder

- [x] 1.1 Add tests proving SITL-backed artifact generation succeeds only from
  external MAVLink evidence with observed vehicle state, blocked hardware
  transmit posture, and raw MAVLink mesh exclusion.
- [x] 1.2 Implement an evidence-derived single-node report/artifact builder
  that maps `/api/evidence` into source metadata and fail-closed assertions.

## 2. CLI And Script Producer Lane

- [x] 2.1 Add a `cmd/semlink-demo` producer mode that fetches local
  `/api/evidence`, writes an evidence-derived report, and emits a
  `sitl-backed` artifact with real SemLink commit/version metadata.
- [x] 2.2 Add or update a SITL demo script/doc path that runs the artifact
  producer after ArduPilot SITL and the SemLink companion runtime are running.

## 3. Validation

- [x] 3.1 Run focused tests for the new builder and demo CLI.
- [x] 3.2 Run broad Go and OpenSpec validation for the change.

## 4. Optional SITL E2E

- [x] 4.1 Add an env-gated ArduPilot SITL e2e test that starts a SemLink
  companion runtime, launches `sim_vehicle.py`, waits for native MAVLink
  evidence, and emits a SITL-backed artifact.
- [x] 4.2 Document the named SITL e2e command and keep it out of the default
  required test lane.
- [x] 4.3 Run focused and broad validation with the SITL lane skipped by
  default.
