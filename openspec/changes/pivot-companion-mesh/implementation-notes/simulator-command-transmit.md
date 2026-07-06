# Simulator Command Transmit Gate

Task 4.4 adds a simulator-only transmit gate behind the 4.3 preflight.

Implementation:

- `internal/mavlink` now supports the narrow MAVLink command surface needed for
  the first simulator proof:
  - `COMMAND_LONG`
  - `COMMAND_ACK`
  - `MAV_CMD_REQUEST_MESSAGE`
  - `AUTOPILOT_VERSION` as the requested readback message ID
- `internal/commandgate.SimulatorTransmitGate` builds a `COMMAND_LONG`
  request-message frame only after preflight accepts the request.
- The gate transmits through an injected `MAVLinkTransmitter`; no UDP, serial,
  hardware, or BlueOS device path is introduced in this slice.
- The gate waits for injected `COMMAND_ACK` evidence, retries only within the
  bounded attempt count accepted by preflight, and increments the
  `COMMAND_LONG.confirmation` field on retry.
- After an accepted ACK, the gate requires injected post-state polling evidence
  before it returns an accepted result.

Evidence boundary:

- The result records preflight checks, transmitted-frame evidence, ACK evidence,
  and post-state evidence.
- The default command remains read-side: request `AUTOPILOT_VERSION`.
- Hardware transmit remains blocked for task 4.5 and requires a separate
  OpenSpec change before any real hardware path exists.
