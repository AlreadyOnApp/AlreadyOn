# TV profile settings

Open the profile menu and choose **Profile Settings**, between **Switch Profile** and **Sign Out**. The full-screen page stops playback and removes the player and guide. It uses the same current-profile API and fields as the WebUI; it does not expose administration of other profiles.

Use Up/Down to select a category, Select to open it, and Right to enter its controls. Focus starts on **Display Name**. A white outline marks the focused control. Choice menus focus and mark the current value. Name and PIN fields use the TV’s on-screen keyboard.

Categories are Display Name, Icon, PIN, Guide Display, Audio and Subtitles. Icon supports presets, removal and a custom image through an installed Android image picker. If the TV has no picker, upload the custom image through the WebUI. Images are cropped and bounded before upload.

**Save Changes** commits only changed name, icon, time-format and language/default-subtitle fields. Drafts survive category changes and are discarded when leaving without saving. PIN updates and removal apply separately; removal requires confirmation. **Cancel**, **Back To Guide** or remote Back returns to the guide and restores playback.

Only the current category is mounted; there are no animated assets or background guide refreshes while settings is open.

For a signed-in review device, run `python3 scripts/android/review-tv-profile-settings.py --serial DEVICE_SERIAL --sdk /path/to/Android/Sdk`. It checks menu placement, all categories, language-menu focus, draft editing and Back to the guide. It discards changes without saving or changing the PIN.
