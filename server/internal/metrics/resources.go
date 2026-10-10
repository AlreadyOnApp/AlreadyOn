package metrics

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CoreSample is one logical CPU's utilization, named by its kernel index.
type CoreSample struct {
	Core        int      `json:"core"`
	Utilization *float64 `json:"utilization"`
}

// parseCPUList expands a kernel CPU list such as "0-3,8,10-11".
func parseCPUList(raw string) map[int]bool {
	out := map[int]bool{}
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		if part == "" {
			continue
		}
		low, high, ranged := strings.Cut(part, "-")
		first, err := strconv.Atoi(low)
		if err != nil {
			return nil
		}
		last := first
		if ranged {
			if last, err = strconv.Atoi(high); err != nil || last < first {
				return nil
			}
		}
		for cpu := first; cpu <= last; cpu++ {
			out[cpu] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// allowedCPUs returns the cgroup cpuset AlreadyOn may run on, or nil when
// unknown (every CPU listed in /proc/stat is then included).
func allowedCPUs(cgroup string) map[int]bool {
	for _, name := range []string{"cpuset.cpus.effective", "cpuset/cpuset.effective_cpus", "cpuset/cpuset.cpus"} {
		if raw, err := os.ReadFile(filepath.Join(cgroup, name)); err == nil {
			if cpus := parseCPUList(string(raw)); cpus != nil {
				return cpus
			}
		}
	}
	return nil
}

type cpuTicks struct{ total, idle uint64 }

func (t cpuTicks) utilization(previous cpuTicks, ok bool) *float64 {
	if !ok || t.total <= previous.total || t.idle < previous.idle {
		return nil
	}
	return percent(100 * (1 - float64(t.idle-previous.idle)/float64(t.total-previous.total)))
}

type cpuCounter struct {
	aggregate cpuTicks
	cores     map[int]cpuTicks
}

// sample reads /proc/stat. Aggregate utilization is the mean of the allowed
// cores so a cpuset-limited container is not diluted by host CPUs it cannot
// use; without per-core lines it falls back to the kernel's "cpu" total.
func (c *cpuCounter) sample(raw string, allowed map[int]bool) (*float64, []CoreSample) {
	var aggregate cpuTicks
	hasAggregate := false
	current := map[int]cpuTicks{}
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) < 9 || !strings.HasPrefix(f[0], "cpu") {
			continue
		}
		var ticks cpuTicks
		for i := 1; i <= 8; i++ {
			n, _ := strconv.ParseUint(f[i], 10, 64)
			ticks.total += n
			if i == 4 || i == 5 {
				ticks.idle += n
			}
		}
		if f[0] == "cpu" {
			aggregate, hasAggregate = ticks, true
			continue
		}
		core, err := strconv.Atoi(f[0][3:])
		if err != nil || (allowed != nil && !allowed[core]) {
			continue
		}
		current[core] = ticks
	}
	cores := make([]CoreSample, 0, len(current))
	sum, valid := 0.0, 0
	for core, ticks := range current {
		previous, ok := c.cores[core]
		value := ticks.utilization(previous, ok)
		if value != nil {
			sum += *value
			valid++
		}
		cores = append(cores, CoreSample{core, value})
	}
	sort.Slice(cores, func(i, j int) bool { return cores[i].Core < cores[j].Core })
	var out *float64
	if len(cores) > 0 && valid == len(cores) {
		out = percent(sum / float64(valid))
	} else if len(cores) == 0 && hasAggregate {
		out = aggregate.utilization(c.aggregate, c.aggregate.total > 0)
	}
	c.cores, c.aggregate = current, aggregate
	if len(cores) == 0 {
		return out, nil
	}
	return out, cores
}

// loadAverage is the one-minute run-queue average; /proc is not namespaced,
// so inside a container it describes the whole host.
func loadAverage(proc string) *float64 {
	raw, err := os.ReadFile(filepath.Join(proc, "loadavg"))
	if err != nil {
		return nil
	}
	f := strings.Fields(string(raw))
	if len(f) == 0 {
		return nil
	}
	value, err := strconv.ParseFloat(f[0], 64)
	if err != nil || value < 0 {
		return nil
	}
	return &value
}

// pressure returns the PSI "some avg10" value: the percentage of the last ten
// seconds in which at least one task stalled on the resource. Kernels without
// PSI, or with it disabled, have no file and yield nil.
func pressure(path string) *float64 {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "some" {
			continue
		}
		for _, field := range f[1:] {
			if value, ok := strings.CutPrefix(field, "avg10="); ok {
				if n, err := strconv.ParseFloat(value, 64); err == nil {
					return percent(n)
				}
			}
		}
	}
	return nil
}

func readUint(path string) (uint64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	return n, err == nil
}

func statValue(path, key string) uint64 {
	raw, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(raw), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == key {
			n, _ := strconv.ParseUint(f[1], 10, 64)
			return n
		}
	}
	return 0
}

type memoryReading struct {
	used, limit int64
	limited     bool
}

// memoryUsage prefers the container's cgroup: used excludes reclaimable
// inactive file cache (as docker stats does) and a limit at or above host RAM
// counts as unlimited. Native servers, or containers without a readable
// cgroup, use host MemTotal - MemAvailable.
func memoryUsage(proc, cgroup string, container bool) (memoryReading, bool) {
	values := map[string]uint64{}
	if raw, err := os.ReadFile(filepath.Join(proc, "meminfo")); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if f := strings.Fields(line); len(f) > 1 {
				values[strings.TrimSuffix(f[0], ":")], _ = strconv.ParseUint(f[1], 10, 64)
			}
		}
	}
	// Kernel counters are unsigned; check multiplication and conversion before
	// computing signed byte counts. A corrupt sample is unavailable, not wrapped.
	var host int64
	if total := values["MemTotal"]; total <= math.MaxInt64/1024 {
		host = int64(total * 1024)
	}
	if container {
		limitPath, currentPath, statPath, inactive := "memory.max", "memory.current", "memory.stat", "inactive_file"
		if _, err := os.Stat(filepath.Join(cgroup, currentPath)); err != nil {
			limitPath, currentPath, statPath, inactive = "memory/memory.limit_in_bytes", "memory/memory.usage_in_bytes", "memory/memory.stat", "total_inactive_file"
		}
		if current, ok := readUint(filepath.Join(cgroup, currentPath)); ok && current <= math.MaxInt64 {
			cache := statValue(filepath.Join(cgroup, statPath), inactive)
			var used int64
			if cache < current {
				used = int64(current - cache)
			}
			if limit, ok := readUint(filepath.Join(cgroup, limitPath)); ok && limit > 0 && limit <= math.MaxInt64 && (host == 0 || limit < uint64(host)) {
				return memoryReading{used, int64(limit), true}, true
			}
			if host > 0 {
				return memoryReading{used, host, false}, true
			}
		}
	}
	available, ok := values["MemAvailable"]
	if host == 0 || !ok || available > uint64(host)/1024 {
		return memoryReading{}, false
	}
	return memoryReading{host - int64(available*1024), host, false}, true
}

type ioCounter struct {
	source      string
	read, write uint64
	at          time.Time
}

// sample converts cumulative byte counters into decimal MB/s. First readings,
// source changes, resets and gaps longer than 30 seconds yield nil, not spikes.
func (c *ioCounter) sample(now time.Time, source string, read, write uint64) (*float64, *float64) {
	elapsed := now.Sub(c.at).Seconds()
	var r, w *float64
	if c.source == source && !c.at.IsZero() && elapsed > 0 && elapsed <= 30 && read >= c.read && write >= c.write {
		rv, wv := float64(read-c.read)/elapsed/1e6, float64(write-c.write)/elapsed/1e6
		r, w = &rv, &wv
	}
	*c = ioCounter{source, read, write, now}
	return r, w
}

// cgroupIO totals bytes read and written by AlreadyOn's cgroup across block
// devices (cgroup v2 io.stat, else v1 blkio). Network filesystems are not
// block devices and do not appear here.
func cgroupIO(cgroup string) (uint64, uint64, bool) {
	if raw, err := os.ReadFile(filepath.Join(cgroup, "io.stat")); err == nil {
		var read, write uint64
		for _, line := range strings.Split(string(raw), "\n") {
			for _, field := range strings.Fields(line) {
				key, value, _ := strings.Cut(field, "=")
				n, _ := strconv.ParseUint(value, 10, 64)
				switch key {
				case "rbytes":
					read += n
				case "wbytes":
					write += n
				}
			}
		}
		return read, write, true
	}
	raw, err := os.ReadFile(filepath.Join(cgroup, "blkio", "blkio.throttle.io_service_bytes"))
	if err != nil {
		return 0, 0, false
	}
	var read, write uint64
	for _, line := range strings.Split(string(raw), "\n") {
		if f := strings.Fields(line); len(f) == 3 {
			n, _ := strconv.ParseUint(f[2], 10, 64)
			switch f[1] {
			case "Read":
				read += n
			case "Write":
				write += n
			}
		}
	}
	return read, write, true
}

// physicalIO totals sectors read and written on whole physical disks; the
// kernel reports 512-byte sectors regardless of the device's block size.
func physicalIO(raw string, devices map[string]string) (uint64, uint64) {
	var read, write uint64
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		if _, ok := devices[f[2]]; !ok {
			continue
		}
		r, _ := strconv.ParseUint(f[5], 10, 64)
		w, _ := strconv.ParseUint(f[9], 10, 64)
		read += r * 512
		write += w * 512
	}
	return read, write
}
