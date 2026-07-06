# Evidence API Contract

This note closes task 5.2 for `pivot-companion-mesh`.

SemLink now exposes a versioned local evidence bundle:

```text
GET /api/evidence
```

The bundle is intended for SemOps, semstreams-ui, and CLI/debug automation. It
keeps SemLink useful to UI products without making SemLink own GCS glass.

The `v1` bundle includes:

- `contract`: name, version, intended consumers, and related API paths
- `node`: node ID, health status, runtime, SemStreams mode, NATS URL, uptime,
  frame counters, graph writes, graph errors, and buffer drops
- `vehicles`: MAVLink current-state summaries with graph revision and indexing
  profile
- `mesh`: configured/unconfigured state, summary count, watermark count,
  current watermarks, and explicit raw-MAVLink non-replication evidence
- `rule_traces`: compact summaries of rule trace entities
- `commands`: command intents and command-gate evidence, including hardware
  block evidence

Implementation files:

- `internal/gcs/evidence.go`
- `internal/gcs/evidence_test.go`
- `internal/gcs/server.go`
- `internal/gcs/store.go`
- `docs/evidence-api.md`

The endpoint is read-only. It does not add a SemLink-owned dashboard and does
not create a hardware transmit path.
