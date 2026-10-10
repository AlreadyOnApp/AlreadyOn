# Android APK releases

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

**Publish Android Release** publishes a signed mobile and TV APKs when a push to `main` changes Android source, Gradle inputs, the wrapper or the release workflow/helper. Ordinary web/server changes do not trigger Android publication. Branch promotion to `main` remains maintainer-controlled. The workflow does not deploy a server or publish container images.

## Candidates and stable promotion

Candidates use `YYYY-MM-DD-betaN` in America/New_York, starting at `beta1` and incrementing past occupied tags and drafts. Legacy RC releases remain available and can still be verified or promoted. Candidates are GitHub prereleases and do not replace the latest stable release. A completed rerun for the same source commit verifies its existing assets and does not create another candidate. A failed upload leaves a recoverable draft.

Each release attaches `alreadyon-android.apk`, `alreadyon-tv.apk`, `SHA256SUMS` and `android-release.json`, recording the source SHA, Android version and APK checksums. Uploads are verified before the draft becomes public. The APK uses the candidate tag as `versionName` and seconds since 2020 as an increasing `versionCode`; developer builds retain their existing version defaults.

After device testing, run **Publish Android Release** on `main`, choose `stable`, and enter the exact tested candidate tag. This explicit dispatch approves promotion. The workflow verifies its source is on main and checks downloaded asset hashes, then creates `YYYY-MM-DD-stable` at the candidate's commit with identical assets. It never rebuilds or re-signs. The embedded APK version remains `YYYY-MM-DD-betaN`, so devices already running that candidate need no update. The stable release is marked latest. Existing published assets cannot be overwritten; only one distinct stable APK may use a given date.

Dispatch `candidate` on main to retry publication. Other branches cannot publish. Release writers are serialized; promote completed changes deliberately.

Release descriptions use reviewed [user-facing changelog entries](RELEASE_NOTES.md), grouped into Features, Improvements and Bug fixes. Candidates include changes since the last reachable stable release; stable promotion preserves the tested candidate’s notes. Build details are collapsed below download links.

## Signing and platform scope

Configure the four repository secrets in [Android signing](ANDROID_SIGNING.md) before publishing. Missing secrets fail clearly. The release app uses `io.github.lynchbros.alreadyon`; the developer `.debug` app stays separate.

The mobile `:app` and [Google TV `:tv`](GOOGLE_TV.md) targets share playback/networking code and signing, with both APKs attached to one release. Tablet and physical-TV validation remain required. Both release APKs retain the established package name; debug targets use separate installation identities.
