## Why

The companion-mesh baseline is now accepted, but an early adopter still needs a
concrete handoff shape: a deployable companion service they can run beside a
MAVLink boat stack and inspect through local evidence APIs. This change turns
the archived architecture into a runnable BlueOS-style/SITL-ready package
without making hardware command transmit or SemLink-owned GCS glass part of the
handoff.

## What Changes

- Define the first deployable companion handoff contract for SemLink.
- Add a BlueOS-style package profile that can be built, launched, and smoked
  locally.
- Add a configuration profile for node identity, MAVLink UDP input, local
  SemStreams runtime, mesh peers, and downstream evidence consumers.
- Add a no-Gazebo SITL/UDP evidence lane that proves the package can ingest
  ArduRover/boat MAVLink and expose current local evidence.
- Require `/api/health`, `/register_service`, and `/api/evidence` readiness
  checks for the handoff.
- Keep SemOps and semstreams-ui as downstream consumers of local APIs rather
  than dependencies of the package.
- Preserve the existing hardware command-transmit block; this handoff may run
  near hardware but must not enable hardware MAVLink transmit.

## Capabilities

### New Capabilities

- `companion-deployment-handoff`: Covers the deployable early-adopter package,
  configuration profile, smoke evidence, and release/tag readiness boundary.

### Modified Capabilities

- `companion-mesh-product`: Clarify that deployable handoffs expose local
  APIs/configuration and downstream metadata without adding SemLink-owned GCS
  glass.
- `mavlink-companion-runtime`: Require package-level BlueOS-style and SITL/UDP
  evidence lanes for MAVLink companion runtime claims.
- `mesh-synchronization`: Require deployment configuration to expose peer and
  mesh-readiness metadata without replicating raw MAVLink.
- `rules-and-command-safety`: Require deployable handoffs to keep hardware
  command transmit disabled unless a later accepted OpenSpec change authorizes
  it.

## Impact

- BlueOS extension Docker image, metadata, and lifecycle smoke scripts.
- CLI/config/env handling for node identity, MAVLink UDP input, local
  SemStreams, peers, and downstream consumer metadata.
- Local HTTP evidence/readiness API documentation and tests.
- SITL/UDP demo scripts and handoff documentation.
- Release/tag guidance for the first companion package checkpoint.
