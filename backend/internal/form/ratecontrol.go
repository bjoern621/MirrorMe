package form

import (
	"slices"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/capabilities"
)

// modeReason states why the selected codec cannot be driven in a rate-control mode.
// The capability table keys its gaps by publish engine where only one builder reaches the mode:
// libvpx codes lossless VP9 and the vp9enc element has no such property.
func (av availability) modeReason(mode string) *mirrormev1.Text {
	if !av.knownCodec || av.engine == "" {
		return nil
	}
	if reasons := av.verdicts.ValueReasons(KeyMode, mode); len(reasons) > 0 {
		return reasons[0]
	}
	return nil
}

// tuneReason states why this engine's wrapper cannot aim at a step the codec's ladder carries.
//
// The ladder is the encoder's and the knob is the wrapper's, so the two can differ:
// libaom takes a tune on ffmpeg where av1enc has no such property,
// and oneVPL's scenario reaches Intel's runtime through ffmpeg alone.
// The step stays offered and greys with the engine named, as a chroma one engine codes does,
// and the untuned step is on no gap so a greyed control always has somewhere to walk to.
func (av availability) tuneReason(value string) *mirrormev1.Text {
	if !av.knownCodec || av.engine == "" {
		return nil
	}
	if reasons := av.verdicts.ValueReasons(KeyTune, value); len(reasons) > 0 {
		return reasons[0]
	}
	return nil
}

// knob weighs the three facts that decide a rate-control control: the mode's concept uses it,
// the codec's encoder has it, and the capture backend's engine forwards the value.
// uses carries the first two, already weighed by the caller, reason being their statement.
// The engine's own rule is read here.
//
// An engine that forwards a knob the mode marks unused leaves a note rather than a greying,
// so the field states what the value does there instead
// of feeding the encoder a number the form never showed.
// An engine that drops the knob in every mode outranks the mode's own reason:
// no rate control brings the control back, so naming the mode would send the user hunting
// for a switch that changes nothing.
func (av availability) knob(key string, uses bool, reason *mirrormev1.Text) state {
	rule, ruled := av.engineRule(key)
	switch {
	case ruled && rule.forwards:
		return availabilityNoted(say(rule.reason))
	case ruled && len(rule.modes) == 0:
		return availabilityDisabled(say(rule.reason))
	case !uses:
		return availabilityDisabled(reason)
	case ruled:
		return availabilityDisabled(say(rule.reason))
	}
	return availabilityLive()
}

// engineRule finds the rule governing a knob for this engine, codec and mode,
// false where the builder treats the knob exactly as the mode table says.
// Earlier rows win, so a mode-specific reason precedes a codec-wide one.
func (av availability) engineRule(key string) (availabilityEngineRule, bool) {
	for _, r := range availabilityEngineRules {
		if r.knob != key || (r.engine != "" && r.engine != av.engine) {
			continue
		}
		if len(r.modes) > 0 && !slices.Contains(r.modes, av.s.Publish.Mode) {
			continue
		}
		// A rule naming neither codecs nor families covers every codec the engine builds.
		// One naming either covers the union.
		if len(r.codecs) == 0 && len(r.families) == 0 {
			return r, true
		}
		if slices.Contains(r.codecs, av.s.Publish.Codec()) {
			return r, true
		}
		if av.knownCodec && slices.Contains(r.families, av.codec.Family) {
			return r, true
		}
	}
	return availabilityEngineRule{}, false
}

// encoderBounds is whether this encoder bounds the selected mode's rate at all.
// Only constant quality asks: a bitrate mode is bounded by the target it names,
// where a quality target is free of the rate unless the element holds it inside a VBV.
func (av availability) encoderBounds() bool {
	if av.s.Publish.Mode != capabilities.ModeCrf {
		return true
	}
	return capabilities.QualityCeiling(av.s.Publish.Codec(), av.engine)
}

// ceilingStated is whether the settings name a ceiling for a buffer to hold.
// Zero is a value here, an unbounded quality target, so the window that would size it has nothing
// to do (internal/form, fieldMaxrateBounds).
func (av availability) ceilingStated() bool {
	return av.s.Publish.Mode != capabilities.ModeCrf || av.s.Publish.MaxrateM > 0
}

// ceilingNote is what the ceiling carries while it is editable:
// that this encoder cannot code constant quality without one,
// or what the VAAPI elements derive theirs from.
// The two cannot both apply, the va rows taking no ceiling in constant quality at all.
func (av availability) ceilingNote() *mirrormev1.Text {
	if capabilities.QualityCeilingRequired(av.s.Publish.Codec(), av.engine) &&
		av.s.Publish.Mode == capabilities.ModeCrf {
		return say(qualityCeilingRequired, argCodec(av.s.Publish.Codec()))
	}
	return av.vaapiCeilingNote()
}

// ceilingReason names which of the two facts withholds the ceiling: the mode bounding nothing,
// or this encoder having no form of a bounded quality target.
func (av availability) ceilingReason() *mirrormev1.Text {
	if !av.mode().usesMaxrate {
		return say(maxrateOnlyInConstrained)
	}
	return say(noCeilingInConstantQuality, argCodec(av.s.Publish.Codec()))
}

// windowReason names which of the three facts withholds the buffer, the ceiling's own two first:
// a window sizes a ceiling, so it has nothing to say where there is none to hold.
func (av availability) windowReason() *mirrormev1.Text {
	switch {
	case !av.mode().usesVbv:
		return say(vbvOnlyInBoundedModes)
	case !av.encoderBounds():
		return say(noCeilingInConstantQuality, argCodec(av.s.Publish.Codec()))
	default:
		return say(vbvNeedsACeiling)
	}
}

// vaapiCeilingNote states the bound the GStreamer VAAPI elements place on a VBR burst ceiling,
// nil where the settings sit inside it.
// Those elements express the target as a percentage of the ceiling and take 50% at the lowest,
// so a ceiling above twice the target has no form there and the GStreamer builder refuses it.
// A note rather than a greying, since the knob is forwarded:
// the field stays live and carries the bound.
func (av availability) vaapiCeilingNote() *mirrormev1.Text {
	if av.engine != capabilities.EngineGst || av.s.Publish.Mode != capabilities.ModeVbr {
		return nil
	}
	if !av.knownCodec || av.codec.Family != capabilities.FamilyVaapi {
		return nil
	}
	if av.s.Publish.BitrateM <= 0 || av.s.Publish.MaxrateM <= av.s.Publish.BitrateM*2 {
		return nil
	}
	return say(vaapiCeilingBound, argMaxrateMbps(av.s.Publish.BitrateM*2), argBitrateMbps(float64(av.s.Publish.BitrateM)))
}
