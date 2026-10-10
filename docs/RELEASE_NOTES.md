# Release notes

Release descriptions explain what users can do and what problems were fixed. Avoid raw commit messages, issue numbers without context, implementation jargon and CI transcripts.

For each user-facing change, add a uniquely named Markdown file under `docs/release-notes/` in the same commit. Use one or more of these headings:

```markdown
## Features

- Connect to a nearby server without typing its address.

## Improvements

- Show or hide the PIN you entered with the eye button.

## Bug fixes

- Keep mute and subtitle settings when changing channels fullscreen.
```

Use one complete sentence per bullet on a single line. Name the affected app or screen where useful. Describe visible behavior, not the code used to implement it. Omit empty categories and changes that only affect development tooling. Once an entry reaches stable, leave it intact and add a new file for later changes.

Android candidate publication reads entries from its selected source commit and includes only release-note files changed since the newest published Android release from an earlier date. For example, a beta after `2026-10-09-rc3` describes the APK changes since that RC, not the entire product history. A maintenance-only candidate explicitly says there are no new user-facing changes.

Stable promotion copies the selected beta’s changelog rather than reading newer source changes. It retains the tested artifacts and date; only RC tags increment. See [release policy](ANDROID_RELEASES.md). Download links appear below the changes; source and build information remain in a collapsed details section. APKs, checksums and signing behavior are unchanged.

Editing existing release descriptions does not rebuild APKs or change their source tags. Keep each description accurate for that release. Older candidates without structured notes need their descriptions updated before stable promotion.
