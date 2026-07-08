## ADDED Requirements

### Requirement: Handoff Packages Fail Closed On Hardware Command Transmit

SemLink deployable handoffs SHALL keep hardware MAVLink command transmit
disabled unless a later accepted OpenSpec change defines authorization and
safety evidence.

#### Scenario: Hardware command transmit is attempted from a handoff package

- **WHEN** a handoff package is running in a hardware or non-simulator context
  and a MAVLink command transmit is requested
- **THEN** SemLink rejects the transmit attempt
- **AND** local evidence records that hardware command transmit is outside the
  accepted handoff scope

#### Scenario: Simulator command evidence remains available

- **WHEN** a handoff package runs a simulator-only command gate
- **THEN** SemLink records preflight, ACK, and post-state evidence
- **AND** that simulator evidence does not authorize hardware transmit
