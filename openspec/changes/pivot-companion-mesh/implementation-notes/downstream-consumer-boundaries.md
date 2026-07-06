# Downstream Consumer Boundaries

This note closes task 5.3 for `pivot-companion-mesh`.

SemLink keeps downstream consumers optional and outside the product boundary by
advertising them in the local evidence bundle instead of wiring them as required
runtime dependencies.

`GET /api/evidence` now includes a `downstream` section:

- `semops`: optional `pull-local-api` consumer for GCS/COP glass and fusion
- `semstreams-ui`: optional `pull-local-api` consumer for generic ops/debug
  views
- `semconnect-csapi`: optional `egress-http` standards projection that is
  `disabled` until `CS_API_URL` is configured

The metadata records ownership, role, direction, enabled/status, relevant API
paths, and boundary claims. SemOps and semstreams-ui are available consumers,
but SemLink does not depend on them and does not own their glass. SemConnect
remains curated low-rate standards egress, not raw MAVLink transport and not
mesh synchronization.

Tests added to `internal/gcs/evidence_test.go` assert both the default disabled
SemConnect state and configured `CS_API_URL` state.
