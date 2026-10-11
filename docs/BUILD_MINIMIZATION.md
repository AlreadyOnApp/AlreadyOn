# Build minimization

Production artifacts retain supported playback, hardware acceleration, TLS, authentication and recovery commands. Build/test tools stay in the CI image. Minimization does not authorize dropping codecs, GPU drivers or supported Android architectures.

## Android

Phone and TV release builds use R8 code optimization and integrated resource shrinking. Debug builds retain their usual diagnostics. The application identity, version inputs and signing process are unchanged.

Comparable unsigned APKs, built with the same source/version/toolchain before and after shrinking:

| Artifact | Before | After | Reduction |
| --- | ---: | ---: | ---: |
| Phone | 10,912,462 bytes | 2,077,077 bytes | 81.0% |
| TV | 2,492,033 bytes | 2,078,457 bytes | 16.6% |

These are local measurements, not published release sizes. Alignment/signing adds a small overhead. Both retain four packaged native libraries; no ABI is excluded. Shrunk release APKs launch on an isolated emulator. Physical playback/update acceptance remains a release gate.

## Containers

The runtime removes Ubuntu's unused Pebble orchestrator, curl and vainfo. AlreadyOn uses its own entrypoint, TLS health command and hardware probing. The cleaned filesystem is copied into a final scratch stage: removed upstream files are absent from downloadable layers, while Ubuntu's libraries, package database, CA certificates and time zones remain available.

The server and local administration commands share one executable through filename aliases, avoiding duplicate Go/SQLite runtime copies. Standalone developer commands remain available. Password-reset regression tests exercise the packaged dispatch against temporary accounts.

Package copyright notices and FFmpeg build/license metadata remain. Unused manual pages and other package documentation are removed. CI installs Chromium's headless shell, matching the automated tests, and discards its npm installation cache.

## Evidence and publication

Record exact image/archive hashes, source revision, sizes and scan database timestamps in private evidence. Recheck the complete media-stack smoke test and disposable packaged-runtime checks before delivery; compare vulnerability counts against the baseline rather than assuming a smaller image is safe.

Public inventory SBOMs are approved for release attachments. Generate them separately from private vulnerability/secret reports, bind them to image digests or APK checksums, and retain the same SBOM bytes when promoting tested artifacts. See [scanning](VULNERABILITY_SCANNING.md) and [release gates](CI_CD.md).
