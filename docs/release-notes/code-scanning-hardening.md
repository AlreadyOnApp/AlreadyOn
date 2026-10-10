## Bug fixes

- Keep sign-in redirects on the AlreadyOn server when a return URL contains hidden control characters.
- Prevent web asset links from exposing files outside the installed web directory.
- Keep malformed media metadata and memory counters from producing wrapped values.
- Store chosen local testing access keys with salted password hashes and upgrade existing keys when they authenticate.
- Serve direct-playback files as inert media bytes even when their contents resemble HTML.
