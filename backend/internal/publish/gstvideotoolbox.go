package publish

import (
	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// vtEncoder maps the one rate-control mode Apple's framework reaches onto one applemedia element,
// the counterpart to videoToolboxArgs in the ffmpeg builder.
//
// abr is all the rows declare, so there is no switch: bitrate is the average the framework codes
// towards, and it counts in kbit/s where the ffmpeg option counts in bits.
// realtime stops the encoder buffering frames to spend longer on them, and frame reordering goes
// off with it, which is the same pair the ffmpeg mapping sets.
func vtEncoder(elem string) func(settings.Settings, gstRates, capabilities.Steps) []string {
	return func(s settings.Settings, r gstRates, _ capabilities.Steps) []string {
		assert.Assert(s.Publish.Mode == capabilities.ModeAbr,
			"a VideoToolbox encode runs the one rate control its rows declare", s.Publish.Mode)

		return []string{
			elem, "realtime=true", "allow-frame-reordering=false",
			"max-keyframe-interval=" + r.gop, "bitrate=" + r.kbps,
		}
	}
}
