# SemOps Contract Recheck

Date: 2026-07-08

Scope: task 4.5, recheck SemLink's draft SemOps readback adapter after the
SemOps contract branch was pushed and the hold-out review was dispositioned.

## Baseline Checked

- SemOps checkout: `/Users/coby/Code/c360/semops`
- Branch: `codex/cop-component-telemetry`
- Head: `e6cba7e docs(openspec): refresh upstream ask deferrals`
- Contract: `docs/contracts/semlink-companion-readback-v0.md`
- Archived spike:
  `openspec/changes/archive/2026-07-08-semlink-semops-contract-spike`

The SemOps spike is archived and its task list records the final SemLink
feedback/disposition work as complete, including the MVP decision: native hot
path, optional CS API / SemConnect projection edge.

## Result

SemLink's local adapter and fixtures still match the pushed SemOps v0 baseline.
`diff -qr` between the SemLink and SemOps
`testdata/contracts/semlink-companion-readback-v0` directories produced no
output.

No adapter schema change is required for SemLink.

## MVP Boundary

SemLink keeps the SemOps readback path native:

- `POST /api/cop/semlink/ardupilot/readback`
- `contract: c360.semops.semlink.ardupilot.readback.v0`
- MAVLink-native `target_system_id`, `target_component_id`, `command_id`, and
  `requested_message_id`
- `companion_node_id`, `correlation_id`, `idempotency_key`, `requested_at`, and
  `ttl_seconds`

SemLink does not send `target_asset_id`, prove born targets, mint trusted
SemOps operator headers, or require CS API / SemConnect in the hot path.

For MVP, CS API remains SemOps / SemConnect standards-edge work. The mirrored
`csapi-projection.accepted.json` fixture is retained only as interop evidence
that SemOps can project the accepted native intent without making SemLink a CS
API runtime dependency.

## Follow-Up

Runtime enablement remains deferred until a later task deliberately wires the
adapter into package execution. The next implementation work should move to
fail-closed command safety evidence before any live hardware transmit path is
considered.
