package gstelements

import (
	"runtime"
	"testing"
	"time"

	"github.com/go-gst/go-gst/pkg/gst"
)

// refCount reads GObject.ref_count, which sits behind the GTypeInstance's one pointer.
func refCount(element gst.Element) uint32 {
	object := (*struct {
		class uintptr
		count uint32
	})(gst.UnsafeElementToGlibNone(element))
	return object.count
}

// A reference left by a walk keeps the element and its device alive after the pipeline is gone.
func TestAWalkLeavesNoReferenceBehind(t *testing.T) {
	gst.Init()

	parsed, err := gst.ParseLaunch("fakesrc ! fakesink name=sink")
	if err != nil {
		t.Fatalf("building a pipeline: %v", err)
	}
	pipeline := parsed.(gst.Pipeline)
	defer pipeline.SetState(gst.StateNull)

	sink := pipeline.GetByName("sink")
	before := refCount(sink)

	if walked := Of(pipeline.IterateRecurse()); len(walked) != 2 {
		t.Fatalf("the walk found %d elements, want the source and the sink", len(walked))
	}

	deadline := time.Now().Add(5 * time.Second)
	for refCount(sink) != before {
		if time.Now().After(deadline) {
			t.Fatalf("the sink holds %d references after the walk, want %d", refCount(sink), before)
		}
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
}
