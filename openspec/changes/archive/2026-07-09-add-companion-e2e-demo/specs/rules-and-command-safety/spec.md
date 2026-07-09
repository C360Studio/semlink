## MODIFIED Requirements

### Requirement: Simulator Command Gate Precedes Native Transmit

SemLink MUST prove command transmit in simulator before any hardware transmit path exists. The e2e/demo proof ladder
MUST preserve simulator command evidence separately from hardware authorization.

#### Scenario: Simulator command transmit is attempted

- **GIVEN** the runtime is in simulator mode
- **WHEN** SemLink attempts a MAVLink command
- **THEN** the command must pass a safety profile, local override, narrow allowlist, ACK requirement, post-state polling
  requirement, simulator-only confirmation, and abort-ready confirmation
- **AND** the command result records ACK and post-state evidence

#### Scenario: Demo preserves simulator command evidence

- **GIVEN** a companion e2e or demo lane runs in simulator mode
- **WHEN** SemLink records command evidence
- **THEN** the generated report records preflight, ACK, post-state, and simulator-only posture
- **AND** that evidence remains explicitly separate from hardware transmit authorization

### Requirement: Handoff Packages Fail Closed On Hardware Command Transmit

SemLink deployable handoffs SHALL keep hardware MAVLink command transmit disabled unless a later accepted OpenSpec
change defines authorization and safety evidence. The e2e/demo proof ladder MUST assert that hardware command transmit
remains fail-closed.

#### Scenario: Hardware command transmit is attempted from a handoff package

- **WHEN** a handoff package is running in a hardware or non-simulator context and a MAVLink command transmit is
  requested
- **THEN** SemLink rejects the transmit attempt
- **AND** local evidence records that hardware command transmit is outside the accepted handoff scope

#### Scenario: Simulator command evidence remains available

- **WHEN** a handoff package runs a simulator-only command gate
- **THEN** SemLink records preflight, ACK, and post-state evidence
- **AND** that simulator evidence does not authorize hardware transmit

#### Scenario: Demo proves hardware transmit remains blocked

- **WHEN** the companion e2e or demo lane probes hardware or non-simulator command transmit posture
- **THEN** SemLink rejects the transmit attempt or reports it unavailable
- **AND** `/api/evidence` and the generated report record that hardware command transmit is blocked
