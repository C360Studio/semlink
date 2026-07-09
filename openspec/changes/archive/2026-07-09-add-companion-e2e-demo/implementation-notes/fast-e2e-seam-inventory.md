# Fast E2E Seam Inventory

Task 1.1 found enough repo-native seams to build the first e2e proof without Docker, Gazebo, SemOps, SemConnect,
semstreams-ui, BlueOS, Navigator hardware, or physical MAVLink devices.

- Companion runtime seam: `internal/companion.NewHarness` already starts N deterministic companion nodes with isolated
  local graphs and MAVLink simulator input.
- Local state/evidence seam: `internal/gcs.Store`, `internal/gcs.NewServer`, and `Server.EvidenceBundle` already expose
  the `/api/evidence` contract and downstream optional posture.
- Mesh seam: `internal/mesh.SummaryIndex` and `internal/mesh.HTTPTransport` already provide selected-summary
  watermarks, bounded diffs, and raw MAVLink exclusion behavior.
- Command-safety seam: `internal/gcs` rejects handoff hardware commands through `internal/commandgate` before normal
  command submission.
- SemOps readback seam: `internal/semops.Client` and v0 readback request types already post the native companion
  contract without CS API or trusted SemOps operator headers.

The first implementation composes these seams in `internal/e2e` and returns a structured fast e2e report with node IDs,
vehicle profile, assertions, evidence bundle, hardware command posture, and fake SemOps readback results.
