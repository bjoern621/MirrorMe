package form

import (
	"time"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/text"
)

// The statement vocabulary this package's answers are built from.
//
// Every reason, note and diagnostic is one code and the identifiers it is about,
// never a sentence (api/proto/mirrorme/v1/text.proto).
// The wrappers exist so a rule reads as the fact it states, say(codecCannotEncodeChroma,
// argCodec(...), argChroma(...)) rather than three lines of enum spelling,
// and so an argument is named here once instead of at every site that fills it.
//
// A code constant is named for what is true, not for the control it greys.
// The same fact greys a pixel format on one screen and explains a diagnostic on another,
// so a name taken from either would read as a lie on the other.
//
// No wrapper here asserts a precondition.
// Every one is a single call to text.Of or text.ID, and both already assert what they require:
// a statement names which statement it is,
// and an argument names which substitution it fills (internal/text).
// A second assert beside them is a second answer to one question.

// say is one statement: text.Of under this package's own name,
// so no rule spells the constructor's package.
func say(code mirrormev1.TextCode, args ...*mirrormev1.TextArg) *mirrormev1.Text {
	return text.Of(code, args...)
}

// Local names for the generated codes, so a rule states its fact on one line.
const (
	captureWrongOS        = mirrormev1.TextCode_TEXT_CODE_CAPTURE_WRONG_OS
	captureTakesNoMonitor = mirrormev1.TextCode_TEXT_CODE_CAPTURE_TAKES_NO_MONITOR
	monitorNotEnumerated  = mirrormev1.TextCode_TEXT_CODE_MONITOR_NOT_ENUMERATED
	// What the capture backend cannot be pointed at,
	// and what a session with no window list cannot offer.
	captureTakesNoShareKind = mirrormev1.TextCode_TEXT_CODE_CAPTURE_TAKES_NO_SHARE_KIND
	captureAsksWhatToShare  = mirrormev1.TextCode_TEXT_CODE_CAPTURE_ASKS_WHAT_TO_SHARE
	noWindowPicked          = mirrormev1.TextCode_TEXT_CODE_NO_WINDOW_PICKED
	windowGone              = mirrormev1.TextCode_TEXT_CODE_WINDOW_GONE
	noRegionPicked          = mirrormev1.TextCode_TEXT_CODE_NO_REGION_PICKED
	regionOffScreen         = mirrormev1.TextCode_TEXT_CODE_REGION_OFF_SCREEN
	scaledFromSource        = mirrormev1.TextCode_TEXT_CODE_SCALED_FROM_SOURCE
	detailStep              = mirrormev1.TextCode_TEXT_CODE_DETAIL_STEP
	effortMost              = mirrormev1.TextCode_TEXT_CODE_EFFORT_MOST
	effortLeast             = mirrormev1.TextCode_TEXT_CODE_EFFORT_LEAST
	engineToolingMissing    = mirrormev1.TextCode_TEXT_CODE_ENGINE_TOOLING_MISSING
	engineHasNoPublishSink  = mirrormev1.TextCode_TEXT_CODE_ENGINE_HAS_NO_PUBLISH_SINK
	// The app builds this leg and the machine cannot run what it builds,
	// which is the encoder probe's question asked about a sink.
	publishSinkElementMissing = mirrormev1.TextCode_TEXT_CODE_PUBLISH_SINK_ELEMENT_MISSING
	engineNotProbed           = mirrormev1.TextCode_TEXT_CODE_ENGINE_NOT_PROBED
	probeNoDevice             = mirrormev1.TextCode_TEXT_CODE_PROBE_NO_DEVICE
	probeNoBuild              = mirrormev1.TextCode_TEXT_CODE_PROBE_NO_BUILD
	probeFailed               = mirrormev1.TextCode_TEXT_CODE_PROBE_FAILED
	codecNotImplemented       = mirrormev1.TextCode_TEXT_CODE_CODEC_NOT_IMPLEMENTED
	noEncoderForFormat        = mirrormev1.TextCode_TEXT_CODE_NO_ENCODER_FOR_FORMAT
	encoderCodesNoFormat      = mirrormev1.TextCode_TEXT_CODE_ENCODER_CODES_NO_FORMAT

	transportCarriesNoFormat    = mirrormev1.TextCode_TEXT_CODE_TRANSPORT_CARRIES_NO_FORMAT
	legCarriesNoAudioCodec      = mirrormev1.TextCode_TEXT_CODE_LEG_CARRIES_NO_AUDIO_CODEC
	engineHasNoAudioEncoder     = mirrormev1.TextCode_TEXT_CODE_ENGINE_HAS_NO_AUDIO_ENCODER
	renderChainElementMissing   = mirrormev1.TextCode_TEXT_CODE_RENDER_CHAIN_ELEMENT_MISSING
	codecCodesNoRGB             = mirrormev1.TextCode_TEXT_CODE_CODEC_CODES_NO_RGB
	codecCannotEncodeChroma     = mirrormev1.TextCode_TEXT_CODE_CODEC_CANNOT_ENCODE_CHROMA
	rgbIsFullRange              = mirrormev1.TextCode_TEXT_CODE_RGB_IS_FULL_RANGE
	decodesInHardware           = mirrormev1.TextCode_TEXT_CODE_DECODES_IN_HARDWARE
	decodesOnCPU                = mirrormev1.TextCode_TEXT_CODE_DECODES_ON_CPU
	decodesInHardwarePartly     = mirrormev1.TextCode_TEXT_CODE_DECODES_IN_HARDWARE_PARTLY
	watchDelay                  = mirrormev1.TextCode_TEXT_CODE_WATCH_DELAY
	decodesInHardwareEverywhere = mirrormev1.TextCode_TEXT_CODE_DECODES_IN_HARDWARE_EVERYWHERE
	encodesOnGPU                = mirrormev1.TextCode_TEXT_CODE_ENCODES_ON_GPU
	fpsMatchesRefresh           = mirrormev1.TextCode_TEXT_CODE_FPS_MATCHES_REFRESH
	joinWait                    = mirrormev1.TextCode_TEXT_CODE_JOIN_WAIT
	pairHasNoDeviceMemory       = mirrormev1.TextCode_TEXT_CODE_PAIR_HAS_NO_DEVICE_MEMORY
	pairConvertsOnDevice        = mirrormev1.TextCode_TEXT_CODE_PAIR_CONVERTS_ON_DEVICE
	pairTradesColour            = mirrormev1.TextCode_TEXT_CODE_PAIR_TRADES_COLOUR
	exactColourReach            = mirrormev1.TextCode_TEXT_CODE_EXACT_COLOUR_REACH
	devicePathHasNoScaler       = mirrormev1.TextCode_TEXT_CODE_DEVICE_PATH_HAS_NO_SCALER
	drmMapUnusedOnDevice        = mirrormev1.TextCode_TEXT_CODE_DRM_MAP_UNUSED_ON_DEVICE
	cqOnlyInConstantQuality     = mirrormev1.TextCode_TEXT_CODE_CQ_ONLY_IN_CONSTANT_QUALITY
	bitrateNotInMode            = mirrormev1.TextCode_TEXT_CODE_BITRATE_NOT_IN_MODE
	maxrateOnlyInConstrained    = mirrormev1.TextCode_TEXT_CODE_MAXRATE_ONLY_IN_CONSTRAINED_VBR
	vbvOnlyInBoundedModes       = mirrormev1.TextCode_TEXT_CODE_VBV_ONLY_IN_BOUNDED_MODES
	noCeilingInConstantQuality  = mirrormev1.TextCode_TEXT_CODE_NO_CEILING_IN_CONSTANT_QUALITY
	vbvNeedsACeiling            = mirrormev1.TextCode_TEXT_CODE_VBV_NEEDS_A_CEILING
	qualityCeilingRequired      = mirrormev1.TextCode_TEXT_CODE_QUALITY_CEILING_REQUIRED
	bframesOffInMode            = mirrormev1.TextCode_TEXT_CODE_BFRAMES_OFF_IN_MODE
	bframesOnlyOnFamilies       = mirrormev1.TextCode_TEXT_CODE_BFRAMES_ONLY_ON_FAMILIES
	codecTakesNoEffortLadder    = mirrormev1.TextCode_TEXT_CODE_CODEC_TAKES_NO_EFFORT_LADDER
	effortPinnedByMode          = mirrormev1.TextCode_TEXT_CODE_EFFORT_PINNED_BY_MODE
	codecTakesNoTuneLadder      = mirrormev1.TextCode_TEXT_CODE_CODEC_TAKES_NO_TUNE_LADDER
	tunePinnedByMode            = mirrormev1.TextCode_TEXT_CODE_TUNE_PINNED_BY_MODE
	audioCodecNeedsSource       = mirrormev1.TextCode_TEXT_CODE_AUDIO_CODEC_NEEDS_SOURCE
	audioDeviceNeedsSource      = mirrormev1.TextCode_TEXT_CODE_AUDIO_ENTRY_NEEDS_SOURCE
	audioSourceHasOneDevice     = mirrormev1.TextCode_TEXT_CODE_AUDIO_SOURCE_HAS_ONE_DEVICE
	audioDeviceNotEnumerated    = mirrormev1.TextCode_TEXT_CODE_AUDIO_DEVICE_NOT_ENUMERATED
	audioTrackCodedAt           = mirrormev1.TextCode_TEXT_CODE_AUDIO_TRACK_CODED_AT
	vaapiCeilingBound           = mirrormev1.TextCode_TEXT_CODE_VAAPI_CEILING_BOUND
	portalServesNoCursorMode    = mirrormev1.TextCode_TEXT_CODE_PORTAL_SERVES_NO_CURSOR_MODE

	groupRequired               = mirrormev1.TextCode_TEXT_CODE_GROUP_REQUIRED
	groupNameMissing            = mirrormev1.TextCode_TEXT_CODE_GROUP_NAME_MISSING
	groupFollowsDiscord         = mirrormev1.TextCode_TEXT_CODE_GROUP_FOLLOWS_DISCORD
	discordNotLinked            = mirrormev1.TextCode_TEXT_CODE_DISCORD_NOT_LINKED
	discordNoVoiceChannel       = mirrormev1.TextCode_TEXT_CODE_DISCORD_NO_VOICE_CHANNEL
	discordLinkRefused          = mirrormev1.TextCode_TEXT_CODE_DISCORD_LINK_REFUSED
	encryptionFollowsTheAddress = mirrormev1.TextCode_TEXT_CODE_ENCRYPTION_FOLLOWS_THE_ADDRESS
	encryptedRtspInterleavesRtp = mirrormev1.TextCode_TEXT_CODE_ENCRYPTED_RTSP_INTERLEAVES_RTP

	trayIconOff  = mirrormev1.TextCode_TEXT_CODE_TRAY_ICON_OFF
	autostartOff = mirrormev1.TextCode_TEXT_CODE_AUTOSTART_OFF

	publishRefused         = mirrormev1.TextCode_TEXT_CODE_PUBLISH_REFUSED
	noUplinkStated         = mirrormev1.TextCode_TEXT_CODE_NO_UPLINK_STATED
	uplinkBelowPrediction  = mirrormev1.TextCode_TEXT_CODE_UPLINK_BELOW_PREDICTION
	burstAboveUplink       = mirrormev1.TextCode_TEXT_CODE_BURST_ABOVE_UPLINK
	fpsAboveRefresh        = mirrormev1.TextCode_TEXT_CODE_FPS_ABOVE_REFRESH
	monitorNotPriced       = mirrormev1.TextCode_TEXT_CODE_MONITOR_NOT_PRICED
	noPictureToPrice       = mirrormev1.TextCode_TEXT_CODE_NO_PICTURE_TO_PRICE
	ceilingHoldsQuality    = mirrormev1.TextCode_TEXT_CODE_CEILING_HOLDS_QUALITY
	nothingLeftToPick      = mirrormev1.TextCode_TEXT_CODE_NOTHING_LEFT_TO_PICK
	settingsStoreUnreadble = mirrormev1.TextCode_TEXT_CODE_SETTINGS_STORE_UNREADABLE
	presetStoreUnreadable  = mirrormev1.TextCode_TEXT_CODE_PRESET_STORE_UNREADABLE
)

// One constructor per substitution the codes above take.
// The axis a constructor names is what a surface resolves the identifier in.

func argCapture(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CAPTURE, v)
}

func argEngine(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_ENGINE, v)
}

func argCursor(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CURSOR, v)
}

func argOtherEngine(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_OTHER_ENGINE, v)
}

func argTransport(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_TRANSPORT, v)
}

// argValue is a settings value with no axis of its own: a render chain is not a codec, a chroma
// or a transport, and giving it one of those names would say it was.
func argValue(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_VALUE, v)
}

// argOption is the settings field a statement is about, spelled as a field key:
// "publish.publish_transport", which is what a statement about a whole list names.
func argOption(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_OPTION, v)
}

// argElement is a GStreamer element factory, which nobody picks:
// it names what this machine lacks rather than a choice that was made.
func argElement(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_ELEMENT, v)
}

func argCodec(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CODEC, v)
}

func argFormat(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_FORMAT, v)
}

func argFamily(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_FAMILY, v)
}

func argEncoder(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_ENCODER, v)
}

func argChroma(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CHROMA, v)
}

func argMode(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_MODE, v)
}

func argMemory(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_MEMORY, v)
}

func argAudioCodec(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_AUDIO_CODEC, v)
}

func argEffort(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_EFFORT, v)
}

func argTune(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_TUNE, v)
}

func argDecodeFamily(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_DECODE_FAMILY, v)
}

func argDecoder(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_DECODER, v)
}

func argFamilies(v []string) *mirrormev1.TextArg {
	return text.IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_FAMILIES, v)
}

func argFormats(v []string) *mirrormev1.TextArg {
	return text.IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_FORMATS, v)
}

func argTransports(v []string) *mirrormev1.TextArg {
	return text.IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_TRANSPORTS, v)
}

func argAudioCodecs(v []string) *mirrormev1.TextArg {
	return text.IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_AUDIO_CODECS, v)
}

func argDecodeFamilies(v []string) *mirrormev1.TextArg {
	return text.IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_DECODE_FAMILIES, v)
}

func argShareKind(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_SHARE_KIND, v)
}

func argWindow(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_WINDOW, v)
}

func argMonitor(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_MONITOR, int64(v))
}

func argDetailStep(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_DETAIL_STEP, v)
}

func argWidth(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_WIDTH, int64(v))
}

func argHeight(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_HEIGHT, int64(v))
}

func argFps(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_FPS, int64(v))
}

func argRefreshHz(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_REFRESH_HZ, int64(v))
}

// One constructor per substitution name, so a surface reading this argument meets one value shape.
// A whole number of Mbit/s crosses as a Decimal too: a reader keyed on the case would otherwise
// find a Number here and a Decimal in the next sentence, under one name.
func argBitrateMbps(v float64) *mirrormev1.TextArg {
	return text.Dec(mirrormev1.TextArgName_TEXT_ARG_NAME_BITRATE_MBPS, v)
}

func argMaxrateMbps(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_MAXRATE_MBPS, int64(v))
}

func argUplinkMbps(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_UPLINK_MBPS, int64(v))
}

func argLowMbps(v float64) *mirrormev1.TextArg {
	return text.Dec(mirrormev1.TextArgName_TEXT_ARG_NAME_LOW_MBPS, v)
}

func argHighMbps(v float64) *mirrormev1.TextArg {
	return text.Dec(mirrormev1.TextArgName_TEXT_ARG_NAME_HIGH_MBPS, v)
}

func argRateHz(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_RATE_HZ, int64(v))
}

func argBitrateKbps(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_BITRATE_KBPS, int64(v))
}

func argOtherCount(v int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_OTHER_COUNT, int64(v))
}

func argCause(v *mirrormev1.Text) *mirrormev1.TextArg {
	return text.Nested(mirrormev1.TextArgName_TEXT_ARG_NAME_CAUSE, v)
}

func argImport(v *mirrormev1.Text) *mirrormev1.TextArg {
	return text.Nested(mirrormev1.TextArgName_TEXT_ARG_NAME_IMPORT, v)
}

func argCost(v *mirrormev1.Text) *mirrormev1.TextArg {
	return text.Nested(mirrormev1.TextArgName_TEXT_ARG_NAME_COST, v)
}

func argReach(v *mirrormev1.Text) *mirrormev1.TextArg {
	return text.Nested(mirrormev1.TextArgName_TEXT_ARG_NAME_REACH, v)
}

func argAudio(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_AUDIO, v)
}

func argDevice(v string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_DEVICE, v)
}

func argDelayMs(v time.Duration) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_DELAY_MS, v.Milliseconds())
}

func argJoinWaitMs(v time.Duration) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_JOIN_WAIT_MS, v.Milliseconds())
}
