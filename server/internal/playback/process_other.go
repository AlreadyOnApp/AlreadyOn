//go:build !linux

package playback

import "os/exec"

// Normal cancellation is portable. Abrupt parent-death supervision on other
// systems is not yet validated and must not be advertised as supported.
func configureProcess(cmd *exec.Cmd) {}
