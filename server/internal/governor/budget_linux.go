//go:build linux

package governor

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AT_CLKTCK reports the kernel's user CPU accounting tick rate. Reading auxv
// avoids assuming 100 Hz or invoking a utility every scheduling window.
var clockTicks = sync.OnceValues(func() (uint64, error) {
	raw, err := os.ReadFile("/proc/self/auxv")
	if err != nil {
		return 0, err
	}
	width := strconv.IntSize / 8
	word := func(b []byte) uint64 {
		if width == 8 {
			return binary.NativeEndian.Uint64(b)
		}
		return uint64(binary.NativeEndian.Uint32(b))
	}
	for len(raw) >= 2*width {
		key, value := word(raw), word(raw[width:])
		if key == 17 && value > 0 {
			return value, nil
		}
		raw = raw[2*width:]
	}
	return 0, fmt.Errorf("governor: missing AT_CLKTCK")
})

func processCPU(pid int) (time.Duration, error) {
	hz, err := clockTicks()
	if err != nil {
		return 0, err
	}
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	return parseProcessCPU(string(raw), hz)
}

// /proc/PID/stat accounts for all threads of that process. The executable name
// can contain spaces and parentheses, so fields start after the LAST ')'.
func parseProcessCPU(raw string, hz uint64) (time.Duration, error) {
	end := strings.LastIndex(raw, ")")
	if end < 0 || hz == 0 {
		return 0, fmt.Errorf("governor: invalid process CPU record")
	}
	fields := strings.Fields(raw[end+1:])
	if len(fields) < 13 {
		return 0, fmt.Errorf("governor: short process CPU record")
	}
	u, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, err
	}
	s, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(float64(u+s) * float64(time.Second) / float64(hz)), nil
}
