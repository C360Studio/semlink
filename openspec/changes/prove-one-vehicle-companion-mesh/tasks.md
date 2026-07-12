## 1. Spec

- [x] 1.1 Add a companion e2e demo requirement for one-vehicle-per-companion
  mesh proof.
- [x] 1.2 Add a mesh synchronization scenario for `N` companion nodes and `N`
  selected summaries.

## 2. Implementation

- [x] 2.1 Make the default mesh demo command/script use `3` nodes with one
  simulated vehicle per node.
- [x] 2.2 Update mesh e2e tests to assert three selected summaries for three
  companions.
- [x] 2.3 Remove the public multi-autopilot demo knob and keep artifact
  generator metadata aligned with the one-vehicle-per-companion proof.

## 3. Docs And Validation

- [x] 3.1 Update docs to describe the multi-companion mesh proof and evidence
  boundary.
- [x] 3.2 Run focused e2e tests.
- [x] 3.3 Run the mesh demo with artifact output.
- [x] 3.4 Run broad Go/OpenSpec validation.
