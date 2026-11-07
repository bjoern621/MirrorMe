package capabilities

import (
	"bjoernblessin.de/go-utils/util/assert"
)

// EveryEngine states one numeric limit under every publish engine,
// for the encoders whose knob is the library's rather than one wrapper's.
// It declares a shared fact once without collapsing the per-engine axis:
// the row still answers per engine,
// and a row that diverges names the engines separately with no consumer changing.
func EveryEngine(limit int) map[string]int {
	out := make(map[string]int, len(Engines))
	for _, engine := range Engines {
		out[engine] = limit
	}

	assert.Assert(len(out) == len(Engines), "a shared limit is stated for every engine", len(out))
	return out
}

// RateFieldM is where a rate field ends on an engine that declares nothing narrower.
//
// Both engines pass the rate as a count of bits per second and every encoder holds that in 32 bits,
// so 2147 Mbit/s is the last value any of them takes.
// BufferFieldKb is the same field read as the rate buffer,
// which reaches an encoder as the rate times the window in kilobits.
//
// A row narrower than this states its own figure per engine,
// and a row stating one for one engine states this for the other:
// a missing key reads as no bound at all (TestNumericLimitsCoverEveryEngineOrNone).
const (
	RateFieldM    = 2147
	BufferFieldKb = 2_147_483
)

// GopFieldFrames is where a keyframe interval ends on an encoder that declares nothing narrower.
// A hundred seconds at sixty frames a second, past any interval a live stream is worth running at.
const GopFieldFrames = 6000

// Where the software H.26x elements' bitrate property ends, in Mbit/s.
// Both count kbit/s in an unsigned field the plugin bounds itself,
// and x265enc's bound is a hundredth of x264enc's,
// so a target the ffmpeg engine takes on the same library is refused on the GStreamer one.
// The libraries impose neither, which is why both are stated against the GStreamer engine alone.
const (
	x264encRateLimitM = 2048
	x265encRateLimitM = 102
)

// VaFieldKb is where the va elements' rate fields end.
// Two properties count in it and both stop there:
// bitrate in kbit/s, and cpb-size, the rate buffer, in Kb.
// Read by the form to narrow what a control offers,
// and by the pipeline builder to refuse what it is handed,
// so the slider's end and the refusal are one figure (publish.vaRateLimits).
const VaFieldKb = 2_048_000

// vaFieldLimit is that figure as the per-engine map a codec row declares limits in.
// It binds on the GStreamer engine alone, the property being the element's:
// ffmpeg drives the same hardware through fields of its own,
// which stop where every other encoder's do.
func vaFieldLimit() map[string]int {
	return map[string]int{EngineFfmpeg: BufferFieldKb, EngineGst: VaFieldKb}
}

// vaRateLimit is the same bound on the rate itself, in the Mbit/s the settings count.
func vaRateLimit() map[string]int {
	return map[string]int{EngineFfmpeg: RateFieldM, EngineGst: VaFieldKb / 1000}
}

// vaGopFrames is where the va elements' key-int-max property ends.
// One property on the plugin's shared base class, so every element of the family stops there,
// and an interval past it is a launch that dies on the property rather than a longer one being coded.
const vaGopFrames = 1024

// VaGopFrames is that bound, for the builder that refuses an interval past it.
func VaGopFrames() int { return vaGopFrames }

// vaGopLimit is that bound as the per-engine map a codec row declares limits in.
func vaGopLimit() map[string]int {
	return map[string]int{EngineFfmpeg: GopFieldFrames, EngineGst: vaGopFrames}
}

// amfGopLimit is where AMF's keyframe-interval field ends.
//
// The interval reaches the encoder as ffmpeg's -g,
// which the AMF encoders map onto their own header_spacing property,
// and that property runs to 1000.
// A longer interval is refused as an option out of range,
// so the encoder never opens and the publish dies at launch.
const amfGopLimit = 1000

// CqMaxOn is the top of this codec's constant-quality scale on the named engine,
// and zero where the row declares none for it.
func (c Codec) CqMaxOn(engine string) int {
	assert.Assert(knownEngine(engine), "a quantizer scale lookup names a publish engine", engine)

	return c.CqMax[engine]
}

// BitrateLimitOn is the highest bitrate target this codec accepts on the named engine, in Mbit/s,
// and zero where that engine imposes no ceiling.
func (c Codec) BitrateLimitOn(engine string) int {
	assert.Assert(knownEngine(engine), "a bitrate ceiling lookup names a publish engine", engine)

	return c.BitrateLimitM[engine]
}

// BufferLimitOn is the largest rate buffer this codec's encoder field holds on the named engine,
// in kilobits, and zero where that engine imposes none of its own.
func (c Codec) BufferLimitOn(engine string) int {
	assert.Assert(knownEngine(engine), "a rate buffer ceiling lookup names a publish engine", engine)

	return c.BufferLimitKb[engine]
}

// GopLimitOn is the longest keyframe interval this codec's encoder holds on the named engine,
// in frames, and zero where that engine imposes no ceiling.
func (c Codec) GopLimitOn(engine string) int {
	assert.Assert(knownEngine(engine), "a keyframe interval ceiling lookup names a publish engine", engine)

	return c.GopLimit[engine]
}

// WidestCqScale is the top of the widest quantizer scale any row declares, on any engine.
//
// What a quantizer control is offered within,
// before the rules narrow it to the selected codec's own scale.
// Derived rather than written down:
// a constant would either clamp a codec counting past it or offer numbers no encoder here reaches,
// and a row declaring a wider scale would leave the constant the one place that did not know.
func WidestCqScale() int {
	widest := 0
	for _, c := range Codecs {
		for _, engine := range Engines {
			if scale := c.CqMaxOn(engine); scale > widest {
				widest = scale
			}
		}
	}
	assert.Assert(widest > 0, "the codec table declares a quantizer scale somewhere")
	return widest
}
