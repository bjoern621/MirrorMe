package transport

import "time"

// pathDelays is how far behind its publisher a stream plays on each watch leg,
// publisher to viewer through MediaMTX on one machine, x264 zerolatency 640x360p30.
// Each figure is the leg's own cost with no network between: a stamp written into each picture,
// read back where the viewer decodes it (docs/delay-measurement.md).
// SRT's figure holds the 200 ms window on each of its two hops.
var pathDelays = map[string]time.Duration{
	RTSP{}.Name():   32 * time.Millisecond,
	WebRTC{}.Name(): 33 * time.Millisecond,
	SRT{}.Name():    444 * time.Millisecond,
	HLS{}.Name():    542 * time.Millisecond,
	RTMP{}.Name():   2700 * time.Millisecond,
}

// PathDelay is how far behind its publisher a stream plays on the named watch leg.
// False for a leg nothing measured.
func PathDelay(name string) (time.Duration, bool) {
	d, ok := pathDelays[name]
	return d, ok
}
