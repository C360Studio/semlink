# SemStreams Beta.160 Migration

This guide is the operational and downstream-consumer contract for SemLink's
breaking move from SemStreams `v1.0.0-beta.141` to
`v1.0.0-beta.160`. The accepted tag resolves to commit
`8403a2218000e45a31c5132fbfe01af42ed04f14`; `go.mod`, `go.sum`, and
`configs/dependencies/semstreams-beta160.json` pin and verify that decision.

The migration must complete before the MAVLink projection surface expands.
It changes graph mutation and predicate contracts, but it does not expand the
supported MAVLink dialect, authorize command transmission, or establish a
hardware-readiness claim.

## Contract Semantics

Beta.160 removes semantic ownership from projection contracts. A SemLink
contract now validates producer intent and graph shape locally. It declares a
contract name, entity pattern, indexing profile, and stable named predicate
groups with an operation mode. It does not:

- reserve a predicate;
- authorize a write;
- grant exclusive runtime ownership; or
- coordinate producers across processes.

SemLink registers every custom predicate before it validates the complete
contract set. All current groups use `projection.ModeReconcile` because each
write supplies the complete desired state for that group:

| Producer | Contract group | Mode |
| --- | --- | --- |
| MAVLink vehicle state | `vehicle-current` | `reconcile` |
| Alert state | `alert-current` | `reconcile` |
| Command intent | `command-current` | `reconcile` |
| CoT operator state | `operator-current` | `reconcile` |
| CoT marker state | `marker-current` | `reconcile` |
| CoT message state | `message-current` | `reconcile` |
| Rule trace state | `trace-current` | `reconcile` |

Runtime startup validates all seven contracts before graph-ingest starts or a
mutation client is constructed. A validation error prevents readiness and is
returned as `validate SemLink projection contracts: ...`.

## Predicate Rename Map

Beta.160 custom predicates must have exactly three dot-separated segments and
use lower-kebab spelling. There are no aliases for the former names. Writers,
queries, rules, fixtures, and downstream consumers must switch together.

### Robot predicates

| Beta.141 predicate | Beta.160 predicate |
| --- | --- |
| `robot.identity.system_id` | `robot.identity.system-id` |
| `robot.identity.vehicle_type` | `robot.identity.vehicle-type` |
| `robot.link.last_seen_unix_ms` | `robot.link.last-seen-unix-ms` |
| `robot.telemetry.sample_unix_ms` | `robot.telemetry.sample-unix-ms` |
| `robot.power.battery_remaining_pct` | `robot.power.battery-remaining-pct` |
| `robot.power.voltage_mv` | `robot.power.voltage-mv` |
| `robot.position.latitude_deg` | `robot.position.latitude-deg` |
| `robot.position.longitude_deg` | `robot.position.longitude-deg` |
| `robot.position.altitude_m` | `robot.position.altitude-m` |
| `robot.position.ground_speed_mps` | `robot.position.ground-speed-mps` |
| `robot.position.heading_deg` | `robot.position.heading-deg` |
| `robot.alert.raised_unix_ms` | `robot.alert.raised-unix-ms` |
| `robot.command.requested_unix_ms` | `robot.command.requested-unix-ms` |

### COP predicates

| Beta.141 predicate | Beta.160 predicate |
| --- | --- |
| `cop.identity.cot_uid` | `cop.identity.cot-uid` |
| `cop.kind` | `cop.identity.kind` |
| `cop.label` | `cop.identity.label` |
| `cop.description` | `cop.content.description` |
| `cop.message.sender_uid` | `cop.message.sender-uid` |
| `cop.message.sender_entity` | `cop.message.sender-entity` |
| `cop.last_seen_unix_ms` | `cop.state.last-seen-unix-ms` |
| `cop.position.latitude_deg` | `cop.position.latitude-deg` |
| `cop.position.longitude_deg` | `cop.position.longitude-deg` |
| `cop.position.altitude_m` | `cop.position.altitude-m` |
| `cop.position.heading_deg` | `cop.position.heading-deg` |
| `cop.position.speed_mps` | `cop.position.speed-mps` |

### Rule predicates

| Beta.141 predicate | Beta.160 predicate |
| --- | --- |
| `rule.trace.rule_id` | `rule.trace.rule-id` |
| `rule.trace.rule_version` | `rule.trace.rule-version` |
| `rule.trace.suggested_action` | `rule.trace.suggested-action` |
| `rule.trace.execution_posture` | `rule.trace.execution-posture` |
| `rule.trace.input_count` | `rule.trace.input-count` |
| `rule.trace.input_hash` | `rule.trace.input-hash` |
| `rule.trace.input_facts_json` | `rule.trace.input-facts-json` |
| `rule.trace.fired_unix_ms` | `rule.trace.fired-unix-ms` |

Predicates absent from this map did not change. JSON evidence field names such
as `system_id` and `vehicle_type` are API fields, not graph predicate names,
and remain unchanged in the `v1` evidence contract.

## Mutation And Read Behavior

SemLink uses the beta.160 `projection.MutationClient`; it does not publish to
legacy graph-ingest create/update subjects or decode the old entity-query
subject. A complete projection follows this bounded sequence:

1. Reconcile the stable named group against authoritative state.
2. If the reconcile proves the entity absent and not committed, strictly
   create an entity envelope with zero triples.
3. Immediately reconcile the complete group.
4. If creation races with another writer, converge through the same group
   reconcile.

The zero-triple envelope is deliberate. Vehicle projections mix MAVLink and
projector-derived triple sources; putting those triples in `Create` would
conflict with the single required creation source. Reconcile preserves each
triple's source and timestamp.

Success requires a verified mutation receipt with a nonzero authoritative KV
revision. Revision conflicts are retried within the adapter's fixed attempt
budget. Commit-unknown results are resolved by an authoritative exact read and
desired-group comparison before retry or success. Exact reads return the
validated entity and the same entry's `KVRevision`; entity logical version and
timestamps are not revision substitutes.

## Runtime Storage And Streams

SemLink no longer provisions graph-owned buckets. Beta.160 graph-ingest owns
`ENTITY_SUFFIX_INDEX`; the removed `COMPONENT_STATUS` bucket has no
replacement. SemLink provisions only its embedding-consumer streams, and both
use file storage with bounded discard-old behavior:

| Stream | Bounds |
| --- | --- |
| `ENTITY` | 24-hour maximum age and 64 MiB maximum bytes |
| `MAVLINK_RAW` | 5-minute age, 128 MiB, 250,000 messages, 5,000 per subject |

Raw MAVLink remains on the local bounded lane and is excluded from mesh
replication by default. This migration adds no hardware command-transmit
authorization; readiness evidence must continue to report that posture
separately.

## Durable Beta.160 Schema

SemLink treats NATS/JetStream as a durable database. An empty managed namespace
is required only for first beta.160 initialization. Before provisioning graph
resources, SemLink compare-and-set creates this exact stamp:

```text
SEMLINK_RUNTIME_META/state-schema-version=beta.160
```

The stamp is durable, has no TTL, and is schema authority for later starts. A
graceful or unexpected restart with the exact stamp validates owned resource
configuration, completes any missing idempotent provisioning, and reopens the
existing triples, authoritative revisions, and bounded stream data.

First initialization behaves as follows:

1. If the stamp is absent and the managed namespace is empty, SemLink creates
   the metadata bucket and CAS-stamps `beta.160` before provisioning.
2. If the metadata bucket exists without the key and no managed resource
   exists, SemLink performs the same CAS stamp and continues.
3. If the stamp is absent but managed resources exist, startup fails unchanged.
4. If the stamp is not exactly `beta.160`, startup fails unchanged and reports
   the expected and observed values.
5. If the stamp is exact but SemLink-owned stream or metadata configuration
   diverges, startup fails unchanged and reports the rejected field.

The stamp CAS serializes only the single metadata key; it is not a transaction
over all streams and buckets. Readiness remains false until the full beta.160
resource inventory and owned configurations validate.

There is no alpha/beta migration runner. SemLink never transforms, copies,
adopts, upgrades, downgrades, or infers compatibility for stored data. A
beta.141, missing-stamp managed, differently stamped, or configuration-divergent
store must be left untouched or explicitly reset by its operator.

## Managed Namespace Inventory

Schema detection enumerates resource names and fails closed if enumeration is
unavailable. Managed streams are `ENTITY` and `MAVLINK_RAW`. Managed graph KV
buckets include the current beta.160 `graph.KVCatalog()` plus the retired
`COMPONENT_STATUS` name, so unversioned legacy state is never adopted:

```text
ENTITY_STATES
ENTITY_SUFFIX_INDEX
GRAPH_INGEST_APPLIED_SEQ
OUTGOING_INDEX
INCOMING_INDEX
ALIAS_INDEX
PREDICATE_INDEX
NAME_INDEX
SPATIAL_INDEX
TEMPORAL_INDEX
TEMPORAL_INDEX_REVERSE
EMBEDDING_INDEX
EMBEDDING_DEDUP
COMMUNITY_INDEX
COMMUNITY_SUMMARIES
ANOMALY_INDEX
GRAPH_STATUS
STORAGE_REPORT
COMPONENT_STATUS
```

A missing-stamp rejection names each conflict as `stream:<name>` or
`kv:<bucket>`. Unrelated application streams and KV buckets are allowed. An
exact beta.160 restart expects and validates managed resources rather than
rejecting them.

## Deployment State Paths

- Packaged and BlueOS embedded mode uses `StateDir`, exposed as
  `SEMLINK_NATS_STATE_DIR`, with the default `/data/nats-beta160`. A configured
  directory is persistent and `Stop` never deletes it.
- An empty `StateDir` is an explicit test/development posture. SemLink creates
  an owned temporary directory and may remove only that directory on stop.
- Repo Compose mounts the beta.160-specific named volume
  `semlink-nats-beta160-data` at NATS `/data`. It does not use tmpfs.
- `scripts/demo-up.sh` performs a normal idempotent deployment. It does not
  remove or force-recreate `semlink-nats`, and it preserves the named volume.
- External NATS uses the same stamp protocol. Deployment orchestration should
  keep a single active starter for a vehicle, while later sequential starts
  validate and reuse the exact beta.160 database.

Container or process termination is not a reset. Graceful and unexpected
restarts within the exact beta.160 schema preserve the stamp and database.
Repository tests statically validate the Compose lifecycle and use embedded
and external process/server restart integration as persistence evidence; they
do not claim an executed Docker restart lane.

## Clear Or Reset An Incompatible Store

SemLink never performs these operations automatically. First stop every
SemLink process that can access the selected store. Confirm the exact target
before running one of the following operator procedures.

For the packaged or BlueOS default directory, use a host or maintenance shell
where `/data` resolves to the package data mount. Quarantine only the selected
beta.160 directory and create an empty replacement:

```bash
# Run only after the SemLink service or extension is stopped.
mv -- /data/nats-beta160 /data/nats-beta160.rejected-20260816T120000Z
install -d -m 0750 /data/nats-beta160
```

For the default repo Compose project, stop only the SemLink services and remove
only its beta.160 NATS volume. Set `SEMCONNECT_ROOT` to the sibling checkout
first; the SemConnect services and their volumes are not targets:

```bash
docker compose -p semlink-demo \
  -f "$SEMCONNECT_ROOT/conformance/compose.yml" \
  -f docs/semconnect-csapi-port.override.yml \
  -f compose.semlink.yml rm -s -f semlink semlink-nats
docker volume rm semlink-demo_semlink-nats-beta160-data
COMPOSE_PROJECT_NAME=semlink-demo ./scripts/demo-up.sh
```

For external NATS, stop SemLink and provision a new dedicated account,
namespace, server, or empty store directory. Point `NATS_URL` at that empty
target and restart one SemLink instance. Do not copy streams, KV buckets, the
metadata stamp, or entity state from the rejected namespace. If unrelated
applications share the old namespace, leave it intact; do not purge the whole
account.

These are destructive reset procedures for explicitly selected SemLink state,
not normal restart steps. A same-schema code rollback may reuse the existing
database only when the selected code expects and validates exact `beta.160`.

## Readiness And Evidence

After startup, `/api/evidence` reports three fields under `node`:

```json
{
  "semstreams_state_schema_version": "beta.160",
  "semstreams_fresh_state": true,
  "semstreams_reused_state": false
}
```

`semstreams_fresh_state=true` means this startup won the CAS stamp on an empty
namespace. `semstreams_reused_state=true` means the exact beta.160 stamp
predated this startup, including partial-bootstrap recovery. The booleans are
mutually exclusive, and a restart with populated state is never reported as
fresh or empty. `semstreams_state_schema_version` comes from the stamp, not an
environment assertion. Readiness is reported only after the complete schema
inventory validates.

Readiness also requires successful startup contract validation, verified graph
mutation receipts, and no legacy predicate or subject dependencies. The
deterministic companion and mesh evidence lanes must still show:

- semantic vehicle state and rule traces;
- bounded raw telemetry with no raw mesh replication;
- no new hardware-readiness claim; and
- no hardware command-transmit authorization.

## Validation Record

On 2026-08-16, the implementation, design, delta specification, and migration
guidance were reviewed as one beta.160 cutover. OpenSpec tasks 5.1 through 5.5
record the mutation-adapter gates. Task section 6 is retained only as
implementation history. The durable schema and restart contract in task
section 7 supersedes its one-shot/tmpfs behavior. The section-7 backend has Go
reviewer approval; final change verification and technical-writer sign-off
remain task 7.9.

Documentation closure passed:

```text
go test ./internal/semstreams -run TestSemStreamsDependencyPin -count=1
go test ./internal/semstreams \
  -run 'Test(ComposeUses|DemoLauncher|BootstrapStateSchema|StateDir|ExternalRuntime)' -count=1
go test ./internal/blueos -run 'TestBlueOS(Compose|Entrypoint)' -count=1
openspec validate migrate-semstreams-beta-160 --strict
local Markdown link check
legacy graph/ownership API scan
removed state-generation/cutover reference scan
git diff --check
```

The remaining beta.141 and legacy API names are limited to migration history,
the explicit predicate rename map, and completed OpenSpec tasks whose earlier
state contracts are explicitly superseded by section 7.

## Rollback

A code rollback within the exact beta.160 schema may reopen the existing
database when that code validates the same stamp and owned configurations. A
code version expecting any other schema must fail closed. Reset to a new empty
store; do not transform, copy, adopt, upgrade, or downgrade alpha/beta data.
