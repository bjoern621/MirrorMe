package capabilities

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/rules"
	"bjoernblessin.de/mirrorme/internal/text"
)

// Validate rejects a codec and option-value combination this table forbids,
// so a settings object nothing repaired cannot reach an encoder.
//
// engine is the caller's own publish engine,
// and decides which codecs and option values are available.
// Both engines call this, so neither path accepts what the other rejects,
// and a capability only one of them reaches is refused for the other rather than approximated.
//
// options carries one value per entry in Options, keyed as Options names them.
// The caller takes them out of its own settings, which keeps this package free of dependencies.
// A value left out asserts rather than being skipped:
// an option gained here and not supplied there would reach an encoder with its gaps unread.
//
// device is the driver the encode would run on (gpu.Device), and decides the DriverDefects rows.
// The zero Device is a machine that named no driver, carrying no defect and refusing nothing.
//
// Whether the publish transport carries the resulting bitstream is the transport package's own
// refusal (transport.ValidatePublish), which the same callers make beside this one.
func Validate(engine, codec string, options map[string]string, cq, bitrateM, gop int, device Device) error {
	// The engine and the option set are the caller's own, so both are Entwicklungsfehler and assert.
	// The values being validated are the user's, and every one of them leaves as an error.
	assert.Assert(knownEngine(engine), "validation names a publish engine", engine)
	for _, option := range Options {
		_, ok := options[option]
		assert.Assert(ok, "validation carries a value for every gappable option", option)
	}
	chroma, mode := options[OptionChroma], options[OptionMode]

	c, ok := Get(codec)
	if !ok {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN, text.Codec(codec))
	}
	if !c.Implemented {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_CODEC_NOT_IMPLEMENTED, text.Codec(codec))
	}
	// A mode outside the set matches no gap and would reach the builders,
	// whose rate-control switches end in the cbr branch,
	// so the encode would run at a rate control the settings never asked for.
	if !contains(Modes, mode) {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN,
			text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_MODE, mode))
	}
	// Every refusal below is the rules' own reason, the statement the form greys the value with,
	// so a greyed control and a refused publish read one evaluation.
	v := rules.EvaluateRules(validationFacts(c, engine, options, cq, bitrateM, gop, device), codecRules())

	if !v.ValueEnabled(rules.AxisCodec, c.Name) {
		return refusal(v.ValueReasons(rules.AxisCodec, c.Name))
	}
	if !contains(c.Chromas, chroma) {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_CODEC_CANNOT_ENCODE_CHROMA, text.Codec(c.Name), text.Chroma(chroma))
	}
	// Every option is read the same way and in table order,
	// so a codec refused on two of them names the first,
	// rather than the one an axis-by-axis validator happened to reach first.
	for _, option := range Options {
		axis, ok := optionAxes[option]
		assert.Assert(ok, "a gappable option names the axis a rule matches it on", option)
		if !v.ValueEnabled(axis, options[option]) {
			return refusal(v.ValueReasons(axis, options[option]), otherReach(c.Name, engine, option, options[option])...)
		}
	}
	// A negative target is refused apart from the rules,
	// no rule stating a floor and a scale starting at zero being one no band below it can express.
	if cq < 0 {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_CQ_ABOVE_CODEC_SCALE, text.Codec(c.Name), text.Engine(engine),
			text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_MODE, mode),
			text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_CQ_MAX, int64(c.CqMaxOn(engine))))
	}
	// The quantizer target reaches the encoder in crf mode alone,
	// and each encoder's knob has its own scale: 60 is a valid libvpx CQ and an error on x264.
	// Which modes read the knob is the rule's own gate, so this asks about the value and nothing else.
	for _, number := range []struct {
		axis  string
		value int
	}{{rules.AxisCq, cq}, {rules.AxisBitrateM, bitrateM}, {rules.AxisGop, gop}} {
		if !v.NumberAllowed(number.axis, number.value) {
			return refusal(v.NumberReasons(number.axis, number.value))
		}
	}
	return nil
}

// refusal states a refused value as the first reason the rules give for it, with extra appended.
// A driver defect leads: the encoder implements the value,
// so any other reason would name the wrong culprit, and the other engine drives the same driver.
func refusal(reasons []*mirrormev1.Text, extra ...*mirrormev1.TextArg) error {
	assert.Assert(len(reasons) > 0, "a refused value carries the reason the rules refused it with")

	for _, reason := range reasons {
		if reason.GetCode() == mirrormev1.TextCode_TEXT_CODE_DRIVER_DEFECT_WITHHOLDS_OPTION {
			return text.Fail(reason.GetCode(), reason.GetArgs()...)
		}
	}
	return text.Fail(reasons[0].GetCode(), append(reasons[0].GetArgs(), extra...)...)
}

// otherReach names the engine that does reach one value of one option, none where no other does.
// A refusal naming the attempted engine alone states half the fact,
// and the half it withholds is the one a settings file that skipped the form can act on.
func otherReach(codec, engine, option, value string) []*mirrormev1.TextArg {
	for _, e := range Engines {
		if e != engine && Reaches(codec, e, option, value) {
			return []*mirrormev1.TextArg{text.OtherEngine(e)}
		}
	}
	return nil
}

// validationFacts is the configuration a validation is about, as the axes read it.
//
// It answers the axes this call was given and leaves the rest empty, which is honest rather than
// lossy: a fact nobody stated matches no rule that names a value, so an empty axis withholds nothing.
// What it must never do is guess,
// which would refuse a publish over a combination the caller never described.
//
// The axes it fills are the ones the codec table's own rules match on,
// rather than every axis the vocabulary declares:
// the cursor mode is a capture fact and no rule registered here names it.
// Rule.binds keeps that honest,
// a rule naming an axis the facts do not carry asserting rather than binding nothing.
func validationFacts(c Codec, engine string, options map[string]string, cq, bitrateM, gop int, device Device) rules.Facts {
	return rules.Facts{
		rules.AxisCodec:            rules.TextValue(c.Name),
		rules.AxisFamily:           rules.TextValue(c.Family),
		rules.AxisFormat:           rules.TextValue(c.Format),
		rules.AxisEncoder:          rules.TextValue(c.Encoder()),
		rules.AxisEngine:           rules.TextValue(engine),
		rules.AxisChroma:           rules.TextValue(options[OptionChroma]),
		rules.AxisMode:             rules.TextValue(options[OptionMode]),
		rules.AxisColorRange:       rules.TextValue(options[OptionColorRange]),
		rules.AxisTune:             rules.TextValue(options[OptionTune]),
		rules.AxisCq:               rules.NumberValue(cq),
		rules.AxisBitrateM:         rules.NumberValue(bitrateM),
		rules.AxisGop:              rules.NumberValue(gop),
		rules.AxisGpuDriver:        rules.TextValue(device.Driver),
		rules.AxisGpuModel:         rules.TextValue(device.Model),
		rules.AxisGpuDriverVersion: rules.NumberValue(device.Version),
		rules.AxisCapture:          rules.TextValue(""),
		rules.AxisMemory:           rules.TextValue(""),
		rules.AxisTransport:        rules.TextValue(""),
		rules.AxisAudioCodec:       rules.TextValue(""),
		rules.AxisOS:               rules.TextValue(""),
		rules.AxisDisplay:          rules.TextValue(""),
	}
}
