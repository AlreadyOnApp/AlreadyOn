# TV remote navigation

Fullscreen Up/Down reveals hidden controls and otherwise retains the selected control. Vertical navigation works inside dot-menu popups. Left/Right navigates visible controls; with controls hidden, it tunes channels. Select activates the selected option.

Entering fullscreen, revealing controls, tuning a channel and pausing/resuming all focus Play/Pause. Quality, Audio Tracks and Subtitles open with the current setting selected and marked. The audio selection comes from the actual playback plan. Back returns from a submenu to the dot menu, then closes the menu, hides controls and returns to the guide.

Guide arrows update the details beside the preview using the selected program’s existing metadata; browsing does not retune playback. Commercials are omitted from TV guide cells and navigation. Continuation segments of a show split by commercials are joined into one cell; fresh airings remain separate. Arrows advance between shows, resolve gaps consistently and keep selection visible. Timeline headers always align to hours and half-hours; 12:44 AM initially shows 12:30 AM, 1:00 AM, 1:30 AM and so on, regardless of program start times.

In the guide, Back first restores the previewed channel and current program if selection has moved. When already there, Back focuses Play/Pause in the mini-player. Left/Right navigates its controls; Up opens the profile menu with **Switch Profile** focused and Down restores the current channel/program in the guide. Back from the open profile menu returns to Play/Pause in the mini-player. Back from the mini-player or profile button exits to the TV home screen without signing out. An open popup handles Back before its underlying screen.

After the first profile selection or restoration, a static remote tutorial appears once per installation. **Got it** or Back dismisses it and opens the guide. The acknowledgement survives updates and sign-out; uninstalling resets it. No player or guide is mounted during the tutorial; it has no animations, image assets or timers.

The optimized review APK is labeled **AlreadyOn TV Test**. Updating this package preserves its encrypted session; uninstalling removes it. See [device review](TV_DEVICE_REVIEW.md) for the physical remote checks.

Routine background loading has no animated status bar on TV; refreshes keep the layout steady. Errors still appear when an operation fails.

TV and phone startup show the centered logo and tagline for two seconds while loading underneath; rotation retains the deadline. The TV guide removes branding, reserves profile-menu space and fits four rows without smaller fonts. Up/Down use identical short transitions to reveal clipped rows.
