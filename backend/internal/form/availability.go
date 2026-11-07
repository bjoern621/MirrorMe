package form

// Which controls the current settings leave usable, and what greys the ones they
// do not (docs/field-availability.md).
//
// Every verdict reads a table (capabilities, transport, gpupath, publish)
// and restates nothing those tables carry.
// What the rules add is the statement: a code and the identifiers it is about,
// never a sentence (statements.go), so the wording stays the surface's and what is true stays with the rule.
//
// Two contracts govern every rule.
// A disabled control says why, which form.go asserts on each field it renders.
// Where two facts block one field, the reason names the one the user can act on:
// B-frames under software x264 name the families that take a count, not the mode.
//
// An unresolved fact withholds nothing.
// The tables are compiled in, so the only facts that can be missing are the machine's:
// a capture backend with no publisher names no engine, an unprobed machine states no verdict.
// Both leave the control live.

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/gpupath"
	"bjoernblessin.de/mirrorme/internal/publish"
	"bjoernblessin.de/mirrorme/internal/rules"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// fieldState decides one control's visibility and availability.
//
// An unknown key is a caller inventing a control: a plain enabled state for it would put a widget
// on screen that no rule here governs and no repair moves.
// The key is the row's own, which for a repeated control is the template rather than one entry's,
// since the statement is about the control.
func fieldState(d Deps, s settings.Settings, key string, entry int) state {
	return fieldStateOf(availabilityOf(d, s), key, entry)
}

// fieldStateOf answers the same question against an evaluation the caller already holds.
//
// A resolve asks it of every control and of every option of every control,
// and availabilityOf walks the whole rule registry, so the evaluation is made once where the draft
// it describes is fixed and handed down (form.go, repair.go).
func fieldStateOf(av availability, key string, entry int) state {
	rule, ok := availabilityRules[key]
	assert.Assert(ok, "an availability question names a field the form declares", key)

	st := rule(av.forEntry(entry))

	assert.Assert(st.enabled || st.reason != nil, "a disabled control says why", key)
	assert.Assert(st.enabled || st.note == nil, "a note rides on a control that is still editable", key)
	return st
}

// optionState decides one entry of a select or radio, within the field named by key.
//
// A field with no option-level rule leaves every entry enabled:
// whole-control greying is fieldState's,
// and repeating it per entry would grey a dropdown twice over.
func optionState(d Deps, s settings.Settings, key, value string, entry int) (enabled bool, reason *mirrormev1.Text) {
	return optionStateOf(availabilityOf(d, s), key, value, entry)
}

// optionStateOf answers the same question against an evaluation the caller already holds,
// for the reason fieldStateOf takes one.
func optionStateOf(av availability, key, value string, entry int) (enabled bool, reason *mirrormev1.Text) {
	_, declared := availabilityRules[key]
	assert.Assert(declared, "an option question names a field the form declares", key)

	rule, ok := availabilityOptionRules[key]
	if !ok {
		return true, nil
	}
	reason = rule(av.forEntry(entry), value)
	return reason == nil, reason
}

// The four ways a state can read, one constructor each, so a rule states which treatment it applies
// rather than assembling a struct that could silently be a fifth.

// availabilityLive is a control this combination blocks in no way.
func availabilityLive() state {
	return state{visible: true, enabled: true}
}

// availabilityNoted is a control that stays editable
// and means something its label does not describe here.
// A nil note yields a plain live control,
// which keeps a caller from branching on whether it has one.
func availabilityNoted(note *mirrormev1.Text) state {
	return state{visible: true, enabled: true, note: note}
}

// availabilityDisabled is a control the combination blocks,
// with the statement shown in place of the value.
func availabilityDisabled(reason *mirrormev1.Text) state {
	assert.IsNotNil(reason, "a disabled control says why")
	return state{visible: true, enabled: false, reason: reason}
}

// availabilityHidden is a backend implementation knob outside the one selection it belongs to.
// Enabled because nothing draws it: the disabled-says-why contract would otherwise demand
// a statement for a widget never on screen.
func availabilityHidden() state {
	return state{visible: false, enabled: true}
}

// availabilityShownFor is the hidden treatment as a condition,
// for a knob that appears with the one protocol it belongs to.
func availabilityShownFor(shown bool) state {
	if !shown {
		return availabilityHidden()
	}
	return availabilityLive()
}

// availability is what a rule reads beyond the fixed tables: the machine, the draft
// and the facts derived from them.
//
// They are derived here rather than inside each rule because each is a lookup with a failure mode
// of its own, and a rule repeating one would be a second
// place the same "not resolved" case is decided.
type availability struct {
	deps Deps
	s    settings.Settings

	// verdicts is what the rule system says about this draft.
	// Every fact the domain tables state arrives through it rather than being looked up per site,
	// so a control's greying and the publish's own refusal
	// are two readings of one evaluation (internal/rules).
	verdicts rules.Verdicts

	// engine is the publish engine the selected capture backend runs.
	// Empty exactly where the settings name a backend this app has no publisher for,
	// which a hand-edited settings file can do.
	// Every rule keyed by engine then withholds nothing, a fact nobody stated being no limit.
	engine string

	// codec is the capability row of the selected codec, knownCodec whether the table has one.
	// A codec outside the table has no family, format or gaps, so the rules reading them state nothing
	// rather than guessing a half.
	codec      capabilities.Codec
	knownCodec bool

	// path is the frame-memory row of the selected capture backend
	// and the codec's family on that engine, onDevicePath whether the pair has one.
	// A pair rather than a property of either end: the portal capture shares memory with a VAAPI
	// encoder and not with an x264 one.
	path         gpupath.Path
	onDevicePath bool

	// entry is which entry of the audio source list the question is about,
	// noEntry for a control belonging to no list.
	// It rides here rather than as a second argument to every rule,
	// since a rule that does not repeat has no use
	// for it and a table with two signatures would be two tables.
	entry int
	// source is that entry, which for the row past the end
	// of the list is the default: no kind, unity gain, unmuted.
	// The row a reader grows the list by holds exactly that, so a rule reading it needs no branch
	// for the row nothing stored.
	source settings.AudioSource
}

func (av availability) forEntry(entry int) availability {
	av.entry = entry
	av.source = audioEntry(av.s, entry)
	return av
}

func availabilityOf(d Deps, s settings.Settings) availability {
	av := availability{deps: d, s: s, verdicts: verdictsOf(d, s)}
	if engine, err := publish.EngineFor(s.Publish.Capture); err == nil {
		av.engine = engine
	}
	av.codec, av.knownCodec = capabilities.Get(s.Publish.Codec())
	if av.engine != "" && av.knownCodec {
		av.path, av.onDevicePath = gpupath.For(av.engine, s.Publish.Capture, av.codec.Family)
	}
	return av
}

// mode is what the selected rate-control concept needs from the encoder.
// A mode the table does not name needs nothing, which greys every knob with its own statement
// rather than offering knobs no builder would read.
func (av availability) mode() availabilityMode {
	return availabilityModes[av.s.Publish.Mode]
}

// family is what the selected codec's encoder family takes.
// A codec outside the table has no family, and the zero row takes neither field,
// so both grey naming who does take them.
func (av availability) family() availabilityFamily {
	if !av.knownCodec {
		return availabilityFamily{}
	}
	return availabilityFamilies[av.codec.Family]
}
