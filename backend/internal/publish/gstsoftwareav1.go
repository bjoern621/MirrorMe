package publish

import (
	"strconv"
	"strings"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// aomEncoder maps the rate-control mode onto av1enc, the libaom encoder and the counterpart
// to the ffmpeg libaom-av1 branch.
// usage-profile=realtime takes libaom off its two-pass defaults, and cpu-used, row-mt and
// lag-in-frames=0 are the same realtime trade as there.
//
// end-usage=q is libaom's unbounded constant-quality mode, which vpxenc's bitrate-capped cq is not.
// The element exposes no cq-level, so the quantizer target is pinned by setting both quantizer
// bounds to it.
// end-usage=vbr takes no ceiling, so vbr is a gap on this engine (gstNoRateCeiling).
// target-bitrate is in kbit/s and buf-sz in milliseconds.
func aomEncoder(s settings.Settings, r gstRates, l capabilities.Steps) []string {
	base := []string{
		"av1enc", "usage-profile=realtime", "cpu-used=" + l.Effort, "row-mt=true",
		"lag-in-frames=0", "keyframe-max-dist=" + r.gop,
	}
	switch s.Publish.Mode {
	case "crf":
		return append(base, "end-usage=q", "min-quantizer="+r.cq, "max-quantizer="+r.cq)
	case "abr":
		return append(base, "end-usage=vbr", "target-bitrate="+r.kbps)
	case "cbr":
		enc := append(base, "end-usage=cbr", "target-bitrate="+r.kbps)
		if s.Publish.VbvMs > 0 {
			enc = append(enc, "buf-sz="+strconv.Itoa(s.Publish.VbvMs))
		}
		return enc
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
}

// svtav1Encoder maps the rate-control mode onto svtav1enc, the counterpart to the ffmpeg libsvtav1
// branch.
// Two of the library's constraints are gaps in capabilities.Codecs rather than letting
// a mode run as another: max-bitrate is refused outside constant-quality mode, so vbr has no form
// on either engine, and cbr's low-delay prediction structure stalls this element,
// so cbr has none on this one.
// What the library takes rather than the element travels in parameters-string, the element's own
// pass-through for the keys it puts no property on, which is where the ffmpeg branch puts the same
// two keys (ffmpeg/encoders.go, svtav1Params).
func svtav1Encoder(s settings.Settings, r gstRates, l capabilities.Steps) []string {
	base := append([]string{"svtav1enc", "preset=" + l.Effort, "intra-period-length=" + r.gop},
		svtav1Parameters(l)...)
	switch s.Publish.Mode {
	case "crf":
		return append(base, "crf="+r.cq)
	case "abr":
		return append(base, "target-bitrate="+r.kbps)
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
}

// svtav1Parameters is the element's parameters-string: the screen-content tools every encode here
// carries, and the tune step where the row's ladder resolved one.
//
// A gst-launch line splits on spaces, so the value carries none, which the colon-separated form it
// takes satisfies on its own.
func svtav1Parameters(l capabilities.Steps) []string {
	params := []string{"scm=1"}
	if tune, named := capabilities.Svtav1TuneValue(l.Tune); named {
		params = append(params, "tune="+tune)
	}
	return []string{"parameters-string=" + strings.Join(params, ":")}
}

// rav1eEncoder maps the rate-control mode onto rav1enc, the counterpart to the ffmpeg librav1e
// branch.
// rav1e's rate control is one bitrate target with no ceiling and no rate buffer,
// so vbr has no form on either engine and cbr differs from abr in dropping frame reordering
// for delay.
// bitrate is in bits/sec and the quantizer counts to 255.
// The element exposes no keyframe interval, so the configured GOP never reaches it and rav1e's own
// default stands.
// The tune travels on a property here where ffmpeg takes it in -rav1e-params, the step being
// rav1e's own on both.
func rav1eEncoder(s settings.Settings, r gstRates, l capabilities.Steps) []string {
	base := append([]string{"rav1enc", "speed-preset=" + l.Effort}, tuneProperty(l.Tune)...)
	switch s.Publish.Mode {
	case "crf":
		return append(base, "quantizer="+r.cq)
	case "abr":
		return append(base, "bitrate="+strconv.Itoa(s.Publish.BitrateM*1_000_000))
	case "cbr":
		// low-latency is not written here: gstLiveDelay pins it on this element in every mode, and
		// a second site writing it puts the property on the launch line twice.
		return append(base, "bitrate="+strconv.Itoa(s.Publish.BitrateM*1_000_000))
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
}

// tuneProperty is one tune step on an element spelling the knob "tune", and nothing where
// the ladder resolved no step or resolved the untuned one.
// Passing no such property is how an element expresses "tune for nothing", every tune enum here
// starting at a real step rather than at an absence.
func tuneProperty(step string) []string {
	if step == "" || step == capabilities.TuneNone {
		return nil
	}
	return []string{"tune=" + step}
}
