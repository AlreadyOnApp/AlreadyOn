# Android APK releases

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

## Release lifecycle

Use one checkout: `python3 scripts/work.py start build` refreshes local `build` from `dev` without losing unique commits. Test at <https://localhost:5173>; then `python3 scripts/work.py publish [YYYY-MM-DD]` pushes the date branch/tag and opens its dev PR. The date is local test completion in America/New_York.

| Promotion | Release tag | Publication |
| --- | --- | --- |
| Merged date PR into `dev` | `YYYY-MM-DD-alphaN` | Fresh signed APKs and image tagged `nightly` plus `YYYY-MM-DD` |
| Merge tested date commit from `dev` to `test` | `YYYY-MM-DD-RCN` | Retag image `latest`; publish/copy tested APK bytes |
| Merge current `test` release into `main` | `YYYY-MM-DD-Stable` | Retag image `stable`; publish the latest RC APK bytes |

Start at alpha1; each dev hotfix increments alpha until RC1. RC1 reuses alpha APK bytes. Later dev hotfixes build a draft for the next RC; test promotion publishes it. RC changelogs include only marked app notes since the prior stage. Stable reuses the latest RC APKs and locks the date.

After dev e2e passes, the plain date tag advances to that merge. Alpha changelogs include marked Android/TV notes since the latest earlier-date Android release, excluding server-only notes. Promotions reuse the tested image digest and signed APKs; no rebuild/re-sign occurs. Earlier dates remain intact.

Each release attaches both APKs, `SHA256SUMS` and `android-release.json`. Promotions retain the APK `versionName`/`versionCode`; titles do not rename tags.

Approved APK-only alpha: dispatch **Android Alpha, RC and Stable** on `dev` with `operation=candidate`, release date and exact SHA. No image publication/deployment.

## Automation and gates

Date PRs run quality, CodeQL and full Go/browser e2e. Dev merge validation publishes the date image and alpha APKs, advances the date tag, then removes the merged date branch. `python3 scripts/promote.py test YYYY-MM-DD` verifies image/alpha APK provenance before merging to test; `python3 scripts/promote.py main` verifies an RC before merging to main. The helpers create exact-tree merge commits when branches have diverged, reuse tested artifacts and skip duplicate e2e. Conflicts stop for resolution on dev. If branch protection rejects a push, stop; never bypass it.

## Signing and scope

Phone/tablet and [TV](GOOGLE_TV.md) APKs share permanent signing credentials and provenance. Validate debug identities and physical devices. See [signing](ANDROID_SIGNING.md), [release notes](RELEASE_NOTES.md) and [delivery](CI_CD.md).

## Application identity

The organization-owned application ID differs from older previews, which cannot update in place. Uninstall the earlier preview before installing this version; app-local settings are not transferred.
