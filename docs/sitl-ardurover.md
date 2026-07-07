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

Start SemLink with a UDP MAVLink listener:

```bash
MAVLINK_UDP_LISTEN=:14550 go run ./cmd/semgcs-demo -mavlink-udp=:14550
```

For the companion handoff profile, the matching field is
`SEMLINK_MAVLINK_UDP_LISTEN=:14550` in
`configs/handoff/companion.env.example`.

In a second terminal, run ArduRover SITL:

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
is large and only needed for this fidelity lane.

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
`SEMCONNECT_ROOT` and `SEMSTREAMS_ROOT`.

## Evidence Boundary

This lane proves that SemLink can listen for ArduPilot Rover/boat MAVLink over
UDP, buffer raw frames, project current vehicle state, and preserve the
MAVLink heartbeat vehicle type in graph state.

It does not prove BlueOS packaging, physical Navigator hardware, Gazebo
physics, or command transmit safety. Those are separate lanes in the
companion-mesh change.
