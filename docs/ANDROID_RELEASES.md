# Android APK releases

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

## Approved release policy

Ordinary `dev` and `test` builds validate source without creating a GitHub release or Git tag. Releases follow an explicitly selected full commit SHA. Select the latest approved `dev` commit for promotion; cherry-pick the selected change set into the destination branch rather than including unrelated work. Record both the selected source SHA and destination SHA because cherry-picking changes commit identity.

| Promotion | Release tag | Publication |
| --- | --- | --- |
| Selected `dev` commit into `test` | `YYYY-MM-DD-rcN` | Signed release candidate; GitHub prerelease |
| Tested release from `test` into `main` | `YYYY-MM-DD-beta` | Beta; GitHub prerelease |
| Tested beta approved as stable | `YYYY-MM-DD-stable` | Stable; marked latest |

The RC date is the promotion date in America/New_York. RC numbering starts at `rc1` and advances past occupied tags and drafts for that date. The beta date is its publication date in America/New_York, with one unnumbered beta per date; stable retains the beta date. For example, `2026-10-09-rc3` can become `2026-10-10-beta`, then `2026-10-10-stable`.

Each beta changelog includes release-note files changed since the latest published Android release from an earlier date. Stable promotion retains the tested APK bytes, signatures, checksums, source provenance and changelog; do not rebuild or re-sign. Cherry-picked source must correspond to the tested release; validate that mapping before publication. Published assets are immutable: a changed same-day candidate requires deleting that date’s beta release and tag before recreating the same tag. Keep all earlier-date releases and tags intact. Reruns for the same source verify existing assets; failed uploads remain recoverable drafts.

Each release attaches `alreadyon-android.apk`, `alreadyon-tv.apk`, `SHA256SUMS` and `android-release.json`. The RC’s Android `versionName` and increasing `versionCode` remain embedded during promotion; renaming a GitHub release does not change installed APK versions. Changing a release title alone does not rename its Git tag.

## Automation

The workflow publishes a signed `YYYY-MM-DD-beta` from `main` and supports direct stable promotion without rebuilding the selected APKs. A source/tag mismatch fails closed; same-day replacement requires explicit release and tag deletion before rerunning.

## Signing and scope

Use the permanent credentials described in [Android signing](ANDROID_SIGNING.md). The mobile and [TV](GOOGLE_TV.md) APKs share signing and release provenance. Debug apps use separate installation identities. Physical phone/tablet and TV validation remains required. [Release notes](RELEASE_NOTES.md) describe user-visible changes. Server deployment and Docker publication follow [delivery](CI_CD.md).
