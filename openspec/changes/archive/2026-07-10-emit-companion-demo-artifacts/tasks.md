## 1. Artifact Envelope Core

- [x] 1.1 Add companion demo artifact envelope types and constants in `internal/e2e`.
- [x] 1.2 Add envelope builder tests for single-node and simple-mesh reports.
- [x] 1.3 Add source-fidelity validation tests proving deterministic defaults and rejection of under-evidenced live
  claims.
- [x] 1.4 Implement artifact envelope construction with timestamp coherence and no-transmit posture.

## 2. CLI And Scripts

- [x] 2.1 Add `cmd/semlink-demo` flags for optional artifact output and producer metadata.
- [x] 2.2 Keep existing raw report output behavior unchanged for single-node and mesh demos.
- [x] 2.3 Update demo wrapper scripts to optionally emit artifact envelopes through environment variables.

## 3. Documentation

- [x] 3.1 Document artifact envelope usage and field semantics in demo docs.
- [x] 3.2 Update evidence/handoff docs to distinguish raw reports from SemOps-ingestable artifact envelopes.

## 4. Validation

- [x] 4.1 Run focused e2e tests.
- [x] 4.2 Run demo commands that emit raw reports and artifact envelopes.
- [x] 4.3 Run `go test ./...`.
- [x] 4.4 Run `go build ./...`.
- [x] 4.5 Run `openspec validate emit-companion-demo-artifacts --strict`.
- [x] 4.6 Run `openspec validate --all --strict`.
