package publish

import (
	"strconv"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/settings"
)

// vaEncoder maps the rate-control mode onto one va plugin element, the counterpart to vaapiArgs
// in the ffmpeg builder.
// quantizers are the properties the constant-quality target is written to, bound per codec
// in gstCodecs: the H.26x elements take one per frame type (qpi for I, qpp for P) and the AV1, VP9
// and VP8 ones a single qp for every frame.
//   - crf: rate-control=cqp with those quantizer properties set.
//   - cbr: rate-control=cbr at the target bitrate, cpb-size its rate buffer.
//   - vbr: rate-control=vbr, where bitrate is the ceiling and target-percentage places the target
//     under it.
//   - abr: the same VBR with the ceiling at twice the target, what ffmpeg derives for a VAAPI VBR
//     encode given no ceiling, so the stream does not change with the capture backend.
//
// VAAPI codes against a maximum either way, so abr is as unbounded as an average gets here.
// bitrate and target-percentage each have a range the settings can fall outside of, and vaLimits
// refuses such a combination ahead of this mapping.
//
// bitrate and cpb-size are in kbit, and a zero cpb-size leaves the element its own calculation, so
// the VBV window appears only where the settings carry one.
// No B-frame count, as on the ffmpeg path: VAAPI B-frame support varies per driver.
// lossless has no VAAPI form (vaapiGaps).
//
// The effort step travels as target-usage, the property carrying the scale on these elements.
// The ffmpeg half spends none: its -quality runs over the range the installed driver reports rather
// than over a fixed seven, so one step cannot mean one thing on both
// (form.availabilityEngineRules).
func vaEncoder(elem string, quantizers ...string) func(settings.Settings, gstRates, capabilities.Steps) []string {
	return func(s settings.Settings, r gstRates, l capabilities.Steps) []string {
		base := []string{elem, "key-int-max=" + r.gop}
		if l.Effort != "" {
			base = append(base, "target-usage="+l.Effort)
		}
		switch s.Publish.Mode {
		case "crf":
			enc := append(base, "rate-control=cqp")
			for _, q := range quantizers {
				enc = append(enc, q+"="+r.cq)
			}
			return enc
		case "abr":
			return append(base, "rate-control=vbr",
				"bitrate="+vaBitrate(s.Publish.BitrateM*vaAbrPeak),
				"target-percentage="+strconv.Itoa(100/vaAbrPeak))
		case "vbr":
			enc := append(base, "rate-control=vbr",
				"bitrate="+vaBitrate(s.Publish.MaxrateM), "target-percentage="+vaTargetPercentage(s))
			if s.Publish.VbvMs > 0 {
				enc = append(enc, "cpb-size="+strconv.Itoa(s.Publish.MaxrateM*s.Publish.VbvMs))
			}
			return enc
		case "cbr":
			enc := append(base, "rate-control=cbr", "bitrate="+vaBitrate(s.Publish.BitrateM))
			if s.Publish.VbvMs > 0 {
				enc = append(enc, "cpb-size="+strconv.Itoa(s.Publish.BitrateM*s.Publish.VbvMs))
			}
			return enc
		default:
			assert.Never("unexpected rate-control mode", s.Publish.Mode)
			return nil
		}
	}
}

// The bounds the va elements' rate-control properties impose: the highest kbit/s bitrate accepts,
// and the floor of target-percentage, which is how far under its ceiling a VBR target may sit.
const (
	vaMaxBitrateKbps      = capabilities.VaFieldKb
	vaMinTargetPercentage = 50
)

// vaAbrPeak is the factor abr places its ceiling at above the target,
// so abr reaches the bitrate bound at half the target the other modes need.
const vaAbrPeak = 2

// vaLimits returns why the va elements cannot express these settings, and nil where they can.
//
// Three properties and one rule: the rate, the buffer sized from it, and the keyframe interval,
// each read into a field with a range the plugin declares, plus the VBR target that is a percentage
// of its ceiling.
// A figure outside a property's range is refused rather than moved into it: the ffmpeg engine
// drives the same hardware at the figure the settings name, so a substitution would make the encode
// a function of the capture backend, with no field on the form stating what it runs at.
func vaLimits(s settings.Settings) error {
	var rateM int
	switch s.Publish.Mode {
	case "cbr":
		rateM = s.Publish.BitrateM
	case "abr":
		rateM = s.Publish.BitrateM * vaAbrPeak
	case "vbr":
		if s.Publish.MaxrateM > 0 && s.Publish.BitrateM*100/s.Publish.MaxrateM < vaMinTargetPercentage {
			return text.Fail(screensharev1.TextCode_TEXT_CODE_VA_VBR_TARGET_BELOW_FLOOR, text.CountLimit(vaMinTargetPercentage),
				text.Num(screensharev1.TextArgName_TEXT_ARG_NAME_BITRATE_MBPS, int64(s.Publish.BitrateM)),
				text.Num(screensharev1.TextArgName_TEXT_ARG_NAME_MAXRATE_MBPS, int64(s.Publish.MaxrateM)))
		}
		rateM = s.Publish.MaxrateM
	case "crf", "lossless":
		// crf drives no rate, and lossless has no VAAPI form.
		return nil
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
	if rateM*1000 > vaMaxBitrateKbps {
		return propertyBound(capabilities.FamilyVaapi, "bitrate", vaMaxBitrateKbps, rateM*1000)
	}
	// cpb-size is the rate times the window, in the same kilobits the bitrate property counts, and
	// stops at the same figure.
	// Refused here rather than left to the element, which takes the property as a GLib value and
	// answers an out-of-range one with a warning and a pipeline that never runs.
	if buffer := bufferKb(s); buffer > vaMaxBitrateKbps {
		return propertyBound(capabilities.FamilyVaapi, "cpb-size", vaMaxBitrateKbps, buffer)
	}
	// The interval the elements are handed rather than the field a reader sets: an unset interval
	// is twice the frame rate (gstGop), so a rate past half the property's range overflows it while
	// the control beside it reads zero.
	if gop := gstGop(s); gop > capabilities.VaGopFrames() {
		return propertyBound(capabilities.FamilyVaapi, "key-int-max", capabilities.VaGopFrames(), gop)
	}
	return nil
}

// bufferKb is the rate buffer these settings ask for, in the kilobits every encoder's buffer field
// counts, and zero where the window is left at the encoder's own default.
func bufferKb(s settings.Settings) int {
	if s.Publish.VbvMs <= 0 {
		return 0
	}
	return rateOfBuffer(s) * s.Publish.VbvMs
}

// rateOfBuffer is the rate the buffer is sized against: the ceiling where one bounds the encode,
// and the target where the mode sends one alone (vaEncoder).
func rateOfBuffer(s settings.Settings) int {
	if s.Publish.Mode == capabilities.ModeVbr {
		return s.Publish.MaxrateM
	}
	return s.Publish.BitrateM
}

// vaBitrate spells a Mbit/s rate as the kbit figure the bitrate property takes.
func vaBitrate(rateM int) string {
	return strconv.Itoa(rateM * 1000)
}

// vaTargetPercentage places a VBR target under the ceiling bitrate carries, as a percentage of it.
// A ceiling more than twice the target falls under the property's floor,
// so it never reaches here (vaLimits); one at or below the target reads as 100.
func vaTargetPercentage(s settings.Settings) string {
	pct := 100
	if s.Publish.MaxrateM > 0 {
		pct = s.Publish.BitrateM * 100 / s.Publish.MaxrateM
	}
	return strconv.Itoa(min(pct, 100))
}
