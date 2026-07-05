# Navigator Read-Only Smoke

Task 2.5 adds the first single-device hardware lane without making hardware
control claims.

Implementation:

- `docs/navigator-readonly-smoke.md` defines the bench-safety envelope,
  read-only evidence, upstream BlueOS/Navigator assumptions, and explicit
  non-goals.
- `scripts/navigator-readonly-smoke.sh` collects local artifacts with HTTP GET
  requests only.
- `.artifacts/` is ignored because the smoke writes run-specific local
  evidence there.

Evidence boundary:

- One Navigator-class BlueOS device is enough.
- No BlueOS endpoint configuration is performed.
- No direct Navigator hardware library access is performed.
- No MAVLink command transmit is enabled.
