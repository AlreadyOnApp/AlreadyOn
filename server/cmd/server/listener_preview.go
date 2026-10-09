//go:build preview || e2e

package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Preview credentials are public development fixtures, so the application listener binds
// loopback. Config addresses are validated port-only strings such as :15900.
func listenerAddress(addr string) string { return "127.0.0.1" + addr }

// Disposable E2E runs use an ephemeral port. Native previews each use their
// own API port plus two, preserving the other assistant's listeners.
func challengeAddress(addr string) string {
	if !periodicJobsEnabled {
		return "127.0.0.1:0"
	}
	port, err := strconv.Atoi(strings.TrimPrefix(addr, ":"))
	if err != nil {
		return "127.0.0.1:0"
	}
	return fmt.Sprintf("127.0.0.1:%d", port+2)
}
