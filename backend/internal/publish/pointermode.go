package publish

import (
	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/framestamp"
	"bjoernblessin.de/screenshare/internal/pointermode"
	"bjoernblessin.de/screenshare/internal/portal"
	"bjoernblessin.de/screenshare/internal/rules"
)

// What each capture backend can do with the pointer.
//
// A table for the reason captureNeeds beside it is one: the same fact is otherwise answered
// by whoever asks.
// The axis is the capture backend rather than the codec, which a Gap cannot express, so these
// are written as rules rather than converted into them (docs/domain-model.md, "One evaluator").

// pointerServes is the modes each capture backend serves, in Modes order.
//
// A backend absent from the map is a bug rather than a backend serving nothing, which init asserts
// against the registry: a backend missing here would otherwise offer every mode and pass a flag
// nothing reads.
var pointerServes = map[string][]string{
	// draw_mouse on the two ffmpeg grabbers, show-cursor on the GStreamer one.
	// None of the three reports a pointer position of its own.
	"ddagrab":               {pointermode.Embedded, pointermode.Hidden},
	"gdigrab":               {pointermode.Embedded, pointermode.Hidden},
	"d3d11screencapturesrc": {pointermode.Embedded, pointermode.Hidden},

	// draw_mouse on the ffmpeg side, show-pointer on the GStreamer one.
	// X11 draws the pointer into the image and answers any client asking where it is, what
	// the metadata mode reads there (internal/pointer).
	// Only the GStreamer row serves that mode: the publish child reports the position, and the ffmpeg
	// engine's child is ffmpeg.
	"x11grab":   {pointermode.Embedded, pointermode.Hidden},
	"ximagesrc": {pointermode.Embedded, pointermode.Hidden, pointermode.Metadata},

	// kmsgrab reads the scanout's primary plane, and the pointer is a hardware plane of its own
	// that the scanout composes at display time.
	// Nothing on that path draws it into the frames, so the capture is cursorless whatever it is asked
	// for, and hidden is the only mode describing what it does.
	"kmsgrab": {pointermode.Hidden},

	// cursor_mode's metadata value sends the position beside each frame instead of drawing it,
	// and pipewiresrc hands it on as a region of interest the publish child reads
	// (internal/portal, gstrun/pointersource.go).
	"portal": {pointermode.Embedded, pointermode.Hidden, pointermode.Metadata},

	// -capture_cursor on the ffmpeg side, capture-screen-cursor on the GStreamer one.
	"avfoundation": {pointermode.Embedded, pointermode.Hidden},
	"avfvideosrc":  {pointermode.Embedded, pointermode.Hidden},
}

// The table describes the same set of backends the registry does, so a row in one and not the other
// is a backend whose pointer behaviour nobody stated, or an entry for a backend nobody registered.
// Both are bugs in this package, so they fail at load.
func init() {
	assert.Assert(len(pointerServes) == len(captureBackends),
		"a capture backend states what it does with the pointer exactly once",
		len(pointerServes), len(captureBackends))
	assert.Assert(len(pointerServes[capturePortal]) > 0, "the portal backend has a pointer row", capturePortal)
	for name, serves := range pointerServes {
		_, ok := captureBackends[name]
		assert.Assert(ok, "a backend with a pointer row is one the registry carries", name)
		assert.Assert(len(serves) > 0, "a capture backend serves at least one pointer mode", name)
		for _, mode := range serves {
			assert.Assert(pointermode.Known(mode), "a served pointer mode is one the settings name", name, mode)
		}
	}
	rules.Register(pointerRules()...)
}

// pointerRules is what each backend cannot do with the pointer, plus the limits that are this app's
// rather than any backend's.
func pointerRules() []rules.Rule {
	var out []rules.Rule
	for _, name := range Captures() {
		for _, mode := range pointermode.Modes {
			if serves(name, mode) {
				continue
			}
			out = append(out, rules.Rule{
				When:    map[string]rules.Match{rules.AxisCapture: rules.OneOf(name)},
				Verdict: rules.Refuse,
				Field:   rules.AxisPointer,
				Values:  rules.OneOf(mode),
				Reason:  pointerRefusal(name, mode),
			})
		}
	}

	// The position reaches a viewer inside the encoded frames, so a bitstream with no unit to write
	// one into carries no pointer however well the capture reads it (internal/framestamp).
	//
	// Stated apart from the backend rows above, the two sending a reader to different places:
	// a capture with no position to report is answered by picking another capture,
	// and a format that cannot carry one by picking another format.
	// Derived from what framestamp writes rather than listed here: a format added to the capability
	// table is refused until that package says it carries a stamp, where a list would silently offer
	// the mode on it.
	for _, format := range capabilities.Formats() {
		if framestamp.Carries(format) {
			continue
		}
		out = append(out, rules.Rule{
			When:    map[string]rules.Match{rules.AxisFormat: rules.OneOf(format)},
			Verdict: rules.Refuse,
			Field:   rules.AxisPointer,
			Values:  rules.OneOf(pointermode.Metadata),
			Reason:  screensharev1.TextCode_TEXT_CODE_FORMAT_CARRIES_NO_CURSOR_METADATA,
		})
	}
	return out
}

// pointerRefusal is why a backend does not serve a mode.
//
// Two facts rather than one sentence with the backend's name in it, the two sending a reader
// to different places.
// A capture that cannot draw the pointer is a property of how it reads the screen, and a capture
// with no position to report is a property of what it reports beside the picture.
func pointerRefusal(capture, mode string) screensharev1.TextCode {
	if mode == pointermode.Metadata {
		return screensharev1.TextCode_TEXT_CODE_CAPTURE_HAS_NO_CURSOR_METADATA
	}
	assert.Assert(capture == "kmsgrab",
		"the only backend that cannot draw the pointer is the scanout one", capture, mode)
	return screensharev1.TextCode_TEXT_CODE_KMSGRAB_HAS_NO_CURSOR_PLANE
}

func serves(capture, mode string) bool {
	for _, m := range pointerServes[capture] {
		if m == mode {
			return true
		}
	}
	return false
}

// gstBool is how a GStreamer element spells a boolean property value.
// One helper, three elements setting the same fact through three differently named properties, and
// a literal typed per site can come out as "1" at one of them.
func gstBool(on bool) string {
	if on {
		return "true"
	}
	return "false"
}

// portalCursor is the pointer setting as the ScreenCast interface's own cursor_mode.
//
// A mapping and not a boolean, the portal's three modes being the settings' three: its metadata
// value sends the position out of band instead of drawing it.
func portalCursor(mode string) portal.CursorMode {
	switch mode {
	case pointermode.Embedded:
		return portal.CursorEmbedded
	case pointermode.Hidden:
		return portal.CursorHidden
	case pointermode.Metadata:
		return portal.CursorMetadata
	default:
		assert.Never("a portal capture names a pointer mode the settings carry", mode)
		return portal.CursorEmbedded
	}
}

// PointerServed reports whether the capture backend serves this pointer mode, for a caller choosing
// a value rather than greying one.
// A form shows the rules' answer.
// This is the same table asked directly, for the repair, which has to land on a mode the backend
// runs.
func PointerServed(capture, mode string) bool {
	assert.Assert(pointermode.Known(mode), "a pointer question names a mode the settings carry", mode)
	return serves(capture, mode)
}

// capturePortal is the one backend whose pointer modes this machine answers for rather than
// the table above.
const capturePortal = "portal"

// PointerServedHere reports whether this machine leaves a pointer mode reachable on a capture
// backend, given what the desktop portal answered (internal/portal, Capabilities).
//
// The rows above state which modes a backend's interface can express,
// which is a fact about the software this app drives.
// Whether the ScreenCast call is answered is the compositor's, so it is asked rather than declared:
// wlroots serves hidden and embedded where KWin serves all three, on one capture backend and one
// table row.
// Every other backend sets a property of its own and needs no asking.
func PointerServedHere(caps portal.Capabilities, capture, mode string) bool {
	assert.Assert(pointermode.Known(mode), "a pointer question names a mode the settings carry", mode)

	if capture != capturePortal {
		return true
	}
	return caps.ServesCursor(portalCursor(mode))
}
