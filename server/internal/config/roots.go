package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RootKind distinguishes the two separately-configured content root
// categories (docs/PROJECT_BRIEF.md: "Read media and commercials from
// separately configured local folders").
type RootKind string

const (
	RootKindMedia      RootKind = "media"
	RootKindCommercial RootKind = "commercial"
)

// ValidatedRoot is a configured content root that has been confirmed to
// exist, be a readable directory, and had its symlinks resolved to a
// canonical real path.
type ValidatedRoot struct {
	Kind           RootKind
	ConfiguredPath string
	// RealPath is ConfiguredPath with all symlinks resolved. Later
	// scanning (#15/#18) should treat this as the trust boundary: a file
	// discovered inside the root whose resolved path (see PathWithinRoot)
	// falls outside RealPath is a symlink escaping the root and must be
	// rejected, not followed.
	RealPath string
}

// ValidateRoots resolves and validates every path in paths as a root of
// the given kind. Every problem is collected and returned together (not
// just the first one found), so an administrator sees every misconfigured
// root in one pass instead of fixing them one at a time. An empty list is
// not an error — it means that category simply isn't configured.
func ValidateRoots(kind RootKind, paths []string) ([]ValidatedRoot, []error) {
	var valid []ValidatedRoot
	var errs []error
	for _, p := range paths {
		v, err := ValidateRoot(kind, p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		valid = append(valid, v)
	}
	return valid, errs
}

// RootProblem classifies why a configured root can't be used.
type RootProblem string

const (
	ProblemNotAbsolute  RootProblem = "notAbsolute"
	ProblemNotFound     RootProblem = "notFound"
	ProblemNotDirectory RootProblem = "notDirectory"
	ProblemUnreadable   RootProblem = "unreadable"
)

// RootError is a root that failed validation. The message text is what
// operators see in logs; Problem lets an API report the same failure
// without parsing it.
type RootError struct {
	Kind    RootKind
	Path    string
	Problem RootProblem
	Err     error
}

func (e *RootError) Error() string {
	switch e.Problem {
	case ProblemNotAbsolute:
		return fmt.Sprintf("%s root %q: must be an absolute path", e.Kind, e.Path)
	case ProblemNotFound:
		return fmt.Sprintf("%s root %q: does not exist", e.Kind, e.Path)
	case ProblemNotDirectory:
		return fmt.Sprintf("%s root %q: not a directory", e.Kind, e.Path)
	case ProblemUnreadable:
		return fmt.Sprintf("%s root %q: not readable: %v", e.Kind, e.Path, e.Err)
	}
	return fmt.Sprintf("%s root %q: %v", e.Kind, e.Path, e.Err)
}

func (e *RootError) Unwrap() error { return e.Err }

// ValidateRoot checks one configured path. Failures are *RootError. The owner
// chooses this boundary through owner-only settings or local configuration;
// it cannot be confined to the old roots without preventing new root selection.
func ValidateRoot(kind RootKind, configuredPath string) (ValidatedRoot, error) {
	fail := func(p RootProblem, err error) (ValidatedRoot, error) {
		return ValidatedRoot{}, &RootError{Kind: kind, Path: configuredPath, Problem: p, Err: err}
	}
	if !filepath.IsAbs(configuredPath) {
		return fail(ProblemNotAbsolute, nil)
	}

	// EvalSymlinks both confirms the path exists and resolves every
	// symlink along it (including the root itself being a symlink) to a
	// canonical real path, which later scanning uses as the boundary a
	// symlink inside the tree must not escape.
	real, err := filepath.EvalSymlinks(configuredPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fail(ProblemNotFound, err)
		}
		return fail(ProblemUnreadable, err)
	}

	info, err := os.Stat(real)
	if err != nil {
		return fail(ProblemUnreadable, err)
	}
	if !info.IsDir() {
		return fail(ProblemNotDirectory, nil)
	}

	f, err := os.Open(real)
	if err != nil {
		return fail(ProblemUnreadable, err)
	}
	defer f.Close()
	if _, err := f.Readdirnames(1); err != nil && err != io.EOF {
		return fail(ProblemUnreadable, err)
	}

	return ValidatedRoot{Kind: kind, ConfiguredPath: configuredPath, RealPath: real}, nil
}

// PathWithinRoot reports whether resolvedPath — already symlink-resolved,
// e.g. via filepath.EvalSymlinks — is root.RealPath itself or lives
// underneath it. Callers use this to reject a symlink discovered inside a
// scanned root that resolves to somewhere outside it.
func PathWithinRoot(root ValidatedRoot, resolvedPath string) bool {
	rel, err := filepath.Rel(root.RealPath, resolvedPath)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
