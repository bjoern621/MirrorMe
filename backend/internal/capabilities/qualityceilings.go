package capabilities

// QualityCeiling states whether a codec's constant-quality mode is bounded by a rate ceiling on this
// engine.
//
// A constant-quality encode spends what the picture asks for,
// so one screen filling with motion codes at many times the rate a still one does.
// The ceiling is what holds that inside a link's budget,
// and an encode with no form of one prices the same setting in delay and dropped frames instead.
//
// True where the mapping hands a stated ceiling to the element.
func QualityCeiling(codec, engine string) bool {
	return qualityCeilings[codec][engine].takes
}

// QualityCeilingRequired states whether this codec's constant-quality mode requires a rate ceiling.
// True for an element whose constant-quality mode is a bounded one,
// where an encode with no ceiling has no form:
// the form offers no zero and the builder refuses one.
func QualityCeilingRequired(codec, engine string) bool {
	return qualityCeilings[codec][engine].required
}

// qualityCeiling is what one engine's constant-quality mapping does with a rate ceiling.
type qualityCeiling struct {
	takes    bool
	required bool
}

// qualityCeilings is per codec and engine,
// a codec absent from it coding constant quality unbounded,
// and greying the ceiling field with the reason (internal/form, ceilingReason).
//
// The x26x elements hold a quality target inside a VBV:
// x264enc reads the ceiling off its bitrate property over vbv-buf-capacity,
// x265enc off vbv-maxrate over vbv-bufsize, and ffmpeg spells both as -maxrate over -bufsize.
// NVENC codes constant quality as VBR against a fixed quantizer,
// so the ceiling is the same -maxrate pair,
// and its GStreamer elements take constqp instead, which carries no rate at all.
// libvpx has one constant-quality mode and it is bounded:
// vpxenc's CQ codes toward a target,
// and a target of zero is a rate derived from the picture size rather than an absence of one,
// which is why the vpx rows require what the others merely take.
var qualityCeilings = map[string]map[string]qualityCeiling{
	"libx264":    {EngineGst: {takes: true}, EngineFfmpeg: {takes: true}},
	"libx265":    {EngineGst: {takes: true}, EngineFfmpeg: {takes: true}},
	"libvpx":     {EngineGst: {takes: true, required: true}, EngineFfmpeg: {takes: true}},
	"libvpx-vp9": {EngineGst: {takes: true, required: true}, EngineFfmpeg: {takes: true}},
	"h264_nvenc": {EngineFfmpeg: {takes: true}},
	"hevc_nvenc": {EngineFfmpeg: {takes: true}},
	"av1_nvenc":  {EngineFfmpeg: {takes: true}},
}
