package metrics

import (
	"fmt"
	"testing"
	"time"
)

func TestDefaultNetworkRoute(t *testing.T) {
	v4 := "eth0 00000000 01020304 0003 0 0 200 00000000\nwlan0 00000000 01020304 0003 0 0 50 00000000\nbad0 00000000 01020304 0201 0 0 1 00000000\ndown0 00000000 01020304 0000 0 0 1 00000000\n"
	if got := defaultInterface(v4, ""); got != "wlan0" {
		t.Fatal(got)
	}
	v6 := "00000000000000000000000000000000 00 00000000000000000000000000000000 00 fe800000000000000000000000000001 00000064 0 0 00000003 enp1s0"
	if got := defaultInterface("", v6); got != "enp1s0" {
		t.Fatal(got)
	}
	if got := defaultInterface("junk", ""); got != "" {
		t.Fatal(got)
	}
}
func TestNetworkRateWarmupResetAndInterfaceChanges(t *testing.T) {
	dev := func(iface string, rx, tx uint64) string {
		return fmt.Sprintf("%s: %d 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0", iface, rx, tx)
	}
	var n networkCounter
	now := time.Now()
	a, b := n.sample(now, "eth0", dev("eth0", 1000, 2000))
	if a != nil || b != nil {
		t.Fatal("warmup should be missing")
	}
	a, b = n.sample(now.Add(time.Second), "eth0", dev("eth0", 126000, 252000))
	if a == nil || b == nil || *a != 1 || *b != 2 {
		t.Fatal(a, b)
	}
	a, b = n.sample(now.Add(2*time.Second), "eth0", dev("eth0", 126000, 252000))
	if a == nil || b == nil || *a != 0 || *b != 0 {
		t.Fatal("idle must be zero")
	}
	a, b = n.sample(now.Add(3*time.Second), "eth0", dev("eth0", 1, 1))
	if a != nil || b != nil {
		t.Fatal("counter reset produced a spike")
	}
	a, b = n.sample(now.Add(4*time.Second), "eth1", dev("eth1", 1000000, 2000000))
	if a != nil || b != nil {
		t.Fatal("interface change produced a spike")
	}
	a, b = n.sample(now.Add(time.Minute), "eth1", dev("eth1", 2000000, 3000000))
	if a != nil || b != nil {
		t.Fatal("gap should be missing")
	}
	a, b = n.sample(now.Add(61*time.Second), "eth1", "malformed")
	if a != nil || b != nil || n.iface != "" {
		t.Fatal("missing device retained baseline")
	}
}
