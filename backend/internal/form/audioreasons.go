package form

import (
	"slices"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/platform"
	"bjoernblessin.de/screenshare/internal/publish"
	"bjoernblessin.de/screenshare/internal/transport"
)

// audioReason states why this capture backend records no such source.
//
// Two facts withhold one and publish.AudioAvailable holds both:
// the platform's session serves it with nothing,
// and the engine the backend runs has no element that opens it.
// It is read through rather than restated, a second copy here being what would let the form grey
// a source the publish opens, or offer one the publish then refuses (docs/domain-model.md).
//
// A capture backend no registry carries reaches neither half, so the platform table answers alone:
// the engine is unknown, and greying on a guess would name a limit the machine may not have.
func (av availability) audioReason(source string) *screensharev1.Text {
	if av.engine == "" {
		_, reason := platform.AudioSourceAvailable(source, av.deps.Platform)
		return reason
	}
	_, reason := publish.AudioAvailable(av.s.Publish.Capture, source)
	return reason
}

// audioCodecReason states why the stream cannot be published with an audio codec.
//
// Two independent facts withhold one: the capture backend's publish engine has no encoder element
// for the codec, and the publish transport carries no track in that codec's bitstream format.
// The first is the audio table's own gap and the second the carriage table's,
// and the statement names whichever applies, the fix differing:
// another capture backend against another codec on the same one.
func (av availability) audioCodecReason(name string) *screensharev1.Text {
	if av.engine == "" {
		return nil
	}
	a, ok := capabilities.GetAudio(name)
	if !ok {
		return nil
	}
	if _, codes := a.EncoderOn(av.engine); !codes {
		// A codec an engine cannot code normally states why as a rule.
		// Where none does, the fact is shown anyway, with no cause invented for it.
		if reasons := av.verdicts.ValueReasons(KeyAudioCodec, a.Name); len(reasons) > 0 {
			return reasons[0]
		}
		return say(engineHasNoAudioEncoder, argEngine(av.engine), argAudioCodec(a.Name))
	}
	carriage, ok := transport.PublishCarriage(av.s.Publish.Transport, av.engine)
	if !ok || slices.Contains(carriage.Audio, a.Format) {
		return nil
	}
	// The codecs that would have worked: what this engine codes and this leg carries.
	// Empty on a leg that carries none,
	// and the statement then reads as the shorter fact rather than trailing off.
	return say(legCarriesNoAudioCodec,
		argTransport(av.s.Publish.Transport),
		argAudioCodec(a.Name),
		argEngine(av.engine),
		argAudioCodecs(capabilities.AudioNamesFor(av.engine, carriage.Audio)))
}

// audioDeviceReason states why one entry cannot record from a device it is offered.
//
// It refuses nothing: what is inside a kind is what the machine answered,
// and a selection it stopped answering for is kept with a note rather than greyed,
// an application that is not running being one that may be running
// when the stream starts (audio.go).
// The row exists because every control that offers entries is answered for here,
// which makes a later refusal a line in this file rather than a new mechanism.
func (av availability) audioDeviceReason(string) *screensharev1.Text {
	return nil
}
