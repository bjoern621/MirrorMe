package control

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/text"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// Local names for the refusal codes, so a refusal states its fact on one line.
const (
	requestNotInThisBuild      = mirrormev1.TextCode_TEXT_CODE_REQUEST_NOT_IN_THIS_BUILD
	frameSourceUnnamed         = mirrormev1.TextCode_TEXT_CODE_FRAME_SOURCE_UNNAMED
	streamUnnamed              = mirrormev1.TextCode_TEXT_CODE_STREAM_UNNAMED
	transportUnnamed           = mirrormev1.TextCode_TEXT_CODE_TRANSPORT_UNNAMED
	settingsAbsent             = mirrormev1.TextCode_TEXT_CODE_SETTINGS_ABSENT
	presetUnnamed              = mirrormev1.TextCode_TEXT_CODE_PRESET_UNNAMED
	readNamesNothing           = mirrormev1.TextCode_TEXT_CODE_READ_NAMES_NOTHING
	kindNotReadable            = mirrormev1.TextCode_TEXT_CODE_KIND_NOT_READABLE
	kindNotWatchable           = mirrormev1.TextCode_TEXT_CODE_KIND_NOT_WATCHABLE
	kindUnknown                = mirrormev1.TextCode_TEXT_CODE_KIND_UNKNOWN
	linkEmpty                  = mirrormev1.TextCode_TEXT_CODE_LINK_EMPTY
	testStreamCountNegative    = mirrormev1.TextCode_TEXT_CODE_TEST_STREAM_COUNT_NEGATIVE
	runKeepsNoLog              = mirrormev1.TextCode_TEXT_CODE_RUN_KEEPS_NO_LOG
	testStreamsAboveLimit      = mirrormev1.TextCode_TEXT_CODE_TEST_STREAMS_ABOVE_LIMIT
	testStreamsReleaseBuild    = mirrormev1.TextCode_TEXT_CODE_TEST_STREAMS_RELEASE_BUILD
	logGone                    = mirrormev1.TextCode_TEXT_CODE_LOG_GONE
	receiveAudioOfNoDecode     = mirrormev1.TextCode_TEXT_CODE_RECEIVE_AUDIO_OF_NO_DECODE
	framesUndrawable           = mirrormev1.TextCode_TEXT_CODE_FRAMES_UNDRAWABLE
	watchRefused               = mirrormev1.TextCode_TEXT_CODE_WATCH_REFUSED
	browserOpenRefused         = mirrormev1.TextCode_TEXT_CODE_BROWSER_OPEN_REFUSED
	receiveRefused             = mirrormev1.TextCode_TEXT_CODE_RECEIVE_REFUSED
	monitorPreviewRefused      = mirrormev1.TextCode_TEXT_CODE_MONITOR_PREVIEW_REFUSED
	reportRefused              = mirrormev1.TextCode_TEXT_CODE_REPORT_REFUSED
	updateCheckRefused         = mirrormev1.TextCode_TEXT_CODE_UPDATE_CHECK_REFUSED
	updateInstallRefused       = mirrormev1.TextCode_TEXT_CODE_UPDATE_INSTALL_REFUSED
	uplinkMeasureWhileLive     = mirrormev1.TextCode_TEXT_CODE_UPLINK_MEASURE_WHILE_LIVE
	encodeRateMeasureWhileLive = mirrormev1.TextCode_TEXT_CODE_ENCODE_RATE_MEASURE_WHILE_LIVE
	protocolMajorMismatch      = mirrormev1.TextCode_TEXT_CODE_PROTOCOL_MAJOR_MISMATCH
	linkUnresolved             = mirrormev1.TextCode_TEXT_CODE_LINK_UNRESOLVED
	presetUnreachable          = mirrormev1.TextCode_TEXT_CODE_PRESET_UNREACHABLE
	settingsUnsaved            = mirrormev1.TextCode_TEXT_CODE_SETTINGS_UNSAVED
	presetUnsaved              = mirrormev1.TextCode_TEXT_CODE_PRESET_UNSAVED
	presetsUnread              = mirrormev1.TextCode_TEXT_CODE_PRESETS_UNREAD
	presetUndeleted            = mirrormev1.TextCode_TEXT_CODE_PRESET_UNDELETED
	publishUnstarted           = mirrormev1.TextCode_TEXT_CODE_PUBLISH_UNSTARTED
	publishUnapplied           = mirrormev1.TextCode_TEXT_CODE_PUBLISH_UNAPPLIED
	testStreamsUnstarted       = mirrormev1.TextCode_TEXT_CODE_TEST_STREAMS_UNSTARTED
	portalConsentKept          = mirrormev1.TextCode_TEXT_CODE_PORTAL_CONSENT_KEPT
	groupKeyUndrawn            = mirrormev1.TextCode_TEXT_CODE_GROUP_KEY_UNDRAWN
	discordUnlinkFailed        = mirrormev1.TextCode_TEXT_CODE_DISCORD_UNLINK_FAILED
	discordLinkFailed          = mirrormev1.TextCode_TEXT_CODE_DISCORD_LINK_FAILED
	logUnopened                = mirrormev1.TextCode_TEXT_CODE_LOG_UNOPENED
	logsFolderUnopened         = mirrormev1.TextCode_TEXT_CODE_LOGS_FOLDER_UNOPENED
	uplinkUnmeasured           = mirrormev1.TextCode_TEXT_CODE_UPLINK_UNMEASURED
	encodeRateUnmeasured       = mirrormev1.TextCode_TEXT_CODE_ENCODE_RATE_UNMEASURED
)

func argStream(name string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_STREAM, name)
}

func argTransport(id string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_TRANSPORT, id)
}

func argMonitor(index int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_MONITOR, int64(index))
}

func argPreset(name string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_PRESET, name)
}

func argPath(path string) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_PATH, path)
}

func argCount(n int) *mirrormev1.TextArg {
	return text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_COUNT, int64(n))
}

func argKind(kind mirrormev1.EventKind) *mirrormev1.TextArg {
	return text.ID(mirrormev1.TextArgName_TEXT_ARG_NAME_KIND, kind.String())
}

// argsRef names one stream on one leg, the pair a viewer, a decode and a page are identified by.
func argsRef(ref wire.StreamRef) []*mirrormev1.TextArg {
	return []*mirrormev1.TextArg{argStream(ref.StreamName), argTransport(ref.Transport)}
}

// argsSource names what a frame stream draws from: a relay decode by its pair, a monitor by its index,
// and the publish preview by naming nothing.
func argsSource(source wire.FrameSource) []*mirrormev1.TextArg {
	switch source.Kind {
	case wire.FrameSourceRelay:
		return argsRef(source.Stream)
	case wire.FrameSourceMonitorPreview:
		return []*mirrormev1.TextArg{argMonitor(source.Monitor)}
	case wire.FrameSourcePublishPreview:
		return nil
	default:
		assert.Never("unexpected frame source kind", int(source.Kind))
		return nil
	}
}
