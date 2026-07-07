# Companion Handoff Profile

The companion handoff profile is the operator-facing configuration surface for
the deployable SemLink companion package. It documents the values needed to run
one boat-local node with local evidence APIs, optional UDP MAVLink input,
optional mesh peers, and optional downstream consumers.

Use the copyable profile at:

```bash
configs/handoff/companion.env.example
```

Task 1.2 defines the profile contract. Parser validation and full runtime
wiring are later tasks in `companion-deployment-handoff`, so this document
labels fields that are already consumed by current scripts versus fields that
are reserved for the next implementation slices.

## Profile Fields

### Identity

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_NODE_ID` | planned | Stable companion node ID, for example `boat-alpha`. |
| `SEMLINK_VEHICLE_ID` | planned | Local vehicle/profile ID when a single node represents one boat. |
| `SEMLINK_CALLSIGN` | planned | Human-readable boat callsign for evidence and demos. |

The current evidence API can carry a node ID through `gcs.ServerOptions`, but
`cmd/semgcs-demo` does not yet expose `-node-id`. Task 1.3 should validate the
identity fields, and task 1.4 should expose them through readiness/evidence.

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
| `SEMLINK_MESH_PEERS` | planned | Comma-separated absolute peer base URLs. |

Leave `SEMLINK_MESH_PEERS` empty for a single-node handoff. Task 4.1 should
wire this into mesh posture evidence and peer count reporting.

### Downstream Consumers

| Field | Status | Purpose |
| --- | --- | --- |
| `CS_API_URL` | wired | Optional SemConnect CS API egress target. |
| `SEMLINK_CSAPI_INTERVAL` | wired | SemConnect bridge sync interval. |
| `SEMLINK_CSAPI_OBSERVATION_INTERVAL` | wired | Observation-rate limit for SemConnect egress. |

SemOps and semstreams-ui are optional pull consumers of local APIs. They do not
need profile endpoints to be available; `/api/evidence` already declares them
as downstream consumers.

### Command Posture

| Field | Status | Purpose |
| --- | --- | --- |
| `SEMLINK_COMMAND_RUNTIME_MODE` | planned | Handoff command posture, initially `hardware-readonly`. |
| `SEMLINK_HARDWARE_TRANSMIT_ENABLED` | planned | Must remain `false` for this OpenSpec change. |

The deployable handoff may run near hardware, but hardware MAVLink command
transmit remains blocked until a later accepted OpenSpec change defines
authorization and safety evidence.

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

## Usage Sketch

For a local BlueOS-style package smoke:

```bash
set -a
. configs/handoff/companion.env.example
set +a
scripts/blueos-extension-smoke.sh
```

For a no-Gazebo SITL/UDP proof, start SemLink with
`SEMLINK_MAVLINK_UDP_LISTEN=:14550`, then run:

```bash
scripts/ardurover-sitl-lane.sh
```

The proof readback should use:

```bash
curl -s http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/api/evidence
```

## Boundary

This profile is a handoff contract, not a hardware-control authorization. It
keeps SemLink focused on local companion APIs, configuration, and evidence.
SemOps owns COP/GCS glass, semstreams-ui can inspect generic ops/debug state,
and SemConnect remains optional standards egress.
