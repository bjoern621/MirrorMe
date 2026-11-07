package publish

import (
	"slices"
	"strings"
	"testing"
	"time"

	"bjoernblessin.de/mirrorme/internal/gstrun"
	"bjoernblessin.de/mirrorme/internal/settings"
)

func congestedStream(t *testing.T, congestion string) settings.Settings {
	t.Helper()
	s := previewStream(t, "portal", "libx264")
	s.Publish.Congestion = congestion
	return s
}

// The rendered command names the gated queue, so SamePipeline sees a change of policy and relaunches.
func TestTheCongestionPolicyChangesTheRenderedPipeline(t *testing.T) {
	gated := "queue name=" + gstRelayName + " max-size-buffers=0 max-size-bytes=0 !"

	for _, tc := range []struct {
		congestion string
		wantGated  bool
	}{
		{settings.CongestionKeyframe, true},
		{settings.CongestionHold, false},
	} {
		pipeline, err := buildPipeline(congestedStream(t, tc.congestion), []string{"videotestsrc"}, "", PreviewLeg{})
		if err != nil {
			t.Fatalf("%s: %v", tc.congestion, err)
		}
		if line := strings.Join(pipeline, " "); strings.Contains(line, gated) != tc.wantGated {
			t.Errorf("%s: gated queue present = %v, want %v: %s", tc.congestion, !tc.wantGated, tc.wantGated, line)
		}
	}
}

// The gated queue is the trunk's last, so every tap off the tee stays ahead of the gate.
func TestTheRelayQueueSitsBehindEveryTap(t *testing.T) {
	pipeline, err := buildPipeline(congestedStream(t, settings.CongestionKeyframe), []string{"videotestsrc"}, "54321", PreviewLeg{Port: 45678})
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Join(pipeline, " ")
	if strings.Index(line, "name="+gstRelayName) < strings.LastIndex(line, gstTeeName+". !") {
		t.Errorf("a tap branches off behind the gate: %s", line)
	}
}

func TestAnUnknownCongestionPolicyIsRefused(t *testing.T) {
	if _, err := buildPipeline(congestedStream(t, "sometimes"), []string{"videotestsrc"}, "", PreviewLeg{}); err == nil {
		t.Error("a policy no table carries built a pipeline")
	}
}

func TestTheChildGatesTheRelayQueueUnderTheKeyframePolicy(t *testing.T) {
	flag := gstrun.RelayGateFlag + gstRelayName

	if !slices.Contains(gstChildArgs(congestedStream(t, settings.CongestionKeyframe), "socket", true), flag) {
		t.Errorf("the keyframe policy leaves %s off the child", flag)
	}
	if slices.Contains(gstChildArgs(congestedStream(t, settings.CongestionHold), "socket", true), flag) {
		t.Errorf("the hold policy passes %s to the child", flag)
	}
}

// Frames the gate keeps off the relay leg are frames no viewer receives,
// so the rate reads what reached that leg and the drop count carries both ends of the shortfall.
func TestGstMeterRateReadsFramesReachingTheRelay(t *testing.T) {
	wall := time.Unix(0, 0)
	var got []Stats
	m := &gstMeter{
		onStats: func(s Stats) { got = append(got, s) },
		now:     func() time.Time { return wall },
	}
	dropped := func(n uint64) *uint64 { return &n }

	m.takeDelay(gstrun.Delay{Frames: 60, Dropped: dropped(2), RelayDropped: dropped(0)})
	m.parse(strings.NewReader("stats (00:00:01): 60 buffers\n"))

	wall = wall.Add(time.Second)
	m.takeDelay(gstrun.Delay{Frames: 120, Dropped: dropped(2), RelayDropped: dropped(45)})
	m.parse(strings.NewReader("stats (00:00:02): 120 buffers\n"))

	if len(got) != 2 {
		t.Fatalf("got %d samples, want 2", len(got))
	}
	if got[1].Fps != 15 {
		t.Errorf("fps = %v, want the 15 of 60 frames that reached the relay", got[1].Fps)
	}
	if got[1].Drop != 47 {
		t.Errorf("drop = %d, want the 2 shed plus the 45 gated", got[1].Drop)
	}
}

// The gate's count lands on its own tick, so one interval can hold more drops than encoded frames.
func TestGstMeterRateStopsAtZero(t *testing.T) {
	wall := time.Unix(0, 0)
	var got []Stats
	m := &gstMeter{
		onStats: func(s Stats) { got = append(got, s) },
		now:     func() time.Time { return wall },
	}
	dropped := func(n uint64) *uint64 { return &n }

	m.takeDelay(gstrun.Delay{Frames: 60, RelayDropped: dropped(0)})
	m.parse(strings.NewReader("stats (00:00:01): 60 buffers\n"))

	wall = wall.Add(time.Second)
	m.takeDelay(gstrun.Delay{Frames: 120, RelayDropped: dropped(70)})
	m.parse(strings.NewReader("stats (00:00:02): 120 buffers\n"))

	if got[1].Fps != 0 {
		t.Errorf("fps = %v, want 0", got[1].Fps)
	}
}
