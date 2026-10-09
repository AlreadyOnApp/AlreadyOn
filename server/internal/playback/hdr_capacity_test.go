package playback

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHDRLimitMatchesPipelineAndLeavesExplicitQualityAlone(t *testing.T) {
	src, caps := detailsFixture()
	v, _ := Tracks(src)
	c := &Capacity{hdr: []HDRCapacity{{SourceFPS: 30, Encoder: EncoderIntelVAAPI, HardwareDecode: true, Transfer: "smpte2084", ToneMapper: "cpu", Threads: 2, MeasuredAt: time.Now(), Samples: []RungSample{{Height: 1080, FPS: 30, Speed: .8}, {Height: 720, FPS: 30, Speed: 1.6}}}}}
	h, f, ok := c.HDRLimit(EncoderIntelVAAPI, true, v, 2)
	if !ok || h != 720 || f != 30 {
		t.Fatalf("limit %d %g %t", h, f, ok)
	}
	for _, tc := range []struct {
		encoder string
		decode  bool
		threads int
	}{{EncoderIntelQSV, true, 2}, {EncoderIntelVAAPI, false, 2}, {EncoderIntelVAAPI, true, 4}} {
		if _, _, ok := c.HDRLimit(tc.encoder, tc.decode, v, tc.threads); ok {
			t.Fatal("mismatched pipeline limited")
		}
	}
	caps.Encoder, caps.HWDecode, caps.HDRMeasured = EncoderIntelVAAPI, true, true
	caps.ServerMaxHeight, caps.ServerMaxFPS = h, f
	p, err := Choose(src, caps, false, true)
	if err != nil || p.Height != 720 || !p.ServerLimited {
		t.Fatalf("auto %+v %v", p, err)
	}
	caps.MaxHeight = 1080
	p, err = Choose(src, caps, false, true)
	if err != nil || p.Height != 1080 || p.ServerLimited {
		t.Fatalf("explicit %+v %v", p, err)
	}
	caps.MaxHeight = 0
	src.Streams[0].ColorTransfer = "bt709"
	p, err = Choose(src, caps, false, true)
	if err != nil || p.Height != 1080 {
		t.Fatalf("SDR %+v %v", p, err)
	}
	c.hdr[0].MeasuredAt = time.Now().Add(-3 * time.Hour)
	if _, _, ok := c.HDRLimit(EncoderIntelVAAPI, true, v, 2); ok {
		t.Fatal("stale capacity used")
	}
}

func TestHDRMixedPipelineUsesCPUFilterThreads(t *testing.T) {
	j := hdrJob(t, EncoderIntelVAAPI, true, false)
	args, err := Arguments(j)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(args, " "), "-filter_threads 4") {
		t.Fatal("CPU tone mapping has no parallel filter budget")
	}
	j.Plan.ToneMapper = "gpu"
	args, err = Arguments(j)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(args, " "), "-filter_threads 4") {
		t.Fatal("GPU path changed")
	}
}

func TestHDRFailureCacheDoesNotDisableDecoder(t *testing.T) {
	now := time.Now()
	h := &Hardware{now: func() time.Time { return now }, filters: " .. tonemap_vaapi V->V"}
	h.filtersOnce.Do(func() {})
	if !h.ToneMapOnGPU(context.Background(), EncoderIntelVAAPI) {
		t.Fatal("initial candidate unavailable")
	}
	h.ToneMapFailed(EncoderIntelVAAPI, errors.New("no HDR VPP"))
	if h.ToneMapOnGPU(context.Background(), EncoderIntelVAAPI) {
		t.Fatal("failed tone mapper retried")
	}
	src, _ := detailsFixture()
	v, _ := Tracks(src)
	if !h.DecodeAllowed(EncoderIntelVAAPI, v) {
		t.Fatal("unrelated decoder disabled")
	}
	now = now.Add(hardwareDecodeFailFor)
	if !h.ToneMapOnGPU(context.Background(), EncoderIntelVAAPI) {
		t.Fatal("retry never allowed")
	}
}

// Long-lived streams must remember unsupported VPP before viewers leave.
func TestHDRFallbackCachesFailureAtFirstSegment(t *testing.T) {
	exe := scriptedFFmpeg(t, "tonemap_vaapi")
	f, err := os.OpenFile(exe, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("sleep 2\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	notified := make(chan struct{}, 1)
	r := Runner{Executable: exe, OnToneMapFailure: func(string, error) {
		select {
		case notified <- struct{}{}:
		default:
		}
	}}
	job := hdrJob(t, EncoderIntelVAAPI, true, true)
	job.Duration = 5 * time.Second
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx, job) }()
	select {
	case <-notified:
		select {
		case err := <-done:
			t.Fatalf("callback arrived only after exit: %v", err)
		default:
		}
	case <-time.After(time.Second):
		t.Fatal("working CPU fallback did not cache GPU failure promptly")
	}
	cancel()
	<-done
}

func TestHDRDropsExcessFramesBeforeCPUToneMapping(t *testing.T) {
	for _, hw := range []bool{false, true} {
		j := hdrJob(t, EncoderIntelVAAPI, hw, false)
		j.Plan.FPS = 30
		args, err := Arguments(j)
		if err != nil {
			t.Fatal(err)
		}
		var filter string
		for i := range args {
			if args[i] == "-vf" {
				filter = args[i+1]
			}
		}
		fps, tone := strings.Index(filter, "fps=30"), strings.Index(filter, "tonemap=")
		if fps < 0 || tone < 0 || fps > tone {
			t.Fatalf("excess frames reach CPU tone map: %s", filter)
		}
	}
}

func TestHDRUsesFilmRateMeasurementsWithoutCappingFasterSourcesByThem(t *testing.T) {
	src, _ := detailsFixture()
	v, _ := Tracks(src)
	c := &Capacity{hdr: []HDRCapacity{
		{SourceFPS: 24, Transfer: "smpte2084", ToneMapper: "cpu", Threads: 2, MeasuredAt: time.Now(), Samples: []RungSample{{Height: 1080, FPS: 24, Speed: 1.5}}},
		{SourceFPS: 30, Transfer: "smpte2084", ToneMapper: "cpu", Threads: 2, MeasuredAt: time.Now(), Samples: []RungSample{{Height: 1080, FPS: 30, Speed: .8}, {Height: 720, FPS: 30, Speed: 1.5}}},
	}}
	if h, _, ok := c.HDRLimit("", false, v, 2); !ok || h != 1080 {
		t.Fatal("film inherited faster source limit")
	}
	v.FrameRate = 30
	if h, _, ok := c.HDRLimit("", false, v, 2); !ok || h != 720 {
		t.Fatal("faster source used film measurement")
	}
	v.Codec = "av1"
	if _, _, ok := c.HDRLimit("", false, v, 2); ok {
		t.Fatal("unmeasured codec used HEVC result")
	}
}
