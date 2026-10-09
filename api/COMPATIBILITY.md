# API compatibility

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

[OpenAPI](openapi.yaml) is the source of truth for endpoint paths, schemas and errors. The API is pre-1.0; consumers should tolerate additive fields and handle documented failures. Breaking changes require an explicit review and migration notes.

## Authentication

Browser sessions use cookies. Management API keys carry owner-level authority. Playback credentials are scoped to media delivery and must not be treated as management keys. Enforce account/profile ownership on every protected operation.

TV pairing adds local HTTPS discovery/trust metadata, PIN login and per-account PIN management. PIN login creates the same account session used by other clients; it does not bypass profile ownership, disabled accounts, password-change restrictions or stream limits. PINs must be unique within an instance. See [Google TV](../docs/GOOGLE_TV.md).

## Playback

Use the tune endpoint to obtain an authorized session, then follow playback status, lease and timing fields. Honor retry hints and server time; do not construct media URLs or assume codec support. Expired credentials require a fresh authorized flow.

## Configuration

A per-channel commercial setting of `enabled: null` inherits the server default. An empty folder selection means all configured commercial folders. Clients must preserve these distinctions rather than substituting convenient defaults.

Server details expose authenticated installation identity and version information. Removed or deprecated routes must not silently return unrelated data. Check OpenAPI before depending on historical endpoints.

## Changes

Update the schema, implementation, consumer and focused compatibility tests together. Record user-facing changes in the relevant guide; keep private build transcripts outside the repository. Never copy real API keys into examples.

Swagger uses a short overview; operation-level role declarations remain in OpenAPI for clients and tooling.

Channel filters accept `excludedGenres` (exclusions win) and `requireKnownYear`. These optional fields default to no exclusions and allowing unknown years. Restricted built-in defaults enable known-year filtering.

Native account PIN sign-in now requires both `username` and `pin` in `POST /api/auth/tv-login`. A PIN supplied with another account’s username is rejected. Phone/tablet and TV use this shared LAN-only flow after discovery or a 30-second manual-address fallback.
