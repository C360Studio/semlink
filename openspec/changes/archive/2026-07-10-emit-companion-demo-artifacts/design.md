## Context

SemLink's demo commands currently produce raw JSON reports for single-node and
simple-mesh companion evidence. SemOps has added a consumer-side artifact
envelope that wraps those reports with provenance, source-fidelity, generator,
no-transmit, and per-node source metadata. SemLink can adopt that envelope as a
native producer output, but only for fields it can substantiate from its own
runtime and demo configuration.

The producer path must preserve the existing raw reports because they are useful
for local debugging, release notes, and downstream consumers that do not need
the SemOps admission wrapper.

## Goals / Non-Goals

**Goals:**

- Emit `semlink-companion-demo-artifact-v0` envelopes from the SemLink demo CLI.
- Keep raw report output available and default-compatible.
- Encode SemLink version/commit, generator profile, generator command,
  no-transmit posture, and source fidelity without requiring SemOps at runtime.
- Represent per-node source metadata only when SemLink is given enough evidence
  to make the claim.
- Make deterministic demo artifacts explicitly deterministic rather than
  SITL-backed or hardware-adjacent.

**Non-Goals:**

- Do not add a SemOps, SemConnect, CS API, SITL, Gazebo, BlueOS, or Navigator
  runtime dependency to the demo commands.
- Do not implement hardware transmit.
- Do not make SemLink own SemOps artifact admission policy or COP/GCS display.
- Do not invent trusted SemOps operator authority headers.

## Decisions

1. **Add a first-class envelope type in `internal/e2e`.**
   The envelope sits next to existing report types and carries the SemOps-facing
   artifact fields. Keeping it in `internal/e2e` lets tests exercise the same
   producer code used by `cmd/semlink-demo` without creating a broader public
   API.

2. **Keep raw reports as the default `-output` behavior and add
   `-artifact-output`.**
   Existing scripts and release checks keep working. A maintainer can request
   a second JSON file for SemOps ingestion without changing the raw report path.

3. **Use deterministic fidelity by default.**
   SemLink's current demos are deterministic local harnesses. They can produce
   useful artifact envelopes, but they MUST NOT claim `sitl-backed` or
   `hardware-adjacent` unless future flags or source adapters provide concrete
   per-node MAVLink source metadata.

4. **Make timestamp coherence explicit.**
   The envelope `generated_at` comes from the embedded report timestamp in this
   slice. That avoids fresh wrappers around stale demo payloads and keeps the
   first producer contract easy to validate.

5. **Carry no-transmit posture as structured producer evidence plus text.**
   The envelope includes the existing `no_transmit_posture` text expected by
   SemOps, and the embedded report continues to carry the machine-readable
   command posture and SemOps readback evidence.

## Risks / Trade-offs

- **Risk:** SemOps may later rename envelope fields after review.
  **Mitigation:** Keep the envelope isolated behind one type and one CLI flag so
  a later rename is mechanical.

- **Risk:** A single top-level `source_fidelity` can be mistaken for a per-node
  live-source claim.
  **Mitigation:** Default to `deterministic`; per-node metadata is included only
  for explicitly supplied source claims.

- **Risk:** Generator commands can leak host-specific paths.
  **Mitigation:** Let callers supply a stable `-generator-profile`; use a concise
  generated command string that contains demo mode and profile rather than shell
  history.
