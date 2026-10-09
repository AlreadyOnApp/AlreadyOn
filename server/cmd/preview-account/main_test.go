package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reject exposed or malformed credential files without touching preview data.
func TestReadCredentials(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		mode       os.FileMode
		valid      bool
	}{
		{"private", `{"username":"example","password":"local-only-password"}`, 0600, true},
		{"public", `{"username":"example","password":"local-only-password"}`, 0644, false},
		{"empty", `{"username":"example","password":""}`, 0600, false},
		{"invalid", `{`, 0600, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "credentials.json")
			if err := os.WriteFile(path, []byte(tc.body), tc.mode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, tc.mode); err != nil {
				t.Fatal(err)
			}
			got, err := readCredentials(path)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
			if tc.valid && got.Username != "example" {
				t.Fatalf("unexpected account: %q", got.Username)
			}
			if err != nil && strings.Contains(err.Error(), "local-only-password") {
				t.Fatal("credential leaked in error")
			}
		})
	}
}

// A fresh contributor checkout keeps first-run setup and existing accounts intact.
func TestMissingOptionalCredentials(t *testing.T) {
	if err := provision(false, filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal(err)
	}
	if err := provision(true, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("explicit reset needs credentials")
	}
}
