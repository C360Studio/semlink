## ADDED Requirements

### Requirement: Standard ArduPilot SITL Container

SemLink SHALL own a standard ArduPilot Rover SITL container recipe for shared
SITL-backed evidence runs.

#### Scenario: Standard container recipe is pinned

- **WHEN** maintainers build the SemLink ArduPilot SITL image without explicit
  overrides
- **THEN** the build uses the SemLink-owned image name
  `c360studio/semlink-ardupilot-sitl`
- **AND** the default image tag is `rover-4.6.3`
- **AND** the default ArduPilot source ref is `Rover-4.6.3`
- **AND** the image prebuilds `bin/ardurover` and uses `sim_vehicle.py` as the
  entrypoint

#### Scenario: Standard container remains headless

- **WHEN** the standard container runs in a SemLink SITL lane
- **THEN** it launches ArduPilot Rover SITL without Gazebo, physical hardware,
  GUI map/console behavior, or a required GCS
- **AND** it sends MAVLink to the configured SemLink UDP listener
- **AND** downstream evidence records any non-standard `ARDUPILOT_REF`, image,
  or tag override used for compatibility testing
