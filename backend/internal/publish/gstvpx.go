package publish

import (
	"slices"
	"strconv"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/settings"
)

// vpxEncoder maps the rate-control mode onto vp8enc or vp9enc, the counterpart to the ffmpeg libvpx
// branches.
// One mapping serves both elements: they expose the same libvpx properties,
// and what differs (VP9's row-mt, the cpu-used point each is worth running at) is bound per codec
// in gstCodecs.
//
// deadline=1 is libvpx's realtime deadline, and static-threshold=100 is the motion threshold
// the elements' own documentation names for screen and window sharing.
// end-usage picks the rate-control family: cq with cq-level is constant quality,
// where the bitrate is a burst cap rather than a target,
// so libvpx has no unbounded constant-quality mode here as the ffmpeg path's -b:v 0 gives.
// end-usage=vbr is the average-bitrate family and takes no ceiling, so vbr has no branch and
// is a gap on this engine (gstNoRateCeiling), as lossless is for want of a property.
// target-bitrate counts in bits/sec and buffer-size in ms, where x264enc and x265enc count kbit.
func vpxEncoder(elem string, extra ...string) func(settings.Settings, gstRates, capabilities.Steps) []string {
	return func(s settings.Settings, r gstRates, l capabilities.Steps) []string {
		bps := strconv.Itoa(s.Publish.BitrateM * 1_000_000)
		base := slices.Concat([]string{elem, "deadline=1", "static-threshold=100",
			"cpu-used=" + l.Effort, "keyframe-max-dist=" + r.gop}, extra, vpxTuneProperty(l.Tune))
		switch s.Publish.Mode {
		case "crf":
			// libvpx codes constant quality toward a target it stays under, so the ceiling is what
			// that target carries: a zero here is a rate derived from the picture size and not an absence
			// of one, so the row requires a ceiling (capabilities.QualityCeilingRequired, vpxRateLimits).
			ceiling := strconv.Itoa(s.Publish.MaxrateM * 1_000_000)
			return append(base, "end-usage=cq", "cq-level="+r.cq, "target-bitrate="+ceiling)
		case "abr":
			return append(base, "end-usage=vbr", "target-bitrate="+bps)
		case "cbr":
			enc := append(base, "end-usage=cbr", "target-bitrate="+bps)
			if s.Publish.VbvMs > 0 {
				enc = append(enc, "buffer-size="+strconv.Itoa(s.Publish.VbvMs))
			}
			return enc
		default:
			assert.Never("unexpected rate-control mode", s.Publish.Mode)
			return nil
		}
	}
}

// vpxRateLimits refuses a constant-quality encode with no ceiling, which the vpx elements have no
// form of: their CQ codes toward a target, and a target of zero is the rate libvpx derives
// from the picture size rather than an absence of one.
// The form offers no zero here either (fieldMaxrateBounds), so this is what a settings file
// that never went through it meets.
func vpxRateLimits(s settings.Settings) error {
	if s.Publish.Mode != capabilities.ModeCrf || s.Publish.MaxrateM > 0 {
		return nil
	}
	return text.Fail(screensharev1.TextCode_TEXT_CODE_QUALITY_CEILING_REQUIRED, text.Codec(s.Publish.Codec()))
}

// vpxTuneProperty spells one tune step on vp8enc and vp9enc, which name the knob "tuning", and
// returns nothing where the ladder resolved no step or the untuned one.
// The steps are libvpx's own two values on both engines, ffmpeg taking them on -tune.
func vpxTuneProperty(step string) []string {
	if step == "" || step == capabilities.TuneNone {
		return nil
	}
	return []string{"tuning=" + step}
}
