package publish

import (
	"strconv"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// nvencTuneProperties spells each NVENC tune step as this engine's elements take it,
// keyed as the capability row declares the step.
//
// The row uses the SDK's abbreviations, as ffmpeg does.
// The elements spell them in full words, the property's own GstNvEncoderTune enum.
// A step with no entry is one the elements do not implement, and the builder then passes no tune
// rather than a word the element would refuse.
//
// The tune stands in for the deprecated half of the elements' preset enum, whose entries name
// presets (default, hp, low-latency-hq) where the p1-p7 half names ladder steps.
// The element's own help points at the ladder with a tune instead.
var nvencTuneProperties = map[string]string{
	"hq":       "high-quality",
	"ll":       "low-latency",
	"ull":      "ultra-low-latency",
	"lossless": "lossless",
}

// nvencEncoder maps the rate-control mode onto one nvcodec element's properties, the counterpart
// to the NVENC branch of encoderArgs.
// These elements express rate control as rc-mode plus a constant-QP target, where ffmpeg does not.
//   - crf: rc-mode=constqp at the quantizer target.
//   - abr: rc-mode=vbr toward the bitrate with no ceiling.
//   - vbr: rc-mode=vbr toward the bitrate with max-bitrate as the ceiling.
//   - cbr: rc-mode=cbr with zerolatency reordering.
//   - lossless: the lossless tune at rc-mode=constqp.
//
// preset takes the row's p1-p7 step and tune the row's step in this engine's spelling,
// so an NVENC stream encodes at the step the form shows whichever engine built it.
// B-frames reach the lossy bursting modes alone.
//
// The element name is bound per codec in gstCodecs and H.264, HEVC and AV1 share this mapping:
// the nvcodec encoders derive from one base class and expose the same properties,
// the codec deciding only which of them the hardware honours.
func nvencEncoder(elem string) func(settings.Settings, gstRates, capabilities.Steps) []string {
	return func(s settings.Settings, r gstRates, l capabilities.Steps) []string {
		base := []string{elem}
		if l.Effort != "" {
			base = append(base, "preset="+l.Effort)
		}
		if tune, spelled := nvencTuneProperties[l.Tune]; spelled {
			base = append(base, "tune="+tune)
		}
		withBframes := func(enc []string) []string {
			if s.Publish.Bframes > 0 {
				// bframes, not b-frames: the elements spell it without the separator,
				// and a property name no element carries fails the launch rather than being ignored.
				return append(enc, "bframes="+strconv.Itoa(s.Publish.Bframes))
			}
			return enc
		}
		switch s.Publish.Mode {
		case "lossless":
			// No B-frames: bit-exact coding gains nothing from them, so the form greys the field here.
			//
			// The tune carries the mode rather than the preset: the enum's lossless preset fails session
			// init with NV_ENC_ERR_INVALID_PARAM on the H.264 encoder at 4:2:0,
			// where the tune reaches every element and chroma the mode is offered on.
			// The row declares that tune for this mode, so it arrives through base.
			//
			// constqp because the driver refuses the transform bypass under any other rate control:
			// "RC Mode / Profile not supported with qpPrimeYZeroTransformBypassFlag" (NVIDIA 615).
			return append(base, "rc-mode=constqp", "gop-size="+r.gop)
		case "crf":
			// The quantizer goes on the three frame types rather than through qp-const: nvav1enc exposes
			// qp-const-i, -p and -b and no combined property, where the H.26x elements expose both.
			// The combined property sets one quantizer on all three anyway,
			// so this is the spelling every element in the family reads.
			return withBframes(append(base, "rc-mode=constqp",
				"qp-const-i="+r.cq, "qp-const-p="+r.cq, "qp-const-b="+r.cq,
				"gop-size="+r.gop))
		case "abr":
			return withBframes(append(base, "rc-mode=vbr", "bitrate="+r.kbps, "gop-size="+r.gop))
		case "vbr":
			return withBframes(append(base, "rc-mode=vbr", "bitrate="+r.kbps, "max-bitrate="+r.maxkbps, "gop-size="+r.gop))
		case "cbr":
			return append(base, "rc-mode=cbr", "bitrate="+r.kbps, "zerolatency=true", "gop-size="+r.gop)
		default:
			assert.Never("unexpected rate-control mode", s.Publish.Mode)
			return nil
		}
	}
}
