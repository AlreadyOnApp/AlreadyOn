# Build minimization

Production artifacts retain supported codecs, GPU drivers, Android architectures, TLS, authentication and recovery commands. Build/test tools stay in CI.

## Android

Phone and TV release builds use R8 code optimization and integrated resource shrinking. Debug builds retain their usual diagnostics. The application identity, version inputs and signing process are unchanged.

Comparable unsigned APKs with identical version/toolchain inputs:

| Artifact | Before | After | Reduction |
| --- | ---: | ---: | ---: |
| Phone | 10,912,462 bytes | 2,077,077 bytes | 81.0% |
| TV | 2,492,033 bytes | 2,078,457 bytes | 16.6% |

Local sizes exclude signing overhead. Both retain four native libraries and launch on an isolated emulator. Physical playback/update acceptance remains a release gate.

## Containers

The runtime removes Ubuntu's unused Pebble orchestrator, curl and vainfo. AlreadyOn uses its own entrypoint, TLS health command and hardware probing. A final scratch stage copies the cleaned filesystem, removing deleted upstream bytes from downloadable layers while retaining Ubuntu libraries, package metadata, CA certificates and time zones.

The server and local administration commands share one executable through filename aliases, avoiding duplicate Go/SQLite runtime copies. Standalone developer commands remain available. Password-reset regression tests exercise the packaged dispatch against temporary accounts.

Copyright notices and FFmpeg build/license metadata remain. Unused package documentation is removed. CI uses Chromium's headless shell and removes installation caches and unused package-manager shims.

Local candidate source `fd0f576` compared with the published alpha2 image:

| Measurement | Before | After |
| --- | ---: | ---: |
| Runtime compressed layers | 210,154,355 bytes | 186,492,160 bytes |
| Runtime packages | 185 | 164 |
| CI scanner image bytes | 1,869,779,456 | 1,391,099,392 |

Runtime scan counts fell from 3 High/42 Medium/7 Low/9 Unknown to 0 High/37 Medium/6 Low/1 Unknown. CI fell from 11 High to 8 High; vulnerable build-tool dependencies still require remediation. Source retains one Unknown finding. No Critical findings or secret matches were detected. These report-only results use Trivy 0.75.0; coverage limits remain.

## Evidence and publication

Exact artifact hashes and scan timestamps stay in private evidence. Media-stack and disposable-runtime checks are required before delivery.

Public inventory SBOMs are approved for release attachments. Generate them separately from private vulnerability/secret reports, bind them to image digests or APK checksums, and retain the same SBOM bytes when promoting tested artifacts. See [scanning](VULNERABILITY_SCANNING.md) and [release gates](CI_CD.md).
