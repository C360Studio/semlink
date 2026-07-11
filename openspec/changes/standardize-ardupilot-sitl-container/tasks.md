## 1. Standard Container Contract

- [x] 1.1 Add a canonical SITL container env file with image, tag, ArduPilot
  ref, vehicle, frame, aircraft, and speedup defaults.
- [x] 1.2 Pin the Dockerfile and Compose defaults to a concrete Rover ref
  instead of moving ArduPilot `master`.
- [x] 1.3 Add a standard image build script that uses the canonical env file.

## 2. Docs And Tests

- [x] 2.1 Document the standard image/tag and downstream e2e usage.
- [x] 2.2 Add static tests that guard the standard container contract and
  prevent accidental drift back to `master`.
- [ ] 2.3 Build the Docker image and run the env-gated real SITL e2e on a host
  or CI runner with Docker network support.

## 3. Validation

- [x] 3.1 Run focused SITL tests.
- [x] 3.2 Run broad Go and OpenSpec validation.
