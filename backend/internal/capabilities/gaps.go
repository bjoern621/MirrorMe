package capabilities

import (
	"slices"

	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// vaapiGaps are the gaps every VAAPI row carries.
//
// A VA encoder quantizes every frame and no VA profile exposes a transform-bypass path,
// so lossless has no VA form on either publish engine.
// The other four modes reach the drivers' CQP, CBR and VBR rate control.
//
// Full range is the colour range the GStreamer engine has no honest form of.
// Measured on all five formats,
// the va elements write no colour description into the bitstream and expose no property that would,
// so what a decoder produces from their streams carries no colorimetry at all.
// A viewer reads an unsignalled stream as limited-range BT.709.
// A full-range publish therefore arrives expanded a second time,
// with crushed blacks and clipped whites, while the form says the stream is full range.
// The ffmpeg engine tags the frames with the whole description (ffmpeg.colourFilter),
// and reaches both ranges on the same hardware.
var vaapiGaps = []Gap{
	{
		Option: OptionMode,
		Value:  ModeLossless,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VAAPI_NO_LOSSLESS,
	},
	{
		Engine: EngineGst,
		Option: OptionColorRange,
		Value:  "pc",
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_GST_VA_NO_COLOUR_DESCRIPTION,
	},
}

// qsvGaps is the rate-control gap every QSV row carries.
// Intel's encoders quantize every frame,
// and neither ffmpeg's QSV encoders nor the qsv elements expose the transform bypass a bit-exact
// stream needs.
// The other four modes map onto oneVPL's CQP, CBR and VBR rate control, which both engines reach.
var qsvGaps = []Gap{{
	Option: OptionMode,
	Value:  ModeLossless,
	Reason: mirrormev1.TextCode_TEXT_CODE_GAP_QSV_NO_LOSSLESS,
}}

// videoToolboxGaps are the gaps both VideoToolbox rows carry,
// and they leave one rate control standing.
//
// What the framework takes on every Mac is an average bitrate,
// so abr is the mode the rows declare and the other three are one fact stated once:
// a constant rate needs a property macOS 13 carries and Apple silicon implements,
// a burst ceiling has no wrapper option on either engine,
// and the constant-quality knob is Apple silicon's alone,
// counting upward where every other row here counts a quantizer downward.
// Declaring a mode half the Macs in the field refuse is a publish that dies at launch,
// and one whose scale runs the other way is a slider that means the opposite of what it says.
//
// Lossless is separate, the fix differing:
// no rate control reaches it, the framework quantizing every frame with no transform bypass,
// as on VAAPI and QSV.
var videoToolboxGaps = []Gap{
	{
		Option: OptionMode,
		Value:  ModeLossless,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VIDEOTOOLBOX_NO_LOSSLESS,
	},
	{
		Option: OptionMode,
		Value:  ModeCbr,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VIDEOTOOLBOX_AVERAGE_BITRATE_ONLY,
	},
	{
		Option: OptionMode,
		Value:  ModeVbr,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VIDEOTOOLBOX_AVERAGE_BITRATE_ONLY,
	},
	{
		Option: OptionMode,
		Value:  ModeCrf,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VIDEOTOOLBOX_AVERAGE_BITRATE_ONLY,
	},
}

// qsvScenarioRowGaps is what the two rows carrying a scenario ladder gap:
// the family's rate-control gap, and the hint the other engine has no property for.
// The AV1 and VP9 rows declare no such ladder at all,
// ffmpeg putting the option on its H.264 and HEVC encoders alone,
// so they take qsvGaps unchanged.
var qsvScenarioRowGaps = slices.Concat(qsvGaps, gstNoQsvScenario)

// amfGaps are the gaps every AMF row carries.
//
// The engine gap is a platform one rather than a codec one:
// GStreamer's amfcodec plugin builds its device layer on D3D11,
// and its meson configuration refuses any other host system,
// so the engine the portal capture backend runs on has no AMF element to name at all.
// That leaves the ffmpeg engine, whose encoders load AMD's libamfrt64 runtime directly.
//
// Lossless is the mode AMD's encoders have no form of, as on VAAPI:
// the quantizer options bottom out at a quantized picture,
// and no AMF property bypasses the transform the way x264's qp 0 or NVENC's lossless tune do.
// The other four modes map onto AMF's CQP, CBR and peak-constrained VBR rate control.
var amfGaps = []Gap{
	{
		Engine: EngineGst,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_GST_AMFCODEC_WINDOWS_ONLY,
	},
	{
		Option: OptionMode,
		Value:  ModeLossless,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_AMF_NO_LOSSLESS,
	},
}

// vulkanGaps are the gaps every Vulkan Video row carries.
//
// The engine gap is a memory one:
// the GStreamer vulkan encoder takes images on a Vulkan device,
// a memory the capture chain would have to upload to,
// and the plugin carries no HEVC or AV1 encoder to take them at all.
// That leaves the ffmpeg engine,
// whose encoders create the Vulkan device themselves and upload each frame through the filter graph.
//
// Vulkan's lossless tuning mode is the one place the options read as if the family coded bit-exact.
// A hint about what to optimize for rather than a coding mode,
// and nothing in the path pins the transform bypass such a stream needs.
// A picture encoded under it decodes back different from the source.
var vulkanGaps = []Gap{
	{
		Engine: EngineGst,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_GST_VULKAN_NO_CAPTURE_MEMORY,
	},
	{
		Option: OptionMode,
		Value:  ModeLossless,
		Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VULKAN_NO_LOSSLESS,
	},
}

// vp8NoFullRange is the colour-range gap both VP8 rows carry, on both publish engines.
// It belongs to the format rather than to an encoder:
// a VP8 keyframe header codes a single colour_space bit, which has one defined value,
// and no colour range field at all.
// Measured on libvpx through both engines,
// there is nothing for an encoder to write the range into and no property that would.
// A full-range publish would arrive at every viewer expanded as limited range,
// with crushed blacks and clipped whites, while the form says the stream is full range.
var vp8NoFullRange = Gap{
	Option: OptionColorRange,
	Value:  "pc",
	Reason: mirrormev1.TextCode_TEXT_CODE_GAP_VP8_HAS_NO_COLOUR_RANGE_FIELD,
}

// gstNoRateCeiling is the constrained-VBR gap the software rows without a ceiling carry,
// on the GStreamer publish engine.
// x264enc's pass=cbr locks the VBV maxrate to the bitrate,
// and vp8enc, vp9enc and av1enc expose a target and no maximum at all.
// Given a ceiling they run the uncapped average abr already names.
//
// x265enc carries no such gap: libx265 takes vbv-maxrate above the target,
// which is the mode's own definition rather than an approximation of it.
//
// It gaps the mode rather than greying the ceiling field, the mode being what the element cannot do:
// a field greyed under a mode that silently becomes another mode still reports the wrong rate
// control in the command and in every verdict derived from it.
var gstNoRateCeiling = Gap{
	Engine: EngineGst,
	Option: OptionMode,
	Value:  ModeVbr,
	Reason: mirrormev1.TextCode_TEXT_CODE_GAP_GST_SOFTWARE_NO_RATE_CEILING,
}

// gstNoPlanarRGB is the planar-RGB gap the software RGB-coding rows carry on the GStreamer publish
// engine.
// x265enc, vp9enc and av1enc take YUV only,
// so the pipeline could only convert RGB to 4:4:4 YUV and spend RGB's bitrate without its exactness.
// The ffmpeg engine codes each of those formats as RGB,
// which is what makes this a gap per engine rather than a pixel format the rows leave out.
//
// Not every RGB-coding row:
// the nvcodec HEVC elements do take a GBR sink format and code it in the Range Extensions profile,
// so hevc_nvenc reaches planar RGB on both engines.
// Which elements take which layout is publish.gstFamilyChromaFormats,
// and this gap states only where none of them does.
var gstNoPlanarRGB = Gap{
	Engine: EngineGst,
	Option: OptionChroma,
	Value:  "gbrp",
	Reason: mirrormev1.TextCode_TEXT_CODE_GAP_GST_ELEMENTS_NO_PLANAR_RGB,
}

// gstNoAomTune takes libaom's tune steps off the GStreamer engine.
// av1enc carries none of libaom's tuning at all,
// where the vpx elements carry the same two steps ffmpeg does,
// so this is one element's limit rather than the plugin's.
var gstNoAomTune = tuneGaps(EngineGst, aomTunes, mirrormev1.TextCode_TEXT_CODE_GAP_GST_AV1ENC_NO_TUNE)

// gstNoQsvScenario takes oneVPL's scenario hint off the GStreamer engine.
// The qsv elements expose target-usage and low-latency and nothing that carries a scenario,
// so the hint reaches Intel's runtime through ffmpeg's encoders alone.
var gstNoQsvScenario = tuneGaps(EngineGst, qsvScenarios, mirrormev1.TextCode_TEXT_CODE_GAP_GST_QSV_NO_SCENARIO)

// tuneGaps withholds every step of a ladder from one engine, under one reason.
//
// A gap names one value,
// so an engine that lacks the knob rather than one step of it is written as the whole ladder minus
// the untuned step:
// TuneNone is what an encoder without the property spends,
// and gapping it would leave the control with nothing to hold.
func tuneGaps(engine string, steps []string, reason mirrormev1.TextCode) []Gap {
	assert.Assert(len(steps) > 1, "a gapped tune ladder carries a step besides the untuned one", len(steps))

	out := make([]Gap, 0, len(steps)-1)
	for _, step := range steps {
		if step == TuneNone {
			continue
		}
		out = append(out, Gap{Engine: engine, Option: OptionTune, Value: step, Reason: reason})
	}

	assert.Assert(len(out) == len(steps)-1, "every step but the untuned one is withheld", len(out), len(steps))
	return out
}
