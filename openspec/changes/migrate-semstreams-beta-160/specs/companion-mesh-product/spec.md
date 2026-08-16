## MODIFIED Requirements

### Requirement: SemStreams Remains The Substrate

SemLink MUST consume SemStreams substrate contracts and canonical graph mutation and authority-read behavior rather
than reimplementing framework-owned graph behavior locally.

#### Scenario: Companion state enters the graph

- **WHEN** SemLink writes vehicle state, alerts, rule traces, or command intent
- **THEN** it first reconciles the complete declared named predicate group
- **AND** if the entity is provably absent, strict creation establishes only the entity envelope with zero triples and
  is followed immediately by reconciliation of the complete named group
- **AND** every group declares whether its operation is reconcile or append
- **AND** the write uses a locally validated projection contract and indexing profile
- **AND** all emitted custom predicates are registered canonical three-segment lower-kebab identifiers
- **AND** reconciliation preserves each triple's source and timestamp provenance
- **AND** projection contracts validate producer intent and graph shape without reserving predicates, authorizing
  writes, or establishing exclusive runtime ownership
- **AND** raw high-rate telemetry remains on bounded lanes rather than becoming one graph entity per frame

#### Scenario: Projection contracts are invalid at startup

- **GIVEN** a SemLink contract has an unnamed group, invalid write mode, undeclared predicate, noncanonical predicate,
  duplicate declaration, or invalid entity pattern
- **WHEN** the companion runtime initializes its graph writer
- **THEN** initialization fails before any graph mutation is attempted
- **AND** readiness evidence identifies the contract validation failure

#### Scenario: Current state already exists

- **GIVEN** a prior write established the target entity
- **WHEN** SemLink publishes the next complete current-state projection
- **THEN** SemLink reconciles only the projection's declared named group against an authoritative entity revision
- **AND** predicates outside that group remain unchanged
- **AND** SemLink reports write success only from a verified mutation receipt

#### Scenario: Current-state entity does not exist

- **GIVEN** named-group reconciliation reports not-found with proven not-committed status
- **WHEN** SemLink establishes the entity
- **THEN** strict creation writes the entity envelope with zero triples
- **AND** verified creation is followed immediately by reconciliation of the complete named group
- **AND** mixed per-triple source and timestamp provenance remains unchanged by creation metadata
- **AND** a concurrent strict-create conflict converges through named-group reconciliation

#### Scenario: Mutation commitment is uncertain

- **WHEN** a graph mutation returns a typed invalid, not-found, conflict, revision-conflict, unavailable,
  commit-unknown, or internal outcome
- **THEN** SemLink preserves the outcome and commitment classification in local error or readiness evidence
- **AND** it does not report the projection as committed unless commitment is verified
- **AND** a commit-unknown result is resolved through an authoritative exact-entity read before the logical mutation is
  retried or reported as successful

#### Scenario: Entity state is read for projection or evidence

- **WHEN** SemLink requires authoritative entity state or a revision for a graph operation
- **THEN** it uses the substrate's exact-entity read contract
- **AND** the result couples a validated entity with the revision of the same authoritative entry
- **AND** entity logical version or timestamps are not substituted for that local authoritative revision

#### Scenario: Deployment initializes beta.160

- **WHEN** a SemLink deployment performs its first beta.160 initialization
- **THEN** it requires an empty managed namespace rather than reusing beta.141 graph or index buckets
- **AND** it atomically stamps `SEMLINK_RUNTIME_META/state-schema-version=beta.160` before provisioning managed
  resources
- **AND** SemLink does not provision the removed component-status bucket
- **AND** SemLink leaves the framework-owned suffix-index bucket to the beta.160 graph component
- **AND** no operator-supplied value is accepted as proof of schema compatibility or freshness

#### Scenario: Durable schema contract supersedes one-shot startup

- **WHEN** the beta.160 deployment contract is evaluated
- **THEN** the schema-stamp and persistent-restart scenarios supersede the completed task-section 6 ephemeral semantics
- **AND** task section 6 is retained only as implementation history

#### Scenario: Persistent embedded runtime restarts

- **GIVEN** a packaged or BlueOS companion uses a configured persistent embedded NATS state directory
- **WHEN** SemLink stops gracefully or restarts after abrupt termination
- **THEN** it reopens the same beta.160 state directory without deleting it
- **AND** the same schema stamp, existing triples, and authoritative revisions remain available
- **AND** deployment profiles default the persistent directory under `/data`

#### Scenario: Test or development embedded runtime starts without a state directory

- **WHEN** an explicit test or development runtime omits the embedded NATS state directory
- **THEN** SemLink creates an owned temporary store
- **AND** only that owned temporary store is eligible for cleanup on stop

#### Scenario: Repo-managed Compose runtime starts

- **WHEN** the repository Compose profile starts NATS
- **THEN** NATS `/data` is backed by a beta.160-specific persistent named volume
- **AND** the SemLink service supplies no schema-version environment assertion
- **AND** normal demo deployment does not remove the NATS container or its volume
- **AND** a graceful or unexpected container restart validates and reopens the beta.160 database

#### Scenario: Managed resources exist without a schema-version stamp

- **GIVEN** an external NATS namespace already contains a SemLink stream or SemStreams graph KV bucket used by the
  runtime but has no `state-schema-version` entry
- **WHEN** SemLink starts
- **THEN** startup fails before stream creation, graph mutation, or graph-ingest initialization
- **AND** the failure identifies the conflicting resources
- **AND** the failure directs the operator to the documented clear/reset procedure
- **AND** SemLink does not delete, purge, or modify the discovered state

#### Scenario: Schema stamp has a different value

- **GIVEN** `state-schema-version` does not exactly equal beta.160
- **WHEN** SemLink starts
- **THEN** startup fails before managed-resource provisioning or graph mutation
- **AND** the failure reports expected and observed values and directs the operator to the clear/reset procedure
- **AND** SemLink does not transform, copy, migrate, upgrade, downgrade, overwrite, or delete existing state

#### Scenario: Clean namespace initializes beta.160

- **GIVEN** an external NATS namespace contains no SemLink stream or SemStreams graph KV bucket used by the runtime
- **AND** no `state-schema-version` entry exists
- **WHEN** SemLink performs first initialization
- **THEN** it stamps beta.160 through compare-and-set before provisioning managed resources
- **AND** it provisions the required resources and starts graph-ingest
- **AND** evidence reports first initialization without relying on configuration input

#### Scenario: Current beta.160 database restarts

- **GIVEN** `state-schema-version` equals beta.160 and managed resources already exist
- **WHEN** SemLink restarts after graceful shutdown or unexpected termination
- **THEN** it validates the beta.160 schema invariants and reuses the existing namespace
- **AND** existing triples, authoritative revisions, and bounded stream contents survive
- **AND** evidence reports reused state rather than fresh or empty state

#### Scenario: Initialization was interrupted

- **GIVEN** a prior process claimed the beta.160 schema but stopped before all resources were provisioned
- **WHEN** SemLink restarts
- **THEN** it validates the stamp and idempotently provisions only missing resources
- **AND** it does not reset existing beta.160 state
- **AND** readiness remains false until all beta.160 schema invariants validate

#### Scenario: Metadata bucket exists without the schema-version key

- **GIVEN** the metadata bucket exists, `state-schema-version` is absent, and no managed resource exists
- **WHEN** SemLink resumes first initialization
- **THEN** it stamps beta.160 through compare-and-set and continues idempotent provisioning

#### Scenario: Concurrent first initializers race

- **GIVEN** two SemLink processes concurrently observe an empty namespace without a schema stamp
- **WHEN** both attempt first initialization
- **THEN** exactly one compare-and-set schema stamp succeeds
- **AND** the other process reads and accepts only the exact beta.160 stamp
- **AND** both resource initialization paths converge without overwriting state

#### Scenario: Schema-stamp atomicity is bounded

- **WHEN** SemLink stamps or validates `state-schema-version`
- **THEN** compare-and-set atomicity applies only to that single KV key
- **AND** stream and graph-bucket provisioning remains a separately validated sequence of idempotent operations
- **AND** a stamp alone never causes readiness to report a complete schema

#### Scenario: A future schema version is introduced

- **WHEN** SemLink changes the current schema version after beta.160
- **THEN** the runtime refuses every existing namespace whose stamp differs from the new exact version
- **AND** the operator initializes a newly empty store through the documented clear/reset procedure
- **AND** no alpha or beta runtime transforms, copies, migrates, upgrades, or downgrades stored data

#### Scenario: Pre-production deployment restarts or rolls back code

- **WHEN** an embedded, Compose, BlueOS, or external deployment restarts within the beta.160 schema version
- **THEN** it reopens the versioned database rather than requiring a clean namespace
- **AND** a code rollback may reuse state only when it honors the same beta.160 schema contract
- **AND** beta.141 or differently stamped state is never reused, transformed, or upgraded in place
