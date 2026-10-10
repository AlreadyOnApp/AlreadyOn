# Validation and delivery

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The maintainer controls release gates. Checkout ownership and deployment credentials belong in ignored `.private/AGENTS.md`.

## Development

Commit changes and preview them. Run `make lint`, `python3 scripts/ci/check-documentation.py` and `python3 scripts/ci/check-syntax.py`. Lint includes public-guide length and link checks. Update contracts when behavior changes.

Documentation checks include the latest commit and pending edits by default. CI supplies `CI_BASE` for the complete push/PR diff.

## Approved release validation

`make release-check` requires a clean checkout and performs:

1. Static, documentation, syntax and CI policy checks.
2. Production-image build (`make image`).
3. Go regressions using patched production FFmpeg (`make test-go-runtime`).
4. Full synthetic browser suite (`make test-e2e`).
5. Disposable packaged-runtime checks (`make test-container`).

Go regressions copy tracked source and non-ignored additions into temporary storage, excluding local databases, credentials and media. Packages run sequentially; multicore accounting checks do not assume minimum throughput from shared runners.

Resolve failures before promotion. Record the commit, image ID/digest and actual results. Live checks are non-destructive; synthetic suites use isolated data.

Before image builds, `scripts/local-bases.py` checks recipe, parent and age. Missing, invalid or seven-day-old bases rebuild. CI refresh pulls remote toolchains separately while retaining its pinned local runtime parent. `make refresh-local-bases` explicitly forces refresh; there is no schedule.

## Delivery

On approval, deploy the exact tested image and verify revision, health, service count, mounts and relevant playback. `scripts/deploy-production.sh <ssh-host> [compose-directory]` transfers it with the updater paused. Preserve live data.

Publish approved images to the project registry: date/nightly first; later approved promotions assign latest/stable to the same digest. Update issues after verified delivery.

Hosted image builds and promotions remain manual. See [retention](IMAGE_RETENTION.md) and [deployment](../deploy/README.md).

## Public automation

Source Validation uses read-only credentials. Its Go job builds the production media base locally with shared caching and no publication credentials. Ubuntu's stock FFmpeg 8.0.x ignores initial burst pacing; the production recipe includes the fix.

CodeQL analyzes Go/TypeScript. Pages publishes documentation from main. [Android releases](ANDROID_RELEASES.md) publish candidates for Android changes on main; stable promotion is explicit and reuses tested assets.

New images use `ghcr.io/alreadyonapp/alreadyon`. Go module and Android package identities stay compatible. **Refresh Base Images** rebuilds shared bases on the selected branch; CI bases pin their runtime digest. Hosted application builds resolve both digests without rebuilding during promotion.
