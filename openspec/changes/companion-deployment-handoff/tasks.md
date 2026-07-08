## 1. Handoff Profile

- [x] 1.1 Inventory current BlueOS, SITL, evidence, and command-gate runtime
      flags against the handoff requirements
- [x] 1.2 Add a documented companion handoff profile for node identity, MAVLink
      UDP input, local SemStreams mode, mesh peers, and downstream consumers
- [x] 1.3 Validate the handoff profile with focused unit tests and clear
      operator-facing errors
- [x] 1.4 Expose handoff profile metadata through local readiness or evidence
      output

## 2. Package Readiness

- [x] 2.1 Wire the BlueOS-style entrypoint and Compose smoke to the handoff
      profile
- [x] 2.2 Extend the package smoke to verify `/api/health`,
      `/register_service`, and `/api/evidence`
- [ ] 2.3 Add tests that package metadata declares companion/mesh behavior and
      does not declare hardware command transmit capability
- [ ] 2.4 Document package run commands, required inputs, and release/tag
      checkpoint criteria
- [ ] 2.5 Plan or begin the SemGCS runtime rename/migration once package
      readiness no longer depends on the historical UI surface

## 3. SITL And UDP Evidence

- [ ] 3.1 Connect the ArduRover/boat SITL lane to the handoff profile without
      requiring Gazebo
- [ ] 3.2 Add a local UDP evidence smoke or test that proves external MAVLink
      disables the internal simulator and updates `/api/evidence`
- [ ] 3.3 Document the hardware-free handoff proof path and any operator-invoked
      Docker/SITL lanes

## 4. Mesh And Downstream Visibility

- [ ] 4.1 Add static mesh peer configuration to the handoff profile and expose
      mesh posture/peer count in evidence
- [ ] 4.2 Keep raw MAVLink exclusion visible in the handoff evidence contract
- [ ] 4.3 Keep SemOps, semstreams-ui, and SemConnect metadata optional and
      downstream in handoff docs and evidence
- [x] 4.4 Add a draft SemOps v0 readback adapter using MAVLink-native contract
      fixtures without CS API in the hot path
- [ ] 4.5 Recheck the adapter after the SemOps contract branch is pushed and
      final hold-out review is complete before runtime enablement

## 5. Command Safety

- [ ] 5.1 Ensure handoff packages reject hardware command transmit with
      handoff-scope evidence
- [ ] 5.2 Preserve simulator-only command evidence without authorizing hardware
      transmit
- [ ] 5.3 Add tests for handoff fail-closed command behavior

## 6. Final Validation

- [ ] 6.1 Run focused package/config/evidence tests
- [ ] 6.2 Run `go test ./...`, `go build ./...`, and
      `openspec validate --all --strict`
- [ ] 6.3 Record the final validation note and mark release/tag readiness or
      explicit waivers
