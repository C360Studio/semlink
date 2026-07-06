# Hardware Transmit Block

Task 4.5 keeps hardware command transmit out of scope for the current
companion-mesh change.

Implementation:

- `internal/commandgate.HardwareTransmitBlocker` records a structured block for
  any hardware transmit request.
- The block evidence includes the policy ID, current OpenSpec scope, requesting
  actor, target entity, requested verb, runtime mode, safety profile, timestamp,
  and the required future OpenSpec change.
- `internal/commandgate.SimulatorTransmitGate` short-circuits hardware-mode
  transmit requests into the blocker before preflight, transmitter dependency
  checks, MAVLink frame construction, ACK observation, or post-state polling.
- Simulator preflight rejection remains separate from hardware policy
  rejection, so tests can tell accidental simulator misconfiguration apart from
  the deliberate hardware boundary.

Evidence boundary:

- No hardware transmitter interface, UDP/serial target, BlueOS device write,
  arming, mode-change, mission-upload, or vehicle-control path is introduced.
- The current accepted change requires a separate OpenSpec change before
  hardware authorization and safety evidence can be designed or implemented.
