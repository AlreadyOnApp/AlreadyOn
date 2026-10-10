# Android APK releases

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

## Release lifecycle

Use one checkout: `python3 scripts/work.py start build` refreshes local `build` from `dev` without losing unique commits. Test at <https://localhost:5173>; then `python3 scripts/work.py publish [YYYY-MM-DD]` pushes the date branch/tag and opens its dev PR. The date is local test completion in America/New_York.

| Promotion | Release tag | Publication |
| --- | --- | --- |
| Merged date PR into `dev` | `YYYY-MM-DD-alphaN` | Fresh signed APKs and image tagged `nightly` plus `YYYY-MM-DD` |
| Fast-forward tested date commit from `dev` to `test` | `YYYY-MM-DD-RCN` | Retag image `latest`; publish/copy tested APK bytes |
| Fast-forward current `test` to `main` | `YYYY-MM-DD-Stable` | Retag image `stable`; publish the latest RC APK bytes |

Start at alpha1; each dev hotfix increments alpha until RC1. RC1 reuses alpha APK bytes. Later dev hotfixes build a draft for the next RC; test promotion publishes it. RC changelogs include only marked app notes since the prior stage. Stable reuses the latest RC APKs and locks the date.

After dev e2e passes, the plain date tag advances to that merge. Alpha changelogs include marked Android/TV notes since the latest earlier-date Android release, excluding server-only notes. Promotions reuse the tested image digest and signed APKs; no rebuild/re-sign occurs. Earlier dates remain intact.

Each release attaches both APKs, `SHA256SUMS` and `android-release.json`. Promotions retain the APK `versionName`/`versionCode`; changing a release title does not rename its tag.

## Automation and gates

Dev PRs run quality, CodeQL and full Go/browser e2e. Dev merges rerun release validation, publish `nightly`/date tags and alpha APKs, then delete the merged date branch. The fast-forward-only commands `python3 scripts/promote.py test YYYY-MM-DD` and `python3 scripts/promote.py main` require tested image/APK provenance and do not rerun e2e. Stop if branch protection rejects a promotion; never bypass it.

## Signing and scope

Phone/tablet and [TV](GOOGLE_TV.md) APKs share permanent signing credentials and provenance. Debug apps use separate identities; validate physical devices. See [signing](ANDROID_SIGNING.md), [release notes](RELEASE_NOTES.md) and [delivery](CI_CD.md).
