// Package capabilities is the single source of truth for the fixed facts about each video codec:
// the encoder family it runs on, which pixel formats it may encode,
// which rate-control modes its encoder implements,
// and the scale its constant-quality knob counts on.
//
// The two publish engines wrap different encoder implementations,
// so a codec, or one value of one settings option, can be one engine's and not the other's.
// Each difference is a Gap naming the engine, the option, the value and the reason,
// rather than a fact narrowed to what both engines do:
// an option value one engine reaches stays offered there, and the engine that lacks it says why.
// Which options a gap may name is the Options list,
// so an axis is added by naming it there rather than by growing the Gap type.
//
// A Gap and a numeric ceiling are how those facts are written, on the codec row they belong to.
// They are not what anything reads:
// rules.go turns each into a rule in the one evaluator (internal/rules),
// and the greying, the offered range and the refusal in Validate read one evaluation.
// The split exists because a fact about a codec and a capture backend together,
// or a codec and a platform, has no row to sit on,
// and a fact with no row grows a table of its own with a consumer written against it.
//
// Which protocol carries a codec is not a fact about the encoder and is not modeled here.
// A protocol carries a bitstream format,
// so the transport package declares its own format set per leg and answers both directions from it.
//
// These facts are consumed on both sides of the wire:
// the argument builders branch on them and reject an impossible combination,
// and a shell reads them (App.Capabilities) to grey out what cannot be picked.
// One definition keeps a codec's constraints from differing between the encoder and a form.
//
// The words are not here.
// Labels, tooltips and refusal sentences are written where the layout is,
// keyed by the identifiers these rows use (docs/ipc-api.md).
//
// Every row is a compiled-in fact,
// so a malformed one is an Entwicklungsfehler and the lookups assert on it.
// What a user picked arrives from a settings file and stays error-returning.
package capabilities

// The publish engines a capability can differ between.
// This package depends on the rule vocabulary and on nothing else in the domain,
// which is what lets every consumer read it,
// so the names are declared here and the publish package's own constants take them from here.
const (
	EngineFfmpeg = "ffmpeg"
	EngineGst    = "gstreamer"
)

// Engines lists every publish engine a Gap may name and a lookup may be asked about.
var Engines = []string{EngineFfmpeg, EngineGst}

// The encoder families a codec's backend belongs to.
// A family is the axis every per-backend fact keys off:
// a surface names the family and the format to tell one catalog row from another,
// and each builder dispatches its family-wide behaviour off a table keyed this way,
// rather than off a per-family flag or a name suffix.
const (
	FamilySoftware     = "software"
	FamilyNvenc        = "nvenc"
	FamilyVaapi        = "vaapi"
	FamilyQsv          = "qsv"
	FamilyAmf          = "amf"
	FamilyV4l2         = "v4l2"
	FamilyRkmpp        = "rkmpp"
	FamilyVulkan       = "vulkan"
	FamilyVideoToolbox = "videotoolbox"
)

// Families lists every encoder family a codec row may declare.
var Families = []string{
	FamilySoftware, FamilyNvenc, FamilyVaapi, FamilyQsv,
	FamilyAmf, FamilyV4l2, FamilyRkmpp, FamilyVulkan, FamilyVideoToolbox,
}

// The rate-control modes an encode runs under.
// Some aim at a bitrate and differ in the ceiling they hold it under,
// one aims at a quality, and one codes bit-exact.
// Which aim at a bitrate is targetsBitrate.
const (
	ModeCbr      = "cbr"
	ModeVbr      = "vbr"
	ModeAbr      = "abr"
	ModeCrf      = "crf"
	ModeLossless = "lossless"
)

// Modes lists every rate-control mode a Gap may name and Validate may be given.
var Modes = []string{ModeCbr, ModeVbr, ModeAbr, ModeCrf, ModeLossless}

// TargetsBitrate is targetsBitrate for the publish engines,
// which ask the same question of a running pipeline:
// a mode that sends the encoder no rate has none to send it again.
func TargetsBitrate(mode string) bool {
	return targetsBitrate(mode)
}

// targetsBitrate is whether a rate-control mode aims at a bitrate somebody sets.
// Constant quality and lossless spend whatever the picture costs,
// so the bitrate field means nothing to them.
func targetsBitrate(mode string) bool {
	return mode == ModeCbr || mode == ModeVbr || mode == ModeAbr
}

// The quantization ranges a picture may be coded at:
// every code value carrying image data, or the 16-235 studio swing a broadcast chain expects.
//
// Declared here for the reason the modes are.
// The colour range is an option a Gap may take a value away from,
// and a table naming the axis without naming its values leaves every consumer to type them,
// where the one that types them differently greys the wrong option.
const (
	ColorRangeFull    = "pc"
	ColorRangeLimited = "tv"
)

// ColorRanges lists every colour range a Gap may name and Validate may be given.
var ColorRanges = []string{ColorRangeFull, ColorRangeLimited}

// The settings options a Gap may take a value away from.
// Each is the JSON field name settings.Settings carries,
// so a gap and the form control it greys are the same identifier on both sides of the wire.
const (
	OptionChroma     = "chroma"
	OptionMode       = "mode"
	OptionColorRange = "colorRange"
	// OptionTune is a step of the row's tune ladder.
	// Gappable because a tune knob is an encoder wrapper's and not the library's:
	// libaom takes a tune on ffmpeg where av1enc exposes no such property,
	// and oneVPL's scenario reaches the qsv encoders through ffmpeg alone.
	// Gapping the steps leaves TuneNone standing, which is what an engine without the knob spends.
	OptionTune = "tune"
)

// Options lists every option a Gap may name and Validate is given a value for.
// Naming one here and a row on the codec that lacks it is the whole of declaring another kind of gap:
// the lookup, the validator and every surface read this list rather than a field per axis.
var Options = []string{OptionChroma, OptionMode, OptionColorRange, OptionTune}

func knownOption(option string) bool {
	return contains(Options, option)
}

// knownEngine draws the line between the two failure kinds.
// Every lookup asks about an engine that the caller names itself, never one read off the settings,
// so an engine outside this set is an Entwicklungsfehler and asserts.
// The codec, pixel format and mode that a lookup asks about come from the user and stay error-returning.
func knownEngine(engine string) bool {
	return contains(Engines, engine)
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
