# Android phone client

[![AlreadyOn](../../web/public/branding/header-doc.svg)](../../README.md)

A Kotlin/Compose client with Media3 playback for an AlreadyOn installation. Enter your own HTTPS server address; no public demo server or shared credentials are supplied.

## Build

Use JDK 17, the committed Gradle wrapper and Android SDK components declared in the build files. The current project compiles against SDK 36 and supports API 26 or later.

From this directory run `./gradlew :app:assembleDebug`. Install the generated debug APK on a selected emulator or test device. Keep SDK paths in ignored `local.properties`; never commit signing keys, credentials or generated APKs.

The application ID is `io.github.lynchbros.alreadyon`. Release signing and distribution belong to the maintainer. Native TV and automatic signed-update work are separate from the phone scaffold.

In fullscreen landscape, swipe up to the next higher available channel or down to the next lower one. Each deliberate swipe changes one channel; the lineup ends do not wrap. Fullscreen, orientation, mute and quality stay unchanged. Subtitle Off or the selected language carries across channels when a matching track is available.

## Test

Use an isolated server and check login/logout, profile permissions, guide navigation, tuning, program boundaries, track selection, rotation and background recovery. Verify physical-device playback independently of emulator behavior.

TLS verification is enabled by default. The optional per-server bypass is under security review; prefer a trusted certificate. See [Android architecture](../../docs/ANDROID.md), [emulator setup](../../docs/ANDROID_EMULATOR.md) and [OpenAPI](../../api/openapi.yaml).
