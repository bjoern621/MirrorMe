package form

import (
	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/screenshare/internal/text"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/publish"
	"bjoernblessin.de/screenshare/internal/settings"
)

// summarize is what a shell shows without deriving it: the command these settings would run,
// and the prediction the diagnostics were ranked against.
//
// The estimate is the one handed in rather than a fresh computation.
// A second computation is a second answer, and a summary disagreeing with the diagnostic beside
// it is the fork this package exists to remove.
//
// No headline and no per-group shorthand.
// Both pick a separator, an abbreviation and a length, which belong where the strip they sit
// in is laid out, and the values are on Form.settings for a surface to compose
// its own at its own width (api/proto/screenshare/v1/form.proto).
func summarize(_ Deps, s settings.Settings, est *screensharev1.Estimate) *screensharev1.Summary {
	command, refusal := formCommand(s)

	sum := &screensharev1.Summary{
		Command:        command,
		CommandRefusal: refusal,
		Estimate:       est,
	}

	assert.Assert((sum.GetCommand() == "") != (sum.GetCommandRefusal() == nil),
		"a summary carries a command or the reason there is none", sum.GetCommand())
	return sum
}

// formCommand renders the pipeline these settings would run, or the reason none renders.
//
// The one place either is asked for, so the command the summary shows
// and the error the diagnostics rank cannot be two answers about one draft.
// publish.Command renders a string and starts nothing: the ffmpeg engine assembles an argument
// list, the GStreamer engine describes its capture chain rather than opening it,
// and neither spawns a process or acquires a portal session.
// It does read this machine, the X display, the monitor list and the render nodes,
// since those reads are what refuse a command the button beside it would refuse.
//
// The refusal is the one a start of these settings meets (docs/ipc-api.md, "Errors").
func formCommand(s settings.Settings) (command string, refusal *screensharev1.Refusal) {
	cmd, err := publish.Command(s)
	if err != nil {
		statement, words := text.Statement(err)
		return "", &screensharev1.Refusal{Text: statement, Message: words}
	}
	return cmd, nil
}
