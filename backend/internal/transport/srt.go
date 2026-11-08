package transport

import (
	"bjoernblessin.de/go-utils/util/assert"
	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"fmt"
	"net/url"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/settings"
	"bjoernblessin.de/mirrorme/internal/text"
)

// SRT streams through the relay's SRT listener, MPEG-TS inside a retransmit window.
// Every constant here is measured, and changing one takes a measurement:
//   - ffmpeg's srt protocol counts latency in MICROSECONDS, not milliseconds.
//   - sndbuf, rcvbuf and ffs are large so a lossless keyframe burst survives while a display-paced
//     player drains slowly.
//   - srtTSPackets sets the datagram size.
//
// A latency set here is what this end asks for, not what the link runs at.
// The handshake negotiates one delay per direction and the larger of the two ends' values wins, so
// the relay's own figure is a floor under every hop (settings.SrtRelayFloorMs).
type SRT struct{}

func init() {
	Register(SRT{})
}

// srtBufBytes sizes the SRT window and the socket buffers, 150 MB.
const srtBufBytes = 150_000_000

// srtPaceFloorBytesPerSec is the lowest send ceiling a publish gets, in libsrt's maxbw unit.
//
// Unpaced, libsrt sends a keyframe as fast as the link takes it.
// The relay's SRT reader queues 1024 packets and drops the rest without a word, so a fast uplink
// loses every large keyframe and a viewer waits for one that never arrives.
// On loopback through MediaMTX 1.20.0, 1.5 MB keyframes crossed at 2, 4 and 8 MB/s and all vanished
// at 12.5 MB/s.
const srtPaceFloorBytesPerSec = 4_000_000

// srtPaceHeadroomPercent keeps the ceiling above the encoder's peak rate, since retransmissions
// spend the same budget and a ceiling under the stream's own rate would starve it.
const srtPaceHeadroomPercent = 125

// srtRelativeOverheadPercent is how far above its measured input rate libsrt sends a publish without a ceiling.
const srtRelativeOverheadPercent = 25

// srtSenderDropOff turns off libsrt's drop of packets that it still holds past the window (SRTO_SNDDROPDELAY).
//
// The relative mode paces near the stream's average.
// On a still screen a keyframe makes up most of that average and leaves over most of a GOP.
// With the drop on, libsrt discards the rest of it and a viewer gets no picture.
// A 158 KB keyframe vanished that way on loopback through MediaMTX 1.20.0 at a 300 ms window.
// A link too slow for the stream delays its frames until the send buffer fills.
const srtSenderDropOff = -1

// srtBytesPerSecPerMbps converts the Mbps of a rate setting to libsrt's bytes per second.
const srtBytesPerSecPerMbps = 125_000

// srtMaxBandwidth is the send ceiling of a publish in bytes per second, and 0 where none applies.
// crf and lossless state no rate, and a ceiling guessed for them could sit under the stream.
func srtMaxBandwidth(p settings.Publish) int64 {
	var peakM int
	switch p.Mode {
	case capabilities.ModeCbr, capabilities.ModeAbr:
		peakM = p.BitrateM
	case capabilities.ModeVbr:
		peakM = max(p.BitrateM, p.MaxrateM)
	default:
		return 0
	}
	return max(srtPaceFloorBytesPerSec, int64(peakM)*srtBytesPerSecPerMbps*srtPaceHeadroomPercent/100)
}

// srtPaceQuery writes the pace as query parameters after sep.
//
// A publish without a ceiling takes libsrt's relative mode, where maxbw=0 paces at the input rate
// that libsrt measures plus the overhead.
// It caps no stream and still spreads a keyframe.
// On loopback through MediaMTX 1.20.0 it carried 2.1 MB keyframes at a 300 ms window.
// Keyframes near 2.9 MB arrived in one run and vanished in another.
func srtPaceQuery(p settings.Publish, sep string) string {
	ceiling := srtMaxBandwidth(p)
	if ceiling == 0 {
		return fmt.Sprintf("%smaxbw=0&oheadbw=%d&snddropdelay=%d", sep, srtRelativeOverheadPercent, srtSenderDropOff)
	}
	return fmt.Sprintf("%smaxbw=%d", sep, ceiling)
}

// srtTSPackets counts the 188-byte MPEG-TS packets in one SRT datagram.
// Six make 1208 bytes behind SRT, UDP and IPv6 headers, which crosses a 1280-byte link whole:
// the IPv6 minimum, and the MTU of the relay's pod network.
// Seven, libsrt's default payload, make 1380 and are dropped wherever nothing fragments them,
// leaving a publish that connects and delivers no frame.
const srtTSPackets = 6

func (SRT) Name() string { return "srt" }

// srtCarriage is what MediaMTX registers an MPEG-TS stream type for: H.264 and H.265, with Opus and
// AAC beside them.
// AV1, VP9 and VP8 have no mapping there, so a stream in one of them reaches no SRT publish and
// comes out of no SRT read, whatever leg the relay ingested it over.
//
// One value covers all four engine legs, MPEG-TS being where the two engines meet: ffmpeg's mpegts
// muxer and mpegtsmux write the same stream types, and libavformat and tsdemux read them back.
// Naming it once makes that agreement a statement rather than four lists that happen to match.
var srtCarriage = Carriage{
	Video: []string{"h264", "hevc"},
	Audio: []string{"opus", "aac"},
}

var srtFormats = Formats{
	Publish: map[string]Carriage{
		capabilities.EngineFfmpeg: srtCarriage,
		capabilities.EngineGst:    srtCarriage,
	},
	Watch: map[string]Carriage{
		capabilities.EngineFfmpeg: srtCarriage,
		capabilities.EngineGst:    srtCarriage,
	},
}

func (SRT) Formats() Formats { return srtFormats }

// ListenerURL is the relay's SRT listener, "srt://relay:8890".
// No TLS on this leg: UDP, protected by the per-prefix passphrase (settings.Relay.SrtPassphrase).
func (SRT) ListenerURL(s settings.Settings) string {
	return fmt.Sprintf("srt://%s:%d", s.Relay.Host, s.Relay.SrtPort)
}

func (SRT) PublishArgs(s settings.Settings) []string {
	// ffmpeg's srt protocol takes every knob as a URL query: latency in MICROSECONDS, and the buffers
	// under ffmpeg's own names (pkt_size, sndbuf, ffs).
	url := SRT{}.ListenerURL(s) + fmt.Sprintf(
		"?streamid=%s&pkt_size=%d&latency=%d&sndbuf=%d&ffs=%d",
		srtStreamID(s, "publish", s.PublishPath()), srtTSPackets*188,
		s.Publish.SrtPublishLatencyMs*1000, srtBufBytes, srtBufBytes) + srtPaceQuery(s.Publish, "&") + srtPassphraseQuery(s)

	return []string{"-f", "mpegts", url}
}

// GstSink configures srtsink the way libsrt takes it, which is not the way ffmpeg's srt protocol
// does: the URI is a bare srt://host:port, and streamid and latency are properties, latency
// in MILLISECONDS rather than microseconds.
// alignment packs srtTSPackets per buffer, one buffer per datagram.
func (SRT) GstSink(s settings.Settings) []string {
	return append([]string{
		"mpegtsmux", "name=" + GstMuxName, fmt.Sprintf("alignment=%d", srtTSPackets),
		"!", "srtsink",
		"uri=" + SRT{}.ListenerURL(s) + srtPaceQuery(s.Publish, "?"),
		"mode=caller",
		"streamid=" + srtStreamID(s, "publish", s.PublishPath()),
		fmt.Sprintf("latency=%d", s.Publish.SrtPublishLatencyMs),
		"wait-for-connection=false",
	}, srtPassphraseProperty(s)...)
}

func (SRT) WatchURL(s settings.Settings, path string) string {
	assert.Assert(path != "", "a watch URL names the path it opens")

	return SRT{}.ListenerURL(s) + fmt.Sprintf(
		"?streamid=%s&latency=%d&rcvbuf=%d&ffs=%d",
		srtStreamID(s, "read", path),
		s.Viewer.SrtWatchLatencyMs*1000, srtBufBytes, srtBufBytes) + srtPassphraseQuery(s)
}

// GstSource splits as GstSink does: srtsrc takes streamid as a property on a bare srt:// URI.
// The buffer options WatchURL carries are ffmpeg protocol knobs with no srtsrc equivalent.
//
// The window goes to the socket as rcvlatency (milliseconds) and the latency property stays at 0.
// srtsrc answers a latency query with that property, and a frame's timestamp at the decoder already
// lies behind the window, so the sink would hold the window a second time.
func (SRT) GstSource(s settings.Settings, path string) []string {
	assert.Assert(path != "", "a receive source names the path it decodes")

	return append([]string{
		"srtsrc",
		fmt.Sprintf("uri=%s?rcvlatency=%d", SRT{}.ListenerURL(s), s.Viewer.SrtWatchLatencyMs),
		"mode=caller",
		"streamid=" + srtStreamID(s, "read", path),
		"latency=0",
	}, srtPassphraseProperty(s)...)
}

// ValidatePublishSettings refuses a stream id that would not survive the wire, and a stream
// that would leave this machine unencrypted.
//
// The id carries the path and the token, and SRT truncates at srtStreamIDBytes rather than
// refusing, so a cut token reaches the relay as a signature error naming nothing a user can act on.
// A settings problem with a settings fix, caught here and named as one.
//
// The passphrase encrypts SRT, there being no TLS on it: UDP, and the reverse proxy wrapping every
// HTTP leg of an encrypted relay never sees this one.
// It derives from the group key, so a machine outside a group derives none, and the leg is refused
// here rather than sent in the clear to a path the relay refuses anyway.
// The refusal names the group, that being what a user sets.
func (SRT) ValidatePublishSettings(s settings.Settings) error {
	id := srtStreamID(s, "publish", s.PublishPath())
	if !srtStreamIDFits(id) {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_STREAM_ID_TOO_LONG,
			text.Count(len(id)), text.CountLimit(srtStreamIDBytes))
	}
	if s.Relay.Tls() && s.Relay.SrtPassphrase() == "" {
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_GROUP_REQUIRED)
	}
	return nil
}

// srtPassphraseQuery is the passphrase as ffmpeg's srt protocol takes it, empty where the path
// is one no derivation keys (settings.Relay.SrtPassphrase).
//
// Both legs carry it, the relay keying both directions of a prefix alike (internal/groupsvc).
// A passphrase on one side only is a stream that connects and never plays.
func srtPassphraseQuery(s settings.Settings) string {
	passphrase := s.Relay.SrtPassphrase()
	if passphrase == "" {
		return ""
	}
	return "&passphrase=" + url.QueryEscape(passphrase)
}

// srtPassphraseProperty is the same value as srtsink and srtsrc take it, a property rather than
// a URI query, the split the latency and the stream id already have between the two engines.
func srtPassphraseProperty(s settings.Settings) []string {
	passphrase := s.Relay.SrtPassphrase()
	if passphrase == "" {
		return nil
	}
	return []string{"passphrase=" + passphrase}
}
