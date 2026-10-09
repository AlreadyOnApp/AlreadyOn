# Android APK releases

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

**Publish Android Release** publishes a signed phone/tablet APK when a push to `main` changes Android source, Gradle inputs, the wrapper or the release workflow/helper. Ordinary web/server changes do not trigger Android publication. Branch promotion to `main` remains maintainer-controlled. The workflow does not deploy a server or publish container images.

## Candidates and stable promotion

Candidates use `YYYY-MM-DD-rcN` in America/New_York, starting at `rc1` and incrementing past occupied tags and drafts. Candidates are GitHub prereleases and do not replace the latest stable release. A completed rerun for the same source commit verifies its existing assets and does not create another candidate. A failed upload leaves a recoverable draft.

Each release attaches `alreadyon-android.apk`, `SHA256SUMS` and `android-release.json`, recording the source SHA, Android version and APK checksum. Uploads are verified before the draft becomes public. The APK uses the candidate tag as `versionName` and seconds since 2020 as an increasing `versionCode`; developer builds retain their existing version defaults.

After device testing, run **Publish Android Release** on `main`, choose `stable`, and enter the exact tested candidate tag. This explicit dispatch approves promotion. The workflow verifies its source is on main and checks downloaded asset hashes, then creates `YYYY-MM-DD-stable` at the candidate's commit with identical assets. It never rebuilds or re-signs. The embedded APK version remains `YYYY-MM-DD-rcN`, so devices already running that candidate need no update. The stable release is marked latest. Existing published assets cannot be overwritten; only one distinct stable APK may use a given date.

You can dispatch `candidate` on main to retry or publish the current source. Dispatches on other branches do not publish. GitHub concurrency serializes the release writer; during a burst, GitHub may replace a pending run with a newer pending run. Maintainers should promote completed changes deliberately rather than rely on every intermediate push producing an APK.

## Signing and platform scope

Configure the four repository secrets in [Android signing](ANDROID_SIGNING.md) before publishing. Missing secrets fail clearly. The release app uses `io.github.lynchbros.alreadyon`; the developer `.debug` app stays separate.

This workflow publishes the phone client, with tablet validation pending. Google TV requires a dedicated remote/focus UI and launcher; it is subsequent product work. Future mobile and TV targets can share playback/networking code and signing, with both APKs attached to one release. Separate builds need not require separate package names or Play listings.
