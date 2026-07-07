# ArduPilot SITL Lane

Task 2.3 adds a headless ArduPilot Rover SITL lane without Gazebo.

Implementation:

- `internal/mavlink.UDPSource` listens for MAVLink UDP datagrams and converts
  validated MAVLink v2 frames into SemLink raw frames.
- `cmd/semgcs-demo -mavlink-udp=:14550` disables the in-process simulator and
  ingests external SITL telemetry.
- `internal/sitl` defines the supported `sim_vehicle.py` Rover command shape,
  including rover, skid-steer, sailboat, and sailboat-motor frames.
- `docker/ardupilot-sitl/Dockerfile` and `compose.sitl.yml` provide an optional
  ArduPilot image for local evidence runs.
- `docs/sitl-ardurover.md` captures local and Docker run commands.

Evidence boundary:

- No Gazebo dependency is required.
- No Navigator hardware is required.
- No command transmit path is enabled.
