# Images and retention

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

Registry tags identify approved image digests; they are separate from GitHub release tags:

- `nightly` and `YYYY-MM-DD`: development releases.
- `latest`: an approved test promotion.
- `stable`: an approved main promotion.

Ordinary dev/test builds create no GitHub releases or Git tags. Approved releases identify a selected source SHA: test promotion creates `YYYY-MM-DD-rcN`, main promotion uses the same date with `-beta`, and later stable approval uses `-stable`. Only RC tags increment; see [release policy](ANDROID_RELEASES.md).

Promotion reuses the tested image; it does not rebuild source under a new tag. Record the source commit and immutable digest before publishing. Mutable tags alone are insufficient to identify a deployed artifact.

Local base images are refreshed on demand when their build timestamp exceeds seven days or their recipe/parent no longer matches. See [delivery](CI_CD.md).

Cleanup must preserve active release digests and anything referenced by a supported tag. PR images and unreferenced intermediates can be removed according to the registry cleanup policy. Check references before deleting artifacts; retention is not a database backup policy.

Store private deployment records and registry credentials outside Git. Future supply-chain work, including provenance and SBOM publication, is tracked separately.
