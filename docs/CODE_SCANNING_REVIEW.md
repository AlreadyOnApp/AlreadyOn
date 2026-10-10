# Code scanning review

This review covers the 14 CodeQL findings reported against `7ae6fa8` in AlreadyOnApp/AlreadyOn. Dispositions below describe code changes and contextual review, not confirmed closure in GitHub. Read [security boundaries](SECURITY_BOUNDARIES.md) for compatibility and operational limits.

| Alerts | Finding | Response |
| --- | --- | --- |
| 1 | Client redirect | Compare parsed URL origin with the current origin. Regressions include browser normalization and permitted local routes. |
| 2 | Sensitive-data hashing | Salted Argon2id for chosen local testing keys; retain indexed SHA-256 only for generated 256-bit random keys. Legacy testing hashes no longer authenticate and are removed during local reprovisioning. |
| 3, 4 | Root-selection paths | Intentional owner-controlled boundary selection. Existing root confinement applies to playback and scanning. Viewer access to folder settings is denied by the authentication middleware. |
| 5, 8 | Folder-browser paths | Intentional owner-only browsing of candidate roots, returning directory names only. Viewer access is denied. Restricting this to existing roots would prevent selecting a new root. |
| 6, 7 | Static-file paths | Root-confined file opens replace joined-path stat and file delivery. Regression coverage rejects traversal and escaping symlinks while preserving static assets and SPA routes. |
| 9, 10 | Reflected XSS in media writers | Generic wrappers forward media bytes. Direct source responses now explicitly select an inert content type; authenticated playback, byte ranges and `nosniff` are exercised with synthetic HTML-looking payloads. |
| 11 | Sample-rate conversion | Parse at native integer width; negative and overflowing metadata become unknown. |
| 12–14 | Memory-counter conversions | Guard multiplication and signed conversions; subtract cache without wrapping and handle invalid readings. |

Owner-path findings need contextual disposition after review; a scanner finding is not itself proof that a viewer can access arbitrary paths. Wrapper findings may remain because analysis does not necessarily model these boundaries. Do not dismiss findings solely to produce a clean dashboard.

CodeQL now includes dev and test in its push and pull-request triggers. Verify the hosted analysis of the eventual reviewed commit before marking any alert fixed. No alert has been dismissed as part of this local work.

Local checks passed: focused regressions, 32-bit sample-rate coverage, web build, lint, documentation and syntax. Full HTTP tests were interrupted at media-refresh tests; hosted CodeQL remains unverified.
