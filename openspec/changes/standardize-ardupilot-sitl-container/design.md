## Context

The SITL artifact lane is producer-owned by SemLink. The previous Docker
recipe was enough to prove the lane locally, but its default `ARDUPILOT_REF` was
`master`. That makes shared demo evidence drift by date and host cache. A
stable image contract gives SemLink a repeatable baseline while still allowing
explicit non-standard ArduPilot refs when compatibility testing requires them.

## Decisions

- Use `Rover-4.6.3` as the default ref for the first standard image. It is a
  concrete ArduPilot Rover tag, not a moving branch.
- Keep the image name SemLink-owned:
  `c360studio/semlink-ardupilot-sitl:rover-4.6.3`.
- Store the image/ref defaults in `docker/ardupilot-sitl/standard.env` so build
  scripts, Compose wrappers, and downstream docs share the same values.
- Preserve the existing Dockerfile shape: Ubuntu base, ArduPilot source under
  `/opt/ardupilot`, `sim_vehicle.py` entrypoint, and prebuilt `bin/ardurover`.
- Do not publish or pull an external image as a hard dependency in tests. The
  repo can build the image locally, and the env-gated e2e can point at either a
  local or published copy of the standard tag.

## Non-Goals

- Do not add Gazebo, GUI map/console behavior, Navigator hardware, or command
  transmit.
- Do not make SemOps build or own the image.
- Do not claim that the standard Rover container covers PX4, Copter, Plane, or
  multi-companion SITL meshes yet.

## Follow-Ups

- Publish the standard tag from CI once registry policy is settled.
- Add a multi-platform build once arm64 runtime evidence is needed for
  Navigator/Pi-like stacks.
- Add separate tags for Copter, Plane, or Sub only when those lanes have
  evidence and consumers.
