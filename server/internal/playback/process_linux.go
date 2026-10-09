//go:build linux

package playback

import (
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	// Linux kills the encoder if the server crashes. FFmpeg does not spawn
	// children; CommandContext kills and Wait reaps it on normal cancellation.
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
