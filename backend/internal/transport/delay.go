package transport

import "time"

// pathDelays is how far behind its publisher a stream plays on each watch leg,
// encoder to decoder through MediaMTX on one machine, x264 zerolatency 640x360p30 with an Opus
// track, published over RTSP and watched on each leg's default window.
// Each figure is the leg's own cost with no network between: a stamp written into each picture,
// read back where the viewer decodes it (docs/delay-measurement.md).
// internal/app, TestEveryLegMeasuresItsDelayThroughARelay, holds a leg to its figure.
var pathDelays = map[string]time.Duration{
	RTSP{}.Name():   160 * time.Millisecond,
	WebRTC{}.Name(): 110 * time.Millisecond,
	SRT{}.Name():    430 * time.Millisecond,
	HLS{}.Name():    3350 * time.Millisecond,
	RTMP{}.Name():   32 * time.Millisecond,
}

// PathDelay is how far behind its publisher a stream plays on the named watch leg.
// False for a leg nothing measured.
func PathDelay(name string) (time.Duration, bool) {
	d, ok := pathDelays[name]
	return d, ok
}
