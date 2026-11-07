package capabilities

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/rules"
)

// DriverDefect is one value this codec's encoder declares, other drivers run,
// and the driver named here gets wrong badly enough to withhold it.
//
// Separate from Gap, the two answering different questions.
// A gap is what an encoder cannot do and holds wherever this app runs,
// so the engine-scoped lookups read Gaps alone,
// and a builder enumerating what an element implements gets the same answer on every machine.
// A defect is what one driver gets wrong and holds only while that driver sits under the encoder,
// so it reaches a form and a publish through the evaluator,
// the reader that carries the machine's own facts.
//
// A defect is written down rather than probed for where trying it is the damage:
// an encode that hangs the graphics device cannot be looked up the way a missing element is,
// so the driver is identified (internal/gpu) and matched.
type DriverDefect struct {
	// Driver is the implementation carrying the defect, spelled as it names itself: "radeonsi".
	Driver string `json:"driver"`
	// Models narrows the defect to the adapters that carry it,
	// each matched whole against the name the driver reports.
	// Empty covers every adapter the driver drives,
	// which is what a defect in the driver's own code rather than in one generation of silicon takes.
	Models []string `json:"models"`
	// Option and Value are what the defect withholds, named as a Gap names them.
	Option string `json:"option"`
	Value  string `json:"value"`
	// FixedIn is the first driver release without the defect, packed as gpu.Version packs one.
	// Zero withholds the value on every release, for a defect no version is known to have fixed.
	FixedIn int `json:"fixedIn"`
	// Reason names the defect, as Gap.Reason names a gap.
	Reason mirrormev1.TextCode `json:"reason"`
}

// Device is the video driver a capability question is asked about.
//
// The reading is internal/gpu's and the shape is declared here,
// so this package depends on the rule vocabulary and on nothing else in the domain.
// The zero Device is a driver nothing identified, which matches no defect:
// a machine that could not name its driver keeps every option the encoder declares.
type Device struct {
	// Driver is the implementation, spelled as it names itself: "radeonsi".
	Driver string `json:"driver"`
	// Model is the adapter it drives.
	// Example: "AMD Radeon 780M Graphics".
	Model string `json:"model"`
	// Version is the driver's release packed into one comparable figure: 26.1.6 reads 26001006.
	Version int `json:"version"`
}

// WithheldByDriver reports whether the driver on device takes this option value away,
// where the encoder itself implements it.
//
// The question OptionGap does not answer:
// a gap is the encoder's and holds everywhere,
// a defect is one driver's and holds where that driver is installed.
// Routed through the evaluator rather than walking DriverDefects,
// so this and the refusal Validate returns cannot come apart.
func (c Codec) WithheldByDriver(device Device, engine, option, value string) bool {
	assert.Assert(knownEngine(engine), "a driver question names a publish engine", engine)
	axis, ok := optionAxes[option]
	assert.Assert(ok, "a driver question names a gappable option", option)

	v := rules.EvaluateRules(
		validationFacts(c, engine, map[string]string{option: value}, 0, 0, 0, device), codecRules())
	return refusedByDriver(v, axis, value)
}

// refusedByDriver reports whether the installed driver took this value away rather than the encoder.
//
// It reads the reasons the evaluation already attached,
// so what decides the verdict and what words it are one pass:
// a second walk of the DriverDefects rows could refuse under one reading and explain under another.
func refusedByDriver(v rules.Verdicts, axis, value string) bool {
	for _, reason := range v.ValueReasons(axis, value) {
		if reason.GetCode() == mirrormev1.TextCode_TEXT_CODE_DRIVER_DEFECT_WITHHOLDS_OPTION {
			return true
		}
	}
	return false
}

// av1VaapiDriverDefects is what radeonsi gets wrong about this codec.
//
// Constant bitrate hangs the video block.
// The rate control pads each frame up to the target,
// and once that padding passes roughly 300 kbit per frame the VCN ring stops answering,
// the kernel resets it, and the encode dies with it.
// A screen holding still is what reaches the figure:
// it codes to almost nothing, so nearly the whole target is padding.
//
// Both publish engines drive the same driver and hang alike, so the row names no engine.
// The other three rate controls code the same pictures on it without hanging,
// and the H.264 and HEVC encoders hold constant bitrate on it,
// so what carries the defect is this codec's rate control.
//
// Withheld whole rather than bounded, what the driver survives being a figure per frame:
// the target over the frame rate.
// A ceiling on this table states Mbit/s and reads no frame rate,
// so there is no figure it could carry that holds across the rates the form offers.
var av1VaapiDriverDefects = []DriverDefect{{
	Driver: "radeonsi",
	Option: OptionMode,
	Value:  ModeCbr,
	Reason: mirrormev1.TextCode_TEXT_CODE_DRIVER_DEFECT_WITHHOLDS_OPTION,
}}
