package capabilities

// Codec describes one video codec's fixed capabilities.
type Codec struct {
	// Name is the ffmpeg encoder name: "hevc_nvenc".
	Name string `json:"name"`
	// Family is the backend the codecs in it share, one of Families.
	// A surface names the family and the format to tell one row from another,
	// and each builder binds what a backend does once per family rather than once per codec.
	Family string `json:"family"`
	// Library is what this row's encoder is called where its family holds more than one,
	// spelled as the project names itself: "x264", "svt-av1".
	// Empty on a family that is one encoder, whose own name is what names it (Encoder).
	Library string `json:"library"`
	// Format is the coding format independent of the backend: "h264", "hevc", "av1", "vp9", "vp8".
	// Facts that follow the format rather than the backend key off this,
	// coding efficiency and decodability among them.
	Format string `json:"format"`
	// Implemented is true where the argument builders map this codec to a working command
	// (encoderArgs, gstEncoder).
	// A false row is offered greyed, so a codec no builder reaches is visible,
	// and BuildPublishArgs rejects it.
	Implemented bool `json:"implemented"`
	// Chromas is every pixel format this codec may encode,
	// on whichever publish engine reaches the most of them.
	// A format the other engine's encoder will not take stays in the list and is declared as a Gap,
	// so it is offered where it works and greyed with the missing element where it does not.
	Chromas []string `json:"chromas"`
	// CqMax is the top of the constant-quality scale per publish engine,
	// which is what the crf mode's quantizer target counts on.
	// It follows the encoder rather than the format:
	// the H.26x encoders reach 51, libvpx and the software AV1 ones 63,
	// and an encoder whose knob is a raw quantizer index counts to 127 or 255.
	//
	// Keyed by engine, the scale belonging to the property each engine sets and not to the silicon:
	// ffmpeg's QSV encoders state a CQP quantizer on the H.26x scale for every codec but AV1,
	// where the qsv elements pass the format's own index through.
	// An engine with no entry declares no scale, which leaves the quantizer unbounded here.
	CqMax map[string]int `json:"cqMax"`
	// BitrateLimitM is the highest bitrate target the encoder accepts per publish engine, in Mbit/s,
	// for the modes that aim at one.
	// An engine with no entry takes any rate the machine can produce.
	// It bounds the target and not the VBR burst ceiling set above it.
	BitrateLimitM map[string]int `json:"bitrateLimitM"`
	// BufferLimitKb is the largest rate buffer the encoder's own field holds per publish engine,
	// in kilobits.
	// The buffer reaches the encoder as the rate times the window,
	// so this bounds the pair rather than either half,
	// and the window is the half a form narrows, the rate being the figure somebody chose
	// (form.fieldVbvBounds).
	// An engine with no entry holds whatever a 32-bit field does.
	BufferLimitKb map[string]int `json:"bufferLimitKb"`
	// GopLimit is the longest keyframe interval the encoder's own field holds per publish engine,
	// in frames.
	// An engine with no entry takes any interval the control offers.
	// The encoder refuses an interval above it rather than coding a shorter one,
	// so it bounds what the control offers and what a publish is refused for.
	GopLimit map[string]int `json:"gopLimit"`
	// Effort is the speed-against-quality ladder this encoder takes, and where each mode starts on it.
	Effort Ladder `json:"effort"`
	// Tune is what the encoder optimizes for, a different question from how hard it works:
	// a live encode drops the lookahead and the frame reordering a quality one keeps,
	// whatever effort it is spending.
	// An encoder with both takes the tune step beside the effort step rather than instead of it.
	Tune Ladder `json:"tune"`
	// Gaps is what this codec cannot do, per axis and per publish engine.
	// Empty means every chroma above and every rate-control mode reaches the encoder on both engines.
	Gaps []Gap `json:"gaps"`
	// ModeGaps is what this codec cannot do under one rate-control mode.
	ModeGaps []ModeGap `json:"modeGaps"`
	// DriverDefects is what this codec's encoder does reach and one installed driver miscodes.
	DriverDefects []DriverDefect `json:"driverDefects"`
}

// Encoder is which encode backend this row runs on, at the grain somebody picks one at:
// a family wherever that family is one encoder, and the library where several share a family.
//
// The software family is the one that splits.
// Three encoders code AV1 there and two code the H.26x formats,
// so a picker offered the family would name four rows with one entry,
// and a settings file naming it could not say which.
//
// The second axis of the encode selection, format being the first (Row).
func (c Codec) Encoder() string {
	if c.Library != "" {
		return c.Library
	}
	return c.Family
}

// Get is the row under name, and false for a codec no row carries.
func Get(name string) (Codec, bool) {
	for _, c := range Codecs {
		if c.Name == name {
			return c, true
		}
	}
	return Codec{}, false
}

// Row is the encode a format and an encoder name between them, and false for a pair no row carries.
//
// The pair is what the settings hold and this is where it becomes a codec,
// so the two fields a user picks stay independent,
// and neither is a copy of the row they address.
// The grid is sparse: AMD's runtime codes no VP9,
// so that pair names nothing and the form greys it (docs/domain-model.md).
func Row(format, encoder string) (Codec, bool) {
	for _, c := range Codecs {
		if c.Format == format && c.Encoder() == encoder {
			return c, true
		}
	}
	return Codec{}, false
}

// FormatsOn lists the bitstreams one encoder produces, in table order.
// Empty for an encoder no row runs on,
// which is what a statement about a pair the table does not carry names as the way out of it.
func FormatsOn(encoder string) []string {
	var out []string
	for _, c := range Codecs {
		if c.Encoder() == encoder && !contains(out, c.Format) {
			out = append(out, c.Format)
		}
	}
	return out
}

// Encoders lists every encoder the table carries, in table order and once each.
// Order is the codec table's,
// so the picker offers them the way the rows are authored and a row brings its encoder with it.
func Encoders() []string {
	var out []string
	for _, c := range Codecs {
		if !contains(out, c.Encoder()) {
			out = append(out, c.Encoder())
		}
	}
	return out
}

// HasFormat is asked by the transport package before it narrows a watch choice by format,
// so a relay path in a format this app never encodes narrows nothing rather than to nothing.
func HasFormat(format string) bool {
	return contains(Formats(), format)
}

// Formats lists the bitstream formats implemented codecs produce, in table order and once each.
func Formats() []string {
	var out []string
	for _, c := range Codecs {
		if c.Implemented && !contains(out, c.Format) {
			out = append(out, c.Format)
		}
	}
	return out
}
