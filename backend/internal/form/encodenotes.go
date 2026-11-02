package form

import (
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/settings"
)

// autoGopSeconds is the keyframe interval a gop of 0 leaves the encoder at (settings.Publish.Gop).
const autoGopSeconds = 2

// diagnosticsAboutTheEncode says where the encode runs and how long a late viewer waits for a picture.
// Both are notes: neither costs the stream anything, and each tells two entries of its control apart.
func diagnosticsAboutTheEncode(s settings.Settings) []*screensharev1.Diagnostic {
	var out []*screensharev1.Diagnostic

	if c, known := capabilities.Get(s.Publish.Codec()); known && c.Family != capabilities.FamilySoftware {
		out = append(out, diagnosticFor(screensharev1.Severity_SEVERITY_INFO, KeyEncoder,
			say(encodesOnGPU, argFamily(c.Family))))
	}

	// A decoder starts on a keyframe, so a viewer opening mid-interval waits for the next one.
	if s.Publish.Fps > 0 {
		wait := time.Duration(autoGopSeconds) * time.Second
		if s.Publish.Gop > 0 {
			wait = time.Duration(s.Publish.Gop) * time.Second / time.Duration(s.Publish.Fps)
		}
		out = append(out, diagnosticFor(screensharev1.Severity_SEVERITY_INFO, KeyGop, say(joinWait, argJoinWaitMs(wait))))
	}
	return out
}
