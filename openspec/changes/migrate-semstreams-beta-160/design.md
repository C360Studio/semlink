## Context

See `proposal.md` for motivation. SemLink currently pins SemStreams `v1.0.0-beta.141`. Its projection contracts import
the removed `pkg/ownership` package, omit beta.160 group names, and are tested through the removed
`projection.Derive` function. Most SemLink custom predicates also contain underscores, which beta.160 rejects under
the canonical `domain.category.property` lower-kebab grammar.

The live graph writer in `internal/semstreams/client.go` is the SemStreams cutover census's W=1/R=2 holdout. It sends
requests directly to the removed create-with-triples and update-with-triples subjects, caches an unverified local
"known entity" hint, and decodes the raw exact-query response without its same-entry KV revision. The embedded runtime
also consumer-provisions `ENTITY_SUFFIX_INDEX`, which beta.160 graph-ingest owns, and `COMPONENT_STATUS`, which
beta.160 removes.

The first implementation accepted `SEMLINK_SEMSTREAMS_STATE_GENERATION=beta.160-fresh` as an external freshness
assertion. The completed task-section 6 implementation removed that input but also made every startup require an empty
namespace and changed Compose to tmpfs. That treated durable NATS/JetStream database state as disposable process state.
This design explicitly supersedes task section 6: graceful and abrupt process or container restarts must reopen the same
beta.160 database.

## Goals / Non-Goals

**Goals:**

- Pin SemStreams `v1.0.0-beta.160` and its tag commit as one reproducible dependency decision.
- Route every SemLink entity birth and current-state update through the beta.160 contract-validating mutation client.
- Make projection contract name, predicate-group name, write mode, and canonical vocabulary declaration explicit at
  the producer boundary.
- Preserve typed mutation outcomes, authoritative revisions, and commitment evidence through SemLink's graph adapter.
- Model NATS/JetStream resources as a versioned durable database, not an every-start freshness condition.
- Initialize beta.160 only on an empty managed namespace without trusting an operator-supplied value.
- Validate and reopen beta.160 state after graceful or abrupt restarts, including partially initialized state.
- Serialize the schema stamp and concurrent starters through crash-safe compare-and-set.
- Make bootstrap steps idempotent and expose readiness only after the beta.160 schema invariants hold.
- Reject any other schema version with actionable clear/reset guidance.
- Persist embedded, BlueOS, Compose, and external NATS state across normal deployment restarts.
- Derive stable first-initialized and reused-state evidence from the schema-stamp protocol.
- Leave framework-owned bucket provisioning to SemStreams.

**Non-Goals:**

- Add append-mode event writers, revision-fenced deletion, or a general graph administration client in this change.
- Preserve, export, migrate, or reuse beta.141 graph state, or silently alias noncanonical underscore predicates.
- Delete or reset state discovered in an external NATS namespace.
- Accept a caller environment variable or option as schema-version authority.
- Transform, copy, migrate, upgrade, or downgrade stored data between any alpha or beta schema versions.
- Change mesh causality, raw MAVLink retention, command authorization, CS API ownership, or SemOps COP ownership.
- Claim new MAVLink, SITL, or hardware fidelity from a substrate-only cutover.

## Decisions

### 1. Pin the exact beta.160 release

Use `github.com/c360studio/semstreams v1.0.0-beta.160`, whose tag resolves to
`8403a2218000e45a31c5132fbfe01af42ed04f14`. Record both values in cutover evidence and keep `go.mod` and `go.sum`
authoritative.

Alternatives considered:

- **Track a branch or pseudo-version:** rejected because the dependency would not be reproducible across CI and
  deployment builds.
- **Patch beta.141 locally:** rejected because it would preserve removed contracts and increase divergence before the
  MAVLink expansion.

### 2. Register canonical vocabulary before validating contracts

Rename every SemLink-owned custom predicate by replacing underscore-separated property words with lower-kebab words,
for example `robot.identity.system_id` becomes `robot.identity.system-id`. Explicit SemLink vocabulary registration
runs before the complete projector, rule, and COP contract set is validated. Contract construction then uses a stable
single-token group name and `projection.ModeReconcile` or `projection.ModeAppend`.

All current SemLink projections represent either complete current state or a complete newly born evidence entity, so
their mutable groups use `ModeReconcile`. No producer uses `ModeAppend` until it has a genuine set-valued evidence
operation and tests its retry identity. Contract validation is local shape validation only; it is not a registry lease,
write authorization, or cross-process ownership claim.

Vehicle projections intentionally mix per-triple provenance such as MAVLink-origin facts and projector-derived facts.
That provenance remains on each triple; it is not collapsed into one mutation-level source.

Alternatives considered:

- **Register aliases for underscore predicates:** rejected because beta.160 rejects their grammar before alias
  semantics can make them writable.
- **Keep unnamed groups and infer mode from indexing profile:** rejected because profile describes indexing, not write
  intent, and beta.160 requires the group contract explicitly.

### 3. Replace the legacy upsert client with canonical create and reconcile

Construct one concurrency-safe `projection.MutationClient` from the NATS client and the complete validated contract
set. Extend each SemLink projection value with its contract name and reconcile-group name so the adapter does not infer
authority from message type strings or predicate inspection.

For each complete projection:

1. Attempt named-group `Reconcile` first. It performs the authoritative exact read and revision-fenced reconcile.
2. On typed `not-found` with proven `not-committed`, perform strict `Create` with the entity envelope and zero triples.
3. After verified create, immediately `Reconcile` the complete named group. Creation metadata has its own adapter
   source, while reconcile leaves mutation-level source and timestamp unset so each triple retains its original
   provenance.
4. If strict create races and returns typed `conflict`, retry the named-group reconcile with a bounded attempt budget.
5. Remove the local `known` entity cache as an authority signal. A later bounded cache may optimize routing, but it
   cannot replace an exact read or mutation receipt.

This ordering makes steady-state updates one authoritative read plus one reconcile and makes first birth one failed
read plus strict create plus reconcile. It also converges safely if another producer creates the same entity between
operations and avoids beta.160 `Create` metadata conflicts with mixed per-triple sources.

Alternatives considered:

- **Create the envelope and complete triples atomically:** rejected because beta.160 requires one create metadata source
  and rejects nonempty triple sources that differ; SemLink projections intentionally contain mixed provenance.
- **Call strict create on every projection, then reconcile on conflict:** rejected because every steady-state update
  would intentionally generate a conflict.
- **Keep direct graph-ingest subjects behind the current wrapper:** rejected because those write subjects were removed
  and bypass beta.160 contract and outcome handling.
- **Publish the projection as a Graphable event:** rejected for this live write lane because it would change its
  request/reply commitment semantics and would not provide named-group reconciliation.

### 4. Treat mutation results as typed commitment evidence

SemLink retains the `projection.MutationError` operation, kind, and commit state and treats a mutation as successful
only when its receipt reports `CommitVerified`.

| Outcome | Adapter behavior |
| --- | --- |
| `invalid` | Fail without retry; surface a programming or contract error. |
| `not-found` | Create only when the failed reconcile is proven not committed. |
| `conflict` | Reconcile after a strict-create race; otherwise surface the semantic conflict. |
| `revision-conflict` | Re-read and retry reconcile within a bounded attempt budget. |
| `unavailable` | Return degraded/unavailable evidence; retry only under the caller's bounded policy. |
| `commit-unknown` | Do not blind-retry; read authoritative state and resolve the logical operation first. |
| `internal` | Fail and surface the classified error without claiming commitment. |

Request ID, source, and timestamp are generated once for the zero-triple create and reused across its safe retries.
Reconcile carries the logical request/trace identity but leaves mutation-level source and timestamp unset, preserving
the nonempty source and timestamp already carried by every desired triple.

### 5. Use embedded authoritative readers, never raw subjects

Projection reconciliation and readback use `MutationClient.ReadAuthoritative`. Read-only consumers that do not need
the mutation surface receive a narrow `graph.ExactEntityReader`. Both return a validated entity coupled to the
same-entry `KVRevision`; SemLink does not decode `graph.ingest.query.entity` itself and does not substitute
`EntityState.Version` or timestamps for the authoritative local revision.

Alternatives considered:

- **Retain the literal query subject in SemLink:** rejected because it duplicates wire decoding and loses the typed
  error and revision contract.
- **Read ENTITY_STATES KV directly:** rejected because it bypasses the substrate's validation and classification seam.

### 6. Manage durable NATS state as a versioned database schema

SemLink bootstrap stops consumer-provisioning `ENTITY_SUFFIX_INDEX`; beta.160 graph-ingest creates and owns that derived
bucket through its catalog. SemLink also stops provisioning the removed `COMPONENT_STATUS` bucket, which has no
replacement. SemLink continues to provision only stores that beta.160 explicitly assigns to the embedding consumer.

SemLink owns one small JetStream KV bucket, `SEMLINK_RUNTIME_META`, with the key `state-schema-version` and exact value
`beta.160`. This durable schema stamp has no TTL and survives for the life of the NATS database. It records
compatibility with the running code; it does not claim that the database is empty.

After connecting, but before `ensureStreams`, mutation-client use, or graph-ingest initialization, startup inventories
SemLink streams and SemStreams graph KV buckets and resolves the stamp as follows:

1. If the stamp contains `beta.160`, accept the database as the current schema and idempotently ensure and validate all
   beta.160 resources. Existing triples, authoritative revisions, and bounded stream data remain intact.
2. If the stamp is absent and any managed stream or graph KV bucket exists, reject the database as unknown or legacy
   without modifying it. The error identifies the conflicting resources and directs the operator to the documented
   clear/reset procedure.
3. If the stamp is absent and no managed resource exists, create the metadata bucket if necessary and use KV `Create`
   compare-and-set semantics to stamp `beta.160` before idempotently creating and validating managed resources.
4. If the stamp contains any other value, reject startup without overwriting the stamp or managed resources. The error
   reports expected and observed versions and directs the operator to the documented clear/reset procedure.

Across the Sem* alpha/beta ecosystem, schema detection never starts a data migration. No runtime transforms, copies,
upgrades, downgrades, or adopts stored data from another schema version. A future schema version uses a newly empty
store selected by an explicit operator clear/reset procedure and receives its own exact stamp.

JetStream KV gives atomic compare-and-set only for the single stamp key; NATS does not provide one transaction over the
stamp, streams, and graph buckets. The CAS is therefore a serialization and recovery boundary, not proof that
provisioning is complete. Readiness remains false until every idempotent bootstrap step succeeds and the complete
beta.160 resource inventory validates. A crash after metadata-bucket creation but before the stamp can retry CAS only
while the managed namespace remains empty. A crash after the stamp or during provisioning is recoverable: the next
process validates beta.160 and completes only missing work. Managed resources without a stamp are never adopted.

Concurrent first starters may both inspect an empty namespace, but only one KV `Create` wins. A loser rereads the stamp
and continues only if it is exactly `beta.160`; both processes use idempotent ensure operations, and neither rewrites
existing state. This serializes schema stamping without claiming a cross-resource transaction or trusting an environment
assertion. Deployment orchestration still prevents two companion runtimes from remaining active for the same vehicle.

`RuntimeOptions` gains `StateDir` for embedded NATS. A caller-provided directory is persistent and is never removed by
`Stop`; packaged and BlueOS profiles default it to `/data/nats-beta160`. An empty `StateDir` explicitly selects an owned
temporary store for tests or development, and only that owned temporary directory is removed on stop. The same schema
stamp protocol runs after either embedded server connects.

Repo-managed Compose mounts a beta.160-specific named volume at NATS `/data`; it does not use tmpfs. Normal demo deploy
and restart paths preserve the `semlink-nats` container and volume. Destructive reset is an explicit operator action for
initial beta.160 setup or a future schema-version change; it is never a normal restart step.

Remove `RuntimeOptions.StateGeneration` and `SEMLINK_SEMSTREAMS_STATE_GENERATION`. Evidence reads the stamp into
`StateSchemaVersion`. `FreshState=true` only for the startup that wins CAS on an empty namespace; add `ReusedState=true`
when the beta.160 stamp predates the current startup. These flags are mutually exclusive, and a restart with populated
state is never described as fresh or empty. A preexisting beta.160 stamp with partial resources is reported as reused
while startup completes idempotent recovery. Readiness separately proves schema validation.

Alternatives considered:

- **Trust a schema-version environment variable:** rejected because it attests operator intent but does not inspect NATS
  state.
- **Preserve or transform beta.141 state:** rejected because there is no production state and no compatibility promise
  worth the operational ceremony.
- **Delete dirty external resources automatically:** rejected because SemLink must not destroy caller-managed data.
- **Require an empty namespace on every start:** rejected because normal power loss, process restart, and container
  restart must preserve local companion state.
- **Write the schema version after provisioning:** rejected because a crash would leave
  indistinguishable unversioned managed state that safe startup must reject.
- **Treat the stamp CAS as a database transaction:** rejected because the atomic guarantee covers one KV key, not
  creation or mutation of the remaining JetStream resources; idempotent bootstrap and final validation are required.
- **Transform or adopt a differently versioned store:** rejected by the alpha/beta reset-only policy; the operator must
  select an empty store and restart initialization.
- **Have SemLink pre-create every framework bucket:** rejected because bucket ownership and retention are SemStreams
  catalog decisions.

### 7. Gate the substrate cutover before MAVLink expansion

Unit tests validate vocabulary registration, every projection contract, group completeness, mutation outcome mapping,
and authoritative read behavior. An embedded beta.160 integration test exercises first create, repeated reconcile,
conflict recovery, not-found recovery, and commit-unknown resolution against fresh state. Existing deterministic
companion and mesh e2e lanes then prove semantic state, rule traces, evidence APIs, and raw-MAVLink exclusion.

ArduPilot SITL remains a separate MAVLink-native evidence lane and no hardware-readiness or command-transmit claim is
derived from this substrate cutover.

## Risks / Trade-offs

- [Predicate renames break stored queries and downstream fixtures] → Publish an explicit rename map, update all repo
  consumers atomically, and require fresh-state evidence.
- [Reconcile removes a predicate omitted from the desired group] → Treat every reconcile payload as the complete
  desired group and test optional-field removal deliberately.
- [A retry after uncertain delivery duplicates a logical write] → Preserve request metadata and resolve
  `commit-unknown` through authoritative read before retry.
- [Create succeeds but its immediate reconcile fails] → Leave the valid empty envelope in place, surface the failed
  projection, and let the next bounded reconcile complete the declared group without recreating the entity.
- [A crash leaves only the metadata bucket or a subset of managed resources] → Make schema stamping and resource
  ensure idempotent, and test interruption after every bootstrap phase.
- [Concurrent starters race first initialization] → Use KV key creation CAS; losers accept only the exact winning
  schema version and then use idempotent resource ensure operations.
- [A schema stamp is lost while managed resources remain] → Reject as unknown state; never reconstruct or overwrite
  the schema version from resource contents.
- [A caller points persistent embedded mode at the wrong directory] → Reject missing or unsupported versions beside
  managed resources and expose the configured state path in diagnostics without deleting it.
- [A future release opens an older or newer schema] → Reject before provisioning and direct the operator to
  clear/reset an explicitly selected store; never transform or adopt its data.
- [Persistent local state grows on a mobile companion] → Retain the existing bounded stream policies and add storage
  health evidence; persistence does not remove retention limits.
- [A pinned beta remains pre-release software] → Pin the tag commit, keep integration tests at the adapter boundary,
  and require a newly empty, version-specific store when a future SemStreams beta changes the schema.

## Rollout Plan

1. Replace the one-shot empty-namespace preflight with the
   `SEMLINK_RUNTIME_META/state-schema-version` detection-and-stamp protocol while retaining the
   no-environment-attestation rule.
2. Add persistent embedded `StateDir` handling and set packaged/BlueOS deployment defaults to `/data/nats-beta160`.
3. Restore the beta.160-specific Compose NATS named volume and keep NATS running across normal demo deployments.
4. Derive schema version and first-initialized versus reused-state evidence from the stamp outcome.
5. Prove graceful restart, forced process termination, abrupt container restart, partial initialization, and concurrent
   initialization without losing existing triples or authoritative revisions.
6. Document that first beta.160 initialization uses an empty namespace, while every later beta.160 start validates and
   reopens that database. Beta.141 and any other schema version require an operator-selected empty store; no data is
   transformed, copied, migrated, upgraded, or downgraded.
7. Run unit, integration, deterministic companion, and mesh evidence gates; run the separate ArduPilot SITL lane before
   making later MAVLink compatibility claims.

Rollback within beta.160 reuses the database when the selected code honors the same schema contract. Code whose expected
schema version differs refuses startup and directs the operator to clear/reset an explicitly selected store. No
beta.141/beta.160 compatibility is promised, no stored data is transformed or copied, and SemLink never deletes an
incompatible namespace automatically.
