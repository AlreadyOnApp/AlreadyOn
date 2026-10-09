# Container deployment

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

Use [docker-compose.example.yml](docker-compose.example.yml) as a starting point. Review every mount, port and identity setting before starting it. Published project images live at `ghcr.io/alreadyonapp/alreadyon`; availability depends on registry access.

`stable` is the approved main release, `latest` the approved test release and `nightly` the development release. Pin a digest when reproducibility matters. Do not assume a newer mutable tag has passed your deployment's validation.

## Configure

Persist writable application configuration and database storage. Mount library media read-only at `/data/media`. Configure the actual media root in the application. Optional GPU access must match the host and container permissions; software processing remains a separate path.

The runtime uses HTTPS port 19900 for UI, API and media. Port 19902 is reserved for HTTP-01 challenges when needed. See [TLS](../docs/TLS.md) for local trust and public certificates. Create the owner through first-run setup.

The example documents supported identity/timezone/port settings. Keep tokens, custom domains and private host configuration in local files excluded from Git. Do not add undocumented environment variables.

## Validate and update

Inspect Compose configuration, then start the existing service. Verify image digest/revision, health, UI, mounts and relevant codec behavior. Preserve live data during updates. Use one service per installation; do not leave duplicate instances running accidentally.

For development builds, `make image` builds locally and `make test-container` validates a disposable runtime. Full release checks and maintainer-controlled publication are described in [delivery](../docs/CI_CD.md). Automatic update scripts are optional; review their timing and target before enabling them.

Review [LICENSE](../LICENSE) and third-party codec/dependency obligations before redistributing images.
