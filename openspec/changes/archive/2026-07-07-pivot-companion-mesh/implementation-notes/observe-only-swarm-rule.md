# Observe-Only Swarm Rule

Task 4.2 adds one deterministic rule over mesh-visible state.

Implementation:

- `LowBatteryPeerHoldRule` evaluates local and mesh-visible battery facts.
- The rule fires only when the local vehicle is at or below the low-battery
  threshold and at least one peer mesh summary reports reserve capacity.
- Firing emits a `TracePayload` with decision
  `suggest-hold-for-peer-coverage`, suggested action `hold-position`, and
  execution posture `observe-only`.
- The trace includes both local and mesh input facts plus the input hash from
  task 4.1.

Evidence boundary:

- The rule returns trace evidence only.
- No command intent is written.
- No MAVLink transmit or simulator command gate is introduced in this slice.
