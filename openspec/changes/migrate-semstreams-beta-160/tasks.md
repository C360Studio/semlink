## 1. Pin Beta.160 And Establish The Red Baseline

- [x] 1.1 Pin `github.com/c360studio/semstreams` to `v1.0.0-beta.160`, verify tag commit
  `8403a2218000e45a31c5132fbfe01af42ed04f14`, update `go.sum`, and record the expected removed-API compile failures.
- [x] 1.2 Add a dependency assertion or release-evidence check that fails when the SemStreams version or expected tag
  commit drifts from the accepted pin.

## 2. Canonical Predicates And Projection Contracts

- [x] 2.1 Write failing table-driven tests for the complete underscore-to-lower-kebab predicate rename map, explicit
  vocabulary registration, and rejection of unregistered or noncanonical predicates.
- [x] 2.2 Rename SemLink predicates and update projector, rule, COP, query, fixture, and evidence consumers to use the
  canonical identifiers; register the vocabulary before any contract validation or graph write.
- [x] 2.3 Write failing contract tests requiring unique contract names, stable nonempty group names,
  `projection.ModeReconcile` for all current complete-state groups, declared predicates, entity patterns, and indexing
  profiles.
- [x] 2.4 Replace `pkg/ownership` and `projection.Derive` usage with beta.160 contracts and
  `projection.ValidateContracts`, and make complete-set validation a fail-fast runtime readiness gate.

## 3. Canonical Mutation And Authority-Read Adapter

- [x] 3.1 Write failing adapter tests for reconcile-first writes, proven not-found handling, zero-triple strict
  creation, immediate complete-group reconcile, and create-conflict convergence through reconcile.
- [x] 3.2 Add contract and group identity to SemLink projection values and replace legacy create/update subjects
  with one `projection.MutationClient` that implements the tested reconcile-create-reconcile sequence.
- [x] 3.3 Add failing mixed-provenance tests proving zero-triple creation uses required creation metadata while
  reconcile preserves each MAVLink/projector triple source and timestamp unchanged, then implement the metadata mapping.
- [x] 3.4 Write failing table-driven tests for invalid, not-found, conflict, revision-conflict, unavailable,
  commit-unknown, and internal outcomes, including bounded retries and no success claim without verified commitment.
- [x] 3.5 Implement typed outcome mapping and resolve commit-unknown through authoritative state comparison before retry
  or logical success; remove the local known-entity cache as an authority signal.
- [x] 3.6 Write failing read tests for validated entity plus same-entry KV revision, then replace raw entity-query
  subject use with `MutationClient.ReadAuthoritative` or a narrow `graph.ExactEntityReader`.

## 4. Runtime Buckets And Fresh-State Cutover

- [x] 4.1 Write failing runtime topology tests proving SemLink does not provision removed `COMPONENT_STATUS` or
  framework-owned `ENTITY_SUFFIX_INDEX`, while beta.160 graph-ingest provisions its own suffix index.
- [x] 4.2 Remove the two consumer bucket declarations, preserve only beta.160 embedding-consumer stores, and expose
  fresh-state posture in startup/readiness evidence.
- [x] 4.3 Add an embedded beta.160 integration test using a fresh state directory that proves envelope-only create,
  immediate reconcile, repeated reconcile, optional-predicate removal, exact revision reads, and restart behavior.
- [x] 4.4 Add upgrade and rollback tests or scripts that reject reuse of beta.141 state, retain any old export for audit
  only, and keep beta.141 and beta.160 state directories isolated.

## 5. Verification, Review, And Documentation

- [x] 5.1 Run focused Go tests for vocabulary, projector, rules, COP, SemStreams adapter, and runtime bootstrap; confirm
  critical mutation and fresh-state paths meet the repository's behavior-focused coverage gate.
- [x] 5.2 Run `gofmt`, `go test ./...`, `go build ./...`, and the deterministic companion/mesh e2e lanes; record that
  raw MAVLink remains local and that no hardware or command-transmit claim is introduced.
- [x] 5.3 Obtain go-reviewer sign-off on contract compliance, context propagation, retry bounds, typed error handling,
  concurrency safety, provenance preservation, and test quality; resolve all blocking findings.
- [x] 5.4 Update migration, evidence, and downstream-consumer documentation with the predicate rename map, beta.160 pin,
  fresh-state/rollback procedure, framework-owned bucket boundary, and revised non-ownership contract semantics.
- [x] 5.5 Obtain technical-writer sign-off, run `openspec validate migrate-semstreams-beta-160 --strict`, and record the
  final implementation/spec/design coherence result before archive or MAVLink expansion resumes.

## 6. Enforced Ephemeral Fresh-State Topology

- [x] 6.1 Write failing external-runtime tests proving a dirty namespace is rejected before `ensureStreams`, mutation,
  or graph-ingest startup; cover `ENTITY`, `MAVLINK_RAW`, every applicable graph KV bucket, enumeration failure, and
  confirmation that preflight never deletes or mutates discovered resources.
- [x] 6.2 Implement the read-only external NATS clean-namespace preflight and add integration coverage proving a clean
  namespace is accepted, required beta.160 resources are provisioned, unrelated NATS resources are allowed, and a
  second start against the same namespace fails closed.
- [x] 6.3 Write failing configuration tests requiring Compose NATS `/data` tmpfs, no named NATS data volume, and no
  `SEMLINK_SEMSTREAMS_STATE_GENERATION`; update the Compose profile to satisfy the enforced ephemeral topology.
- [x] 6.4 Write failing runtime/evidence tests proving no caller-supplied state-generation attestation is accepted,
  then remove `RuntimeOptions.StateGeneration` and its environment/config plumbing and derive `FreshState=true` plus
  the stable `beta.160-fresh` label only from embedded temp-store creation or successful external preflight.
- [x] 6.5 Add a repository reference test for the removed beta.141 preservation/cutover workflow, then delete
  `scripts/check-semstreams-state-cutover.sh` and remove its audit-export, preservation, rollback-volume, and generation
  environment references without deleting any discovered NATS data.
- [x] 6.6 Update migration, handoff, Compose, evidence, and restart documentation to require a newly empty NATS instance
  for every pre-production start or code rollback and to state that dirty external namespaces are rejected, not reset.
- [x] 6.7 Run focused and full Go tests, Compose configuration validation, deterministic companion/mesh e2e gates,
  `go build ./...`, and strict OpenSpec validation; obtain go-reviewer and technical-writer sign-off on the superseding
  clean-state contract before archiving or resuming MAVLink expansion.

## 7. Durable Beta.160 Schema Detection Across Restarts

Section 7 supersedes the completed one-shot startup semantics in section 6. Section 6 remains only as implementation
history and MUST NOT be treated as the current deployment contract.

- [x] 7.1 Write failing table-driven schema-stamp tests for empty first initialization, exact beta.160 restart, a
  missing stamp beside managed resources, every non-beta.160 stamp, metadata bucket without the key, partial
  provisioning, CAS loss, and enumeration failure. Assert that rejection identifies expected/observed state, supplies
  clear/reset guidance, and never deletes, rewrites, transforms, copies, or adopts stored data.
- [x] 7.2 Implement `SEMLINK_RUNTIME_META/state-schema-version=beta.160`: CAS-stamp only an empty namespace before
  managed-resource provisioning, converge CAS losers on the exact stamp, reopen exact beta.160 state, and recover
  partial bootstrap idempotently. Replace the section 6 empty-on-every-start preflight; do not add a data migration
  path.
- [x] 7.3 Write failing persistent embedded-NATS tests covering configured `StateDir`, graceful restart, forced process
  termination, schema-stamp reuse, and survival of exact triples and KV revisions; separately prove that an omitted
  `StateDir` uses an owned temporary directory eligible for cleanup.
- [x] 7.4 Implement persistent embedded `StateDir` without deletion on `Stop`, set packaged and BlueOS defaults under
  `/data/nats-beta160`, and keep empty-StateDir temporary storage limited to explicit test/development posture.
- [x] 7.5 Write failing static Compose and demo-lifecycle tests requiring a beta.160-specific named NATS volume, no
  tmpfs or schema-version environment assertion, and no normal-deploy removal of `semlink-nats`; implement the durable
  Compose and demo-up configuration. Keep graceful and unexpected container restart survival as a deployment
  requirement; use the test-owned embedded and external process/server restart integration in 7.3 and 7.7 as the
  executable persistence evidence rather than claiming an executed Docker restart lane.
- [x] 7.6 Write failing evidence tests for stamp-derived `StateSchemaVersion`, mutually exclusive first-initialized and
  reused posture, and restart reporting that never labels populated state fresh or empty; implement the evidence model,
  including a stable `ReusedState` field and readiness gated on complete schema validation.
- [x] 7.7 Add external-NATS integration tests for graceful and abrupt restart, existing triple/revision survival,
  interrupted bootstrap recovery after each stamp/provisioning phase, different or missing stamp rejection, concurrent
  initializer convergence, and absence of any data transform/copy/upgrade/downgrade path.
- [x] 7.8 Update cutover, restart, BlueOS, handoff, and demo documentation to supersede one-shot/tmpfs guidance: require
  an empty namespace only for first beta.160 initialization, retain exactly stamped beta.160 state on restart, document
  actionable clear/reset steps for every mismatch, and prohibit alpha/beta data migration or in-place reuse.
- [x] 7.9 Run focused and full Go tests, race-sensitive bootstrap tests, Compose/BlueOS configuration validation,
  deterministic companion/mesh e2e gates, `go build ./...`, and strict OpenSpec validation; obtain go-reviewer and
  technical-writer sign-off before archive or MAVLink expansion resumes.
