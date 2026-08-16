# BlueOS Extension Packaging

SemLink's BlueOS lane packages the companion service as an Extension-style
Docker image. This is a packaging and lifecycle proof, not a Navigator hardware
claim.

## Product Boundary

BlueOS is treated as an optional vehicle appliance layer. It is valuable for
Blue Robotics operators because it owns the onboard-computer UX around hardware
bring-up, firmware and parameter management, networking, logs, MAVLink routing,
camera/video services, and extension lifecycle.

SemLink's BlueOS package is an adoption path for that ecosystem, not the center
of the runtime architecture. The same SemLink companion contract must run as a
plain native process, a SITL/UDP service, a companion-Pi service, or a BlueOS
extension. Native MAVLink evidence remains the compatibility boundary;
`/register_service` and BlueOS package lifecycle evidence are deployment
metadata.

Do not depend on BlueOS REST, MAVLink2REST, endpoint-manager state, SemOps,
SemConnect, or CS API to prove package readiness. SemOps consumes SemLink as
fleet COP/GCS glass, and SemConnect remains optional standards egress.

SemStreams beta.160 packages default persistent embedded state to
`SEMLINK_NATS_STATE_DIR=/data/nats-beta160`. First initialization stamps an
empty namespace; graceful and unexpected restarts validate the exact stamp and
reopen the same database. See
[`semstreams-beta160-migration.md`](semstreams-beta160-migration.md) before
changing package versions. No alpha/beta release transforms, copies, adopts,
upgrades, or downgrades stored data, and incompatible state requires an
explicit operator reset.

Current BlueOS extension guidance treats an extension as a Docker image plus
metadata. The image uses Docker labels for version, permissions, authors,
maintainer/company, readme, links, type, and tags. SemLink uses
`register_service` for BlueOS discoverability, while the forward product surface
remains local APIs and CLI/config rather than a repo-owned dashboard.

The BlueOS handoff image is API/evidence-only by default. It does not build or
bundle the SemGCS Svelte demo UI; the root demo image can keep that dashboard
surface until a later migration removes or rehomes it.

Sources:

- [BlueOS extension development](https://blueos.cloud/docs/latest/development/extensions/)
- [BlueOS Extensions Repository metadata][blueos-extensions-repo]

## Local Lifecycle Smoke

Before running the BlueOS-style package smoke, use the repo-owned lightweight
demos when you only need companion/runtime evidence:

```bash
./scripts/demo-single-companion.sh
./scripts/demo-mesh-companions.sh
```

Those commands write JSON reports under `.artifacts` and do not require Docker,
BlueOS, Navigator hardware, Gazebo, SITL, SemOps, semstreams-ui, or
SemConnect/CS API. They prove the companion API/evidence shape, command-safety
posture, native SemOps readback adapter compatibility, and simple selected-state
mesh catch-up before package lifecycle enters the picture. They still run
through the SemLink Go module and the pinned SemStreams module version in
`go.mod`.

The local smoke runs the extension container with embedded SemStreams runtime
using the companion handoff profile, then verifies:

- `/api/health`
- `/register_service`
- `/api/evidence`

```bash
scripts/blueos-extension-smoke.sh
```

The smoke defaults to `configs/handoff/companion.env.example`. Override with
`SEMLINK_HANDOFF_PROFILE_FILE=/path/to/companion.env` to run a copied local
profile. Compose mounts that file as `/data/companion.env`, and the BlueOS-style
entrypoint sources it before translating environment values into runtime flags.
The evidence check asserts the SemLink companion evidence contract, configured
profile, handoff node ID, optional downstream posture, and fail-closed hardware
transmit posture.

## Required Inputs

Run package smokes from the SemLink checkout with these local inputs:

- Docker with Compose v2 access.
- A handoff profile file. The default is
  `configs/handoff/companion.env.example`; use a copied profile for local
  boats or alternate ports.
- A free host HTTP port for `SEMLINK_BLUEOS_HOST_PORT`, defaulting to `8081`.

The profile must keep `SEMLINK_HARDWARE_TRANSMIT_ENABLED=false` for this
handoff. The validator rejects `true`, and the smoke checks evidence for the
blocked hardware-transmit posture. In the default `hardware-readonly` command
mode, local command attempts are rejected before command-intent writes and
recorded as handoff-scope hardware block evidence.

## Run Commands

Preflight Compose interpolation without building the image:

```bash
docker compose -f compose.blueos.yml config
```

Run the full local BlueOS-style lifecycle smoke:

```bash
scripts/blueos-extension-smoke.sh
```

Run with an explicit profile and host port:

```bash
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
- `./scripts/demo-single-companion.sh`
- `./scripts/demo-mesh-companions.sh`
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
- `docker/blueos-extension/entrypoint.sh`: extension runtime flags with static
  UI serving disabled by default
- `compose.blueos.yml`: local lifecycle Compose target
- `blueos/extension/metadata.json`: Bazaar repository metadata skeleton
- `blueos/extension/README.md`: submission note and evidence boundary

## Evidence Boundary

This slice proves that SemLink exposes BlueOS-compatible service registration
and has an image/lifecycle shape that can be tested locally.

It does not publish to Docker Hub or the BlueOS Bazaar. It also does not access
Navigator hardware or enable hardware command transmit; those remain separate
OpenSpec tasks. SemOps, semstreams-ui, and SemConnect are downstream consumers
of the local evidence/API surface, not services the package must start for
readiness.

[blueos-extensions-repo]: https://raw.githubusercontent.com/bluerobotics/BlueOS-Extensions-Repository/master/README.md
