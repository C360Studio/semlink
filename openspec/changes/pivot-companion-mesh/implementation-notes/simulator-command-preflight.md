# Simulator Command Preflight

Task 4.3 adds a fail-closed preflight gate before any simulator command
transmit path exists.

Implementation:

- `internal/commandgate.PreflightGate` evaluates command transmit readiness.
- The default gate is simulator-only and allowlists only
  `request-autopilot-version`, a read-side command shape suitable for a later
  `MAV_CMD_REQUEST_MESSAGE` / `AUTOPILOT_VERSION` proof.
- Preflight requires a safety profile, target entity, requester, stable sender
  identity, bounded attempts, local override, ACK requirement, post-state
  polling requirement, simulator-only confirmation, and abort-ready
  confirmation.
- The default safety profile is explicitly allowlisted as
  `simulator-readback-v1`; arbitrary non-empty profile names do not pass.
- The result records each check plus structured evidence for the eventual
  command evidence entity.

Evidence boundary:

- No MAVLink frame is generated.
- No command intent is executed.
- Hardware mode is rejected before any transmit-capable path can use it.
