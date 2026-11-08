package gstrun

import (
	"testing"
	"time"

	"github.com/go-gst/go-gst/pkg/gst"
)

func audioGateDrops(t *testing.T, description string) bool {
	t.Helper()
	gst.Init()
	el, err := gst.ParseLaunch(description)
	if err != nil {
		t.Fatalf("building the pipeline: %v", err)
	}
	pipeline := el.(gst.Pipeline)
	defer pipeline.SetState(gst.StateNull)

	holdAudioUntilVideo(pipeline)
	pipeline.SetState(gst.StatePlaying)

	gate := pipeline.GetByName(AudioGateName)
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if drop, _ := gate.ObjectProperty("drop").(bool); !drop {
			return false
		}
	}
	return true
}

func TestTheAudioGateOpensOnTheFirstFrame(t *testing.T) {
	if audioGateDrops(t, "fakesrc num-buffers=3 ! queue name="+RelayQueueName+" ! fakesink async=false "+
		"fakesrc is-live=true ! valve name="+AudioGateName+" drop=true ! fakesink async=false") {
		t.Fatal("the gate stayed shut after a frame left the relay queue")
	}
}

func TestTheAudioGateStaysShutUntilAFrameArrives(t *testing.T) {
	if !audioGateDrops(t, "fakesrc num-buffers=0 ! queue name="+RelayQueueName+" ! fakesink async=false "+
		"fakesrc is-live=true ! valve name="+AudioGateName+" drop=true ! fakesink async=false") {
		t.Fatal("the gate opened with no frame in the relay queue")
	}
}
