# SemGCS Runtime Migration

Task 2.5 starts the migration away from SemGCS-named runtime/package surfaces
without deleting working companion capability.

## Begun In This Slice

- `docker/blueos-extension/Dockerfile` no longer builds the historical Svelte
  UI or copies `ui/dist` into the BlueOS handoff image.
- `docker/blueos-extension/entrypoint.sh` passes an empty static directory by
  default, so `/api/health`, `/register_service`, and `/api/evidence` prove
  package readiness without serving repo-owned glass.
- Static tests in `internal/blueos/packaging_test.go` guard the BlueOS package
  against accidentally reintroducing the historical UI build or default static
  serving.

The root `Dockerfile`, `compose.semlink.yml`, `compose.sitl.yml`,
`cmd/semgcs-demo`, `internal/gcs`, and `ui/` remain intact for the legacy demo
and developer flows.

## Migration Order

1. Keep BlueOS handoff packaging API/evidence-only.
2. Add `cmd/semlink-companion` as the forward binary name, initially reusing the
   existing runtime wiring.
3. Split `internal/gcs` into companion-oriented packages:
   - local API and evidence contract
   - in-memory store and snapshot/SSE surfaces
   - simulator/demo orchestration
   - command evidence and safety gates
4. Keep compatibility wrappers for `cmd/semgcs-demo` and `internal/gcs` until
   the demo scripts, docs, and tests move to the companion names.
5. Retire the historical Svelte UI/static serving only after equivalent
   readiness remains covered by `/api/health`, `/register_service`, and
   `/api/evidence`.

## Boundaries

- Do not introduce new SemLink-owned GCS glass.
- Do not remove MAVLink ingest, evidence APIs, optional CS API egress, TAK
  bridge behavior, or simulator command evidence while renaming.
- Do not enable hardware MAVLink command transmit as part of this migration.
