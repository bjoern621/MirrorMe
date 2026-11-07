package capabilities

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// Gap is one thing a codec cannot do, with the reason a surface shows in place of the option.
//
// A gap takes one value of one option away: Option names which option, Value which of its values.
// A gap naming no option takes the codec off that engine altogether,
// for a format that engine has no encoder for at all.
//
// Which options exist is the Options list and nothing else,
// so a codec that cannot encode at a colour range is declared like one that cannot encode a chroma.
type Gap struct {
	// Engine is "ffmpeg", "gstreamer", or empty for both.
	// Empty says the format or the library has no such capability,
	// rather than one builder failing to reach it.
	Engine string `json:"engine"`
	// Option is one of Options, and empty exactly when the gap takes the codec off the engine.
	Option string `json:"option"`
	// Value is the option value the engine's encoder will not take, empty exactly when Option is.
	Value string `json:"value"`
	// Reason names which library or element lacks the capability,
	// as a code rather than a sentence about it (api/proto/mirrorme/v1/text.proto).
	// One code per row and no arguments:
	// the three fields above already say which codec, which engine and which value,
	// so a surface rendering the code has everything the sentence names.
	Reason mirrormev1.TextCode `json:"reason"`
}

func (g Gap) covers(engine string) bool {
	return g.Engine == "" || g.Engine == engine
}

// OptionGap is the gap keeping this codec from encoding with value for the named option,
// on the named engine, and false where that value reaches its encoder there.
// The one gap lookup:
// chroma, rate-control mode, colour range and any option Options gains are asked about the same way.
//
// A chroma the codec encodes on neither engine is absent from Chromas instead,
// which Validate rejects on its own.
func (c Codec) OptionGap(engine, option, value string) (Gap, bool) {
	assert.Assert(knownEngine(engine), "a gap lookup names a publish engine", engine)
	assert.Assert(knownOption(option), "a gap lookup names a gappable option", option)

	for _, g := range c.Gaps {
		if g.Option == option && g.Value == value && g.covers(engine) {
			return g, true
		}
	}
	return Gap{}, false
}

// EngineGap is the gap that takes this codec off the named engine altogether,
// and false where that engine has an encoder for it.
// The gap naming no option, no value of any reaching an encoder that is not there.
func (c Codec) EngineGap(engine string) (Gap, bool) {
	assert.Assert(knownEngine(engine), "a gap lookup names a publish engine", engine)

	for _, g := range c.Gaps {
		if g.Option == "" && g.covers(engine) {
			return g, true
		}
	}
	return Gap{}, false
}

// EngineChromas is the pixel formats this codec encodes on the named engine, in table order:
// Chromas minus the formats gapped there.
// An engine-wide gap answers with nothing rather than with the formats the other engine reaches,
// since an engine with no encoder for the codec encodes none of them.
func (c Codec) EngineChromas(engine string) []string {
	assert.Assert(knownEngine(engine), "a chroma list names a publish engine", engine)

	if _, gap := c.EngineGap(engine); gap {
		return nil
	}
	out := make([]string, 0, len(c.Chromas))
	for _, chroma := range c.Chromas {
		if _, gap := c.OptionGap(engine, OptionChroma, chroma); !gap {
			out = append(out, chroma)
		}
	}
	return out
}

// SupportsChroma answers per engine,
// so a format the codec encodes on the other engine alone reports false here,
// matching what Validate accepts on the engine asked about.
func SupportsChroma(name, engine, chroma string) bool {
	assert.Assert(knownEngine(engine), "a chroma question names a publish engine", engine)

	c, ok := Get(name)
	if !ok {
		return false
	}
	if !contains(c.Chromas, chroma) {
		return false
	}
	_, gap := c.OptionGap(engine, OptionChroma, chroma)
	return !gap
}
