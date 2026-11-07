package form

import (
	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/publish"
)

// captureReason states why this machine cannot run a capture backend.
//
// The verdict and the statement are both publish's, read through rather than restated:
// the catalog shows the same list and has to give the same answer.
//
// A privilege the backend needs is no greying (publish.Grant).
// The process either holds it or the capture dies at launch and nothing can tell which in advance,
// so the entry stays selectable and what it needs granted rides on the option as a note.
func (av availability) captureReason(capture string) *screensharev1.Text {
	available, reason := publish.Available(capture, av.deps.Platform)
	if available {
		return nil
	}
	assert.IsNotNil(reason, "an unavailable capture backend says why", capture)
	return reason
}

// pointerReason states why the selected capture backend does not serve a pointer mode.
//
// Two sources, and the machine's is asked first.
// What a backend does with the pointer is a per-backend table written as rules
// (internal/publish/pointermode.go), and the one limit that is this app's rather than any backend's,
// that a bitstream carries no position, is a rule beside them.
// The desktop portal's own list is neither: it is the compositor's answer, so no table here holds it
// and a machine that refuses the mode outright is named ahead of a format a reader could change.
func (av availability) pointerReason(value string) *screensharev1.Text {
	if !publish.PointerServedHere(av.deps.Portal, av.s.Publish.Capture, value) {
		return say(portalServesNoCursorMode, argCapture(av.s.Publish.Capture), argCursor(value))
	}
	if reasons := av.verdicts.ValueReasons(KeyPointer, value); len(reasons) > 0 {
		return reasons[0]
	}
	return nil
}
