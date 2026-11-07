package publish

import (
	"slices"
	"strconv"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
	"bjoernblessin.de/mirrorme/internal/text"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// gstRates carries the rate-control figures as the element property values they are spelled in:
// kbit/s for the x26x elements, the constant-quality target, and the GOP in frames.
type gstRates struct {
	kbps, maxkbps, cq, gop string
}

func gstRatesFor(s settings.Settings, gop int) gstRates {
	return gstRates{
		kbps:    strconv.Itoa(s.Publish.BitrateM * 1000),
		maxkbps: strconv.Itoa(s.Publish.MaxrateM * 1000),
		cq:      strconv.Itoa(s.Publish.Cq),
		gop:     strconv.Itoa(gop),
	}
}

// The elements linking an encoder to the sink.
//
// config-interval=-1 inserts SPS/PPS (H.264) or VPS/SPS/PPS (H.265) ahead of every IDR.
// Sent once at stream start they never reach a viewer joining the relay mid-stream,
// whose decoder then cannot start.
// The property is this engine's alone: the ffmpeg engine repeats the parameter sets by default.
// VP9 and AV1 carry no out-of-band parameter sets, so their parsers need none and rtspclientsink
// payloads the parsed video/x-vp9 and video/x-av1.
//
// VP8 has no parser element.
// vp8enc leaves profile a list on its output caps and then rejects its own unfixed caps
// with "Invalid vpx profile", so a capsfilter pins profile 0,
// the only profile an 8-bit 4:2:0 VP8 bitstream carries.
var (
	h264Parser = []string{"h264parse", "config-interval=-1"}
	h265Parser = []string{"h265parse", "config-interval=-1"}
	vp9Parser  = []string{"vp9parse"}
	av1Parser  = []string{"av1parse"}
	vp8Caps    = []string{"video/x-vp8,profile=(string)0"}
)

// gstCodec is one codec's GStreamer half: the encoding element with the rate-control mode mapped
// onto its own knobs, and the elements linking it to the sink.
type gstCodec struct {
	// encode builds the element and its properties.
	// l carries the two ladder steps the encode spends, resolved once against the codec's own row,
	// so a mapping states which property carries a step.
	encode func(s settings.Settings, r gstRates, l capabilities.Steps) []string
	link   []string
	// limits refuses a settings combination the element cannot express, beyond what the capability
	// table and gstFamilyLimits already cover.
	// Per codec where the bound is one element's property range: two elements of one plugin can
	// declare the same property differently.
	limits func(settings.Settings) error
}

// gstCodecs is this engine's half of the codec facts capabilities.Codecs declares once,
// and the counterpart to encoderArgs in the ffmpeg builder.
// A rate-control mode or pixel format an element has no form of is a Gap on that codec's row,
// carrying Engine "gstreamer" where the ffmpeg encoder reaches it,
// and is rejected before a pipeline is built, so no mapping approximates one.
//
// The VAAPI rows target the stateless "va" plugin from gst-plugins-bad,
// not gstreamer-vaapi's older vaapih264enc and vaapih265enc.
// The va plugin is the maintained one, it holds the only AV1 encoder,
// and it negotiates the VAMemory caps the portal backend's GPU path runs on:
// vapostproc imports the compositor's dmabuf and converts into surfaces these elements read
// (gpupath.Paths).
// Its elements register per detected device, so a name below exists only where the driver exposes
// that encode entrypoint, the condition the codec's probe tests (encoders.Detect).
//
// The QSV rows target the "qsv" plugin, which drives Intel's oneVPL runtime over VA on Linux and
// D3D11 on Windows, and registers per detected device as the va plugin does.
//
// A family with no entry is rejected before a pipeline is built, either as Implemented:false
// in capabilities.Codecs (v4l2, rkmpp) or as a row gapping it off this engine (amf, vulkan).
var gstCodecs = map[string]gstCodec{
	"libx264":    {encode: x264Encoder, link: h264Parser},
	"libx265":    {encode: x265Encoder, link: h265Parser},
	"h264_nvenc": {encode: nvencEncoder("nvh264enc"), link: h264Parser},
	"hevc_nvenc": {encode: nvencEncoder("nvh265enc"), link: h265Parser},
	"av1_nvenc":  {encode: nvencEncoder("nvav1enc"), link: av1Parser},
	"libvpx-vp9": {encode: vpxEncoder("vp9enc", "row-mt=true"), link: vp9Parser, limits: vpxRateLimits},
	"libvpx":     {encode: vpxEncoder("vp8enc"), link: vp8Caps, limits: vpxRateLimits},
	"libaom-av1": {encode: aomEncoder, link: av1Parser},
	"libsvtav1":  {encode: svtav1Encoder, link: av1Parser},
	"librav1e":   {encode: rav1eEncoder, link: av1Parser},
	"h264_vaapi": {encode: vaEncoder("vah264enc", "qpi", "qpp"), link: h264Parser},
	"hevc_vaapi": {encode: vaEncoder("vah265enc", "qpi", "qpp"), link: h265Parser},
	"av1_vaapi":  {encode: vaEncoder("vaav1enc", "qp"), link: av1Parser},
	"vp9_vaapi":  {encode: vaEncoder("vavp9enc", "qp"), link: vp9Parser},
	// No link element: VP8 has no parser, and vavp8enc fixes its output caps where vp8enc leaves
	// the profile a list, so rtspclientsink payloads its video/x-vp8 directly.
	"vp8_vaapi": {encode: vaEncoder("vavp8enc", "qp")},
	"h264_qsv":  {encode: qsvEncoder("qsvh264enc"), link: h264Parser},
	"hevc_qsv":  {encode: qsvEncoder("qsvh265enc"), link: h265Parser},
	// bitrate is an unsigned 16-bit property on the AV1 and VP9 elements where the H.26x ones take any
	// rate (qsvShortBitrateKbps).
	"av1_qsv": {encode: qsvEncoder("qsvav1enc"), link: av1Parser, limits: qsvShortRateLimits},
	"vp9_qsv": {encode: qsvEncoder("qsvvp9enc"), link: vp9Parser, limits: qsvShortRateLimits},
	// The applemedia elements, which are what a Mac reaches its media block through on this engine.
	"h264_videotoolbox": {encode: vtEncoder("vtenc_h264"), link: h264Parser},
	"hevc_videotoolbox": {encode: vtEncoder("vtenc_h265"), link: h265Parser},
}

// gstLiveDelay is the lookahead an element holds by default, pinned off per element.
//
// A lookahead is frames the encoder keeps before it hands the first one on, and those frames leave
// at whatever rate the transport takes: a leg draining three a second turns fifty held frames
// into seventeen seconds of picture nobody has seen, which is what internal/pipedelay reports
// as the publish transit.
// The effort ladder is paid in compression alone, so a pin stands whatever step it resolved to.
//
// No row where the element holds nothing to begin with: av1enc lags no frames until asked, and
// the hardware families queue on the device rather than in an element.
// x265enc's pins travel in the option-string its mode composes (x265LiveDelay).
// svtav1enc has none: it takes lookahead=0 in its parameters-string and holds the frames anyway.
var gstLiveDelay = map[string][]string{
	// rc-lookahead has no property of its own and travels in the option-string no x264 mode writes,
	// and sliced-threads is what keeps frame threading from holding a frame per thread.
	"x264enc": {"sliced-threads=true", "option-string=rc-lookahead=0"},
	"vp8enc":  {"lag-in-frames=0"},
	"vp9enc":  {"lag-in-frames=0"},
	// rav1e holds frames past this too, so the pin shortens the wait rather than ending it.
	"rav1enc": {"low-latency=true"},
}

// gstEncoder returns the encoder element with its properties, and the elements linking it
// to the sink, for the selected codec reading frames in the resolved memory.
// A rate outside an element's property range is refused rather than moved into it.
func gstEncoder(s settings.Settings, gop int, memory string, device capabilities.Device) (encoder []string, link []string, err error) {
	gst, ok := gstCodecs[s.Publish.Codec()]
	if !ok {
		return nil, nil, text.Fail(mirrormev1.TextCode_TEXT_CODE_ENGINE_HAS_NO_FORM, text.Engine(EngineGst), text.Codec(s.Publish.Codec()))
	}
	c, ok := capabilities.Get(s.Publish.Codec())
	if !ok {
		return nil, nil, text.Fail(mirrormev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN, text.Codec(s.Publish.Codec()))
	}
	if err := gstEncoderRefusal(s, c, gst); err != nil {
		return nil, nil, err
	}
	l, err := c.ResolveSteps(s.Publish.Mode, s.Publish.Effort, s.Publish.Tune, device)
	if err != nil {
		return nil, nil, err
	}
	encoder = gst.encode(s, gstRatesFor(s, gop), l)
	assert.Assert(len(encoder) > 0, "a codec mapping names its element ahead of its properties", s.Publish.Codec())
	// The element name is the first word every mapping writes, how GstEncoderElement reads it back
	// out.
	// A plugin shipping one element per memory kind changes that word and nothing after it: both
	// elements derive from one base class and take the same rate-control properties, the memory
	// deciding only which of them links to the conversion ahead of it.
	if device, named := gstDeviceEncoderElement(c.Family, s.Publish.Codec(), memory); named {
		encoder[0] = device
	}
	// Read off the element the mapping ended up naming, a device element holding no lookahead
	// of its own (gstLiveDelay).
	encoder = append(encoder, gstLiveDelay[encoder[0]]...)
	// The name goes on every encoder whatever the element is: a write to a playing pipeline addresses
	// "enc" over the control socket, and the codec decides which element wears it (gstlive.go).
	encoder = slices.Insert(encoder, 1, "name="+gstEncoderName)
	return encoder, gst.link, nil
}
