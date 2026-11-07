package capabilities

import (
	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/text"
)

// Ladder is one encoder knob's steps, and where each rate-control mode starts on it.
//
// The steps are the encoder's own identifiers rather than a normalized scale,
// a normalized one lying:
// x264's "slow" and SVT-AV1's preset 8 are not the same trade,
// and a number carried across a codec change would land on a different real setting.
// A shell names each step and renders one it has no name for as the identifier,
// which lets a codec gain a ladder with nothing edited there (docs/domain-model.md).
//
// Steps run from the most effort to the least, whatever direction the encoder's own numbering takes,
// so a control is drawn without knowing which way each ladder counts.
type Ladder struct {
	// Steps are the values the encoder takes, most effort first.
	// Empty is an encoder without the knob, which greys the control with the reason.
	Steps []string `json:"steps"`
	// Defaults is the step each rate-control mode starts on, keyed as Modes names them.
	// A mode with no entry leaves the knob unset,
	// which is the encoder's own default rather than a step of this ladder.
	Defaults map[string]string `json:"defaults"`
	// Pins are the modes that fix the step rather than starting on it,
	// so the control is greyed there and the statement names the step in force.
	//
	// A fact about the encoder and not about the mode:
	// NVENC pins its preset in CBR, a low-latency preset being what lets it hold a constant rate,
	// and x264 in the same mode takes whatever step it is given.
	// A flag on the mode would grey every codec's control wherever one of them pins.
	Pins []string `json:"pins"`
	// Drivers holds the value a driver reads each step as, keyed by driver and then by step.
	// A driver with no entry reads every step as written.
	Drivers map[string]map[string]string `json:"-"`
}

// SpelledFor is the value the driver on device reads step as.
// The empty step stays empty: a knob left alone needs no spelling.
func (l Ladder) SpelledFor(device Device, step string) string {
	assert.Assert(step == "" || l.Has(step), "a spelled step lies on its ladder", step)

	spellings, ok := l.Drivers[device.Driver]
	if !ok || step == "" {
		return step
	}
	spelled, ok := spellings[step]
	assert.Assert(ok, "a driver spelling covers every step of its ladder", device.Driver, step)
	return spelled
}

func (l Ladder) PinsIn(mode string) bool {
	return contains(l.Pins, mode)
}

// StepFor is where a mode starts on this ladder,
// and false where the ladder leaves the knob to the encoder in that mode.
func (l Ladder) StepFor(mode string) (string, bool) {
	step, ok := l.Defaults[mode]
	return step, ok
}

func (l Ladder) Has(step string) bool {
	return contains(l.Steps, step)
}

// Resolve is what one encode spends on this ladder, and false for a step the encoder does not take.
//
// An empty answer means spend nothing and leave the knob at the encoder's own default.
// A ladder with no steps is an encoder without the knob,
// and a mode the Defaults leave out is one that default is right for.
//
// A held step wins wherever the ladder carries it,
// which is what makes the control a choice rather than a display of the row.
// It loses in a mode that pins the knob, and a pinned mode refuses nothing:
// the control is greyed there, so the value behind the greying is one nothing spends,
// and refusing a publish over it would name a field nobody can reach.
func (l Ladder) Resolve(mode, held string) (string, bool) {
	if len(l.Steps) == 0 {
		return "", true
	}
	if l.PinsIn(mode) || held == "" {
		step, _ := l.StepFor(mode)
		return step, true
	}
	if l.Has(held) {
		return held, true
	}
	return "", false
}

// Steps are the two ladder steps one encode spends, in the encoder's own identifiers.
// Either is empty where its ladder leaves the knob alone.
type Steps struct {
	Effort, Tune string
}

// ResolveSteps is what an encode of this codec in this mode spends on both ladders,
// and a refusal for a step the encoder does not take.
//
// Both builders read it, which is the point.
// The step is a fact about the encoder, not of the engine that builds the command.
// An encode that followed the engine would tie a stream's picture to its capture backend.
// Each engine still owns the spelling: ffmpeg's -preset, the x264 element's speed-preset,
// and the empty step, which every builder answers by passing no such option at all.
// device decides the effort value its driver reads (Ladder.Drivers).
//
// A refusal is an Umgebungsfehler,
// keeping a hand-edited settings file off an encoder's own error path:
// the field is free-form text,
// the repair walks a step off the ladder back onto it,
// and nothing between the two bounds a value that reached the settings by another route.
func (c Codec) ResolveSteps(mode, effort, tune string, device Device) (Steps, error) {
	e, ok := c.Effort.Resolve(mode, effort)
	if !ok {
		return Steps{}, text.Fail(screensharev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN,
			text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_EFFORT, effort))
	}
	t, ok := c.Tune.Resolve(mode, tune)
	if !ok {
		return Steps{}, text.Fail(screensharev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN,
			text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_TUNE, tune))
	}
	return Steps{Effort: c.Effort.SpelledFor(device, e), Tune: t}, nil
}

// everyMode spreads one step across every rate-control mode,
// for a ladder whose step does not follow what the encoder is aiming at.
func everyMode(step string) map[string]string {
	assert.Assert(step != "", "a ladder default names the step it spends")

	out := make(map[string]string, len(Modes))
	for _, mode := range Modes {
		out[mode] = step
	}

	assert.Assert(len(out) == len(Modes), "a spread step covers every rate control", len(out))
	return out
}
