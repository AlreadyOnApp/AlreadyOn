package metrics

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Disk utilization is elapsed time with I/O in progress, not space used or a
// universal saturation measure (especially for parallel devices such as NVMe).
type DiskSample struct {
	Name        string   `json:"name"`
	Utilization *float64 `json:"utilization"`
}
type diskReading struct {
	identity string
	busy     uint64
	at       time.Time
}
type diskCounter struct{ previous map[string]diskReading }

// Only whole, device-backed disks with no lower-layer devices are included.
// Partitions, RAID/device-mapper stacks, loop/zram and MMC boot areas would
// otherwise double-count I/O or describe virtual storage rather than disks.
func physicalDisks(root string) map[string]string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(root, name)
		if strings.HasPrefix(name, "mmcblk") && (strings.Contains(name, "boot") || strings.Contains(name, "rpmb")) {
			continue
		}
		if _, err := os.Stat(filepath.Join(path, "device")); err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(path, "partition")); err == nil {
			continue
		}
		slaves, err := os.ReadDir(filepath.Join(path, "slaves"))
		if err != nil || len(slaves) != 0 {
			continue
		}
		dev, err := os.ReadFile(filepath.Join(path, "dev"))
		if err != nil {
			continue
		}
		seq, _ := os.ReadFile(filepath.Join(path, "diskseq"))
		out[name] = strings.TrimSpace(string(dev)) + ":" + strings.TrimSpace(string(seq))
	}
	return out
}
func (d *diskCounter) sample(now time.Time, devices map[string]string, raw string) []DiskSample {
	counters := map[string]uint64{}
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		if _, ok := devices[f[2]]; !ok {
			continue
		}
		busy, err := strconv.ParseUint(f[12], 10, 64)
		if err == nil {
			counters[f[2]] = busy
		}
	}
	next := map[string]diskReading{}
	out := make([]DiskSample, 0, len(devices))
	for name, identity := range devices {
		p := DiskSample{Name: name}
		if busy, ok := counters[name]; ok {
			prev, exists := d.previous[name]
			elapsed := now.Sub(prev.at).Seconds()
			if exists && prev.identity == identity && elapsed > 0 && elapsed <= 30 && busy >= prev.busy {
				p.Utilization = percent(float64(busy-prev.busy) / 10 / elapsed)
			}
			next[name] = diskReading{identity, busy, now}
		}
		out = append(out, p)
	}
	d.previous = next
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (d *diskCounter) read(now time.Time) []DiskSample {
	raw, _ := os.ReadFile("/proc/diskstats")
	return d.sample(now, physicalDisks("/sys/block"), string(raw))
}

// ResourceSampler exposes the same physical-disk and network-namespace readings
// used by the dashboard to the background governor. Each caller owns its history.
type ResourceSampler struct {
	disks   diskCounter
	network networkCounter
}

func (s *ResourceSampler) Read(now time.Time) (disk, network float64) {
	for _, d := range s.disks.read(now) {
		if d.Utilization != nil {
			disk = max(disk, *d.Utilization)
		}
	}
	_, in, out := s.network.read(now)
	if in != nil {
		network += *in
	}
	if out != nil {
		network += *out
	}
	return
}
