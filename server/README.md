# Server

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The Go server provides authentication, catalog scanning, scheduling, administration and scoped playback delivery. SQLite persists installation state; FFmpeg and FFprobe provide media inspection and processing.

Run commands from this directory: `go test ./...` for unit tests and `go build ./cmd/server` for a native build. The repository's `make preview` starts the coordinated browser/server development preview.

Keep migrations versioned and preserve existing databases. Confine filesystem access to configured roots and bound background work, subprocesses and request lifetimes.

See [the project brief](../docs/PROJECT_BRIEF.md), [playback](../docs/PLAYBACK.md) and [OpenAPI](../api/openapi.yaml). Runtime images are built from `deploy/Dockerfile`; development account provisioning is excluded from them.
