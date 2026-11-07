package form

import (
	"slices"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/gpupath"
	"bjoernblessin.de/screenshare/internal/wire"
)

// chromaReason states why the selected codec cannot be handed a pixel format.
//
// Two facts block one and the capability table carries both: a format the codec's encoder codes
// on no engine is absent from its Chromas, and one only the other engine's encoder takes carries
// a gap naming that engine.
// The gap's code passes through, since it already says which library or element is the limit.
func (av availability) chromaReason(chroma string) *screensharev1.Text {
	if !av.knownCodec {
		return nil
	}
	if !slices.Contains(av.codec.Chromas, chroma) {
		return availabilityChromaBlock(chroma, av.codec.Name)
	}
	if av.engine == "" {
		return nil
	}
	if reasons := av.verdicts.ValueReasons(KeyChroma, chroma); len(reasons) > 0 {
		return reasons[0]
	}
	return nil
}

// colorRangeReason states why the stream would not carry a colour range.
// The capability table declares it per codec and engine: an encoder signalling no colour
// description, and a format with no colour range field, both leave a full-range publish watched
// as limited whatever the form said.
func (av availability) colorRangeReason(value string) *screensharev1.Text {
	if !av.knownCodec || av.engine == "" {
		return nil
	}
	if reasons := av.verdicts.ValueReasons(KeyColorRange, value); len(reasons) > 0 {
		return reasons[0]
	}
	return nil
}

// availabilityChromaBlock states that a codec cannot encode a pixel format on either engine.
// Planar RGB carries its own code: RGB reaches an encoder only through HEVC's Range Extensions
// or VP9's identity matrix, a coding-tool fact rather than a subsampling limit,
// and a reader told only "cannot encode gbrp" would go looking for a setting that would fix it.
func availabilityChromaBlock(chroma, codec string) *screensharev1.Text {
	if chroma == "gbrp" {
		return say(codecCodesNoRGB, argCodec(codec))
	}
	return say(codecCannotEncodeChroma, argCodec(codec), argChroma(chroma))
}

// encoderColourReason states why the colour fields do not reach this stream, nil where they do.
// On the path that converts nothing the encoder reads the captured surface on its own terms
// and signals what it chose, so both fields grey with the row's cost
// and Repair moves each onto what the encoder signals: a greyed field showing something
// other than what the run produces is the disagreement the form and the publish exist to prevent.
func (av availability) encoderColourReason() *screensharev1.Text {
	if !av.onDevicePath || av.s.Publish.CaptureMemory != gpupath.MemoryGpuEncoderColor {
		return nil
	}
	if !av.path.Colour.TradesColour() {
		return nil
	}
	return wire.GpuPathCost(av.path)
}

// decodeNote states what decoding this stream costs a viewer,
// nil for a codec the table does not carry.
//
// A note, because every format has a software decoder,
// so the choice is between a viewer's GPU and a viewer's cores.
// Where some hardware decodes the pair the statement names those decode families,
// which ones they are being the whole point of the choice.
// Where none does, it names the software element and one family whose limit stands for the rest,
// since the others are out for reasons of their own and listing
// them states again what the first already shows.
func (av availability) decodeNote() *screensharev1.Text {
	if !av.knownCodec {
		return nil
	}
	var families []string
	for _, d := range capabilities.Decoders {
		if d.Format != av.codec.Format || !d.Hardware() || !slices.Contains(d.Chromas, av.s.Publish.Chroma) {
			continue
		}
		if !slices.Contains(families, d.Family) {
			families = append(families, d.Family)
		}
	}
	if len(families) > 0 {
		return say(decodesInHardware, argDecodeFamilies(families))
	}

	software, limited := "", ""
	for _, d := range capabilities.Decoders {
		if d.Format != av.codec.Format {
			continue
		}
		if !d.Hardware() && software == "" {
			software = d.Element
		}
		if d.Hardware() && limited == "" && !slices.Contains(d.Chromas, av.s.Publish.Chroma) {
			limited = d.Family
		}
	}
	return say(decodesOnCPU,
		argFormat(av.codec.Format), argChroma(av.s.Publish.Chroma), argDecoder(software), argDecodeFamily(limited))
}
