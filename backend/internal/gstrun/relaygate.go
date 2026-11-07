package gstrun

import (
	"sync/atomic"
	"time"

	"github.com/go-gst/go-gst/pkg/gst"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/padprobe"
)

// RelayGateFlag names the queue ahead of the relay sink, the name following it.
const RelayGateFlag = "--relay-gate="

// relayBacklog is how far the relay leg falls behind before the gate drops frames.
// It sits under the queue's 1 s bound, since a full queue blocks the tee and every branch off it.
const relayBacklog = 500 * time.Millisecond

// relayGate keeps a relay that cannot keep up from holding up the encoder.
//
// One encoder feeds every branch off the tee, so a relay queue that fills stops the local preview
// and the meter as well.
// Past relayBacklog the gate drops frames ahead of that queue, up to the next keyframe with room
// behind it.
// A decoder needs every frame back to the last keyframe, so the relay leg freezes for the rest of
// the GOP and sends no frame that it cannot decode.
type relayGate struct {
	// dropping holds across frames.
	// Only the streaming thread of this branch touches it.
	dropping bool
	// dropped is written on the streaming thread and read on the reporting one.
	dropped atomic.Uint64
}

// admit reports whether one frame enters the relay queue.
// The caller must ask once per frame, in stream order.
func (g *relayGate) admit(keyframe, backedUp bool) bool {
	if g.dropping && keyframe && !backedUp {
		g.dropping = false
	}
	if backedUp {
		g.dropping = true
	}
	if g.dropping {
		g.dropped.Add(1)
	}
	return !g.dropping
}

// Read is what the gate has dropped, and false for a pipeline carrying none, which a nil gate stands for.
func (g *relayGate) Read() (uint64, bool) {
	if g == nil {
		return 0, false
	}
	return g.dropped.Load(), true
}

// watchRelayGate gates the named queue, and answers nil where the pipeline holds no such element.
// A missing gate leaves the queue holding the encoder up, so nothing here fails a run.
func watchRelayGate(pipeline gst.Pipeline, element string) *relayGate {
	assert.Assert(element != "", "a relay gate names the queue it sits at")

	el := pipeline.GetByName(element)
	if el == nil {
		return nil
	}
	sink := el.GetStaticPad("sink")
	if sink == nil {
		return nil
	}

	gate := &relayGate{}
	sink.AddProbe(gst.PadProbeTypeBuffer, func(_ gst.Pad, info *gst.PadProbeInfo) gst.PadProbeReturn {
		buffer := padprobe.Buffer(info)
		if buffer == nil {
			return gst.PadProbeOK
		}
		if gate.admit(!buffer.HasFlags(gst.BufferFlagDeltaUnit), queueBacklog(el) >= relayBacklog) {
			return gst.PadProbeOK
		}
		return gst.PadProbeDrop
	})
	return gate
}

// queueBacklog reads how much running time a queue holds.
// The property is a guint64 in nanoseconds.
func queueBacklog(el gst.Element) time.Duration {
	level, ok := el.ObjectProperty("current-level-time").(uint64)
	assert.Assert(ok, "a gated queue states its level in time", el.GetName())
	return time.Duration(level)
}
