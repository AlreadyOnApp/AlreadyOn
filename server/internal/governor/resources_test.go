package governor

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestResourceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sample Sample
		cpu    float64
		want   string
	}{
		{"at CPU ceiling", Sample{}, 80, LimitCPU},
		{"below CPU ceiling", Sample{}, 79.9, ""},
		{"disk busy is not CPU wait", Sample{DiskUtil: 80}, 5, LimitDisk},
		{"old five percent IO wait is allowed", Sample{IOWait: 5, DiskUtil: 30}, 5, ""},
		{"RAM at 80 percent", Sample{MemTotal: 100, MemAvailable: 20}, 5, ""},
		{"RAM above 80 percent", Sample{MemTotal: 100, MemAvailable: 19}, 5, LimitRAM},
		{"network Mbps ceiling", Sample{NetworkMbps: 750}, 5, LimitNetwork},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := pressureReason(Sample{}, tc.sample, tc.cpu, false); got != tc.want {
				t.Fatalf("%q != %q", got, tc.want)
			}
		})
	}
	if got := pressureReason(Sample{SwapOut: 42}, Sample{SwapOut: 42}, 5, false); got != "" {
		t.Fatal("historical swap blocks work")
	}
}

func TestSwapPressureRequiresSustainedPagingRate(t *testing.T) {
	pagesPerMiB := uint64(SwapActivityThresholdBytesPerSecond / os.Getpagesize())
	for _, tc := range []struct {
		name     string
		pages    uint64
		interval time.Duration
		want     string
	}{
		{"single page is transient", 1, time.Second, ""},
		{"low rate is transient", pagesPerMiB, 2 * time.Second, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := swapPressureState{}
			got := ""
			for range swapPressureSamples {
				limited := state.update(Sample{}, Sample{SwapOut: tc.pages}, tc.interval)
				got = pressureReason(Sample{}, Sample{SwapOut: tc.pages}, 5, limited)
			}
			if got != tc.want {
				t.Fatalf("pressure reason %q, want %q", got, tc.want)
			}
		})
	}

	state := swapPressureState{}
	for range swapPressureSamples {
		state.update(Sample{}, Sample{SwapOut: pagesPerMiB}, time.Second)
	}
	for range swapPressureSamples - 1 {
		if !state.update(Sample{}, Sample{SwapOut: pagesPerMiB / 4}, time.Second) {
			t.Fatal("transient clear sample released sustained swap pressure")
		}
	}
	if state.update(Sample{}, Sample{SwapOut: pagesPerMiB / 4}, time.Second) {
		t.Fatal("sustained quiet samples did not release swap pressure")
	}
}
func TestCPUReservation(t *testing.T) {
	for _, tc := range []struct {
		cores int
		want  float64
	}{{1, .8}, {2, 1}, {4, 3}, {8, 6.4}, {32, 25.6}} {
		if got := cpuCapacity(tc.cores); got != tc.want {
			t.Fatalf("%d CPUs: %f != %f", tc.cores, got, tc.want)
		}
	}
}
func TestCheckpointCancellationAndRecovery(t *testing.T) {
	g := Fixed(1)
	g.target = 0
	SetDefault(g)
	defer SetDefault(nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if Checkpoint(ctx) == nil {
		t.Fatal("cancelled checkpoint proceeded")
	}
	done := make(chan error, 1)
	go func() { done <- Checkpoint(t.Context()) }()
	select {
	case <-done:
		t.Fatal("checkpoint bypassed pause")
	case <-time.After(20 * time.Millisecond):
	}
	g.mu.Lock()
	g.target = 1
	g.signalLocked()
	g.mu.Unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("recovery did not wake Go work")
	}
}
func TestSharedNetworkPacing(t *testing.T) {
	var p bytePacer
	start := time.Now()
	// Two concurrent reservations share one 750-Mbps allowance, not one each.
	done := make(chan error, 2)
	for range 2 {
		go func() { done <- p.wait(t.Context(), NetworkBytesPerSecond/20) }()
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) < 95*time.Millisecond {
		t.Fatal("concurrent transfers exceeded shared allowance")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if p.wait(ctx, NetworkBytesPerSecond) == nil {
		t.Fatal("cancelled network wait succeeded")
	}
}
