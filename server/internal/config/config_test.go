package config

import (
	"testing"
	"time"
)

func TestLoad_UnsetListeners_UseDefaults(t *testing.T) {
	t.Setenv("ALREADYON_WEB_PORT", "")
	t.Setenv("ALREADYON_STREAM_PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ManagementAddr != ":19900" {
		t.Errorf("addr = %q; want :19900", cfg.ManagementAddr)
	}
}

func TestLoad_PortOverrides(t *testing.T) {
	t.Setenv("ALREADYON_WEB_PORT", "8123")
	t.Setenv("ALREADYON_STREAM_PORT", "8124")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ManagementAddr != ":8123" {
		t.Errorf("addr = %q; want :8123", cfg.ManagementAddr)
	}
}

func TestLoad_InvalidPort_FailsFast(t *testing.T) {
	for _, bad := range []string{"abc", "0", "70000", ":8080", "-1"} {
		t.Setenv("ALREADYON_WEB_PORT", bad)
		t.Setenv("ALREADYON_STREAM_PORT", "")
		if _, err := Load(); err == nil {
			t.Errorf("Load accepted ALREADYON_WEB_PORT=%q, want an error", bad)
		}
	}
}

func TestLoad_DirectoriesAreBuildTimeValues(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != dataDir || cfg.WebDir != webDir {
		t.Errorf("dirs = %q, %q; want the build-time values %q, %q", cfg.DataDir, cfg.WebDir, dataDir, webDir)
	}
}

// Existing deployments may retain the old stream-port override after upgrading.
// It must not reject startup or open a second application listener.
func TestLoad_ObsoleteStreamPortIsIgnored(t *testing.T) {
	t.Setenv("ALREADYON_WEB_PORT", "19900")
	t.Setenv("ALREADYON_STREAM_PORT", "19900")

	_, err := Load()
	if err != nil {
		t.Fatalf("obsolete stream port must not prevent startup: %v", err)
	}
}

func TestLoad_DistinctAddrs_Succeeds(t *testing.T) {
	t.Setenv("ALREADYON_WEB_PORT", "19900")
	t.Setenv("ALREADYON_STREAM_PORT", "19901")

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestNextHourlyAtIsOffsetPastTheHour(t *testing.T) {
	loc := time.FixedZone("test", -4*3600)
	at := func(h, m, s int) time.Time { return time.Date(2026, 10, 1, h, m, s, 0, loc) }
	for _, c := range []struct {
		now    time.Time
		offset time.Duration
		want   time.Time
	}{
		{at(12, 0, 0), 15 * time.Minute, at(12, 15, 0)},
		{at(12, 14, 59), 15 * time.Minute, at(12, 15, 0)},
		{at(12, 15, 0), 15 * time.Minute, at(13, 15, 0)}, // never "now"
		{at(12, 45, 0), 30 * time.Minute, at(13, 30, 0)},
		{at(23, 50, 0), 0, time.Date(2026, 10, 2, 0, 0, 0, 0, loc)},
	} {
		if got := NextHourlyAt(c.now, c.offset); !got.Equal(c.want) {
			t.Errorf("NextHourlyAt(%v, %v) = %v, want %v", c.now, c.offset, got, c.want)
		}
	}
}
