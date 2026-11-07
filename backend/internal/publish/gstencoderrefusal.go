package publish

import (
	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/settings"
)

// EncoderRefusal is why this engine's encoder mapping cannot express these settings, and nil where
// it can.
//
// What one element imposes beyond the capability table: a rate past its property's range, a rate
// buffer past the field it is read into, a VBR ceiling more than twice its target.
// The capture backend fixes the engine, and nothing here reads the machine, the display or
// the relay, so a caller weighing a configuration ahead of building one asks this rather than
// rendering a command, whose refusal can be about a transport the caller was not choosing
// (form.presetResolve).
//
// The ffmpeg engine states no such pair rule: what its encoders refuse is what the capability table
// already carries.
func EncoderRefusal(s settings.Settings) error {
	engine, err := EngineFor(s.Publish.Capture)
	if err != nil {
		return err
	}
	if engine != capabilities.EngineGst {
		return nil
	}
	c, known := capabilities.Get(s.Publish.Codec())
	if !known {
		return text.Fail(screensharev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN, text.Codec(s.Publish.Codec()))
	}
	gst, mapped := gstCodecs[s.Publish.Codec()]
	if !mapped {
		return text.Fail(screensharev1.TextCode_TEXT_CODE_ENGINE_HAS_NO_FORM, text.Engine(EngineGst), text.Codec(s.Publish.Codec()))
	}
	return gstEncoderRefusal(s, c, gst)
}

// gstEncoderRefusal is the limit half of gstEncoder, answering about the settings rather than about
// the pipeline being built.
//
// Both bounds apply where both exist: the family's comes from what its elements share, the row's
// from one element's own property range, so neither stands in for the other.
func gstEncoderRefusal(s settings.Settings, c capabilities.Codec, gst gstCodec) error {
	if limits, ok := gstFamilyLimits[c.Family]; ok {
		if err := limits(s); err != nil {
			return err
		}
	}
	if gst.limits != nil {
		return gst.limits(s)
	}
	return nil
}

// gstFamilyLimits are the settings bounds a family's elements impose beyond capabilities.Codecs,
// keyed as that table names the family.
// A family absent here takes whatever the capability table already approved.
//
// Per family because the bound comes from the properties a plugin's elements share,
// and here rather than in the capability table because it binds two settings against each other
// (a ceiling against its target) instead of taking one value out of a set.
var gstFamilyLimits = map[string]func(settings.Settings) error{
	capabilities.FamilyVaapi: vaLimits,
}

// propertyBound refuses a value past where an element property stops, both in the property's unit.
func propertyBound(family, property string, limit, asked int) error {
	assert.Assert(asked > limit, "a bound refusal names a value past the bound", property, limit, asked)
	return text.Fail(screensharev1.TextCode_TEXT_CODE_ENCODER_PROPERTY_BOUND,
		text.Family(family), text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_PROPERTY, property),
		text.CountLimit(limit), text.Count(asked))
}
