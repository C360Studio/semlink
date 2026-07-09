## 1. Harness And Fast E2E

- [x] 1.1 Inventory companion runtime, evidence API, mesh transport, command safety, and SemOps readback adapter seams.
- [x] 1.2 Add a fast e2e harness that starts deterministic companion nodes without Docker or external services.
- [x] 1.3 Verify local state projection, evidence-compatible state, downstream optional posture, and no CS API hot-path
  dependency in the fast e2e lane.
- [x] 1.4 Add a fake SemOps readback receiver that validates the v0 native contract fixtures without SemOps runtime.
- [x] 1.5 Emit a structured fast e2e report or test artifact with node IDs, profile, assertions, and adapter results.

## 2. Single-Node Demo

- [x] 2.1 Add a single-node demo runner that launches one companion runtime with deterministic local simulated input.
- [x] 2.2 Add probe assertions for runtime readiness, `/register_service`, `/api/evidence`, companion profile, and
  downstream dependency posture.
- [x] 2.3 Add command-safety probes proving hardware transmit is blocked and simulator command evidence remains distinct.
- [x] 2.4 Write single-node report artifacts under a deterministic `.artifacts` path.
- [x] 2.5 Document the single-node demo command, expected report fields, and waiver rules.

## 3. Simple Mesh Demo

- [x] 3.1 Add a local multi-node runner with deterministic companion identities and static peer URLs.
- [x] 3.2 Drive selected simulated vehicle state changes and trigger bounded mesh synchronization.
- [x] 3.3 Assert watermark exchange, bounded diff counts, peer counts, TTL/merge posture, and selected summary visibility.
- [x] 3.4 Assert raw MAVLink frame exclusion from mesh replication in evidence and reports.
- [x] 3.5 Write simple mesh report artifacts under a deterministic `.artifacts` path.
- [x] 3.6 Document the simple mesh demo command, expected report fields, and waiver rules.

## 4. Boundary And Docs

- [x] 4.1 Update evidence API and handoff docs to describe SemLink-owned demo reports as consumer-oriented evidence.
- [x] 4.2 Update BlueOS/package docs to state that single-node and simple mesh demos do not require Navigator hardware,
  Gazebo, SemOps, SemConnect, or semstreams-ui.
- [x] 4.3 Ensure docs keep SemOps, semstreams-ui, and SemConnect as optional downstream consumers.
- [x] 4.4 Ensure docs state CS API/SemConnect is not a SemLink runtime hot-path dependency for MVP demos.

## 5. Validation

- [x] 5.1 Run the focused fast e2e test command and record the result.
- [x] 5.2 Run the single-node demo command and record the generated report path or waiver.
- [x] 5.3 Run the simple mesh demo command and record the generated report path or waiver.
- [x] 5.4 Run `go test ./...`.
- [x] 5.5 Run `go build ./...`.
- [x] 5.6 Run `openspec validate add-companion-e2e-demo --strict`.
- [x] 5.7 Run `openspec validate --all --strict`.
