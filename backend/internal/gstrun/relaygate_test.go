package gstrun

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-gst/go-gst/pkg/gst"

	"bjoernblessin.de/mirrorme/internal/padprobe"
)

// A decoder needs every frame back to the last keyframe, so a drop runs to the next keyframe
// and no frame after a hole gets through.
func TestTheGateDropsFromABacklogToTheNextKeyframeWithRoom(t *testing.T) {
	const (
		keyframe = true
		delta    = false
		backedUp = true
		room     = false
	)
	gate := &relayGate{}
	for _, step := range []struct {
		keyframe, backlog, admitted bool
		dropped                     uint64
	}{
		{delta, room, true, 0},
		{keyframe, room, true, 0},
		{delta, backedUp, false, 1},
		{delta, room, false, 2},
		{keyframe, backedUp, false, 3},
		{delta, room, false, 4},
		{keyframe, room, true, 4},
		{delta, room, true, 4},
	} {
		if got := gate.admit(step.keyframe, step.backlog); got != step.admitted {
			t.Fatalf("admit(keyframe=%v, backlog=%v) = %v at %d dropped", step.keyframe, step.backlog, got, gate.dropped.Load())
		}
		if got, _ := gate.Read(); got != step.dropped {
			t.Fatalf("dropped %d, want %d", got, step.dropped)
		}
	}
}

func TestAPipelineWithNoGateReportsNoDrop(t *testing.T) {
	var gate *relayGate
	if _, counted := gate.Read(); counted {
		t.Error("a run with no gate reported a drop count")
	}
}

// A relay leg at 10 fps costs the preview branch nothing and the encoder keeps its rate.
func TestASlowRelayLegLeavesThePreviewBranchItsRate(t *testing.T) {
	gst.Init()
	if gst.ElementFactoryFind("x264enc") == nil {
		t.Skip("this GStreamer registers no x264enc")
	}

	const description = "videotestsrc is-live=true pattern=ball ! video/x-raw,format=I420,width=320,height=180,framerate=60/1" +
		" ! queue max-size-buffers=1 leaky=downstream" +
		" ! x264enc tune=zerolatency speed-preset=ultrafast key-int-max=30 ! h264parse config-interval=-1 ! tee name=t" +
		" t. ! queue max-size-buffers=8 leaky=downstream ! fakesink name=preview sync=false" +
		" t. ! queue name=relay max-size-buffers=0 max-size-bytes=0 ! identity sleep-time=100000 ! fakesink sync=false"
	el, err := gst.ParseLaunch(description)
	if err != nil {
		t.Fatal(err)
	}
	pipeline := el.(gst.Pipeline)
	defer pipeline.SetState(gst.StateNull)

	gate := watchRelayGate(pipeline, "relay")
	if gate == nil {
		t.Fatal("the pipeline holds the queue and no gate went on")
	}
	var frames atomic.Uint64
	pipeline.GetByName("preview").GetStaticPad("sink").AddProbe(gst.PadProbeTypeBuffer,
		func(_ gst.Pad, info *gst.PadProbeInfo) gst.PadProbeReturn {
			if padprobe.Buffer(info) != nil {
				frames.Add(1)
			}
			return gst.PadProbeOK
		})

	pipeline.SetState(gst.StatePlaying)
	const window = 4 * time.Second
	time.Sleep(window)

	// The first second goes to startup, the relay queue filling inside it.
	if got, want := frames.Load(), uint64(window.Seconds()-1)*60; got < want {
		t.Errorf("the preview branch took %d frames, want at least %d", got, want)
	}
	if dropped, _ := gate.Read(); dropped == 0 {
		t.Error("a relay leg at 10 fps dropped nothing, so it held the encoder up")
	}
}

// A viewer decodes from the last keyframe, so a frame behind a gap must be a keyframe.
// The muxer waits on both pads, so the audio branch must survive the video pad going quiet.
func TestARelayLegResumesOnAKeyframe(t *testing.T) {
	gst.Init()
	if gst.ElementFactoryFind("x264enc") == nil {
		t.Skip("this GStreamer registers no x264enc")
	}

	const encoded = "videotestsrc is-live=true ! video/x-raw,format=I420,width=640,height=360,framerate=60/1" +
		" ! x264enc tune=zerolatency speed-preset=ultrafast key-int-max=30 ! h264parse config-interval=-1"
	const slow = "identity sleep-time=100000 ! fakesink sync=false"

	for name, description := range map[string]string{
		"video alone": encoded + " ! queue name=relay max-size-buffers=0 max-size-bytes=0 ! " + slow,
		"with audio": encoded + " ! queue name=relay max-size-buffers=0 max-size-bytes=0 ! mpegtsmux name=mux ! identity datarate=200000 ! fakesink sync=true" +
			" audiotestsrc is-live=true ! audioconvert ! opusenc ! queue ! mux.",
	} {
		t.Run(name, func(t *testing.T) {
			el, err := gst.ParseLaunch(description)
			if err != nil {
				t.Fatal(err)
			}
			pipeline := el.(gst.Pipeline)
			defer pipeline.SetState(gst.StateNull)

			gate := watchRelayGate(pipeline, "relay")
			var frames, gaps atomic.Uint64
			var previous gst.ClockTime
			pipeline.GetByName("relay").GetStaticPad("src").AddProbe(gst.PadProbeTypeBuffer,
				func(_ gst.Pad, info *gst.PadProbeInfo) gst.PadProbeReturn {
					buffer := padprobe.Buffer(info)
					if buffer == nil {
						return gst.PadProbeOK
					}
					if frames.Add(1) > 1 && buffer.PTS()-previous > gst.ClockTime(30*time.Millisecond) {
						gaps.Add(1)
						if buffer.HasFlags(gst.BufferFlagDeltaUnit) {
							t.Error("a delta frame follows a gap, which a viewer decodes without its references")
						}
					}
					previous = buffer.PTS()
					return gst.PadProbeOK
				})

			pipeline.SetState(gst.StatePlaying)
			time.Sleep(3 * time.Second)
			before := frames.Load()
			time.Sleep(2 * time.Second)

			if frames.Load() == before {
				t.Errorf("the relay leg stopped at %d frames", before)
			}
			if gaps.Load() == 0 {
				t.Error("the gate left no gap, so nothing here tested a resume")
			}
			if dropped, _ := gate.Read(); dropped == 0 {
				t.Error("a relay leg at 10 fps dropped nothing")
			}
		})
	}
}
