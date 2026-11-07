package publish

import (
	"strconv"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// qsvAbrPeak is the factor abr places its ceiling at above the target, the derivation the ffmpeg
// builder's QSV mapping makes too, so an unbounded average means one thing on both publish engines.
const qsvAbrPeak = 2

// qsvShortBitrateKbps is the highest rate the AV1 and VP9 qsv elements take: their bitrate and
// max-bitrate properties are unsigned 16-bit where the H.264 and H.265 elements' are unbounded.
const qsvShortBitrateKbps = 65535

// qsvShortRateLimits returns why an element with the 16-bit bitrate property cannot express
// the settings' rates, and nil where it can.
// Refused rather than moved into range, as on the va elements: the ffmpeg engine drives the same
// hardware at the rate the settings name, so a substitution would make the bitrate a function
// of the capture backend.
func qsvShortRateLimits(s settings.Settings) error {
	var rateM int
	switch s.Publish.Mode {
	case "cbr":
		rateM = s.Publish.BitrateM
	case "abr":
		rateM = s.Publish.BitrateM * qsvAbrPeak
	case "vbr":
		rateM = s.Publish.MaxrateM
	case "crf", "lossless":
		// crf drives no rate, and lossless has no QSV form.
		return nil
	default:
		assert.Never("unexpected rate-control mode", s.Publish.Mode)
		return nil
	}
	if rateM*1000 > qsvShortBitrateKbps {
		return propertyBound(capabilities.FamilyQsv, "bitrate", qsvShortBitrateKbps, rateM*1000)
	}
	return nil
}

// qsvEncoder maps the rate-control mode onto one qsv element's properties, the counterpart
// to qsvArgs in the ffmpeg builder.
// The mode is named on rate-control rather than derived from which options carry a value, the one
// place this engine's QSV path is the plainer of the two.
//   - crf: rate-control=cqp with the quantizer on qp-i and qp-p.
//   - cbr: rate-control=cbr at the target bitrate, plus low-latency, which drops the encoder
//     from four frames in flight to one.
//   - vbr: rate-control=vbr at the target with max-bitrate as the ceiling.
//   - abr: the same VBR with the ceiling at twice the target, since oneVPL codes against a maximum
//     either way and picks one the settings never stated where it is left unset.
//
// One mapping serves every qsv codec: the elements derive from one base class and spell
// the quantizer properties alike.
// The H.264 and H.265 elements count their quantizer to 51 and the AV1 and VP9 ones to 255, each
// capability row's CqMax.
//
// bitrate and max-bitrate are in kbit/s.
// The VBV window reaches no qsv element: they expose no rate-buffer property,
// so the settings' figure binds on the ffmpeg engine alone.
// The effort step travels as target-usage, the property oneVPL's scale is spelled on here.
// No B-frame count: the elements code no B-pictures unless asked, as the ffmpeg mapping pins them
// off.
func qsvEncoder(elem string) func(settings.Settings, gstRates, capabilities.Steps) []string {
	return func(s settings.Settings, r gstRates, l capabilities.Steps) []string {
		base := []string{elem, "gop-size=" + r.gop}
		if l.Effort != "" {
			base = append(base, "target-usage="+l.Effort)
		}
		switch s.Publish.Mode {
		case "crf":
			return append(base, "rate-control=cqp",
				"qp-i="+r.cq, "qp-p="+r.cq)
		case "abr":
			return append(base, "rate-control=vbr",
				"bitrate="+r.kbps, "max-bitrate="+strconv.Itoa(s.Publish.BitrateM*1000*qsvAbrPeak))
		case "vbr":
			return append(base, "rate-control=vbr",
				"bitrate="+r.kbps, "max-bitrate="+r.maxkbps)
		case "cbr":
			return append(base, "rate-control=cbr",
				"bitrate="+r.kbps, "low-latency=true")
		default:
			assert.Never("unexpected rate-control mode", s.Publish.Mode)
			return nil
		}
	}
}
