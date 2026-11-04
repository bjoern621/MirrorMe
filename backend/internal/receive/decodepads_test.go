package receive

import (
	"testing"
	"time"

	"github.com/go-gst/go-gst/pkg/gst"
)

// TestDecodedAudioPadReadsAsAudio holds the audio branch to the pad the decode bin exposes.
// decodebin3 exposes a track's pad with no caps yet at pad-added.
// A check reading caps alone leaves an MPEG-TS stream's sound off.
func TestDecodedAudioPadReadsAsAudio(t *testing.T) {
	initGStreamer()
	for _, need := range []string{"opusenc", decodeFactory} {
		if gst.ElementFactoryFind(need) == nil {
			t.Skipf("this GStreamer registers no %s", need)
		}
	}

	el, err := gst.ParseLaunch("audiotestsrc num-buffers=50 ! opusenc ! " + decodeFactory + " name=" + decodeName)
	if err != nil {
		t.Fatalf("the audio decode does not parse: %v", err)
	}
	pipeline := el.(gst.Pipeline)
	defer pipeline.SetState(gst.StateNull)

	audio := make(chan bool, 4)
	pipeline.GetByName(decodeName).Connect("pad-added", func(_ gst.Element, pad gst.Pad) {
		audio <- isAudioPad(pad)
	})
	pipeline.SetState(gst.StatePlaying)

	select {
	case got := <-audio:
		if !got {
			t.Error("the pad an Opus track was exposed on reads as no audio")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the decode bin exposed no pad for an Opus track")
	}
}
