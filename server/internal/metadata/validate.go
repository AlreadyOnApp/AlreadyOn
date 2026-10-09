package metadata

import "regexp"

// tmdbKeyPattern matches the shape of a TMDb v4 "API Read Access
// Token": a JWT, i.e. three dot-separated base64url segments. The
// accompanying length floor (see IsValidTMDbKey) exists specifically to
// reject the short 32-character v3 "API Key" that sits right next to it
// on TMDb's own settings page and won't work here — that key happens to
// contain no dots, so the pattern alone already rejects it, but the
// length check makes the intent explicit and guards against any
// dotted-but-still-wrong paste.
var tmdbKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

// IsValidTMDbKey reports whether value has the shape of a real TMDb v4
// API Read Access Token. This is a format check only — it can't confirm
// the key actually works without calling TMDb.
func IsValidTMDbKey(value string) bool {
	return len(value) >= 100 && tmdbKeyPattern.MatchString(value)
}
