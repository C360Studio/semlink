# ArduRover SITL Lane

This lane feeds real ArduPilot Rover SITL telemetry into SemLink without
Gazebo, physical Navigator hardware, or command transmit.

ArduPilot's SITL workflow starts `sim_vehicle.py` for a selected vehicle and
frame. The official docs list Rover frames such as `rover`, `rover-skid`,
`sailboat`, and `sailboat-motor`, and document a headless UDP path with
`--no-mavproxy` plus `-A "--serial0=udpclient:<gcs ip>:14550"`.

Sources:

- [ArduPilot SITL testing](https://ardupilot.org/dev/docs/using-sitl-for-ardupilot-testing.html)

## Local Run

Both SITL wrappers load the companion handoff profile before launching. The
default profile is `configs/handoff/companion.env.example`; use
`SEMLINK_HANDOFF_PROFILE_FILE=/path/to/companion.env` to point at a local copy.

Start SemLink with the profile's UDP MAVLink listener:

```bash
set -a
. configs/handoff/companion.env.example
set +a
go run ./cmd/semgcs-demo -mavlink-udp="${SEMLINK_MAVLINK_UDP_LISTEN}"
```

In a second terminal, run ArduRover SITL. The launcher reads
`SEMLINK_MAVLINK_UDP_HOST` and `SEMLINK_MAVLINK_UDP_PORT` from the same
profile and passes them to `sim_vehicle.py` as a headless UDP client target:

```bash
scripts/ardurover-sitl-lane.sh
```

Useful frame overrides:

```bash
ARDUPILOT_FRAME=sailboat-motor scripts/ardurover-sitl-lane.sh
ARDUPILOT_FRAME=rover-skid scripts/ardurover-sitl-lane.sh
```

## Docker Run

The ArduPilot image is intentionally separate from the SemLink image because it
is large and only needed for this fidelity lane. The wrapper sources the
handoff profile before Compose interpolation so SemLink listens on
`SEMLINK_MAVLINK_UDP_LISTEN` and ArduPilot sends to the matching
`SEMLINK_MAVLINK_UDP_PORT` inside the Compose network.

```bash
scripts/ardurover-sitl-compose-up.sh
```

Optional build and run pins:

```bash
ARDUPILOT_REF=master ARDUPILOT_FRAME=sailboat-motor \
  scripts/ardurover-sitl-compose-up.sh
```

The wrapper follows the existing demo compose shape, so it expects sibling
`semconnect` and `semstreams` checkouts. Override their locations with
`SEMCONNECT_ROOT` and `SEMSTREAMS_ROOT`. Override the profile with
`SEMLINK_HANDOFF_PROFILE_FILE=/path/to/companion.env`.

## Evidence Checks

The local and Docker lanes are successful when `/api/evidence` shows that
SemLink is using external MAVLink input and has projected current vehicle state:

```bash
curl -s http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/api/evidence
```

The expected handoff signals are:

- `profile.mavlink.external_input_configured=true`
- `profile.simulator.enabled=false`
- `profile.simulator.source=external-mavlink-udp`
- `node.raw_frames` and `node.decoded_frames` greater than zero
- at least one `vehicles[]` item with `vehicle_type` set from the MAVLink
  heartbeat

The quick non-Docker proof for those signals is:

```bash
go test ./internal/gcs -run TestUDPEvidenceSmokeProjectsExternalMAVLinkState
```

## Optional E2E Test

The real ArduPilot SITL scenario is part of the Go test suite, but skipped by
default so local and PR checks do not require ArduPilot. Enable it explicitly
when `sim_vehicle.py` is installed:

```bash
SEMLINK_E2E_SITL=1 go test ./internal/e2e -run TestArduPilotSITLArtifactE2E -count=1 -v
```

Useful overrides:

```bash
SIM_VEHICLE=/path/to/sim_vehicle.py \
SEMLINK_E2E_SITL_TIMEOUT=2m \
SEMLINK_E2E_SITL_FRAME=sailboat-motor \
SEMLINK_E2E_SITL=1 \
go test ./internal/e2e -run TestArduPilotSITLArtifactE2E -count=1 -v
```

If host ArduPilot is not installed, build the repo-owned Rover SITL image and
point the e2e test at it:

```bash
docker build -f docker/ardupilot-sitl/Dockerfile -t c360studio/semlink-ardupilot-sitl:local .

SEMLINK_E2E_SITL=1 \
SEMLINK_E2E_SITL_DOCKER_IMAGE=c360studio/semlink-ardupilot-sitl:local \
SEMLINK_E2E_SITL_TIMEOUT=4m \
go test ./internal/e2e -run TestArduPilotSITLArtifactE2E -count=1 -v -timeout 5m
```

The image includes ArduPilot's SITL prerequisites and a prebuilt `bin/ardurover`
binary so repeated e2e runs spend their time on runtime evidence instead of a
container-local compile.

Docker mode defaults SemLink's UDP listener to `0.0.0.0` and the SITL output
target to `host.docker.internal`. It uses MAVProxy only as a non-interactive
MAVLink bridge with the output module loaded, so the run stays headless and
does not pull in map, console, terrain, or GCS behavior. Override those only
when the host route or network shape is different:

```bash
SEMLINK_E2E_SITL_DOCKER_PLATFORM=linux/arm64 \
SEMLINK_E2E_SITL_DOCKER_NETWORK=host \
SEMLINK_E2E_SITL_UDP_LISTEN_HOST=0.0.0.0 \
SEMLINK_E2E_SITL_OUTPUT_HOST=host.docker.internal \
SEMLINK_E2E_SITL_MAVPROXY_ARGS="--non-interactive --default-modules=output --retries=30" \
SEMLINK_E2E_SITL_DELAY_START_SECONDS=5 \
SEMLINK_E2E_SITL=1 \
SEMLINK_E2E_SITL_DOCKER_IMAGE=c360studio/semlink-ardupilot-sitl:local \
go test ./internal/e2e -run TestArduPilotSITLArtifactE2E -count=1 -v
```

When enabled, the test starts embedded SemStreams, a SemLink companion runtime
with external MAVLink UDP input, and ArduPilot Rover SITL without Gazebo. It
polls `/api/evidence` until native MAVLink state appears, then emits and
validates a temp `sitl-backed` artifact with the current SemLink commit.

## SemOps Artifact Handoff

After SemLink and ArduPilot SITL are running and `/api/evidence` shows external
MAVLink frames, produce the SemOps-facing artifact from observed evidence:

```bash
scripts/demo-sitl-artifact.sh
```

By default the wrapper reads
`http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/api/evidence` and writes:

- `.artifacts/semlink-demo-sitl/report.json`
- `.artifacts/semlink-demo-sitl/artifact.json`

Override the runtime URL or output locations with:

```bash
SEMLINK_RUNTIME_URL=http://127.0.0.1:8081 \
SEMLINK_SITL_DEMO_REPORT=.artifacts/semlink-demo-sitl/report.json \
SEMLINK_SITL_DEMO_ARTIFACT=.artifacts/semlink-demo-sitl/artifact.json \
scripts/demo-sitl-artifact.sh
```

The artifact producer fails before writing a `sitl-backed` artifact unless
`/api/evidence` proves external MAVLink input, observed raw and decoded frames,
at least one MAVLink system ID, raw MAVLink mesh exclusion, blocked hardware
transmit posture, and a real SemLink commit or version. This keeps SemLink's
SITL fidelity claim producer-owned; SemOps can consume and reject the artifact,
but it does not synthesize SITL evidence.

## Evidence Boundary

This lane proves that SemLink can listen for ArduPilot Rover/boat MAVLink over
UDP, buffer raw frames, project current vehicle state, and preserve the
MAVLink heartbeat vehicle type in graph state.

It does not prove BlueOS packaging, physical Navigator hardware, Gazebo
physics, or command transmit safety. Those are separate lanes in the
companion-mesh change.
