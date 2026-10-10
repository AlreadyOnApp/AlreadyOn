package metrics

import "testing"

// Corrupted kernel fixtures must not become negative or wrapped byte counts.
func TestMemoryUsageRejectsOverflowingCounters(t *testing.T) {
	for _, tc := range []struct {
		name, total, available, current, cache, limit string
		ok                                            bool
		used, denominator                             int64
	}{
		{"host multiplication overflow", "18446744073709551615", "1", "", "", "", false, 0, 0},
		{"available exceeds total", "100", "101", "", "", "", false, 0, 0},
		{"current overflows signed range", "100", "25", "18446744073709551615", "0", "1000", true, 75 * 1024, 100 * 1024},
		{"cache exceeds signed range", "100", "25", "800", "18446744073709551615", "1000", true, 0, 1000},
		{"unlimited sentinel without host", "0", "0", "800", "0", "18446744073709551615", false, 0, 0},
		{"valid cgroup without host", "0", "0", "800", "100", "1000", true, 700, 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proc, cgroup := t.TempDir(), t.TempDir()
			write(t, proc, map[string]string{"meminfo": "MemTotal: " + tc.total + " kB\nMemAvailable: " + tc.available + " kB\n"})
			if tc.current != "" {
				write(t, cgroup, map[string]string{"memory.current": tc.current, "memory.stat": "inactive_file " + tc.cache, "memory.max": tc.limit})
			}
			got, ok := memoryUsage(proc, cgroup, tc.current != "")
			if ok != tc.ok || (ok && (got.used != tc.used || got.limit != tc.denominator)) {
				t.Fatalf("got %+v, %v", got, ok)
			}
		})
	}
}
