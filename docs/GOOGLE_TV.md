# Google TV client

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The `:tv` target shares networking, profiles, guide, branding, quality, mute, audio and subtitles with mobile `:app`. TV uses fixed landscape and a Leanback launcher. It starts with a compact player above the guide. Channel selection opens fullscreen; Back restores the guide without retuning.

The centered chooser says **Select Profile**. Guide Up/Down reaches offscreen channels; offscreen programs skip rendering. Channel selection opens fullscreen and unmutes.

Fullscreen Left/Right tunes channels with controls hidden, or navigates visible controls. Up/Down reveals controls, then navigates them; Select activates the focused control. Playback settings survive tuning and presentation changes.

## Connect

First-run WebUI setup asks for **Server Name**, username and password. Owners can rename it under **Server Settings → Server → Server Name**. Discovery reflects changes without restarting. Existing servers default to `AlreadyOn`. Names are 1–64 characters after trimming, without control characters.

Each account manages its unique 6–12 digit numeric PIN under **Admin Settings → TV Pairing Pin**, below the password fields. Leading zeros are supported. Use **Save Changes** to set it and **Remove Pin** to remove it. PINs are hashed and never returned. Replacing/removing a PIN prevents new sign-ins; revoke existing devices through **Signed-In Browsers**.

Both apps scan private IPv4 HTTPS port 19900 with 32 concurrent probes. The recent server appears first and its name refreshes when found. Responses must identify AlreadyOn protocol 1. Larger networks search the device's /24 first, considering at most /16 worth of addresses. After 30 seconds, requests are cancelled and manual entry appears with `https://` prefilled. Bare addresses assume HTTPS and port 19900; explicit ports remain. Server selection cancels scanning; Rescan restarts it. Large or IPv6-only networks may require manual entry.

Clients accept the selected instance certificate automatically, then send username/PIN over validated HTTPS. Pairing requires a direct LAN peer, limits attempts and rejects disabled accounts or required password changes. Standard Docker bridge networking with published TCP 19900 works; no multicast or extra discovery port is needed. Guest isolation/firewalls can prevent discovery. See [deployment](../deploy/README.md).

## Validate

Build `./gradlew :tv:assembleDebug` in `apps/android`. Install `tv/build/outputs/apk/debug/tv-debug.apk` (debug identity ends in `.tv.debug`).

Use a 3840×2160 Google TV AVD. Run `scripts/android/test-tv.py --serial emulator-5554 --sdk /path/to/Android/Sdk` with JDK 17 and FFmpeg. It clears only the selected emulator's debug app and uses disposable HTTPS/database/media fixtures on port 19900. `--interactive` keeps fixtures available until Enter. Physical-TV validation is still needed for LAN discovery and hardware 4K/HDR decoding.
