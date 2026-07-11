# Companion Handoff Profile

The companion handoff profile is the operator-facing configuration surface for
the deployable SemLink companion package. It documents the values needed to run
one vehicle-local node with local evidence APIs, optional UDP MAVLink input,
optional mesh peers, and optional downstream consumers.

Use the copyable profile at:

```bash
configs/handoff/companion.env.example
```

Task 1.2 defined the profile contract. Task 1.3 adds parser validation in
`internal/handoff`, task 1.4 exposes active profile metadata through
`/api/evidence`, and task 2.1 wires the BlueOS-style entrypoint plus local
Compose smoke to load this profile.

## Profile Fields

### Identity

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_NODE_ID` | wired through handoff profile | Stable companion node ID, for example `vehicle-alpha`. |
| `SEMLINK_VEHICLE_ID` | wired through handoff profile | Single vehicle/profile ID. |
| `SEMLINK_CALLSIGN` | wired through handoff profile | Human-readable callsign for evidence and demos. |

The current evidence API carries identity from the loaded handoff profile
through `gcs.ServerOptions`. The handoff validator rejects blank or
whitespace-separated identity tokens, and `/api/evidence` exposes the accepted
identity under `profile`.

### Local API

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_HTTP_LISTEN` | wired in BlueOS entrypoint | HTTP listen address inside the package. |
| `SEMLINK_BLUEOS_HOST_PORT` | wired in smoke script | Local host port for package smoke runs. |

The readiness endpoints for the handoff are:

- `/api/health`
- `/register_service`
- `/api/evidence`

### SemStreams Runtime

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_EMBEDDED_NATS` | wired in BlueOS entrypoint | Runs embedded local SemStreams runtime when true. |
| `NATS_URL` | wired in binary and entrypoint | External NATS URL when embedded runtime is false. |

The first handoff profile defaults to local embedded runtime so a single node
can be smoked without external infrastructure.

### MAVLink Input

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_MAVLINK_UDP_LISTEN` | wired in entrypoints and Compose | UDP listen address for external MAVLink. |
| `SEMLINK_MAVLINK_UDP_HOST` | wired in SITL helper | Hostname or IP that SITL sends UDP to. |
| `SEMLINK_MAVLINK_UDP_PORT` | wired in SITL helper | UDP port that SITL sends MAVLink to. |

When `SEMLINK_MAVLINK_UDP_LISTEN` is set, SemLink disables the internal
simulator and treats UDP MAVLink as the input source.

### Simulator Fallback

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_VEHICLES` | wired | Internal simulator vehicle count. |
| `SEMLINK_HZ` | wired | Internal simulator tick rate. |
| `SEMLINK_BUFFER` | wired in BlueOS entrypoint | Raw telemetry buffer capacity. |

These fields support local smoke runs without a live MAVLink source. Such runs
prove package readiness, but they do not prove autopilot wire compatibility.

### Mesh Peers

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_MESH_PEERS` | wired | Comma-separated absolute peer base URLs. |

Leave `SEMLINK_MESH_PEERS` empty for a single-node handoff. When static peers
are supplied, `/api/evidence` reports `profile.mesh.mode`, `mesh.posture`,
`mesh.configured_peer_count`, and `mesh.configured_peers`. The handoff
validator accepts only absolute `http` or `https` peer URLs.

### Downstream Consumers

| Field | Status | Purpose |
| --- | --- | --- |
| `CS_API_URL` | wired | Optional SemConnect CS API egress target. |
| `SEMLINK_CSAPI_INTERVAL` | wired | SemConnect bridge sync interval. |
| `SEMLINK_CSAPI_OBSERVATION_INTERVAL` | wired | Observation-rate limit for SemConnect egress. |

SemOps and semstreams-ui are optional pull consumers of local APIs. They do not
need profile endpoints to be available; `/api/evidence` already declares them
as downstream consumers. SemConnect remains optional standards egress: setting
`CS_API_URL` enables that egress path, but none of SemOps, semstreams-ui, or
SemConnect are runtime dependencies or package readiness requirements.

The machine-readable evidence posture for each of those entries is
`dependency_mode=optional-downstream`, `runtime_dependency=false`, and
`required_for_readiness=false`.

### Command Posture

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_COMMAND_RUNTIME_MODE` | wired | Handoff command posture, initially `hardware-readonly`. |
| `SEMLINK_HARDWARE_TRANSMIT_ENABLED` | wired fail-closed | Must remain `false` for this OpenSpec change. |

The deployable handoff may run near hardware, but hardware MAVLink command
transmit remains blocked until a later accepted OpenSpec change defines
authorization and safety evidence. The handoff validator rejects
`SEMLINK_HARDWARE_TRANSMIT_ENABLED=true`. When the package runs in
`hardware-readonly` mode, local `/api/commands` attempts are rejected before
command-intent graph writes and recorded in `/api/evidence.commands[]` as a
`hardware_block` with `scope=companion-deployment-handoff`.

Simulator command gates remain useful as evidence, but they are not hardware
authorization. `/api/evidence.commands[]` preserves compact simulator-only
preflight, ACK, and post-state proof while keeping
`hardware_transmit_authorized=false`.

### TAK Bridge

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_TAK_ENABLED` | wired | Optional TAK outbound bridge. |
| `SEMLINK_TAK_MULTICAST_ADDR` | wired | TAK multicast target. |
| `SEMLINK_TAK_TCP_LISTEN` | wired | Optional outbound TCP listener. |
| `SEMLINK_TAK_INBOUND_UDP_LISTEN` | wired | Optional inbound UDP listener. |
| `SEMLINK_TAK_INBOUND_TCP_LISTEN` | wired | Optional inbound TCP listener. |
| `SEMLINK_TAK_INTERVAL` | wired | TAK publish interval. |

The companion handoff profile keeps TAK disabled by default. TAK remains an
adapter path, not the core companion deployment proof.

## Hardware-Free Proof Path

Use these lanes in increasing fidelity:

- Fast companion e2e: `go test ./internal/e2e`
- Single-node companion demo: `./scripts/demo-single-companion.sh`
- Simple local mesh demo: `./scripts/demo-mesh-companions.sh`
- Local UDP evidence test:
  `go test ./internal/gcs -run TestUDPEvidenceSmokeProjectsExternalMAVLinkState`
- BlueOS-style package smoke: `scripts/blueos-extension-smoke.sh`
- Local ArduRover SITL lane: `scripts/ardurover-sitl-lane.sh`
- Dockerized ArduRover/SemLink lane: `scripts/ardurover-sitl-compose-up.sh`

The fast companion e2e and lightweight demo scripts are SemLink-owned
proofs. They use deterministic simulated MAVLink vehicles and local
httptest-style runtimes, write JSON reports under `.artifacts`, and do not
require SemOps, semstreams-ui, SemConnect/CS API, BlueOS, Navigator hardware,
Gazebo, SITL, or physical MAVLink devices. Boat/ArduRover is the first demo
profile, not the architecture limit.

Set `SEMLINK_DEMO_ARTIFACT` on the single-node or simple-mesh demo scripts to
write a `semlink-companion-demo-artifact-v0` envelope beside the raw report.
That envelope is the native producer handoff shape for SemOps ingestion: it
adds source fidelity, a real SemLink commit or version, generator metadata,
no-transmit posture, and optional per-node source metadata. Artifact output
uses explicit source-ref flags, Go VCS build metadata, or checkout `HEAD`, and
fails if none can provide a real `semlink_commit` or `semlink_version`; it
still does not make SemOps, SemConnect, or CS API part of the package readiness
path.

The local UDP evidence test is the fastest handoff proof. It opens a UDP
listener, sends one MAVLink heartbeat frame, verifies that the internal
simulator is disabled for an external MAVLink input, and reads the projected
vehicle state back through `/api/evidence`. It does not require Docker,
ArduPilot, BlueOS, sibling checkouts, or physical Navigator hardware.

The BlueOS-style package smoke proves the deployable container lifecycle and
local API surface. It starts the package with the companion profile, then checks
`/api/health`, `/register_service`, and `/api/evidence`. That smoke may use the
simulator fallback unless the profile points at an external MAVLink source, so
the SITL lanes remain the autopilot wire-compatibility proof.

For a local BlueOS-style package smoke:

```bash
scripts/blueos-extension-smoke.sh
```

The smoke defaults to `configs/handoff/companion.env.example`. To use a local
copy, set `SEMLINK_HANDOFF_PROFILE_FILE=/path/to/companion.env` before running
the script. The Compose target mounts that file at `/data/companion.env`, and
the BlueOS-style entrypoint sources it before starting the companion service.

For a no-Gazebo SITL/UDP proof, both ArduRover launchers source the same
handoff profile. The local lane requires ArduPilot's `sim_vehicle.py` on
`PATH`, but does not require Docker or Gazebo. Start SemLink with the profile's
`SEMLINK_MAVLINK_UDP_LISTEN`, then run:

```bash
scripts/ardurover-sitl-lane.sh
```

The local launcher sends `sim_vehicle.py` traffic to
`SEMLINK_MAVLINK_UDP_HOST:SEMLINK_MAVLINK_UDP_PORT`. The Docker wrapper sources
the profile before Compose interpolation and maps the ArduPilot container to
the same profile port:

```bash
scripts/ardurover-sitl-compose-up.sh
```

The Docker lane is operator-invoked because it builds/runs the ArduPilot SITL
container and expects sibling `semconnect` and `semstreams` checkouts. Use it
when the local host does not have `sim_vehicle.py`, or when the SemLink,
SemConnect, and SemStreams demo stack should be exercised together.

The proof readback should use:

```bash
curl -s http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/api/evidence
```

For an external MAVLink proof, the evidence should show
`profile.mavlink.external_input_configured=true`,
`profile.simulator.enabled=false`,
`profile.simulator.source=external-mavlink-udp`, increasing frame counters, and
at least one MAVLink-derived vehicle with a `vehicle_type`.

## Boundary

This profile is a handoff contract, not a hardware-control authorization. It
keeps SemLink focused on local companion APIs, configuration, and evidence.
SemOps owns COP/GCS glass, semstreams-ui can inspect generic ops/debug state,
and SemConnect remains optional standards egress. Package readiness is proven
by SemLink local APIs and evidence, not by starting any downstream consumer.
