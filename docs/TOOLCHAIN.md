# Toolchain

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The production baseline is defined by [the runtime Dockerfile](../deploy/Dockerfile), [build base](../deploy/base.Dockerfile), [CI base](../deploy/ci-base.Dockerfile), `server/go.mod` and `web/package.json`. Current build recipes use Go 1.27, Node 24 and Ubuntu 26.04.

Run `make check-toolchain` for consistency checks. `make check-ubuntu-lts` additionally requires network access to inspect the upstream Ubuntu baseline. A native workstation may differ; validate release behavior in the production image.

Use `npm ci` with the committed lockfile and normal Go module tooling. Do not replace dependencies or toolchain versions opportunistically during unrelated work. Explain upgrades and run affected checks.

SQLite schema changes use versioned migrations. Preserve existing data and test upgrades from supported older schemas. Avoid destructive migration shortcuts.

Android uses its committed Gradle wrapper and the versions in its build files; see [Android](ANDROID.md). FFmpeg build configuration and patches are maintained under `deploy/` and must retain their licensing and runtime checks.
