# Browser mobile UX

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The same browser application serves desktop, tablet and phone layouts. The native Android client is separate; do not equate their validation results.

## Guide and player

Keep channel selection, current program and playback controls usable on small screens. Large-player behavior must preserve navigation and program information without obscuring playback. Descriptions may truncate visually, but essential actions and labels must remain accessible.

Support touch and keyboard input. Use visible focus states, meaningful labels and adequately sized controls. Avoid hover-only actions. Respect safe areas, screen rotation and viewport changes.

## State

Show loading, empty, disconnected and error states explicitly. A channel change cancels stale playback work. Backgrounding, autoplay restrictions and network recovery must not produce runaway retries or misleading playback status.

## Validation

Check narrow and wide layouts, portrait/landscape, fullscreen, channel transitions, track menus and metadata overlays. Automated browser checks should be supplemented with real devices where gesture, decoder or operating-system behavior matters.

Keep exact requested labels and timings consistent with the product contract. See [playback](PLAYBACK.md) and [browser tests](../tests/e2e/README.md).
