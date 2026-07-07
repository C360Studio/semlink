# Final Validation

Task 5.4 closes the companion-mesh pivot with a repo-local validation pass.

Validated on 2026-07-06:

- `go test ./...`
- `go build ./...`
- `openspec validate --all --strict`
- `git diff --check`

The local test suite covers the implementation slices added by this change:

- companion multi-node harness behavior
- mesh envelope, watermark, unreliable-link, HTTP transport, and raw-MAVLink
  exclusion behavior
- observe-only rule traces and simulator command gates
- hardware transmit blocking
- BlueOS metadata/read-only smoke contracts
- ArduPilot SITL command-shape contracts
- local evidence API and downstream consumer metadata

The demo script now includes the same local validation backstop. Docker Compose,
ArduPilot SITL, BlueOS extension smoke, and Navigator read-only hardware checks
remain operator-invoked fidelity lanes because they require Docker, sibling
checkouts, SITL tooling, or attached hardware.
