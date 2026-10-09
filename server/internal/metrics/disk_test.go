package metrics

// Physical-disk busy time now feeds only the background governor.
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func diskstats(name string, busy uint64) string {
	return fmt.Sprintf("8 0 %s 0 0 0 0 0 0 0 0 0 %d 0\n", name, busy)
}
func TestDiskBusyTimeAndMissingResetSamples(t *testing.T) {
	var d diskCounter
	now := time.Unix(1800000000, 0)
	devices := map[string]string{"sda": "8:0:1", "nvme0n1": "259:0:2"}
	read := func(at time.Time, raw string) []DiskSample { return d.sample(at, devices, raw) }
	first := read(now, diskstats("sda", 100)+diskstats("nvme0n1", 500))
	if len(first) != 2 || first[0].Utilization != nil || first[1].Utilization != nil {
		t.Fatal(first)
	}
	got := read(now.Add(2*time.Second), diskstats("sda", 1100)+diskstats("nvme0n1", 500))
	if *got[0].Utilization != 0 || *got[1].Utilization != 50 {
		t.Fatal(got)
	}
	got = read(now.Add(3*time.Second), diskstats("sda", 10))
	if got[0].Utilization != nil || got[1].Utilization != nil {
		t.Fatal("missing or reset counters must be null", got)
	}
	got = read(now.Add(4*time.Second), diskstats("sda", 5010)+diskstats("nvme0n1", 600))
	if got[0].Utilization != nil || *got[1].Utilization != 100 {
		t.Fatal("reappearing device baseline or clamp", got)
	}
	devices["sda"] = "8:0:99"
	got = read(now.Add(5*time.Second), diskstats("sda", 6010))
	if got[1].Utilization != nil {
		t.Fatal("replaced disk reused baseline")
	}
	got = read(now.Add(40*time.Second), diskstats("sda", 7010))
	if got[1].Utilization != nil {
		t.Fatal("long sample gap should be missing")
	}
	got = read(now.Add(40*time.Second), diskstats("sda", 8010))
	if got[1].Utilization != nil {
		t.Fatal("nonpositive interval should be missing")
	}
}
func TestPhysicalDiskSelection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"sda", "sda1", "nvme0n1", "mmcblk0", "mmcblk0boot0", "loop0", "md0", "dm-0"} {
		base := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(base, "slaves"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "dev"), []byte("8:0"), 0644); err != nil {
			t.Fatal(err)
		}
		if name != "loop0" {
			if err := os.Mkdir(filepath.Join(base, "device"), 0755); err != nil {
				t.Fatal(err)
			}
		}
		if name == "sda1" {
			if err := os.WriteFile(filepath.Join(base, "partition"), []byte("1"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		if name == "md0" || name == "dm-0" {
			if err := os.Mkdir(filepath.Join(base, "slaves", "sda"), 0755); err != nil {
				t.Fatal(err)
			}
		}
	}
	got := physicalDisks(root)
	if len(got) != 3 || got["sda"] == "" || got["nvme0n1"] == "" || got["mmcblk0"] == "" {
		t.Fatal(got)
	}
}
