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
and verifies:

- `/api/health`
- `/register_service`

```bash
scripts/blueos-extension-smoke.sh
```

The smoke expects a sibling SemStreams checkout. Override with
`SEMSTREAMS_ROOT=/path/to/semstreams` when needed.

Use `configs/handoff/companion.env.example` as the companion handoff profile
for node identity, MAVLink UDP input, local runtime, mesh peers, and downstream
consumer posture. The current smoke consumes the already-wired environment
fields; parser validation and full profile wiring are tracked by OpenSpec
change `companion-deployment-handoff`.

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
