# Navigator Read-Only Hardware Smoke

This is the first single-device hardware lane for SemLink. It is intentionally
read-only: the smoke checks that a Navigator-class BlueOS device and the
SemLink companion service are visible, but it does not configure endpoints,
change autopilot parameters, arm, change modes, upload missions, or transmit
MAVLink commands.

Current upstream context:

- Blue Robotics describes Navigator as a Raspberry Pi 4 flight-controller kit
  shipping with BlueOS, intended for ROVs, USVs, and other robotics, with
  BlueOS support for ArduPilot firmware management and user-defined extensions.
- BlueOS documents an Available Services page for developer access to service
  HTTP interfaces, a MAVLink Endpoints manager for serial/UDP/TCP endpoint
  configuration, and a MAVLink Inspector backed by MAVLink2REST on port `6040`.
- BlueOS also notes that endpoint configuration for external use is an
  intentional setup action; this smoke does not perform that action.

Sources:

- [Navigator product page](https://bluerobotics.com/store/comm-control-power/control/navigator/)
- [BlueOS advanced usage](https://blueos.cloud/docs/latest/usage/advanced/)

## Preconditions

- One powered Navigator-class BlueOS device on a bench network.
- Vehicle is safe: props/thrusters cannot spin, command transmit is not under
  test, and the operator can power down the vehicle.
- The control station can reach BlueOS, usually `http://blueos.local`.
- Optional: the SemLink BlueOS extension container is running and reachable via
  its assigned extension URL or host/port.

## Smoke Command

Run from the SemLink checkout:

```bash
scripts/navigator-readonly-smoke.sh
```

Useful overrides:

```bash
BLUEOS_URL=http://blueos.local \
ARDUPILOT_MANAGER_URL=http://blueos.local:8000 \
MAVLINK2REST_URL=http://blueos.local:6040 \
SEMLINK_URL=http://blueos.local/extensionv2/semlinkcompanion \
scripts/navigator-readonly-smoke.sh
```

The script writes local artifacts under `.artifacts/navigator-readonly-smoke/`
by default. Override with `ARTIFACT_DIR=/path/to/dir` when collecting evidence
for a run log.

## Required Evidence

- BlueOS web root responds.
- If SemLink is installed, `/api/health` responds.
- If SemLink is installed, `/register_service` responds with the BlueOS service
  registration.
- ArduPilot Manager docs and MAVLink2REST root are collected when available.

## Explicit Non-Goals

- No BlueOS MAVLink endpoint creation or modification.
- No autopilot parameter reads that require special tooling, and no parameter
  writes.
- No direct Navigator hardware library access.
- No MAVLink command transmit.
- No firmware install, reboot, mode change, arming, mission upload, or actuator
  output.

Those actions require separate tasking and, for hardware command transmit, a
new accepted OpenSpec change.
