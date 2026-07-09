## Context

SemLink is pivoting toward a MAVLink companion service that can run beside ArduPilot/ArduRover devices and exchange
selected semantic state with peers. The handoff and runtime specs now make SemOps, semstreams-ui, and SemConnect useful
downstream consumers, but not package startup dependencies or hot-path requirements.

The missing piece is a SemLink-owned proof path. Today a reviewer can read the companion, mesh, safety, and SemOps
contract specs, but there is no single local demo ladder that proves the same story without borrowing SemOps glass,
SemConnect/CS API, Gazebo, Navigator hardware, or a full BlueOS stack. For MVP, the demo artifact should be evidence and
repeatable commands, not another GCS UI.

## Goals / Non-Goals

**Goals:**

- Provide a fast local e2e lane that runs without Docker, hardware, Gazebo, SemOps, SemConnect, or semstreams-ui.
- Provide a single-node demo that proves companion runtime readiness, evidence API posture, native SemOps readback
  adapter semantics, and fail-closed command posture.
- Provide a simple local mesh demo with two or more companion nodes, static peers, bounded watermark/diff catch-up, and
  raw MAVLink exclusion evidence.
- Generate machine-readable demo reports that SemOps, semstreams-ui, SemConnect, or release notes can consume later.
- Keep BlueOS, SITL, and Navigator hardware as higher-fidelity evidence lanes that do not block the default demo.

**Non-Goals:**

- Build or revive a SemLink-owned GCS/dashboard.
- Make CS API/SemConnect part of the companion runtime hot path.
- Require SemOps, semstreams-ui, BlueOS, Navigator hardware, Gazebo, or ArduPilot SITL for the fast e2e lane.
- Add hardware command transmit.
- Mint SemOps trusted-operator headers, own SemOps target asset identifiers, or own born-target proof.

## Decisions

### Decision: Use A Three-Lane Proof Ladder

SemLink will define three runnable proof lanes:

1. Fast in-process e2e tests that start deterministic companion nodes through repo-owned harnesses.
2. A single-node demo script that launches one companion runtime and verifies local evidence.
3. A simple mesh demo script that launches multiple local companion runtimes with static peers.

This keeps the shortest feedback loop cheap while still producing something a human can run during a demo. SITL and
BlueOS stay above this ladder as optional fidelity lanes.

Alternative considered: start directly with SITL or BlueOS. That gives better realism, but it makes every contract
iteration depend on a heavy environment and turns hardware-adjacent setup into the default proof path.

### Decision: Treat Reports As The Demo Surface

Each demo lane will produce a JSON report under a deterministic artifact directory. Reports should include node IDs,
runtime profile, endpoint URLs, peer posture, evidence checks, SemOps contract probe results, command posture, and
assertion status. A small Markdown or HTML summary can be generated later, but the canonical artifact is structured
evidence.

Alternative considered: a local UI. That duplicates SemOps and semstreams-ui, and it shifts the repo back toward owning
glass instead of proving companion behavior.

### Decision: Use A Fake Downstream Receiver For SemOps Compatibility

The demo will prove the native SemOps companion contract by posting to a fake local receiver that validates fields and
records what SemLink attempted. This verifies the adapter boundary without making SemOps a runtime dependency or pulling
CS API into the hot path.

Alternative considered: require a real SemOps stack during demos. That is useful for later integration tests, but it
couples SemLink's basic proof to a sibling product and makes failures harder to attribute.

### Decision: Mesh Demo Uses Static Peers And HTTP-Level Probes First

The simple mesh demo will use local static peer URLs and the existing mesh synchronization surface to prove watermarks,
bounded diffs, TTL/merge posture, and raw MAVLink exclusion. RF behavior, ad hoc discovery, and lossy-link simulation
can be later slices once the local contract is stable.

Alternative considered: emulate radios or build a richer network simulator immediately. That is interesting, but it
would obscure whether the core mesh contract works.

## Risks / Trade-offs

- Fast demos can become too synthetic -> Keep SITL/UDP and BlueOS lanes as separate fidelity evidence and make reports
  clearly state the profile that ran.
- Reports may drift from `/api/evidence` -> Prefer probing real runtime endpoints and reusing evidence types instead of
  inventing a separate report model.
- Static peers do not prove ad hoc networking -> Name the lane "simple mesh" and reserve RF/ad hoc behavior for a later
  spec.
- Fake SemOps can overfit the contract -> Keep fixtures aligned with the SemOps v0 contract and recheck against a pushed
  SemOps baseline before accepting integration as final.

## Migration Plan

1. Add the fast in-process e2e harness and report writer.
2. Add the single-node demo script and probe assertions.
3. Add the simple mesh demo script and peer catch-up assertions.
4. Wire demo docs and evidence report examples into handoff documentation.
5. Keep CI running the fast e2e lane by default; make script-based demos opt-in until they are stable enough for release
   gates.

Rollback is straightforward because the change adds tests, scripts, docs, and optional reports without changing the
required runtime dependency graph.

## Open Questions

- Should the first implementation put fast e2e under `test/e2e`, `internal/e2e`, or a narrow `cmd` probe package?
- Should the generated report be JSON-only in the first slice, or JSON plus Markdown for human demos?
- Should script demos start real OS processes immediately, or should the first slice prove the behavior through
  `httptest`-style in-process servers before adding process orchestration?
