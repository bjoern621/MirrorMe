package form

import (
	"slices"

	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/capabilities"
)

// The rate-control tables state what no Go domain package carries.

// availabilityMode is what one rate-control concept needs from the encoder,
// the first of the three facts that decide a rate-control field.
// It states which controls the mode uses, not what any encoder does with them.
type availabilityMode struct {
	// usesCq: crf alone targets a quantizer.
	usesCq bool
	// usesBitrate: cbr, vbr and abr target a bitrate, crf and lossless none.
	usesBitrate bool
	// usesMaxrate: vbr sets a burst ceiling over the target, and crf one over a target it has not,
	// which is the same ceiling and the same rate buffer holding it.
	// Whether the encoder has a form of it is the codec's answer
	// (capabilities.QualityCeiling), weighed after this one.
	usesMaxrate bool
	// usesVbv: cbr and vbr bound the rate with a buffer of tunable size, and so does a bounded crf.
	usesVbv bool
	// usesBframes: the lossy bitrate and quality modes gain from B-frames,
	// where the family counts them.
	usesBframes bool
}

// availabilityModes is what each rate-control mode needs, keyed as capabilities names the modes.
// A mode outside the table needs nothing, which answers a hand-edited settings file naming
// one no builder implements.
var availabilityModes = map[string]availabilityMode{
	capabilities.ModeCbr:      {usesBitrate: true, usesVbv: true},
	capabilities.ModeVbr:      {usesBitrate: true, usesMaxrate: true, usesVbv: true, usesBframes: true},
	capabilities.ModeAbr:      {usesBitrate: true, usesBframes: true},
	capabilities.ModeCrf:      {usesCq: true, usesMaxrate: true, usesVbv: true, usesBframes: true},
	capabilities.ModeLossless: {},
}

// availabilityFamily is what an encoder family's encoders take and where they come from,
// the second of the three facts a rate-control field is decided by.
type availabilityFamily struct {
	// takesBframes: a family with no property for the count greys the field whatever the rate-control
	// mode, and the builders pin it off rather than forwarding a value the encoder would ignore.
	// The effort step is no such field: it follows the codec's own ladder,
	// one family's codecs being free to declare different ones.
	takesBframes bool
	// needsDevice: whether the encoders come with a device rather than with a build.
	// An absent encoder is then the machine's answer (no such GPU,
	// or no driver exposing that encode entrypoint) where a software one's is the build's.
	needsDevice bool
}

// availabilityFamilies is one row per encoder family capabilities declares.
// Every family has a row (TestEveryEncoderFamilyStatesWhatItsEncodersTake):
// one missing here reaches no verdict of its own and would grey under the engine's name instead.
var availabilityFamilies = map[string]availabilityFamily{
	capabilities.FamilySoftware: {},
	capabilities.FamilyNvenc:    {takesBframes: true, needsDevice: true},
	capabilities.FamilyVaapi:    {needsDevice: true},
	capabilities.FamilyQsv:      {needsDevice: true},
	capabilities.FamilyAmf:      {needsDevice: true},
	capabilities.FamilyV4l2:     {needsDevice: true},
	capabilities.FamilyRkmpp:    {needsDevice: true},
	capabilities.FamilyVulkan:   {needsDevice: true},
	// The device is the Mac, so the encoders are there on every machine that runs the framework
	// at all and absent on every machine that does not.
	capabilities.FamilyVideoToolbox: {needsDevice: true},
}

// availabilityFamiliesWith lists the families whose row sets a flag,
// for a statement naming who takes a field rather than one family written in by hand.
// The list follows the table when a family gains the field.
//
// The order is capabilities.Families' own, so the statement reads the same on every call rather
// than in the map's iteration order of the moment.
func availabilityFamiliesWith(flag func(availabilityFamily) bool) []string {
	var out []string
	for _, name := range capabilities.Families {
		family, ok := availabilityFamilies[name]
		assert.Assert(ok, "every encoder family states what its encoders take", name)
		if flag(family) {
			out = append(out, name)
		}
	}
	return out
}

// availabilityEngineRule is one place where a publish engine's encoder builder departs
// from the mode table: a knob the mode uses that the engine ignores,
// or one the engine forwards in a mode the table marks unused.
// It is the third of the three facts.
//
// Whether a value the mode needs reaches the encoder depends on which engine builds the command,
// the two expressing the same modes through different properties: the GStreamer nvcodec
// and QSV elements carry no rate buffer,
// and the vpx elements read a constant-quality bitrate as a cap.
//
// A knob an encoder library has no form of at all is a rule with no engine,
// both builders hitting the same wall.
// The rules mirror encoderArgs in ffmpeg/encoders.go and gstEncoder in publish/gstencoders.go.
type availabilityEngineRule struct {
	// engine is empty for a rule describing the library rather than a builder, and then holds on both.
	engine string
	// knob is the field key the rule governs, so a rule and the control it greys are one identifier.
	knob string
	// codecs and families select what the rule covers, unioned.
	// Both empty covers every codec the engine builds.
	codecs   []string
	families []string
	// modes selects the rate-control modes the rule covers.
	// Empty covers every mode, so such a rule outranks the mode's own reason.
	modes []string
	// forwards is true where the builder sends the value though the mode table marks the knob unused,
	// false where it ignores a value the mode would use.
	forwards bool
	// reason names why an ignored value never reaches the encoder, or what a forwarded one does there.
	// A code and not a sentence, one per row: each states a fact about one named element.
	reason screensharev1.TextCode
}

// The table decides which knobs a builder never reads,
// so a malformed row is an Entwicklungsfehler and fails at load.
//
// A row naming no knob governs nothing and would sit in the table doing nothing visible.
// One carrying no reason would grey or annotate a control with no code behind the sentence.
func init() {
	for i, r := range availabilityEngineRules {
		assert.Assert(r.knob != "", "a departure names the control it governs", i)
		assert.Assert(r.reason != screensharev1.TextCode_TEXT_CODE_UNSPECIFIED,
			"a departure states the fact behind it", r.knob, i)
		assert.Assert(r.engine == "" || slices.Contains(capabilities.Engines, r.engine),
			"a departure that names an engine names a publish engine", r.engine, r.knob)
	}
}

// availabilityEngineRules is the departure table, earlier rows winning.
var availabilityEngineRules = []availabilityEngineRule{
	// VAAPI is the one family whose effort step reaches a single engine.
	// Every other element that codes a laddered codec takes its steps through a property of its own,
	// speed-preset, cpu-used or preset,
	// and the nvcodec elements take the same p1-p7 steps ffmpeg does.
	//
	// An encoder that cannot bound the burst at all has no VBR mode,
	// declared as a mode gap in the capability table, so no row withholds the ceiling field
	// for that case: a rule would grey a field under a mode that cannot be selected.
	{
		engine:   capabilities.EngineFfmpeg,
		knob:     KeyEffort,
		families: []string{capabilities.FamilyVaapi},
		reason:   screensharev1.TextCode_TEXT_CODE_FFMPEG_VAAPI_QUALITY_IS_THE_DRIVERS_SCALE,
	},
	{
		knob:   KeyVbvMs,
		codecs: []string{"librav1e"},
		reason: screensharev1.TextCode_TEXT_CODE_RAV1E_SIZES_NO_RATE_BUFFER,
	},
	{
		engine: capabilities.EngineGst,
		knob:   KeyGop,
		codecs: []string{"librav1e"},
		reason: screensharev1.TextCode_TEXT_CODE_GST_RAV1ENC_NO_KEYFRAME_INTERVAL,
	},
	{
		engine:   capabilities.EngineGst,
		knob:     KeyVbvMs,
		families: []string{capabilities.FamilyNvenc},
		reason:   screensharev1.TextCode_TEXT_CODE_GST_NVENC_NO_RATE_BUFFER,
	},
	{
		engine:   capabilities.EngineGst,
		knob:     KeyVbvMs,
		families: []string{capabilities.FamilyQsv},
		reason:   screensharev1.TextCode_TEXT_CODE_GST_QSV_NO_RATE_BUFFER,
	},
	{
		knob:     KeyBitrateM,
		families: []string{capabilities.FamilyVaapi, capabilities.FamilyAmf, capabilities.FamilyVulkan, capabilities.FamilyQsv},
		modes:    []string{capabilities.ModeAbr},
		forwards: true,
		reason:   screensharev1.TextCode_TEXT_CODE_FIXED_FUNCTION_ABR_DERIVES_CEILING,
	},
	{
		knob:     KeyBframes,
		families: []string{capabilities.FamilyAmf},
		reason:   screensharev1.TextCode_TEXT_CODE_AMF_CODES_NO_BFRAMES,
	},
}
