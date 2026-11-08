package gstrun

import (
	"sync"
	"time"

	"github.com/go-gst/go-gst/pkg/gst"

	"bjoernblessin.de/mirrorme/internal/padprobe"
)

// AudioGateName names the valve that keeps a publish's sound off the muxer until video reaches it.
const AudioGateName = "audiogate"

// RelayQueueName names the queue that hands the video to the muxer.
const RelayQueueName = "relay"

// audioGateWait bounds how long the first frame waits for sound that may never come.
const audioGateWait = 500 * time.Millisecond

// holdAudioUntilVideo makes both tracks known to the muxer before its first program table.
// Nothing happens in a pipeline without a gate.
//
// A muxer's first table declares the tracks whose caps have arrived by then, and a relay holds to that table.
// Sound starting ahead of the encoder would be the only track that viewers get, and sound starting after it would be lost.
// The first frame's caps have passed the queue when its buffer gets here.
// So the gate opens at that point, and the frame waits until the sound's caps have passed the gate.
func holdAudioUntilVideo(pipeline gst.Pipeline) {
	gate := pipeline.GetByName(AudioGateName)
	if gate == nil {
		return
	}
	queue := pipeline.GetByName(RelayQueueName)
	if queue == nil {
		gate.SetObjectProperty("drop", false)
		return
	}

	soundPassed := make(chan struct{})
	var once sync.Once
	gate.GetStaticPad("src").AddProbe(gst.PadProbeTypeBuffer, func(_ gst.Pad, info *gst.PadProbeInfo) gst.PadProbeReturn {
		if padprobe.Buffer(info) == nil {
			return gst.PadProbeOK
		}
		once.Do(func() { close(soundPassed) })
		return gst.PadProbeRemove
	})
	queue.GetStaticPad("src").AddProbe(gst.PadProbeTypeBuffer, func(_ gst.Pad, info *gst.PadProbeInfo) gst.PadProbeReturn {
		if padprobe.Buffer(info) == nil {
			return gst.PadProbeOK
		}
		gate.SetObjectProperty("drop", false)
		select {
		case <-soundPassed:
		case <-time.After(audioGateWait):
		}
		return gst.PadProbeRemove
	})
}
