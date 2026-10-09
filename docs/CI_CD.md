# Validation and delivery

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The maintainer controls release gates. Local checkout ownership, private targets and publication credentials belong in ignored `.private/AGENTS.md`.

## Development

Make focused changes, commit and push according to the active branch instructions, then preview and review them. Run relevant static checks: `make lint`, `python3 scripts/ci/check-documentation.py` and `python3 scripts/ci/check-syntax.py`. `make lint` also checks public guide length and local links. Update public contracts when behavior changes.

## Approved release validation

`make release-check` requires a clean checkout and performs:

1. Production-image build (`make image`).
2. Full end-to-end tests against an isolated synthetic server (`make test-e2e`).
3. Disposable packaged-runtime tests (`make test-container`).

Resolve failures and repeat affected gates before promotion. Record the commit, exact image ID/digest and actual test results. Live deployment verification uses non-destructive checks; synthetic tests must never mutate live data.

Before each local image build, `scripts/local-bases.py` checks the base recipe, parent and timestamp. Missing, invalid or more-than-seven-day-old bases rebuild. There is no scheduled local refresh. `make refresh-local-bases` requests an explicit refresh.

## Deploy and publish

Only on approval, deploy the exact validated image and verify revision, health, service count, configuration mounts and relevant playback behavior. Generic manual transfer is available through `scripts/deploy-production.sh <ssh-host> [compose-directory]`; it requires existing host configuration and a paused automatic updater. Never create duplicate live instances or reset live data.

Publish approved images to the project registry. Development releases use the date and `nightly`; later approved promotions assign `latest` and `stable` to the same recorded digest. Branch promotion and issue completion happen only after the corresponding approved gates.

Hosted workflows are optional execution infrastructure, not permission to skip checks. Consult [image retention](IMAGE_RETENTION.md) and [deployment](../deploy/README.md). Keep build transcripts and private operational records outside Git.

## Public automation

Pushes and pull requests validate source with read-only credentials; maintainers merge manually. CodeQL scans Go and TypeScript. Pages publishes a minimal documentation landing page from `main`. Image builds and promotions remain manually dispatched after approval.

The Go module and Android application ID retain their established identities for compatibility. New image references use `ghcr.io/alreadyonapp/alreadyon`; migrate existing deployments only after a new image is available.
