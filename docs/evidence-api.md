# SemLink Evidence API

SemLink exposes a local evidence bundle for external operator surfaces. This is
the preferred UI integration point for SemOps, semstreams-ui, and CLI/debug
automation.

The API is read-only:

```bash
curl -s http://127.0.0.1:8080/api/evidence
```

## Contract

The response includes a versioned contract header:

```json
{
  "contract": {
    "name": "c360.semlink.companion.evidence",
    "version": "v1"
  }
}
```

`v1` is intentionally compact. It is meant for scanning companion-node state,
not replacing SemStreams graph queries, SemConnect standards egress, or SemOps
COP fusion.

## Sections

- `node`: node ID, runtime health, SemStreams mode, NATS URL, uptime, frame
  counters, graph writes, graph errors, and buffer drops.
- `profile`: handoff profile metadata for identity, HTTP listen address,
  BlueOS host port, SemStreams mode, MAVLink UDP input, simulator fallback,
  static mesh peers, optional CS API egress, command posture, and TAK bridge
  posture. When external MAVLink UDP is configured, `profile.simulator.enabled`
  is `false` and `profile.simulator.source` is `external-mavlink-udp`.
- `downstream`: optional downstream consumers and their boundary metadata.
  SemOps and semstreams-ui pull local API evidence. SemConnect is disabled
  until `CS_API_URL` is configured, then acts as curated standards egress.
  Each entry declares `dependency_mode=optional-downstream`,
  `runtime_dependency=false`, and `required_for_readiness=false`.
- `vehicles`: current MAVLink-derived vehicle summaries with MAVLink vehicle
  type, graph revision, indexing profile, link status, battery, position, and
  evidence class.
- `mesh`: mesh deployment posture (`single-node` or `static-peers`),
  configured peer count/URLs, configured/unconfigured summary-index status,
  summary count, watermark count, current watermarks, and the explicit
  `raw_mavlink_replicates_by_default=false` and
  `raw_mavlink_replication_policy=excluded-by-default` claims.
- `rule_traces`: append-limited rule trace summaries with rule ID/version,
  node, vehicle, decision, suggested action, posture, input hash, and input
  count.
- `commands`: command intents and command-gate evidence, including simulator
  gate status or hardware block evidence. Simulator gates preserve compact
  `preflight_accepted`, `ack_accepted`, and `post_state_observed` proof while
  keeping `hardware_transmit_authorized=false`. In the MVP handoff,
  hardware-readonly command attempts are rejected before command-intent writes
  and recorded with `hardware_block.scope=companion-deployment-handoff`.

## Boundary

This API keeps SemLink useful to UI products without making SemLink own the UI.
SemOps owns GCS/COP glass. semstreams-ui can use the same response for
ops/debug views. SemConnect remains the optional standards-facing egress path.

The `downstream` section is deliberately declarative. It lets external tools
discover whether a consumer path is available without making that consumer a
required runtime dependency for SemLink. `enabled=true` means the local path is
available or configured; it does not mean the consumer is required for package
readiness.

## Demo Reports

SemLink-owned e2e/demo commands generate structured JSON reports that wrap this
evidence contract with probe assertions. They are consumer-oriented evidence,
not a repo-owned GCS UI.

```bash
./scripts/demo-single-companion.sh
./scripts/demo-mesh-companions.sh
```

The single-node report records `/api/health`, `/register_service`,
`/api/evidence`, command-safety posture, simulator-only command evidence, and
fake SemOps native readback results. The simple mesh report defaults to three
companion nodes with one simulated vehicle per node and records static peer URLs,
watermark/diff catch-up, TTL/merge posture, selected summary counts, and raw
MAVLink exclusion. These demos do not require SemOps,
semstreams-ui, SemConnect/CS API, BlueOS, Navigator hardware, Gazebo, SITL, or
physical MAVLink devices. They do use the SemLink Go module and the pinned
SemStreams module version in `go.mod`.

The demo commands can also emit a `semlink-companion-demo-artifact-v0` envelope
for downstream SemOps ingestion by setting `SEMLINK_DEMO_ARTIFACT` in the
wrapper scripts or `-artifact-output` on `cmd/semlink-demo`. The envelope wraps
the raw report with SemLink producer provenance, source fidelity, generator
metadata, timestamp coherence, a real `semlink_commit` or `semlink_version`,
and no-transmit posture. Artifact output fails if the CLI cannot resolve a real
source reference from explicit flags, Go VCS build metadata, or checkout
`HEAD`. Raw reports remain the repo-owned local proof; artifact envelopes are
optional downstream handoff files and do not make SemOps a runtime dependency.
