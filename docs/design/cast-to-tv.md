# Casting design

[![AlreadyOn](../../web/public/branding/header-doc.svg)](../../README.md)

This is a design proposal, not a claim that every receiver platform is implemented.

A sender chooses an airing; a receiver plays an authorized stream. Receiver credentials must be short-lived and scoped to playback. Never send an owner's management key, browser session cookie or unrelated filesystem path to a television.

The sender and receiver should recover from disconnects, expiry and program boundaries without duplicate sessions or unbounded retries. Capability negotiation determines direct, remux or transcode delivery. Remote navigation and profile permissions remain explicit design work.

Web casting SDK integration, a receiver application and Safari/AirPlay behavior have different platform constraints. Validate them separately on physical devices. Plan pairing, logout, token revocation and failure states before enabling a public receiver.

Keep implementation proposals small and review security boundaries against [playback](../PLAYBACK.md) and [OpenAPI](../../api/openapi.yaml).
