# Android

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The Android phone client uses Kotlin, Compose and Media3. It connects to an owner-specified HTTPS installation and supports sign-in, account profiles, guide browsing and playback. Native TV work is a separate roadmap item.

Use [the app guide](../apps/android/README.md) to build. Keep `io.github.lynchbros.alreadyon` as the application identity. Signing keys and update credentials belong to the maintainer and must remain private.

## Boundaries

Management calls use authenticated sessions; media delivery uses scoped credentials. Profile and account permissions remain server-enforced. Persist only necessary session state and review storage, backup and logout behavior before release.

Certificate verification is enabled by default. The current explicit per-server bypass disables certificate-chain validation; it is a security review item, not equivalent to trusting a local CA. Prefer a valid certificate or installing the instance CA.

## Validation

Check sign-in, profile selection/PIN behavior, channel changes, transitions, tracks, rotation, picture-in-picture and background recovery. Use synthetic fixtures; actual device validation is distinct from emulator or browser testing. Record device/codec limitations without publishing personal host or device details.

See [emulator setup](ANDROID_EMULATOR.md), [playback](PLAYBACK.md) and [API compatibility](../api/COMPATIBILITY.md).

The [Google TV target](GOOGLE_TV.md) shares this client infrastructure, with fixed TV presentation, remote navigation and local account PIN pairing.
