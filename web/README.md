# Web client

[![AlreadyOn](public/branding/header-doc.svg)](../README.md)

The browser client uses React, TypeScript and Vite. It provides the program guide, player, profiles and owner administration.

Run `npm ci` in this directory to install locked dependencies. `npm run lint` performs frontend linting; `npm run build` type-checks and builds assets. Use `make preview` from the repository root for a coordinated backend and HTTPS browser preview.

Keep server-authoritative permissions, timing and playback credentials intact. Cancel obsolete requests when changing channels. Handle loading, empty and failure states explicitly; preserve keyboard and touch access.

Use [OpenAPI](../api/openapi.yaml) for server contracts. [Mobile UX](../docs/WEB_MOBILE_UX.md) and [playback](../docs/PLAYBACK.md) describe interface boundaries. Browser tests live under `tests/e2e/` and use isolated synthetic instances.

Sign-in recovery guidance starts with another owner resetting the password, followed by the Docker reset tool. Profile and account back navigation sits below the branded header; server settings keep their header navigation.
