package publish

import (
	"slices"
	"strconv"
	"strings"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// x264Encoder maps the rate-control mode onto x264enc's pass property, the counterpart
// to the libx264 branch of encoderArgs.
//   - crf: pass=qual at a constant quantizer, bounded by the stated ceiling (x264Ceiling).
//   - lossless: pass=quant at quantizer 0, x264's bit-exact coding mode.
//   - abr: pass=cbr with vbv-buf-capacity=0, which disables the VBV and leaves one-pass ABR toward
//     the target.
//   - cbr: pass=cbr, which targets the bitrate and bounds the VBV to it.
//
// vbr has no branch: pass=cbr locks the VBV maxrate to the bitrate and x264enc cannot raise it
// above, so the mode is a gap on this engine (gstNoRateCeiling) rather than run as this one.
// speed-preset carries x264's named effort ladder, and the tune travels in whichever
// of the element's two tune properties declares it (x264TuneProperty).
func x264Encoder(s settings.Settings, r gstRates, l capabilities.Steps) []string {
	base := slices.Concat([]string{"x264enc", "speed-preset=" + l.Effort}, x264TuneProperty(l.Tune),
		[]string{"key-int-max=" + r.gop})
	switch s.Publish.Mode {
	case "crf":
		return x264Ceiling(append(base, "pass=qual", "quantizer="+r.cq), s, r)
	case "lossless":
		return append(base, "pass=quant", "quantizer=0")
	case "abr":
		return append(base, "bitrate="+r.kbps, "pass=cbr", "vbv-buf-capacity=0")
	case "cbr":
		enc := append(base, "bitrate="+r.kbps, "pass=cbr")
		if s.Publish.VbvMs > 0 {
			enc = append(enc, "vbv-buf-capacity="+strconv.Itoa(s.Publish.VbvMs))
		}
		return enc
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
}

// x264Ceiling bounds a constant-quality encode, or takes the element's rate buffer away where
// the settings state no ceiling.
//
// pass=qual reads the ceiling off the bitrate property, held over the vbv-buf-capacity window.
// Both carry a default, so an encode that sets neither is bounded at 2 Mbit/s over 600 ms whatever
// quantizer it asked for: a buffer of zero makes a quality target free of the rate.
// A window of zero beside a ceiling is the element's own, the settings stating no size rather than
// stating none is wanted (internal/form, fieldVbvBounds).
func x264Ceiling(enc []string, s settings.Settings, r gstRates) []string {
	if s.Publish.MaxrateM <= 0 {
		return append(enc, "vbv-buf-capacity=0")
	}

	enc = append(enc, "bitrate="+r.maxkbps)
	if s.Publish.VbvMs > 0 {
		enc = append(enc, "vbv-buf-capacity="+strconv.Itoa(s.Publish.VbvMs))
	}
	return enc
}

// x264PsyTunes are the steps x264enc takes on its psy-tune property rather than on its tune one.
//
// The element splits x264's tune list across two properties: tune is a flagset of the tunes
// that change what the encoder codes, psy-tune an enum of the ones weighing what the eye sees.
// ffmpeg spells the whole list through one -tune, so which property carries a step is this engine's
// while the step stays the encoder's own identifier.
var x264PsyTunes = []string{"film", "animation", "grain", "psnr", "ssim"}

// x264TuneProperty is the property one tune step travels in on x264enc, and nothing where
// the ladder leaves the knob alone.
// Both properties default to no tuning, so an untuned encode is what neither of them set yields.
func x264TuneProperty(step string) []string {
	switch {
	case step == "" || step == capabilities.TuneNone:
		return nil
	case slices.Contains(x264PsyTunes, step):
		return []string{"psy-tune=" + step}
	default:
		return []string{"tune=" + step}
	}
}

// x265Encoder maps the rate-control mode onto x265enc, the HEVC counterpart to x264Encoder.
// The element has no pass property: rate control comes from bitrate and qp plus an option-string
// of libx265 knobs.
//   - crf: option-string crf, libx265's rate factor, bounded by the stated ceiling.
//   - lossless: option-string lossless=1, qp 0 not being bit-exact on x265 as it is on x264.
//   - abr: bitrate alone, one-pass average bitrate.
//   - vbr: bitrate with a vbv-maxrate ceiling above it, libx265's constrained VBR.
//   - cbr: the same pair with the ceiling at the target.
//
// The quantizer travels as a rate factor rather than on the qp property, which is libx265's CQP:
// a fixed quantizer takes no VBV, so a ceiling stated beside it would reach the element and bound
// nothing, and the same setting codes at one quality on the ffmpeg engine and another here.
func x265Encoder(s settings.Settings, r gstRates, l capabilities.Steps) []string {
	base := slices.Concat([]string{"x265enc", "speed-preset=" + l.Effort}, x265TuneProperty(l.Tune),
		[]string{"key-int-max=" + r.gop})
	switch s.Publish.Mode {
	case "crf":
		quality := []string{"crf=" + r.cq}
		if s.Publish.MaxrateM > 0 {
			quality = append(quality, x265Vbv(r.maxkbps, s.Publish.MaxrateM, s.Publish.VbvMs)...)
		}
		return append(base, x265Options(quality...))
	case "lossless":
		return append(base, x265Options("lossless=1"))
	case "abr":
		return append(base, "bitrate="+r.kbps, x265Options())
	case "vbr":
		return append(base, "bitrate="+r.kbps,
			x265Options(x265Vbv(r.maxkbps, s.Publish.MaxrateM, s.Publish.VbvMs)...))
	case "cbr":
		return append(base, "bitrate="+r.kbps,
			x265Options(x265Vbv(r.kbps, s.Publish.BitrateM, s.Publish.VbvMs)...))
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
}

// x265Vbv is a rate ceiling and the buffer it is held over, as libx265 spells the pair.
// vbv-bufsize counts in kbit rather than in milliseconds, so the window is the ceiling held
// for that long, and one second where the settings state no window, matching ffmpeg's bufsizeArg.
func x265Vbv(ceilingKbps string, ceilingMbps, vbvMs int) []string {
	assert.Assert(ceilingKbps != "", "a VBV names the ceiling it holds")

	bufKbit := ceilingKbps
	if vbvMs > 0 {
		bufKbit = strconv.Itoa(ceilingMbps * vbvMs)
	}
	return []string{"vbv-maxrate=" + ceilingKbps, "vbv-bufsize=" + bufKbit}
}

// x265LiveDelay is the lookahead pin for x265enc, the rule gstLiveDelay states for the elements
// that carry it in properties of their own.
// frame-threads holds one frame per thread as x264's frame threading does, and rc-lookahead needs
// the b-frames off with it: x265 raises the lookahead back to cover them.
var x265LiveDelay = []string{"rc-lookahead=0", "bframes=0", "frame-threads=1"}

// x265Options is the element's one option-string, the live pins first and the mode's own keys
// after.
// One property carries every libx265 knob, so a mode that wrote its own would drop the pins.
func x265Options(mode ...string) string {
	return "option-string=" + strings.Join(slices.Concat(x265LiveDelay, mode), ":")
}

// x265TuneProperty is the tune step as x265enc spells it, and nothing where the ladder leaves
// the knob to the encoder.
//
// The untuned step is the one that has to be spelled: this element's tune enum starts at ssim,
// so leaving the property off is a tuning choice here where it is none on ffmpeg.
// The enum's zero entry is named with a space no gst-launch line carries, so it travels as tune=0.
func x265TuneProperty(step string) []string {
	switch step {
	case "":
		return nil
	case capabilities.TuneNone:
		return []string{"tune=0"}
	default:
		return []string{"tune=" + step}
	}
}
