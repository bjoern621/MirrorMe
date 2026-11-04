package receive

import (
	"strings"
	"testing"

	"github.com/go-gst/go-gst/pkg/gst"
	"github.com/go-gst/go-gst/pkg/gstapp"
)

// TestGLChainDecodesIntoDeviceMemory holds the GL chain to the copy it exists to avoid.
// A hardware decoder handing over system memory has the CPU read every frame off the GPU
// for glupload to put straight back (the gl row in chains.go says what that costs).
//
// Skips where this GStreamer registers no VA H.264 encoder and decoder,
// the bitstream being made here and the claim being about a hardware decoder.
func TestGLChainDecodesIntoDeviceMemory(t *testing.T) {
	initGStreamer()
	c := chainNamed("gl")
	for _, need := range append([]string{"vah264enc", "vah264dec"}, c.needs...) {
		if gst.ElementFactoryFind(need) == nil {
			t.Skipf("this GStreamer registers no %s", need)
		}
	}

	src := "videotestsrc num-buffers=30 ! video/x-raw,width=640,height=360,framerate=30/1 ! vah264enc ! h264parse"
	el, err := gst.ParseLaunch(c.launch(Stream{Source: src}, toneMapRung{}))
	if err != nil {
		t.Fatalf("the GL launch line does not parse: %v", err)
	}
	pipeline := el.(gst.Pipeline)
	defer pipeline.SetState(gst.StateNull)
	sink, ok := pipeline.GetByName(sinkName).(gstapp.AppSink)
	if !ok {
		t.Fatalf("the GL launch line holds no appsink named %q", sinkName)
	}
	pipeline.SetState(gst.StatePlaying)

	sample := sink.PullSample()
	if sample == nil {
		t.Fatal("the GL chain drew no frame of the VA-encoded stream")
	}
	defer gst.UnsafeSampleUnref(sample)

	dec := videoDecoderOf(pipeline)
	if dec == nil {
		t.Fatal("the pipeline drew a frame and holds no video decoder")
	}
	if got := memoryOf(padCaps(dec, "src")); got != "memory:DMABuf" {
		t.Errorf("%s hands the GL chain %s, which the CPU copies off the GPU for every frame",
			dec.GetFactory().GetName(), got)
	}

	// dmabuf caps name their layout in drm-format and carry DMA_DRM as the format,
	// which names no depth and no subsampling.
	var s Stats
	readDecoded(&s, dec)
	if s.Format != "NV12" || s.Depth != 8 || s.Subsampling != "4:2:0" {
		t.Errorf("an 8-bit 4:2:0 encode reads back as format %q, depth %d, subsampling %q",
			s.Format, s.Depth, s.Subsampling)
	}
}

// videoDecoderOf is the video decoder the bin autoplugged, nil where it built none.
func videoDecoderOf(pipeline gst.Pipeline) gst.Element {
	for v := range pipeline.IterateRecurse().Values() {
		e, ok := v.(gst.Element)
		if !ok || e.GetFactory() == nil {
			continue
		}
		klass := e.GetFactory().GetMetadata("klass")
		if strings.Contains(klass, klassDecoder) && strings.Contains(klass, "Video") {
			return e
		}
	}
	return nil
}
