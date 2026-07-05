# SemLink BlueOS Extension

This directory is the Bazaar submission skeleton for the SemLink Companion
extension. The image metadata is carried as Docker labels in
`docker/blueos-extension/Dockerfile`; this file and `metadata.json` are the
repo-side artifacts that would be copied into the BlueOS Extensions Repository
when the image is published.

The first packaging slice is lifecycle-only:

- container starts with embedded SemStreams runtime
- `/api/health` responds
- `/register_service` returns the BlueOS sidebar registration
- hardware command transmit remains blocked

Navigator hardware access and read-only hardware smoke are tracked by the next
OpenSpec task.
