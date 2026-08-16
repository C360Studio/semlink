## Why

SemLink's former SemStreams `v1.0.0-beta.141` projection APIs, predicate rules, and durable NATS/JetStream schema are
incompatible with `v1.0.0-beta.160`. Alpha and beta releases never transform, copy, or upgrade stored data. Because
there is no production beta.141 state, beta.160 is initialized only on an empty namespace; the resulting database must
then survive planned and unexpected companion-computer restarts.

## What Changes

This amendment explicitly supersedes the completed task-section 6 one-shot/ephemeral startup contract. That section is
retained only as implementation history; the schema-stamp and persistent-restart behavior below is authoritative.

- Pin SemStreams `v1.0.0-beta.160` at tag commit `8403a2218000e45a31c5132fbfe01af42ed04f14` and update SemLink's
  integration to the beta.160 graph, projection, vocabulary, and generated dependency surface.
- Replace the removed `pkg/ownership` and `projection.Derive` usage with beta.160 projection contracts that give
  every predicate group a stable name and an explicit `reconcile` or `append` write mode, then validate the complete
  contract set locally.
- Correct SemLink's contract semantics: projection declarations validate producer intent and graph shape; they do
  not reserve predicates, authorize writes, or establish runtime ownership against other producers.
- **BREAKING**: Rename SemLink custom predicates from underscore-bearing identifiers to canonical three-segment,
  lower-kebab identifiers and register them with the SemStreams vocabulary before contract validation or graph
  publication. Update all SemLink writers, readers, rules, fixtures, and downstream evidence that names those
  predicates; persisted beta.141 graph state is not compatible with the beta.160 runtime and is not reused.
- Replace SemLink's direct calls to the removed graph-ingest create/update subjects with
  `projection.MutationClient`: reconcile the complete named group first; on a proven not-found result, use strict create
  with zero triples to establish only the entity envelope, then immediately reconcile the complete group. This preserves
  each triple's source and timestamp provenance, and a concurrent create conflict converges through reconcile. Replace
  raw entity-query requests with `MutationClient.ReadAuthoritative` or `graph.ExactEntityReader`, and handle typed
  mutation outcome and commit states without treating delivery as success.
- **BREAKING**: Treat the NATS/JetStream namespace as durable database state and initialize the beta.160 schema only on
  an empty namespace rather than reusing beta.141 graph or index buckets. SemLink stops
  consumer-provisioning the framework-owned `ENTITY_SUFFIX_INDEX` bucket and stops provisioning the removed
  `COMPONENT_STATUS` bucket.
- Add a SemLink-owned schema-version stamp at `SEMLINK_RUNTIME_META/state-schema-version` with value `beta.160`.
  First initialization stamps an empty namespace through compare-and-set before idempotent resource provisioning;
  later starts validate that exact value and reopen the database. Missing-version managed resources or any different
  version fail closed without deletion and direct the operator to an explicit clear/reset procedure. Partial bootstrap
  and concurrent starters recover through the same stamp protocol.
- Add configurable persistent embedded NATS storage. Deployment and BlueOS profiles use a beta.160-specific directory
  under `/data`, such as `/data/nats-beta160`, and `Stop` never deletes caller-provided state. Empty state-directory
  input remains an owned temporary-store option only for tests and explicit development runs.
- Restore a beta.160-specific persistent Compose NATS volume and keep the NATS service and volume across normal demo
  deployments so both graceful and abrupt container restarts reopen the marked state.
- Remove caller schema attestation such as `SEMLINK_SEMSTREAMS_STATE_GENERATION`. Schema version, first-initialized
  posture, and reused-state posture are derived from the durable stamp result rather than configuration input.
- Re-prove startup contract validation, semantic graph projection, rule traces, local evidence, and mesh-selected
  summaries with deterministic simulator coverage. MAVLink/SITL compatibility evidence remains distinct, and this
  substrate cutover makes no hardware-readiness or command-transmit claim.
- Preserve product boundaries: SemStreams continues to own the graph substrate and validation rules, SemLink owns
  companion-domain producers and predicates, SemOps owns broad COP/fusion, and SemConnect owns CS API conformance.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `companion-mesh-product`: Update the SemStreams substrate requirement from removed projection-ownership semantics
  to beta.160 local projection-contract validation, canonical mutation and authoritative-read adapters, explicit group
  write modes, declared canonical predicates, typed outcomes, and fail-fast handling of invalid producer contracts.

## Impact

- Dependency: `github.com/c360studio/semstreams` advances from `v1.0.0-beta.141` to the pinned beta.160 tag and
  corresponding transitive module set.
- Code: projection contracts and tests in `internal/projector`, `internal/rules`, and `internal/cop`; predicate
  declarations and consumers across those packages; SemStreams runtime registration and graph integration; fixtures,
  e2e assertions, and evidence/readback tests.
- Runtime storage: packaged embedded/BlueOS mode uses a persistent beta.160 state directory, repo Compose uses a
  beta.160-specific named NATS volume, and external NATS uses the same durable schema-stamp protocol.
- Data/API compatibility: canonical predicate renames affect graph queries, rule inputs, serialized triples, and any
  downstream consumer keyed to the former underscore identifiers. Beta.141 state is never reused in place, while marked
  beta.160 state is the authoritative restart source.
- Operations: first initialization rejects unversioned or differently versioned managed state without deleting it and
  supplies an actionable clear/reset instruction. Exact beta.160 starts reopen the same schema and preserve triples,
  revisions, and bounded streams. Alpha/beta releases never transform, copy, migrate, upgrade, or downgrade stored data.
- Delivery sequencing: this substrate cutover is a prerequisite for dialect-capable MAVLink work so new message
  adapters do not expand an obsolete or invalid projection surface.
