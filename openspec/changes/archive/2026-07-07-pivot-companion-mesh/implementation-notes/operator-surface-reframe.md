# Operator Surface Reframe

This note closes task 5.1 for `pivot-companion-mesh`.

SemLink no longer treats a repo-owned dashboard or GCS as the forward product
surface. The forward surface is:

- CLI/config for local setup and companion-node operation
- local JSON/SSE status and evidence APIs
- SemStreams-backed semantic evidence for node, vehicle, mesh, rule, and
  command state
- downstream consumption by SemOps, semstreams-ui, or standards egress

The current `cmd/semgcs-demo`, `internal/gcs`, and `ui/` tree remain historical
demo implementation. They are useful prior art and still run, but docs now call
them legacy or historical demo surfaces instead of the product direction.

Files updated:

- `README.md`
- `CLAUDE.md`
- `docs/adr/002-tak-cot-bridge.md`
- `docs/adr/003-companion-mesh-product-boundary.md`
- `docs/blueos-extension.md`
- `docs/demo-script.md`
- `configs/flows/semgcs-demo.json`
- `tickets/FEAT-002.yaml`

Follow-up 5.2 should make the API/evidence contracts concrete enough for
SemOps and semstreams-ui consumers.
