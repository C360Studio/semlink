# BlueOS Extension Packaging

Task 2.4 adds a BlueOS-style packaging skeleton and local lifecycle smoke.

Implementation:

- `/register_service` returns BlueOS sidebar registration metadata.
- `docker/blueos-extension/Dockerfile` builds a SemLink extension image with
  BlueOS labels for permissions, version, maintainer, links, type, and tags.
- `compose.blueos.yml` runs the extension image locally with embedded
  SemStreams runtime.
- `scripts/blueos-extension-smoke.sh` builds the image, starts it, and checks
  `/api/health` plus `/register_service`.
- `blueos/extension/metadata.json` is the Bazaar repository metadata skeleton.

Evidence boundary:

- No Navigator hardware is required.
- No Docker Hub or BlueOS Bazaar publish is performed.
- Hardware command transmit remains blocked.
