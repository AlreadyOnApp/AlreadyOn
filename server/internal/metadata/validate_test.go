package metadata

import "testing"

func TestIsValidTMDbKey(t *testing.T) {
	longSegment := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	validJWTShaped := longSegment + "." + longSegment + "." + longSegment

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid JWT-shaped token", validJWTShaped, true},
		{"empty", "", false},
		{"short v3 API key (32 hex chars, no dots)", "1234567890abcdef1234567890abcdef", false},
		{"has dots but too short overall", "a.b.c", false},
		{"long enough but no dots", longSegment + longSegment + longSegment, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsValidTMDbKey(c.value); got != c.want {
				t.Errorf("IsValidTMDbKey(%q) = %v, want %v", c.value, got, c.want)
			}
		})
	}
}
