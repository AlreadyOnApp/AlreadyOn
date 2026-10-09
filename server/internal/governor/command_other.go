//go:build !linux

package governor

import "os/exec"

// Unsupported platforms do not admit production background work.
func configureCommand(cmd *exec.Cmd) {}
