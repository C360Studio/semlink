## 1. OpenSpec And Product Boundary

- [x] 1.1 Initialize OpenSpec project structure and Codex helper prompts
- [x] 1.2 Add SemLink project context and companion-mesh change set
- [x] 1.3 Supersede ADR 001 with ADR 003 as the forward product boundary
- [x] 1.4 Keep ADR 002 as a TAK / CoT adapter decision without COP ownership
- [x] 1.5 Review and accept the `pivot-companion-mesh` OpenSpec change

## 2. Companion Runtime Slice

- [x] 2.1 Inventory SemOps MAVLink parser, UDP listener, replay, and command
      helper code for porting or shared-module extraction
- [x] 2.2 Add a deterministic multi-boat simulator harness with per-node local
      SemStreams state
- [x] 2.3 Add ArduRover / ArduPilot SITL telemetry lane without Gazebo
- [x] 2.4 Add BlueOS extension packaging skeleton and local lifecycle smoke
- [x] 2.5 Add read-only hardware smoke plan for a single Navigator-class device

## 3. Mesh Synchronization Slice

- [x] 3.1 Define mesh envelope types with origin, ordering, expiry, confidence,
      merge policy, and payload hash
- [x] 3.2 Define per-origin watermarks and diff generation
- [x] 3.3 Add unreliable-link harness tests for reconnect, duplicate delivery,
      stale telemetry expiry, and bounded catch-up
- [x] 3.4 Add one concrete demo transport
- [x] 3.5 Prove raw MAVLink frames do not replicate over the mesh by default

## 4. Rules And Command Safety Slice

- [x] 4.1 Define local rule evaluation input and trace entity shape
- [ ] 4.2 Add one observe-only swarm coordination rule over mesh-visible state
- [ ] 4.3 Add simulator-only command preflight gate
- [ ] 4.4 Add simulator command transmit gate with COMMAND_ACK and post-state
      polling
- [ ] 4.5 Keep hardware transmit blocked until a separate OpenSpec change
      defines authorization and safety evidence

## 5. Operator Experience And Egress

- [ ] 5.1 Reframe UI language from SemGCS toward companion / mesh operations
- [ ] 5.2 Add node, vehicle, mesh, rule trace, and command evidence views
- [ ] 5.3 Keep SemOps consumption and SemConnect standards egress optional and
      downstream
- [ ] 5.4 Validate the final slice with docs, unit tests, integration harness,
      and demo-script updates
