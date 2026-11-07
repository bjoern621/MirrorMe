package form

import (
	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/rules"
	"bjoernblessin.de/mirrorme/internal/transport"
)

// formatReason states why this combination cannot publish a bitstream format.
//
// The format's own two facts, in the order the user can act on them.
// Whether anything here produces it comes first, since "no AV1 encoder on this machine"
// is what stops a search for the right transport, and the publish leg's carriage second,
// that being the one a different transport fixes.
//
// Nothing about one encoder is stated here.
// Several encoders reach a format and they are out for reasons of their own,
// so a format greys under one statement covering all of them and the control below says
// which encoder is missing what.
//
// A leg this engine cannot serialize at all states nothing about any format:
// that refusal belongs to the transport control,
// and greying every format under it would name the wrong one.
func (av availability) formatReason(format string) *mirrormev1.Text {
	if av.engine == "" {
		return nil
	}
	// An engine whose own tooling is missing runs nothing, the encoders no probe is spent on included,
	// so one statement covers every format.
	if _, ok := av.deps.Encoders.Unprobed[av.engine]; ok {
		return say(engineToolingMissing, argEngine(av.engine))
	}
	if !av.formatRuns(format) {
		return say(noEncoderForFormat, argFormat(format), argEngine(av.engine))
	}
	if !transport.CanPublish(av.s.Publish.Transport, av.engine) {
		return nil
	}
	if transport.CanPublishFormat(av.s.Publish.Transport, av.engine, format) {
		return nil
	}
	return av.carryBlockReason(format)
}

// formatRuns reports whether some encoder here produces the format on the selected engine.
// Every row of it is asked, since the control offers the format and not one encoder's way to it.
func (av availability) formatRuns(format string) bool {
	for _, c := range capabilities.Codecs {
		if c.Format == format && av.rowReason(c) == nil {
			return true
		}
	}
	return false
}

// carryBlockReason states that the publish leg has no mapping for a bitstream,
// naming the engine that lacks it and where the combination would have worked.
//
// The engine belongs in the statement because the same protocol carries a format on one engine
// and not the other: ffmpeg's WHIP muxer publishes H.264
// alone where the GStreamer one payloads VP8 and VP9 with it.
// A reason naming the protocol alone would send a user hunting for another transport
// where another capture backend is the fix, so both ways out are named where the tables hold them,
// and each is an argument the surface may or may not receive.
func (av availability) carryBlockReason(format string) *mirrormev1.Text {
	other := capabilities.EngineGst
	if av.engine == capabilities.EngineGst {
		other = capabilities.EngineFfmpeg
	}
	otherEngine := ""
	if transport.CanPublishFormat(av.s.Publish.Transport, other, format) {
		otherEngine = other
	}
	return say(transportCarriesNoFormat,
		argTransport(av.s.Publish.Transport),
		argFormat(format),
		argEngine(av.engine),
		argTransports(transport.PublishNamesFor(av.engine, format)),
		argOtherEngine(otherEngine))
}

// encoderReason states why this combination cannot produce the selected format with an encoder.
//
// The pair is what greys: an encoder that codes the format elsewhere and not here says
// so through its row, and one that codes the format nowhere says that instead,
// naming the formats it does produce so the way out is on the statement.
//
// What the publish leg carries is absent, being a fact about the format alone (formatReason).
// Greying every encoder under it would send a user
// through the whole list for a refusal none of them can lift.
func (av availability) encoderReason(encoder string) *mirrormev1.Text {
	if av.engine == "" {
		return nil
	}
	if _, ok := av.deps.Encoders.Unprobed[av.engine]; ok {
		return say(engineToolingMissing, argEngine(av.engine))
	}
	c, ok := capabilities.Row(av.s.Publish.Format, encoder)
	if !ok {
		return say(encoderCodesNoFormat,
			argEncoder(encoder),
			argFormat(av.s.Publish.Format),
			argFormats(capabilities.FormatsOn(encoder)))
	}
	return av.rowReason(c)
}

// rowReason states why this machine cannot run one row of the capability table on the selected
// engine, nil where it runs.
//
// Three facts withhold one, weighed by what the user can act on.
// A probe that could not run the encoder outranks every table fact,
// since "no NVIDIA encoder on this machine" is the message that stops a search elsewhere.
// Below it sit the engine's own gap and the roadmap.
func (av availability) rowReason(c capabilities.Codec) *mirrormev1.Text {
	if reason := av.probeReason(c); reason != nil {
		return reason
	}
	// The rule binds on the engine alone and names the codec it takes,
	// so one evaluation answers for every entry
	// of both dropdowns rather than for the selected row alone.
	if reasons := av.verdicts.ValueReasons(rules.AxisCodec, c.Name); len(reasons) > 0 {
		return reasons[0]
	}
	if !c.Implemented {
		return say(codecNotImplemented)
	}
	return nil
}

// probeReason states why this machine's probe could not run a codec on the selected engine,
// nil where it ran or was never asked.
//
// A failed probe means different things per engine and family,
// and naming the wrong one sends the user after the wrong fix.
// On the ffmpeg engine a hardware codec needs a card the machine may not have and a software
// one needs its library compiled into the build.
// On the GStreamer engine the element is missing from the registry instead,
// its plugin not installed or, for the hardware families, finding no device to register it for.
// Which of the two a verdict is follows the family and not the engine.
//
// An engine with no verdicts at all is an unprobed machine
// rather than one with nothing usable, so it greys nothing.
func (av availability) probeReason(c capabilities.Codec) *mirrormev1.Text {
	probed, ok := av.deps.Encoders.Usable[av.engine]
	if !ok {
		return nil
	}
	if ran, tested := probed[c.Name]; !tested || ran {
		return nil
	}
	family, ok := availabilityFamilies[c.Family]
	if !ok {
		// Which half is missing is the family's fact, and this row names a family no table here carries.
		// The probe's verdict still holds, so the codec greys under what is known rather
		// than under a guessed half.
		return say(probeFailed, argEngine(av.engine), argCodec(c.Name))
	}
	if family.needsDevice {
		return say(probeNoDevice, argEngine(av.engine), argFamily(c.Family))
	}
	return say(probeNoBuild, argEngine(av.engine), argCodec(c.Name))
}
