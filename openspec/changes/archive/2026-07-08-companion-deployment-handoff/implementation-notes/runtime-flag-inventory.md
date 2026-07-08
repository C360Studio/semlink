# Runtime Flag Inventory

Task 1.1 inventories the current BlueOS, SITL, evidence, and command-gate
runtime knobs against the handoff profile requirements.

## Current Binary Flags

`cmd/semgcs-demo` currently exposes these relevant flags:

- `-listen`: local HTTP listen address.
- `-nats-url`: external NATS URL when embedded runtime is disabled.
- `-embedded-nats`: starts local NATS/SemStreams runtime when true.
- `-vehicles`: number of internally simulated vehicles.
- `-hz`: internal simulator tick rate.
- `-buffer`: raw telemetry buffer capacity.
- `-mavlink-udp`: external MAVLink UDP listen address; disables the internal
  simulator when set.
- `-static`: historical UI static directory.
- `-csapi-url`: optional SemConnect CS API egress target.
- `-csapi-interval`: SemConnect bridge sync interval.
- `-csapi-observation-interval`: minimum interval between CS API observations.
- `-tak`, `-tak-multicast`, `-tak-tcp`, `-tak-inbound-udp`,
  `-tak-inbound-tcp`, and `-tak-interval`: optional TAK bridge controls.

The binary does not yet expose handoff profile flags for node identity, vehicle
identity, static mesh peers, downstream consumer declarations beyond
`CS_API_URL`, or command runtime posture.

## BlueOS Package Surface

`docker/blueos-extension/entrypoint.sh` maps environment variables to the
binary flags:

- `SEMLINK_HTTP_LISTEN`
- `SEMLINK_EMBEDDED_NATS`
- `NATS_URL`
- `SEMLINK_VEHICLES`
- `SEMLINK_HZ`
- `SEMLINK_BUFFER`
- `SEMLINK_MAVLINK_UDP_LISTEN`
- `CS_API_URL`
- `SEMLINK_CSAPI_INTERVAL`
- `SEMLINK_CSAPI_OBSERVATION_INTERVAL`
- TAK bridge variables

`compose.blueos.yml` currently passes only the embedded runtime, HTTP listen,
simulator count/rate, and MAVLink UDP listen variables. It does not yet pass a
node ID, mesh peers, downstream metadata, or a single named handoff profile.

The existing `scripts/blueos-extension-smoke.sh` verifies `/api/health` and
`/register_service`. It does not yet verify `/api/evidence`.

## SITL And UDP Surface

The local SITL lane is hardware-free and no-Gazebo:

- `scripts/ardurover-sitl-lane.sh` accepts `SIM_VEHICLE`, `ARDUPILOT_FRAME`,
  `ARDUPILOT_AIRCRAFT`, `ARDUPILOT_SPEEDUP`, `SEMLINK_MAVLINK_UDP_HOST`, and
  `SEMLINK_MAVLINK_UDP_PORT`.
- `scripts/ardurover-sitl-compose-up.sh` wires SemLink, SemConnect, local
  NATS/SemStreams, and the ArduPilot SITL container. It defaults
  `SEMLINK_MAVLINK_UDP_LISTEN` to `:14550`.
- `compose.sitl.yml` forces `-mavlink-udp` for the SemLink service and points
  SITL serial output at `semlink:<port>`.

This proves the external MAVLink UDP shape, but the scripts do not yet consume
a common handoff profile or query `/api/evidence` as the final proof.

## Evidence Surface

`/api/evidence` already exposes useful handoff material:

- `node.node_id`, with `semlink-local` as the default.
- runtime and NATS evidence through `node.runtime`, `node.nats_url`, and
  `node.semstreams_embedded`.
- raw frame, decoded frame, graph write, graph error, and buffer-drop metrics.
- downstream metadata for SemOps, semstreams-ui, and optional SemConnect.
- mesh status, summary count, watermark count, and
  `raw_mavlink_replicates_by_default: false`.
- command intent and command-gate evidence, including hardware block evidence.

`gcs.ServerOptions` can accept `NodeID`, `CSAPIURL`, and `MeshIndex`, but
`cmd/semgcs-demo` does not yet wire node identity or a mesh index/peer profile
from runtime configuration.

`/api/health` currently reports only `ok` and `time`. It is good as a liveness
check, but it is not yet a profile/readiness summary.

## Command-Gate Surface

The simulator command gate is implemented in `internal/commandgate`:

- simulator preflight requires runtime mode, safety profile, target, requestor,
  stable sender identity, bounded attempts, local override, ACK requirement,
  post-state polling, simulator confirmation, and abort readiness.
- hardware mode returns `hardware-transmit-blocked` with explicit block
  evidence and does not transmit.

The HTTP command endpoint currently records command intents. Command-gate
results are stored as evidence when code calls `RecordCommandGateResult`, but
there is not yet a deployment-profile flag that declares simulator versus
hardware posture.

The hardware block scope still names `pivot-companion-mesh`; task 5.1 should
refresh that evidence for the deployment handoff scope without enabling
hardware transmit.

## Handoff Profile Gaps For Task 1.2

Task 1.2 should introduce one documented profile surface that covers:

- node ID, likely `SEMLINK_NODE_ID` and `-node-id`;
- optional vehicle ID or callsign defaults, if needed for boat-class handoff;
- MAVLink UDP listen address and simulator disablement;
- embedded versus external SemStreams mode and NATS URL;
- static mesh peer URLs and peer count evidence;
- optional downstream consumer metadata for SemOps, semstreams-ui, and
  SemConnect;
- command runtime posture, with hardware transmit remaining blocked.

The immediate implementation target should be a small profile parser/normalizer
with tests, then wiring into the binary, BlueOS entrypoint, Compose smoke, and
evidence output in later tasks.
