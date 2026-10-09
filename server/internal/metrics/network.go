package metrics

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type networkCounter struct {
	iface  string
	rx, tx uint64
	at     time.Time
}

// Prefer the lowest-metric active IPv4 default route, then an IPv6 default.
func defaultInterface(v4, v6 string) string {
	name := ""
	best := uint64(^uint64(0))
	for _, line := range strings.Split(v4, "\n") {
		f := strings.Fields(line)
		if len(f) < 8 || f[1] != "00000000" || f[7] != "00000000" || f[0] == "lo" {
			continue
		}
		flags, e1 := strconv.ParseUint(f[3], 16, 64)
		metric, e2 := strconv.ParseUint(f[6], 10, 64)
		if e1 == nil && e2 == nil && flags&1 != 0 && flags&0x200 == 0 && metric < best {
			name, best = f[0], metric
		}
	}
	if name != "" {
		return name
	}
	for _, line := range strings.Split(v6, "\n") {
		f := strings.Fields(line)
		if len(f) < 10 || f[0] != strings.Repeat("0", 32) || f[1] != "00" || f[3] != "00" || f[9] == "lo" {
			continue
		}
		flags, e1 := strconv.ParseUint(f[8], 16, 64)
		metric, e2 := strconv.ParseUint(f[5], 16, 64)
		if e1 == nil && e2 == nil && flags&1 != 0 && flags&0x200 == 0 && metric < best {
			name, best = f[9], metric
		}
	}
	return name
}
func (n *networkCounter) sample(now time.Time, iface, dev string) (*float64, *float64) {
	for _, line := range strings.Split(dev, "\n") {
		name, values, ok := strings.Cut(line, ":")
		if !ok || iface == "" || strings.TrimSpace(name) != iface {
			continue
		}
		f := strings.Fields(values)
		if len(f) < 16 {
			break
		}
		rx, e1 := strconv.ParseUint(f[0], 10, 64)
		tx, e2 := strconv.ParseUint(f[8], 10, 64)
		if e1 != nil || e2 != nil {
			break
		}
		var ingress, egress *float64
		elapsed := now.Sub(n.at).Seconds()
		if n.iface == iface && !n.at.IsZero() && elapsed > 0 && elapsed <= 30 && rx >= n.rx && tx >= n.tx {
			in, out := float64(rx-n.rx)*8/elapsed/1e6, float64(tx-n.tx)*8/elapsed/1e6
			ingress, egress = &in, &out
		}
		*n = networkCounter{iface: iface, rx: rx, tx: tx, at: now}
		return ingress, egress
	}
	*n = networkCounter{}
	return nil, nil
}
func (n *networkCounter) read(now time.Time) (string, *float64, *float64) {
	v4, _ := os.ReadFile("/proc/net/route")
	v6, _ := os.ReadFile("/proc/net/ipv6_route")
	dev, _ := os.ReadFile("/proc/net/dev")
	iface := defaultInterface(string(v4), string(v6))
	in, out := n.sample(now, iface, string(dev))
	return iface, in, out
}
