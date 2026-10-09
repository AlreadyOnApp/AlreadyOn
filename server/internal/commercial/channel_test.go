package commercial

import (
	"os"
	"path/filepath"
	"testing"
)

// Scanner paths are canonical even when the administrator selects a symlinked
// root. A sibling directory sharing its prefix must never enter the ad pool.
func TestChannelPolicyCanonicalFolderBoundary(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "ads")
	if err := os.Mkdir(selected, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(selected, alias); err != nil {
		t.Fatal(err)
	}
	policy := ChannelPolicy{ChannelSettings: ChannelSettings{Folders: []string{alias}}, EffectiveFolders: []string{alias}}
	for path, want := range map[string]bool{
		filepath.Join(selected, "one.mp4"):           true,
		filepath.Join(selected, "nested", "two.mp4"): true,
		filepath.Join(root, "ads-other", "bad.mp4"):  false,
		filepath.Join(root, "outside.mp4"):           false,
	} {
		if got := policy.IncludesPath(path); got != want {
			t.Errorf("IncludesPath(%q)=%v want %v", path, got, want)
		}
	}
}
