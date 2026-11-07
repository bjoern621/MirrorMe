package publish

import (
	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/gpupath"
	"bjoernblessin.de/screenshare/internal/settings"
)

// GstEncoderElement returns the element that encodes codec from frames in system memory,
// and false where this engine has no mapping for it.
//
// The element is what the encoder probe asks the plugin registry about: unlike the ffmpeg encoders,
// each lives in a plugin an install may not carry, and the hardware ones register per detected
// device.
//
// System memory is the path every pair has, so a caller asking whether this machine encodes a codec
// at all needs no memory to ask with.
// A caller holding a resolved memory takes GstEncoderElementOn: where a plugin ships an element per
// memory kind the two names differ, and a registry query for the wrong one reports the family
// present while the element a run launches is missing.
func GstEncoderElement(codec string) (string, bool) {
	return GstEncoderElementOn(codec, gpupath.MemorySystem)
}

// GstEncoderElementOn returns the element that encodes codec from frames in the resolved memory,
// and false where this engine has no mapping for the codec.
//
// The system-memory name is read back off a built encoder rather than stored beside the mapping,
// so it cannot drift from the element a pipeline runs.
// The device name comes from the family's device row, the lookup gstEncoder builds with.
func GstEncoderElementOn(codec, memory string) (string, bool) {
	gst, ok := gstCodecs[codec]
	if !ok {
		return "", false
	}
	c, ok := capabilities.Get(codec)
	if !ok {
		return "", false
	}
	if device, named := gstDeviceEncoderElement(c.Family, codec, memory); named {
		return device, true
	}
	mode, ok := firstGstMode(codec)
	if !ok {
		return "", false
	}
	s := settings.Defaults()
	s.Publish.Mode = mode
	// The codec's own steps for that mode: the defaults carry another codec's,
	// and a step off this ladder spells a property value no element takes.
	// Harmless for the element name read back below, and a trap for a caller that reads further.
	l, err := c.ResolveSteps(mode, "", "", capabilities.Device{})
	assert.Assert(err == nil, "a codec's own row resolves against its own ladders", codec, err)
	// The defaults' own rates rather than an empty set: a mapping composes properties out of them, and
	// a figure that is not there is one a mapping cannot state a property from.
	return gst.encode(s, gstRatesFor(s, gstGop(s)), l)[0], true
}

// firstGstMode names a rate-control mode this codec's element implements on this engine,
// and false where the table gaps every one of them off.
//
// Reading an element name means building an encoder, and a mapping dispatches on the mode to decide
// which properties to set, so the mode handed to it has to be one the element implements:
// the settings' own default is lossless, which the hardware families have no form of,
// and asking for it would reach the assert.Never every mapping ends in.
func firstGstMode(codec string) (string, bool) {
	for _, mode := range capabilities.Modes {
		if capabilities.Reaches(codec, capabilities.EngineGst, capabilities.OptionMode, mode) {
			return mode, true
		}
	}
	return "", false
}
