# Rule Trace Shape

Task 4.1 defines the rule evaluation input and trace entity shape without
introducing a rule evaluator or command gate.

Implementation:

- `internal/rules.EvaluationInput` carries the local node, vehicle context,
  evaluation time, and input facts.
- `InputFact.Scope` distinguishes local vehicle facts from peer mesh-visible
  facts.
- `TracePayload` records rule ID/version, decision, suggested action, execution
  posture, fired timestamp, input facts, and an input hash.
- `TracePayload.Projection` writes a trace-profiled SemStreams entity with
  queryable triples for the decision, suggestion, posture, input count/hash,
  input facts JSON, and fired timestamp.
- `rules.RegisterPayloads` is registered with the local SemStreams runtime.

Evidence boundary:

- Rule outputs default to observe-only posture.
- No MAVLink command transmit path is introduced.
- Actual rule firing remains for task 4.2.
