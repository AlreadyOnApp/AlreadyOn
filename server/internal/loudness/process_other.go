//go:build !linux

package loudness

import "os/exec"

func configure(cmd *exec.Cmd) {}
