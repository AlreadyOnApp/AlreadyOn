//go:build linux

package metrics

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type gpuEngine struct {
	fd          int
	count, time uint64
}
type GPUCounter struct{ engines []gpuEngine }

func NewGPUCounter() *GPUCounter {
	g := &GPUCounter{}
	raw, err := os.ReadFile("/sys/bus/event_source/devices/i915/type")
	if err != nil {
		return g
	}
	typ, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 32)
	if err != nil {
		return g
	}
	paths, _ := filepath.Glob("/sys/bus/event_source/devices/i915/events/*-busy")
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		config, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(string(data)), "config="), 0, 64)
		if err != nil {
			continue
		}
		attr := unix.PerfEventAttr{Type: uint32(typ), Size: unix.PERF_ATTR_SIZE_VER0, Config: config, Read_format: unix.PERF_FORMAT_TOTAL_TIME_ENABLED}
		fd, err := unix.PerfEventOpen(&attr, -1, 0, -1, unix.PERF_FLAG_FD_CLOEXEC)
		if err == nil {
			g.engines = append(g.engines, gpuEngine{fd: fd})
		}
	}
	return g
}
func (g *GPUCounter) Read() *float64 {
	var maximum *float64
	for i := range g.engines {
		engine := &g.engines[i]
		raw := make([]byte, 16)
		n, err := unix.Read(engine.fd, raw)
		if err != nil || n != 16 {
			continue
		}
		count, elapsed := binary.NativeEndian.Uint64(raw[:8]), binary.NativeEndian.Uint64(raw[8:])
		if engine.time > 0 && elapsed > engine.time && count >= engine.count {
			usage := percent(100 * float64(count-engine.count) / float64(elapsed-engine.time))
			if maximum == nil || *usage > *maximum {
				maximum = usage
			}
		}
		engine.count, engine.time = count, elapsed
	}
	return maximum
}
func (g *GPUCounter) Close() {
	for _, e := range g.engines {
		_ = unix.Close(e.fd)
	}
}
