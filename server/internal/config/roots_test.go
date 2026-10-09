package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRoots_ValidRoots(t *testing.T) {
	valid, errs := ValidateRoots(RootKindMedia, []string{t.TempDir(), t.TempDir()})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(valid) != 2 {
		t.Fatalf("expected 2 validated roots, got %d", len(valid))
	}
	if valid[0].Kind != RootKindMedia {
		t.Errorf("Kind = %q, want %q", valid[0].Kind, RootKindMedia)
	}
}

func TestValidateRoots_EmptyList_IsNotAnError(t *testing.T) {
	valid, errs := ValidateRoots(RootKindCommercial, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for an unconfigured category, got %v", errs)
	}
	if len(valid) != 0 {
		t.Fatalf("expected no validated roots, got %d", len(valid))
	}
}

func TestValidateRoots_NonexistentPath_ProducesClearError(t *testing.T) {
	valid, errs := ValidateRoots(RootKindMedia, []string{filepath.Join(t.TempDir(), "does-not-exist")})
	if len(valid) != 0 {
		t.Fatalf("expected no validated roots, got %d", len(valid))
	}
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "does not exist") {
		t.Errorf("expected a clear 'does not exist' diagnostic, got: %v", errs[0])
	}
	var re *RootError
	if !errors.As(errs[0], &re) || re.Problem != ProblemNotFound {
		t.Errorf("error = %#v, want *RootError with ProblemNotFound", errs[0])
	}
}

func TestValidateRoots_FileInsteadOfDirectory_ProducesClearError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	_, errs := ValidateRoots(RootKindMedia, []string{file})
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "not a directory") {
		t.Errorf("expected a clear 'not a directory' diagnostic, got: %v", errs[0])
	}
}

func TestValidateRoots_RelativePath_ProducesClearError(t *testing.T) {
	_, errs := ValidateRoots(RootKindMedia, []string{"relative/path"})
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "absolute") {
		t.Errorf("expected a clear 'must be an absolute path' diagnostic, got: %v", errs[0])
	}
}

func TestValidateRoots_UnreadableDirectory_ProducesClearError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, which bypasses directory permission checks")
	}

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	_, errs := ValidateRoots(RootKindMedia, []string{dir})
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "not readable") {
		t.Errorf("expected a clear 'not readable' diagnostic, got: %v", errs[0])
	}
}

func TestValidateRoots_MultipleBrokenRoots_ReportsEveryOne(t *testing.T) {
	_, errs := ValidateRoots(RootKindMedia, []string{
		filepath.Join(t.TempDir(), "missing-one"),
		filepath.Join(t.TempDir(), "missing-two"),
	})
	if len(errs) != 2 {
		t.Fatalf("expected both broken roots to be reported, got %d error(s): %v", len(errs), errs)
	}
}

func TestValidateRoots_ResolvesSymlinkedRootToRealPath(t *testing.T) {
	realDir := t.TempDir()
	linkPath := filepath.Join(t.TempDir(), "link-to-real")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	valid, errs := ValidateRoots(RootKindMedia, []string{linkPath})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 validated root, got %d", len(valid))
	}

	resolvedReal, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(realDir): %v", err)
	}
	if valid[0].RealPath != resolvedReal {
		t.Errorf("RealPath = %q, want %q", valid[0].RealPath, resolvedReal)
	}
	if valid[0].ConfiguredPath != linkPath {
		t.Errorf("ConfiguredPath = %q, want %q", valid[0].ConfiguredPath, linkPath)
	}
}

func TestPathWithinRoot(t *testing.T) {
	root := ValidatedRoot{RealPath: "/media/movies"}

	cases := []struct {
		path string
		want bool
	}{
		{"/media/movies", true},
		{"/media/movies/inception.mkv", true},
		{"/media/movies/sub/dir/file.mkv", true},
		{"/media/other", false},
		{"/etc/passwd", false},
		{"/media/movies-evil-twin", false},
	}
	for _, tc := range cases {
		if got := PathWithinRoot(root, tc.path); got != tc.want {
			t.Errorf("PathWithinRoot(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestPathWithinRoot_DetectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	validated := ValidatedRoot{RealPath: root}

	// A symlink inside the root whose target resolves outside it.
	escapePath := filepath.Join(root, "escape")
	if err := os.Symlink(outside, escapePath); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(escapePath)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if PathWithinRoot(validated, resolved) {
		t.Error("expected a symlink resolving outside the root to be rejected")
	}

	// A normal file inside the root, for contrast.
	safePath := filepath.Join(root, "safe.txt")
	if err := os.WriteFile(safePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if !PathWithinRoot(validated, safePath) {
		t.Error("expected a plain file inside the root to be accepted")
	}
}
