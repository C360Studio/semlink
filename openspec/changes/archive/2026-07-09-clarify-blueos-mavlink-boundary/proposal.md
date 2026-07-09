## Why

SemOps now treats BlueOS/Navigator as an optional SemLink deployment profile
rather than the SemLink-to-SemOps protocol boundary. SemLink specs already
separate package readiness from SITL/UDP wire proof, but the accepted baseline
should state that boundary directly before downstream adapter work resumes.

## What Changes

- Clarify that SemLink's compatibility proof remains MAVLink-native and can be
  satisfied by local UDP or ArduPilot SITL evidence without BlueOS.
- Clarify that BlueOS, Navigator, and companion-Pi packaging are deployment
  profiles for SemLink, not required protocol surfaces for SemOps.
- Clarify that BlueOS service registration, Navigator readiness, and package
  lifecycle are deployment metadata only, not command/readback compatibility
  proof.
- Clarify that the SemOps readback adapter keeps the same native request shape
  across BlueOS, Pi, SITL, and plain native deployments.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `mavlink-companion-runtime`: Add the native MAVLink/SITL compatibility proof
  boundary.
- `companion-deployment-handoff`: Add the BlueOS deployment-profile boundary
  for the SemOps readback adapter and handoff package.
- `companion-mesh-product`: Add the product-boundary rule that BlueOS is an
  optional deployment lane, not the mesh or protocol boundary.

## Impact

- OpenSpec baseline requirements for runtime, handoff, and product-boundary
  specs.
- SemOps readback adapter documentation.
- No runtime code, dependency, schema, or wire payload changes.
