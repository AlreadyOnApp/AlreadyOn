# Android

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The Android phone client uses Kotlin, Compose and Media3. Phone/tablet and TV apps discover local HTTPS servers for 30 seconds, then offer manual server address entry. After accepting the server certificate automatically, they sign in with username and the account’s 6–12 digit TV PIN configured in WebUI Admin Settings. They share account profiles, guide browsing and playback.

Use [the app guide](../apps/android/README.md) to build. Keep `io.github.lynchbros.alreadyon` as the application identity. Signing keys and update credentials belong to the maintainer and must remain private.

## Boundaries

Management calls use authenticated sessions; media delivery uses scoped credentials. Profile and account permissions remain server-enforced. Persist only necessary session state and review storage, backup and logout behavior before release.

Native onboarding accepts the instance certificate on first connection without a fingerprint confirmation screen and validates subsequent HTTPS requests against that instance certificate. PIN sign-in requires a direct LAN connection.

## Validation

Check sign-in, profile selection/PIN behavior, channel changes, transitions, tracks, rotation, picture-in-picture and background recovery. Use synthetic fixtures; actual device validation is distinct from emulator or browser testing. Record device/codec limitations without publishing personal host or device details. To test mobile onboarding without changing an existing installation’s sessions, build debug with `-PandroidDebugSuffix=.pairingtest.debug`; release identity stays unchanged.

See [emulator setup](ANDROID_EMULATOR.md), [playback](PLAYBACK.md) and [API compatibility](../api/COMPATIBILITY.md).

The [Google TV target](GOOGLE_TV.md) shares this client infrastructure, with fixed TV presentation, remote navigation and local account PIN pairing.
