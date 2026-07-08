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
- `vehicles`: current MAVLink-derived vehicle summaries with MAVLink vehicle
  type, graph revision, indexing profile, link status, battery, position, and
  evidence class.
- `mesh`: configured/unconfigured mesh status, summary count, watermark count,
  current watermarks, and the explicit `raw_mavlink_replicates_by_default`
  false claim.
- `rule_traces`: append-limited rule trace summaries with rule ID/version,
  node, vehicle, decision, suggested action, posture, input hash, and input
  count.
- `commands`: command intents and command-gate evidence, including simulator
  gate status or hardware block evidence.

## Boundary

This API keeps SemLink useful to UI products without making SemLink own the UI.
SemOps owns GCS/COP glass. semstreams-ui can use the same response for
ops/debug views. SemConnect remains the optional standards-facing egress path.

The `downstream` section is deliberately declarative. It lets external tools
discover whether a consumer path is available without making that consumer a
required runtime dependency for SemLink.
