package playback

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestChooseCapsAutoQualityAtTheServersMeasuredLimit(t *testing.T) {
	s := sourceInfo() // 1080p60
	caps := clientCaps()
	caps.ServerMaxHeight, caps.ServerMaxFPS = 720, 30
	p, err := Choose(s, caps, true, true)
	if err != nil || p.Mode != "transcode" || p.Height != 720 || p.FPS > 30 || p.VideoBitrate > 5000000 || !p.ServerLimited {
		t.Fatalf("plan = %+v %v, want a limited 720p30 conversion within 5 Mbps", p, err)
	}
	d := Describe("/m/a.mp4", s, p, caps)
	if !strings.Contains(d.VideoReason, "720p30") || !strings.Contains(d.VideoReason, "real time") {
		t.Errorf("details do not explain the cap: %q", d.VideoReason)
	}

	// Direct play needs no encoding, so the cap never applies to it.
	if p, err := Choose(s, caps, true, false); err != nil || p.Mode != "direct" {
		t.Errorf("direct play was affected: %+v %v", p, err)
	}
	// A hardware encoder, or a size the viewer picked, is not capped.
	hw := caps
	hw.Encoder = EncoderIntelVAAPI
	if p, _ := Choose(s, hw, true, true); p.Height != 1080 || p.ServerLimited {
		t.Errorf("hardware encoding was capped: %+v", p)
	}
	picked := caps
	picked.MaxHeight = 1080
	if p, _ := Choose(s, picked, true, true); p.Height != 1080 || p.ServerLimited {
		t.Errorf("a viewer's choice was capped: %+v", p)
	}
	// A source already within the limit is not "limited".
	small := sourceInfo()
	small.Streams[0].Width, small.Streams[0].Height, small.Streams[0].FrameRate = 1280, 720, 24
	if p, _ := Choose(small, caps, true, true); p.ServerLimited || p.Height != 720 {
		t.Errorf("a source within the limit was marked limited: %+v", p)
	}
}

func TestCapacityLimitAppliesOnlyToTheMeasuredSoftwareEncoder(t *testing.T) {
	var nilCapacity *Capacity
	if h, _ := nilCapacity.Limit(""); h != 0 {
		t.Fatalf("a missing capacity must not limit: %d", h)
	}
	c := &Capacity{}
	c.Load(CapacityRecord{MeasuredAt: time.Now(), MaxHeight: 720, MaxFPS: 30, Processes: 3})
	if h, f := c.Limit(""); h != 720 || f != 30 {
		t.Errorf("software limit = %dp%g, want 720p30", h, f)
	}
	if h, _ := c.Limit(EncoderIntelQSV); h != 0 {
		t.Errorf("a hardware encoder in use must not be limited by a software measurement: %d", h)
	}
	c.Load(CapacityRecord{MeasuredAt: time.Now(), Encoder: EncoderIntelQSV, MaxHeight: 0, Processes: 5})
	if h, _ := c.Limit(""); h != 0 {
		t.Errorf("a measurement for another encoder must not limit software: %d", h)
	}
}

func TestCapacityMeasuresDownTheLadderAndStopsAtTheFirstSustainedSize(t *testing.T) {
	speeds := map[int]float64{1080: 0.7, 720: 1.0, 480: 2.0, 360: 4.0} // by height; 60 fps is half as fast
	c := &Capacity{
		Threads: func() int { return 4 },
		measure: func(_ context.Context, r RungSample, _ int, _, _ string) (float64, error) {
			s := speeds[r.Height]
			if r.FPS > 30 {
				s /= 2
			}
			return s, nil
		},
		memory: func() uint64 { return 16 << 30 },
		cpus:   8,
	}
	var saved CapacityRecord
	c.Save = func(_ context.Context, r CapacityRecord) { saved = r }
	samples, err := c.benchmark(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	// 1080p60 .35x, 1080p30 .7x, 720p60 .5x, 720p30 1.0x, 480p30 2.0x: first at or above 1.3x is 480p30.
	if h, f := pickRung("", samples); h != 480 || f != 30 || len(samples) != 5 {
		t.Fatalf("picked %dp%g from %d samples %+v, want 480p30 after 5", h, f, len(samples), samples)
	}

	c.Load(CapacityRecord{})
	c.mu.Lock()
	c.status.MeasuredAt, c.status.Samples = &time.Time{}, samples
	c.status.MaxHeight, c.status.MaxFPS = pickRung("", samples)
	c.mu.Unlock()
	c.RecomputeProcesses(context.Background())
	// 0.8 * 8 CPUs * 2.0x / 4 threads = 3.2 conversions.
	if got := c.Processes(); got != 3 {
		t.Errorf("processes = %d, want 3", got)
	}
	if saved.Processes != 3 || saved.MaxHeight != 480 {
		t.Errorf("saved %+v, want the processes and limit stored", saved)
	}
}

func TestCapacityWhenNothingKeepsUpUsesTheSmallestSize(t *testing.T) {
	c := &Capacity{measure: func(context.Context, RungSample, int, string, string) (float64, error) { return 0.5, nil }}
	samples, err := c.benchmark(context.Background(), "")
	if err != nil || len(samples) != len(capacityRungs) {
		t.Fatalf("samples = %+v, %v; want every size tried", samples, err)
	}
	if h, f := pickRung("", samples); h != 360 || f != 30 {
		t.Errorf("picked %dp%g, want the smallest, 360p30", h, f)
	}
}

func TestCapacityHardwareMeasuresOneSizeAndSetsNoLimit(t *testing.T) {
	calls := 0
	c := &Capacity{measure: func(_ context.Context, r RungSample, threads int, encoder, _ string) (float64, error) {
		calls++
		if r.Height != 1080 || r.FPS != 30 || threads != 1 || encoder != EncoderIntelVAAPI {
			t.Errorf("unexpected measurement %+v threads=%d encoder=%q", r, threads, encoder)
		}
		return 6, nil
	}}
	samples, err := c.benchmark(context.Background(), EncoderIntelVAAPI)
	if err != nil || calls != 1 {
		t.Fatalf("samples %+v err %v calls %d", samples, err, calls)
	}
	if h, _ := pickRung(EncoderIntelVAAPI, samples); h != 0 {
		t.Errorf("a hardware encoder must have no size limit, got %d", h)
	}
}

func TestCapacityMeasurementFailureKeepsThePreviousResult(t *testing.T) {
	c := &Capacity{measure: func(context.Context, RungSample, int, string, string) (float64, error) { return 0, errors.New("boom") }}
	if _, err := c.benchmark(context.Background(), ""); err == nil {
		t.Fatal("a failed measurement must be reported")
	}
}

func TestDetectProcesses(t *testing.T) {
	for _, tc := range []struct {
		name          string
		speed         float64
		cpus, threads int
		free          uint64
		encoder       string
		want          int
	}{
		{"small NAS", 1.5, 4, 2, 16 << 30, "", 2},
		{"big desktop", 6, 18, 8, 32 << 30, "", 10},
		{"slower than real time still gets one", 0.3, 4, 4, 16 << 30, "", 1},
		{"memory bound", 6, 18, 2, 1 << 30, "", 2},
		{"never above the supported maximum", 50, 64, 2, 64 << 30, "", 16},
		{"nvenc session limit", 20, 16, 1, 0, EncoderNvidiaNVENC, 8},
		{"hardware ignores free memory", 20, 4, 1, 1 << 20, EncoderIntelVAAPI, 16},
	} {
		if got := DetectProcesses(tc.speed, tc.cpus, tc.threads, tc.free, tc.encoder); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestEffectiveSettingsFollowTheDetectedLimitUnlessTheAdminChoseOne(t *testing.T) {
	c := &Capacity{}
	c.Load(CapacityRecord{MeasuredAt: time.Now(), Processes: 6, MaxHeight: 1080, MaxFPS: 60})
	s := DefaultSettings()
	if got := EffectiveSettings(s, c).MaxProcesses; got != 6 {
		t.Errorf("auto limit = %d, want the detected 6", got)
	}
	s.AutoProcesses, s.MaxProcesses = false, 3
	if got := EffectiveSettings(s, c).MaxProcesses; got != 3 {
		t.Errorf("manual limit = %d, want the administrator's 3", got)
	}
	s = DefaultSettings()
	if got := EffectiveSettings(s, &Capacity{}).MaxProcesses; got != 2 {
		t.Errorf("unmeasured auto limit = %d, want the default 2", got)
	}
	if err := EffectiveSettings(s, c).Validate(); err != nil {
		t.Errorf("effective settings must stay valid: %v", err)
	}
}

func TestCapacityMeasuresRealFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}
	// State becomes idle before the save callback; synchronize on persistence
	// rather than racing the worker's write to an ordinary local variable.
	records := make(chan CapacityRecord, 1)
	c := &Capacity{Context: t.Context(), Threads: func() int { return 2 }, Save: func(_ context.Context, r CapacityRecord) { records <- r }}
	c.Start(true)
	var saved CapacityRecord
	select {
	case saved = <-records:
	case <-time.After(3 * time.Minute):
		t.Fatal("capacity measurement did not save its result")
	}
	s := c.Status()
	if s.MeasuredAt == nil || len(s.Samples) == 0 || s.MaxHeight == 0 || s.Processes < 1 {
		t.Fatalf("status = %+v, want a measurement with a size limit and a parallel limit", s)
	}
	for _, r := range s.Samples {
		if r.Speed <= 0 {
			t.Errorf("sample %+v has no speed", r)
		}
	}
	if saved.Processes != s.Processes || saved.MaxHeight != s.MaxHeight {
		t.Errorf("saved %+v does not match %+v", saved, s)
	}
	t.Logf("measured %s; %d parallel", describeRungs(s.Samples), s.Processes)
}
