# SemOps MAVLink Prior-Art Inventory

Date: 2026-07-04

## Purpose

This note closes task 2.1 for `pivot-companion-mesh`: inventory SemOps MAVLink
parser, UDP listener, replay, and command-helper code before SemLink implements
the companion runtime slice.

The goal is not to make SemLink depend on SemOps. SemOps is the broad COP
product. SemLink should reuse proven ideas and, where useful, port code into a
SemLink-local package or a future shared module.

## Sources Reviewed

- `../semops/pkg/adapters/mavlink/doc.go`
- `../semops/pkg/adapters/mavlink/parser.go`
- `../semops/pkg/adapters/mavlink/generator.go`
- `../semops/pkg/adapters/mavlink/raw_lane.go`
- `../semops/pkg/adapters/mavlink/replay.go`
- `../semops/pkg/adapters/mavlink/commands.go`
- `../semops/internal/adapters/mavlink/udp_listener.go`
- `../semops/internal/adapters/mavlink/adapter.go`
- `../semops/internal/components/mavlink/components.go`
- `../semops/internal/projectors/mavlink/projector.go`
- `../semops/internal/projectors/mavlink/writer.go`
- `../semops/cmd/semops-mavlink-command/main.go`
- `../semops/docs/mavlink-sitl-smoke.md`
- `../semops/docker/ardupilot-sitl/`

## Current SemLink Baseline

SemLink currently has a narrow demo-oriented MAVLink surface:

- `internal/mavlink/frame.go`: validates and emits unsigned MAVLink v2 frames
  only. It rejects signed frames and does not support MAVLink v1.
- `internal/mavlink/messages.go`: decodes `HEARTBEAT`, `SYS_STATUS`, and
  `GLOBAL_POSITION_INT` into strongly typed demo messages.
- `internal/mavlink/sim.go`: deterministic in-process frame generator for the
  current SemGCS demo.
- `internal/projector`: collapses the supported messages into SemLink
  `robotics.fleet` vehicle state, alerts, and command-intent entities.

That baseline is intentionally small and has been good for the first demo. It
is too narrow for the companion-mesh pivot because boat/ArduRover work needs
stream parsing, UDP ingest, more message coverage, replayable raw evidence, and
command ACK/readback discipline.

## Reuse Decisions

### Port First: Codec, Generator, Raw Lane, Replay

SemOps `pkg/adapters/mavlink` is the best first source to port into SemLink.
It is already library-shaped and mostly product-neutral.

What it gives us:

- MAVLink v1 and v2 stream parser.
- Incremental buffer handling for split frames.
- Resync across noise.
- Parser stats for health and debugging.
- MAVLink v2 generator for deterministic tests.
- Message coverage for heartbeat, global position, attitude, battery status,
  `COMMAND_LONG`, and `COMMAND_ACK`.
- Bounded in-memory raw-frame lane with source refs.
- JSONL replay store for captured raw frames.
- Real-frame tests for parser/generator/concurrency behavior.

SemLink decision:

- Port these into `internal/mavlink` or a new `internal/mavlinkwire` package in
  the next implementation slice.
- Keep SemLink's current typed message/projector API temporarily, then adapt it
  to consume SemOps-style `Packet` values.
- Preserve SemLink tests for the existing three-message demo while adding
  SemOps-derived tests for attitude, battery, command-long, command-ack, split
  buffers, noise resync, checksum errors, and concurrent generator/parser use.

Watch-outs:

- SemOps parser accepts MAVLink v2 signatures as opaque bytes but does not
  provide signing/authentication validation. SemLink should keep signed-link
  support out of hardware-readiness claims until a separate spec adds it.
- SemOps `commands.go` currently includes ArduCopter mode helpers. The boat
  lane needs ArduRover / boat-appropriate mode helpers instead of copying the
  ArduCopter map as-is.

### Port Soon: UDP Listener Shape

SemOps has two UDP ingest shapes:

- `internal/adapters/mavlink/udp_listener.go`: small context-aware UDP listener
  that feeds an adapter.
- `internal/components/mavlink/components.go`: larger SemStreams component /
  flowgraph input that publishes raw BaseMessages onto NATS.

SemLink decision:

- Port the small `UDPListener` shape first for the companion node.
- Keep the listener independent of SemOps COP types.
- Add an interface that accepts raw datagrams so tests can drive it without
  binding real UDP ports.
- Defer SemStreams component/flowgraph adoption until SemLink actually needs a
  hosted component topology.

### Adapt, Do Not Copy: Projector And Graph Writer

SemOps `internal/projectors/mavlink` is useful but COP-shaped.

What it gives us:

- Born-first source asset and track entities.
- Restart reconciliation for create conflicts.
- `COMMAND_ACK` projection into control-task evidence.
- Owner-token-aware mutation plans.
- Source references from raw frames.
- Tests for strict source-track edges, no rebirth, unsupported message ignores,
  attitude/battery mapping, raw source references, and command ACK tasks.

SemLink decision:

- Do not copy the COP projector wholesale.
- Reuse the behavioral tests and patterns while mapping into SemLink companion
  vocabulary and ADR 003 boundaries.
- Consider a small SemLink-local mutation-plan writer if the current
  `GraphClient.UpsertProjection` path becomes too narrow for born-first
  command-ACK evidence.

### Adapt Later: Command Helper And Safety Gate

SemOps `cmd/semops-mavlink-command` is valuable as a simulator command-session
pattern, not as a SemLink CLI to copy immediately.

What it gives us:

- Simulator-only confirmation.
- Stable sender identity.
- Optional heartbeat before command.
- Bounded attempts and retry interval.
- `COMMAND_LONG` confirmation increments.
- Route learning from raw telemetry.
- Raw-lane observation of command ACKs.
- Dry-run metadata.

SemLink decision:

- Do not add native command transmit in the companion runtime slice.
- Reuse this discipline when task 4.3/4.4 starts.
- Keep the first command allowlist read-side and simulator-only, probably
  `MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION`, before any vehicle-control
  action.

### Reuse As Evidence Pattern: SITL Gates

SemOps `docs/mavlink-sitl-smoke.md`, `scripts/mavlink-sitl-gate.sh`, and
`docker/ardupilot-sitl/` are the strongest simulator-fidelity prior art.

What it gives us:

- Simulator-family evidence stamping.
- Separate PX4, ArduPilot, Gazebo, MAVSDK/offboard, hardware, and command-control
  claims.
- ArduPilot SITL-only Docker lane without Gazebo.
- Fail-closed command-control preflight.
- Simulator command-live gate with ACK and post-state polling.

SemLink decision:

- Reuse the ArduPilot SITL-only lane pattern for task 2.3.
- Keep Gazebo out of the first SemLink lane.
- Stamp evidence so ArduPilot passes cannot satisfy PX4, MAVSDK/offboard,
  hardware, or command-control claims.
- Keep hardware transmit blocked until a separate OpenSpec change defines it.

## Do Not Port Now

- SemOps `pkg/cop` vocabulary and COP entity identity. SemLink has its own
  companion/product vocabulary and SemOps owns the broad COP.
- SemOps component flowgraph wiring. It is a useful reference, but it would add
  product-topology weight before SemLink proves the companion node harness.
- SemOps CS API egress behavior. SemConnect remains the standards anchor and
  SemLink egress should stay optional and curated.
- Gazebo-heavy ArduPilot lane. Keep it as later evidence only.
- MAVSDK/offboard lane. It does not add a third telemetry protocol and belongs
  in a future command-control proof.

## Recommended Next Implementation Slice

Start with a SemLink-local wire upgrade, before mesh or BlueOS packaging:

1. Add SemOps-derived parser/generator/raw-lane tests under `internal/mavlink`.
2. Port the SemOps stream parser, generator, raw lane, and replay store into
   SemLink.
3. Keep the existing SemLink typed decode/projector path passing.
4. Add battery and attitude support as typed messages only after the stream
   parser lands.
5. Add UDP listener tests and port the small listener shape.
6. Leave command transmit out until the rules/command-safety slice.

This gives the companion pivot a stronger MAVLink spine without dragging SemOps
COP ownership into SemLink.
