# Google TV client

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The `:tv` Android target shares networking, sessions, profiles, guide, branding, quality, mute, audio and subtitles with the mobile `:app` target. TV uses a fixed landscape shell and Leanback launcher. It starts with a compact player beside program metadata above the guide. Selecting a channel opens fullscreen; Back returns to the compact guide without retuning.

In fullscreen, Right tunes the next higher available channel and Left the next lower. Up or Down reveals controls; subsequent Up/Down moves through controls, and Select activates the focused control. Menus and guide use D-pad navigation. Playback settings survive channel changes and presentation changes.

## Connect a TV

In the WebUI, open **Admin Settings → TV Sign-In**. Each signed-in account sets its own unique 6–12 digit numeric PIN, including leading zeros. PINs are stored as Argon2id digests and are never returned. Replacing/removing a PIN prevents new sign-ins; revoke an existing TV through **Signed-In Browsers**.

The server advertises `_alreadyon._tcp.local.` over mDNS on private-network interfaces. The TV scans for servers, shows name/IP, automatically accepts the instance certificate, and prompts for username and account TV PIN over validated HTTPS. TV offers manual HTTPS entry during its 30-second search; typing stops discovery. Mobile offers manual entry after 30 seconds. Passwords are not used for native-app sign-in.

TV pairing requires a direct private/loopback/link-local HTTPS peer and has per-peer and aggregate attempt limits. Disabled accounts and accounts requiring password changes cannot pair. Docker deployments require host networking for LAN discovery; see [deployment](../deploy/README.md). Guest Wi-Fi and firewalls can block mDNS; UDP 5353 must reach the LAN and the advertised HTTPS address must be reachable. Discovery failure does not stop server playback.

## Build and emulator validation

Build with `./gradlew :tv:assembleDebug` in `apps/android`. Install `tv/build/outputs/apk/debug/tv-debug.apk`; its debug identity ends in `.tv.debug`.

Create a Google TV AVD using the 4K hardware profile (3840×2160). Run `scripts/android/test-tv.py --serial emulator-5554 --sdk /path/to/Android/Sdk` with JDK 17 and FFmpeg available. It clears only that emulator's TV debug app, uses temporary HTTPS/database/media fixtures, and bridges the fixture onto the virtual LAN for discovery and immediate manual-entry sign-in/navigation checks. Add `--interactive` to keep a synthetic server available for manual testing until Enter is pressed; the helper prints its emulator address and synthetic credentials. Real home-LAN discovery and hardware 4K/HDR decoding need physical-TV validation; the 4K emulator display does not prove those capabilities.
