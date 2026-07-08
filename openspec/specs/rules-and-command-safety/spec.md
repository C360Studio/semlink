# rules-and-command-safety Specification

## Purpose

Define observe-first swarm rule evidence and command safety gates. Simulator
command transmit must prove preflight, ACK, and post-state evidence before any
future hardware transmit path can be proposed.

## Requirements
### Requirement: Rules Emit Auditable Traces

SemLink rules SHALL emit trace evidence for decisions made from local or
mesh-visible state.

#### Scenario: Rule fires from mesh-visible state

- **GIVEN** a local rule observes local vehicle state and peer mesh summaries
- **WHEN** the rule fires
- **THEN** SemLink records input facts, decision, suggested action, execution
  posture, and timestamp as rule trace evidence
- **AND** the trace is available through local evidence APIs and semantic
  evidence for external operator surfaces

### Requirement: Rules Are Observe-Only By Default

SemLink SHALL default rule outputs to observe-only suggestions.

#### Scenario: Rule suggests a vehicle action

- **WHEN** a rule suggests a course, hold, return, speed, or mission action
- **THEN** SemLink records the suggestion as evidence
- **AND** it does not transmit a MAVLink command unless a command gate accepts
  the action

### Requirement: Simulator Command Gate Precedes Native Transmit

SemLink MUST prove command transmit in simulator before any hardware transmit
path exists.

#### Scenario: Simulator command transmit is attempted

- **GIVEN** the runtime is in simulator mode
- **WHEN** SemLink attempts a MAVLink command
- **THEN** the command must pass a safety profile, local override, narrow
  allowlist, ACK requirement, post-state polling requirement, simulator-only
  confirmation, and abort-ready confirmation
- **AND** the command result records ACK and post-state evidence

### Requirement: Hardware Transmit Is Out Of Scope

SemLink SHALL block hardware command transmit until a separate accepted
OpenSpec change defines the authorization and safety evidence.

#### Scenario: Hardware command path is requested

- **WHEN** a feature or operator path would transmit a command to hardware
- **THEN** the current companion-mesh slice blocks the path
- **AND** the repo requires a new OpenSpec change before implementation

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
