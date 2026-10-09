# Android emulator testing

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

Install Android Studio or the command-line SDK and a compatible emulator image. Confirm hardware virtualization is available on the host. Store local SDK paths and device identifiers outside Git.

Build the phone client using [its instructions](../apps/android/README.md). Select the intended emulator explicitly when installing or collecting logs, especially if a physical device is also connected.

An emulator's loopback is separate from the host's loopback. Use the emulator's documented host-network route and a certificate valid for the selected server name. Do not turn off TLS globally to work around routing or certificate problems.

Use synthetic media and a disposable server for destructive tests. Validate authentication, guide navigation, tuning, transitions, screen rotation and background recovery. Sanitize logs before sharing them: tokens, cookies, usernames, media names and local addresses may be present.

Emulator success does not establish physical-device codec, HDR, hardware-decoder or television behavior. Record those checks separately.
