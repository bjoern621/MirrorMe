package control

import (
	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/text"
	"bjoernblessin.de/screenshare/internal/wire"
)

// Local names for the refusal codes, so a refusal states its fact on one line.
const (
	requestNotInThisBuild      = screensharev1.TextCode_TEXT_CODE_REQUEST_NOT_IN_THIS_BUILD
	frameSourceUnnamed         = screensharev1.TextCode_TEXT_CODE_FRAME_SOURCE_UNNAMED
	streamUnnamed              = screensharev1.TextCode_TEXT_CODE_STREAM_UNNAMED
	transportUnnamed           = screensharev1.TextCode_TEXT_CODE_TRANSPORT_UNNAMED
	settingsAbsent             = screensharev1.TextCode_TEXT_CODE_SETTINGS_ABSENT
	presetUnnamed              = screensharev1.TextCode_TEXT_CODE_PRESET_UNNAMED
	readNamesNothing           = screensharev1.TextCode_TEXT_CODE_READ_NAMES_NOTHING
	kindNotReadable            = screensharev1.TextCode_TEXT_CODE_KIND_NOT_READABLE
	kindNotWatchable           = screensharev1.TextCode_TEXT_CODE_KIND_NOT_WATCHABLE
	kindUnknown                = screensharev1.TextCode_TEXT_CODE_KIND_UNKNOWN
	linkEmpty                  = screensharev1.TextCode_TEXT_CODE_LINK_EMPTY
	testStreamCountNegative    = screensharev1.TextCode_TEXT_CODE_TEST_STREAM_COUNT_NEGATIVE
	runKeepsNoLog              = screensharev1.TextCode_TEXT_CODE_RUN_KEEPS_NO_LOG
	testStreamsAboveLimit      = screensharev1.TextCode_TEXT_CODE_TEST_STREAMS_ABOVE_LIMIT
	logGone                    = screensharev1.TextCode_TEXT_CODE_LOG_GONE
	receiveAudioOfNoDecode     = screensharev1.TextCode_TEXT_CODE_RECEIVE_AUDIO_OF_NO_DECODE
	framesUndrawable           = screensharev1.TextCode_TEXT_CODE_FRAMES_UNDRAWABLE
	watchRefused               = screensharev1.TextCode_TEXT_CODE_WATCH_REFUSED
	browserOpenRefused         = screensharev1.TextCode_TEXT_CODE_BROWSER_OPEN_REFUSED
	receiveRefused             = screensharev1.TextCode_TEXT_CODE_RECEIVE_REFUSED
	monitorPreviewRefused      = screensharev1.TextCode_TEXT_CODE_MONITOR_PREVIEW_REFUSED
	reportRefused              = screensharev1.TextCode_TEXT_CODE_REPORT_REFUSED
	updateCheckRefused         = screensharev1.TextCode_TEXT_CODE_UPDATE_CHECK_REFUSED
	updateInstallRefused       = screensharev1.TextCode_TEXT_CODE_UPDATE_INSTALL_REFUSED
	uplinkMeasureWhileLive     = screensharev1.TextCode_TEXT_CODE_UPLINK_MEASURE_WHILE_LIVE
	encodeRateMeasureWhileLive = screensharev1.TextCode_TEXT_CODE_ENCODE_RATE_MEASURE_WHILE_LIVE
	protocolMajorMismatch      = screensharev1.TextCode_TEXT_CODE_PROTOCOL_MAJOR_MISMATCH
	linkUnresolved             = screensharev1.TextCode_TEXT_CODE_LINK_UNRESOLVED
	presetUnreachable          = screensharev1.TextCode_TEXT_CODE_PRESET_UNREACHABLE
	settingsUnsaved            = screensharev1.TextCode_TEXT_CODE_SETTINGS_UNSAVED
	presetUnsaved              = screensharev1.TextCode_TEXT_CODE_PRESET_UNSAVED
	presetsUnread              = screensharev1.TextCode_TEXT_CODE_PRESETS_UNREAD
	presetUndeleted            = screensharev1.TextCode_TEXT_CODE_PRESET_UNDELETED
	publishUnstarted           = screensharev1.TextCode_TEXT_CODE_PUBLISH_UNSTARTED
	publishUnapplied           = screensharev1.TextCode_TEXT_CODE_PUBLISH_UNAPPLIED
	testStreamsUnstarted       = screensharev1.TextCode_TEXT_CODE_TEST_STREAMS_UNSTARTED
	portalConsentKept          = screensharev1.TextCode_TEXT_CODE_PORTAL_CONSENT_KEPT
	groupKeyUndrawn            = screensharev1.TextCode_TEXT_CODE_GROUP_KEY_UNDRAWN
	discordUnlinkFailed        = screensharev1.TextCode_TEXT_CODE_DISCORD_UNLINK_FAILED
	discordLinkFailed          = screensharev1.TextCode_TEXT_CODE_DISCORD_LINK_FAILED
	logUnopened                = screensharev1.TextCode_TEXT_CODE_LOG_UNOPENED
	logsFolderUnopened         = screensharev1.TextCode_TEXT_CODE_LOGS_FOLDER_UNOPENED
	uplinkUnmeasured           = screensharev1.TextCode_TEXT_CODE_UPLINK_UNMEASURED
	encodeRateUnmeasured       = screensharev1.TextCode_TEXT_CODE_ENCODE_RATE_UNMEASURED
)

func argStream(name string) *screensharev1.TextArg {
	return text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_STREAM, name)
}

func argTransport(id string) *screensharev1.TextArg {
	return text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_TRANSPORT, id)
}

func argMonitor(index int) *screensharev1.TextArg {
	return text.Num(screensharev1.TextArgName_TEXT_ARG_NAME_MONITOR, int64(index))
}

func argPreset(name string) *screensharev1.TextArg {
	return text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_PRESET, name)
}

func argPath(path string) *screensharev1.TextArg {
	return text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_PATH, path)
}

func argCount(n int) *screensharev1.TextArg {
	return text.Num(screensharev1.TextArgName_TEXT_ARG_NAME_COUNT, int64(n))
}

func argKind(kind screensharev1.EventKind) *screensharev1.TextArg {
	return text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_KIND, kind.String())
}

// argsRef names one stream on one leg, the pair a viewer, a decode and a page are identified by.
func argsRef(ref wire.StreamRef) []*screensharev1.TextArg {
	return []*screensharev1.TextArg{argStream(ref.StreamName), argTransport(ref.Transport)}
}

// argsSource names what a frame stream draws from: a relay decode by its pair, a monitor by its index,
// and the publish preview by naming nothing.
func argsSource(source wire.FrameSource) []*screensharev1.TextArg {
	switch source.Kind {
	case wire.FrameSourceRelay:
		return argsRef(source.Stream)
	case wire.FrameSourceMonitorPreview:
		return []*screensharev1.TextArg{argMonitor(source.Monitor)}
	case wire.FrameSourcePublishPreview:
		return nil
	default:
		assert.Never("unexpected frame source kind", int(source.Kind))
		return nil
	}
}
