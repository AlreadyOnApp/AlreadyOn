# Images and retention

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

Release tags identify approved image digests:

- `nightly` and `YYYY-MM-DD`: development releases.
- `latest`: an approved test promotion.
- `stable`: an approved main promotion.

Promotion reuses the tested image; it does not rebuild source under a new tag. Record the source commit and immutable digest before publishing. Mutable tags alone are insufficient to identify a deployed artifact.

Local base images are refreshed on demand when their build timestamp exceeds seven days or their recipe/parent no longer matches. See [delivery](CI_CD.md).

Cleanup must preserve active release digests and anything referenced by a supported tag. PR images and unreferenced intermediates can be removed according to the registry cleanup policy. Check references before deleting artifacts; retention is not a database backup policy.

Store private deployment records and registry credentials outside Git. Future supply-chain work, including provenance and SBOM publication, is tracked separately.
