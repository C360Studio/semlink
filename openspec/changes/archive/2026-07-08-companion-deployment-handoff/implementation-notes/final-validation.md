# Companion Deployment Handoff Final Validation

Run date: 2026-07-08

Profile used: `configs/handoff/companion.env.example`

Result: ready for a repo-local companion handoff checkpoint proposal. This run
does not publish to Docker Hub, publish to the BlueOS Bazaar, or cut a tag.

Waivers: none.

Hardware posture: hardware MAVLink command transmit remains disabled through
`SEMLINK_HARDWARE_TRANSMIT_ENABLED=false`, `SEMLINK_COMMAND_RUNTIME_MODE=hardware-readonly`,
and `/api/evidence` reporting `hardware_transmit_status=blocked`.

## Evidence

- Focused package/config/evidence tests passed:
  `go test ./internal/blueos ./internal/handoff ./internal/gcs ./internal/sitl ./internal/commandgate ./internal/semops`
- UDP handoff evidence lane passed:
  `go test ./internal/gcs -run TestUDPEvidenceSmokeProjectsExternalMAVLinkState`
- Full repo tests passed: `go test ./...`
- Full repo build passed: `go build ./...`
- Change validation passed: `openspec validate companion-deployment-handoff --strict`
- Full OpenSpec validation passed: `openspec validate --all --strict`
- BlueOS Compose config passed: `docker compose -f compose.blueos.yml config`
- BlueOS lifecycle smoke passed: `scripts/blueos-extension-smoke.sh`

The lifecycle smoke built and started the BlueOS-style container, observed the
container healthy state, verified `/api/health`, verified `/register_service`,
and verified `/api/evidence` contains the companion evidence contract, the
configured profile, the configured node ID, and
`hardware_transmit_status=blocked`.
