package form

import (
	"slices"

	"bjoernblessin.de/go-utils/util/assert"
	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/wire"
)

// pick holds what the settings come to with one entry of a control picked and nothing else moved.
type pick struct {
	estimate *screensharev1.Estimate
	impacts  []*screensharev1.Diagnostic
}

// pickOptions writes onto each enabled entry of one control what picking it would do:
// its price (FieldOption.estimate) and its diagnostics (FieldOption.impacts).
//
// Each is kept only where the entries disagree on it:
// a figure or a statement repeated down a list says nothing about the choice it sits beside.
// Which controls carry either is therefore read off the estimate and the rules rather than listed,
// so a setting that a rule starts reading marks its entries with nothing here to edit.
func pickOptions(d Deps, s settings.Settings, key string, options []*screensharev1.FieldOption) {
	picks := make([]*pick, len(options))
	for i, o := range options {
		if o.GetEnabled() {
			picks[i] = picked(d, s, key, o.GetValue())
		}
	}

	pricesDiffer := pickPricesDiffer(picks)
	shared := pickSharedImpacts(picks)
	for i, o := range options {
		if picks[i] == nil {
			continue
		}
		if pricesDiffer {
			o.Estimate = picks[i].estimate
		}
		o.Impacts = slices.DeleteFunc(picks[i].impacts, func(w *screensharev1.Diagnostic) bool {
			return slices.Contains(shared, pickImpactOf(w))
		})
	}
}

// picked evaluates these settings with one control set to one of its entries.
// The estimate is nil where the result cannot be priced (estimate).
func picked(d Deps, s settings.Settings, key, value string) *pick {
	m := wire.Settings(s)
	target, descriptor, ok := settingsField(m, key)
	assert.Assert(ok, "a picked control names a setting the message holds", key)

	v, ok := repairValue(descriptor, value)
	assert.Assert(ok, "a picked entry parses as the setting it offers", key, value)
	target.Set(descriptor, v)

	reached := walked(m, s)
	est := estimate(d, reached)
	return &pick{estimate: est, impacts: ranked(diagnosticsOfTheSettings(d, reached, est))}
}

func pickPricesDiffer(picks []*pick) bool {
	var first *screensharev1.Estimate
	for _, p := range picks {
		if p == nil || p.estimate == nil {
			continue
		}
		if first == nil {
			first = p.estimate
			continue
		}
		if p.estimate.GetBitrateMbps() != first.GetBitrateMbps() {
			return true
		}
	}
	return false
}

// pickImpact names a diagnostic apart from its figures,
// so an uplink shortfall at 30 fps and at 60 fps count as one statement.
// A statement in pickFigureClaims keeps its figure, two figures there being two claims.
type pickImpact struct {
	code     screensharev1.TextCode
	severity screensharev1.Severity
	anchor   string
	figure   string
}

// pickFigureClaims are the statements whose figure is what they claim:
// "0.4 s behind" and "2.7 s behind" tell two entries apart.
var pickFigureClaims = []screensharev1.TextCode{watchDelay}

func pickImpactOf(w *screensharev1.Diagnostic) pickImpact {
	i := pickImpact{code: w.GetText().GetCode(), severity: w.GetSeverity(), anchor: w.GetFieldKey()}
	if slices.Contains(pickFigureClaims, i.code) {
		i.figure = w.GetText().String()
	}
	return i
}

// pickSharedImpacts collects every statement that each evaluated entry raises.
func pickSharedImpacts(picks []*pick) []pickImpact {
	var shared []pickImpact
	first := true
	for _, p := range picks {
		if p == nil {
			continue
		}
		raised := make([]pickImpact, 0, len(p.impacts))
		for _, w := range p.impacts {
			raised = append(raised, pickImpactOf(w))
		}
		if first {
			shared, first = raised, false
			continue
		}
		shared = slices.DeleteFunc(shared, func(i pickImpact) bool { return !slices.Contains(raised, i) })
	}
	return shared
}
