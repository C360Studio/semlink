# SemOps Readback Adapter

Status: draft-contract adapter for the SemOps `semlink-companion-readback-v0`
fixtures.

SemLink can submit the MVP ArduPilot readback intent to SemOps through a native
JSON contract without putting CS API or SemConnect in the hot path. The adapter
is intentionally narrow:

- `POST /api/cop/semlink/ardupilot/readback`
- `contract: c360.semops.semlink.ardupilot.readback.v0`
- `command_id: 512` (`MAV_CMD_REQUEST_MESSAGE`)
- `requested_message_id: 148` (`AUTOPILOT_VERSION`)
- MAVLink-native `target_system_id` and `target_component_id`

The adapter lives in `internal/semops` and is fixture-tested against mirrored
SemOps v0 contract examples in `testdata/contracts/semlink-companion-readback-v0`.
The fixtures are mirrored locally so SemLink tests do not depend on a sibling
checkout or an unpushed SemOps branch.

The mirrored `csapi-projection.accepted.json` fixture is a hold-out review
artifact for SemOps/SemConnect interop. SemLink tests keep it in sync with the
native readback facts, but the adapter does not consume CS API in the hot path.

The 2026-07-08 recheck against the pushed SemOps
`codex/cop-component-telemetry` baseline found no fixture drift. SemOps owns
the standards projection edge for MVP; SemLink keeps this path native and does
not add CS API runtime coupling for the companion handoff.

## Boundary

The adapter does not mint trusted SemOps operator headers. Production trust
translation belongs to a SemOps deployment gateway, sidecar, or test harness.
SemLink only identifies itself as a companion node in the request body.

The adapter does not send `target_asset_id`; SemOps derives and proves the
canonical COP target before graph writes.

The adapter does not authorize hardware transmit. It submits command intent to
SemOps and expects the response to preserve `native_execution_allowed=false`
and `companion_transmit_allowed=false` for this MVP path.

`/api/evidence` remains the read-only pull surface for SemOps, semstreams-ui,
and operator tooling. This adapter is the separate push/admission path for the
draft readback intent contract.
