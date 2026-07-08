# BlueOS Extension Packaging

SemLink's BlueOS lane packages the companion service as an Extension-style
Docker image. This is a packaging and lifecycle proof, not a Navigator hardware
claim.

Current BlueOS extension guidance treats an extension as a Docker image plus
metadata. The image uses Docker labels for version, permissions, authors,
maintainer/company, readme, links, type, and tags. SemLink uses
`register_service` for BlueOS discoverability, while the forward product surface
remains local APIs and CLI/config rather than a repo-owned dashboard.

Sources:

- [BlueOS extension development](https://blueos.cloud/docs/latest/development/extensions/)
- [BlueOS Extensions Repository metadata][blueos-extensions-repo]

## Local Lifecycle Smoke

The local smoke runs the extension container with embedded SemStreams runtime
using the companion handoff profile, then verifies:

- `/api/health`
- `/register_service`
- `/api/evidence`

```bash
scripts/blueos-extension-smoke.sh
```

The smoke expects a sibling SemStreams checkout. Override with
`SEMSTREAMS_ROOT=/path/to/semstreams` when needed.

The smoke defaults to `configs/handoff/companion.env.example`. Override with
`SEMLINK_HANDOFF_PROFILE_FILE=/path/to/companion.env` to run a copied local
profile. Compose mounts that file as `/data/companion.env`, and the BlueOS-style
entrypoint sources it before translating environment values into runtime flags.
The evidence check asserts the SemLink companion evidence contract, configured
profile, handoff node ID, and fail-closed hardware transmit posture.

## Required Inputs

Run package smokes from the SemLink checkout with these local inputs:

- Docker with Compose v2 access.
- A sibling SemStreams checkout, or `SEMSTREAMS_ROOT=/path/to/semstreams`.
- A handoff profile file. The default is
  `configs/handoff/companion.env.example`; use a copied profile for local
  boats or alternate ports.
- A free host HTTP port for `SEMLINK_BLUEOS_HOST_PORT`, defaulting to `8081`.

The profile must keep `SEMLINK_HARDWARE_TRANSMIT_ENABLED=false` for this
handoff. The validator rejects `true`, and the smoke checks evidence for the
blocked hardware-transmit posture.

## Run Commands

Preflight Compose interpolation without building the image:

```bash
docker compose -f compose.blueos.yml config
```

Run the full local BlueOS-style lifecycle smoke:

```bash
scripts/blueos-extension-smoke.sh
```

Run with explicit checkouts, profile, and host port:

```bash
SEMSTREAMS_ROOT=/path/to/semstreams \
SEMLINK_HANDOFF_PROFILE_FILE=/path/to/companion.env \
SEMLINK_BLUEOS_HOST_PORT=8081 \
scripts/blueos-extension-smoke.sh
```

Inspect the package endpoints while the Compose service is running:

```bash
curl -fsS http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/api/health
curl -fsS http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/register_service
curl -fsS http://127.0.0.1:${SEMLINK_BLUEOS_HOST_PORT:-8081}/api/evidence
```

Tear down any leftover local smoke container:

```bash
docker compose -p semlink-blueos-smoke -f compose.blueos.yml down --remove-orphans
```

## Release / Tag Checkpoint

Do not tag or publish a companion handoff checkpoint until the release note or
tag proposal records:

- `go test ./...`
- `go build ./...`
- `openspec validate --all --strict`
- `docker compose -f compose.blueos.yml config`
- `scripts/blueos-extension-smoke.sh`, or an explicit Docker-environment
  waiver such as a resolver/cache timeout before SemLink starts
- The evidence contract observed at `/api/evidence`
- The BlueOS registration metadata observed at `/register_service`
- The exact handoff profile used, with secrets removed if a copied local file
  was used
- A statement that hardware MAVLink command transmit remains disabled

This checkpoint does not publish to Docker Hub or the BlueOS Bazaar. It is a
repo-local readiness gate for deciding whether a later publish/tag action is
safe to propose.

## Packaging Files

- `docker/blueos-extension/Dockerfile`: BlueOS-style image and labels
- `docker/blueos-extension/entrypoint.sh`: extension runtime flags
- `compose.blueos.yml`: local lifecycle Compose target
- `blueos/extension/metadata.json`: Bazaar repository metadata skeleton
- `blueos/extension/README.md`: submission note and evidence boundary

## Evidence Boundary

This slice proves that SemLink exposes BlueOS-compatible service registration
and has an image/lifecycle shape that can be tested locally.

It does not publish to Docker Hub or the BlueOS Bazaar. It also does not access
Navigator hardware or enable hardware command transmit; those remain separate
OpenSpec tasks.

[blueos-extensions-repo]: https://raw.githubusercontent.com/bluerobotics/BlueOS-Extensions-Repository/master/README.md
