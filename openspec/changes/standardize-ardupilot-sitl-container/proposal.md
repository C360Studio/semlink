## Why

SemLink now owns the SITL-backed artifact lane, but the shared container path
must not depend on each maintainer's host `sim_vehicle.py` install or a moving
ArduPilot `master` checkout. SemOps, SemLink, and demo evidence need one named
ArduPilot SITL container contract that can be rebuilt, referenced, and audited.

## What Changes

- Define the standard SemLink ArduPilot SITL image contract in
  `docker/ardupilot-sitl/standard.env`.
- Pin the default ArduPilot source ref to `Rover-4.6.3` while keeping explicit
  overrides available for intentional compatibility runs.
- Tag the default image as `c360studio/semlink-ardupilot-sitl:rover-4.6.3`.
- Keep the image headless: prebuilt `bin/ardurover`, `sim_vehicle.py`
  entrypoint, no Gazebo, no physical hardware, no GCS dependency.
- Route Compose and the SITL artifact e2e docs through the standard image
  contract.

## Impact

- Affected code/scripts: `docker/ardupilot-sitl`, `compose.sitl.yml`, SITL
  runner scripts, and static contract tests.
- Affected docs: ArduRover SITL lane guidance and downstream e2e examples.
- Downstream systems: SemOps can request the standard image/tag for
  SITL-backed artifact evidence instead of asking every host to install
  ArduPilot locally.
