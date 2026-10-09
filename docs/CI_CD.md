# Validation and delivery

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The maintainer controls release gates. Local checkout ownership, private targets and publication credentials belong in ignored `.private/AGENTS.md`.

## Development

Commit approved changes and preview them. Run: `make lint`, `python3 scripts/ci/check-documentation.py` and `python3 scripts/ci/check-syntax.py`. `make lint` also checks public guide length and local links. Update public contracts when behavior changes.

## Approved release validation

`make release-check` requires a clean checkout and performs:

1. Production-image build (`make image`).
2. Full end-to-end tests against an isolated synthetic server (`make test-e2e`).
3. Disposable packaged-runtime tests (`make test-container`).

Resolve failures and repeat affected gates before promotion. Record the commit, exact image ID/digest and actual test results. Live checks are non-destructive; synthetic tests use isolated data.

Before each local image build, `scripts/local-bases.py` checks the base recipe, parent and timestamp. Missing, invalid or more-than-seven-day-old bases rebuild. There is no scheduled local refresh. `make refresh-local-bases` requests an explicit refresh.

## Deploy and publish

Only on approval, deploy the exact validated image and verify revision, health, service count, configuration mounts and relevant playback behavior. Manual transfer uses `scripts/deploy-production.sh <ssh-host> [compose-directory]`; with existing configuration and a paused updater. Never create duplicate live instances or reset live data.

Publish approved images to the project registry. Development releases use the date and `nightly`; later approved promotions assign `latest` and `stable` to the same recorded digest. Branch promotion and issue completion happen only after the corresponding approved gates.

Hosted workflows preserve approval gates. Consult [image retention](IMAGE_RETENTION.md) and [deployment](../deploy/README.md). Keep private records outside Git.

## Public automation

Pushes and pull requests validate source with read-only credentials; maintainers merge. CodeQL scans Go and TypeScript. Pages publishes a documentation landing page from `main`. Container image builds and promotions remain manually dispatched after approval. [Android APK releases](ANDROID_RELEASES.md) publish candidates on Android changes pushed to `main`; stable APK promotion is an explicit manual dispatch that reuses tested candidate assets.

The Go module and Android application ID retain their established identities for compatibility. New image references use `ghcr.io/alreadyonapp/alreadyon`; migrate existing deployments only after a new image is available.

Hosted Go tests install FFmpeg for synthetic checks. CodeQL builds Go explicitly. Registry image names use lowercase. Document workflow changes.

To refresh published bases, dispatch **Refresh Base Images** on the approved branch. It rebuilds `ghcr.io/alreadyonapp/alreadyon-base:base-runtime` and `:base-ci`. The CI base pins the runtime digest; hosted application builds resolve and pass both base digests to the Dockerfile. Dockerfile defaults and local registry fallback use the same organization.
