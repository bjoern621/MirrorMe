package transport

import (
	"bjoernblessin.de/go-utils/util/assert"
	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"fmt"
	"slices"

	"bjoernblessin.de/screenshare/internal/capabilities"
	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/text"
)

// RTSP streams through the relay's RTSP listener, one RTP stream per track inside the session, so
// no MPEG-TS mux appears anywhere here.
//
// Each leg names the RTP lower transport it runs over: settings.Publish RtspPublishProtocol
// on the publish leg, settings.Viewer RtspWatchProtocol on the watch leg.
// Over TCP the tracks are interleaved on the connection the session already holds.
// Over UDP each takes a port pair, which drops the delay in-order delivery adds.
// Lost RTP is never retransmitted either way.
//
// TCP is the default on both legs because of that port pair.
// The pair is negotiated apart from the RTSP connection, and a client behind NAT reaches it only
// by sending from those ports first: sending creates the mapping, and the relay has to answer
// into it rather than at the port SETUP announced, which is the private one the NAT rewrote.
// The media itself does that sending on the publish leg, and probe packets do it on the watch leg
// before any RTP can come back.
// A network dropping outbound UDP ends both, silently: the control connection sets up over TCP
// as usual and no RTP follows.
type RTSP struct{}

func init() {
	Register(RTSP{})
}

func (RTSP) Name() string { return "rtsp" }

// rtspCarriage is what RTP has a payload format for: every format this app encodes and every audio
// codec it offers (docs/domain-model.md).
// The relay ingests and re-serves all of them, so RTSP carries the whole codec table and is the leg
// the other refusals point at.
//
// All four engine legs share one value, RTP payloading being per format and implemented for every
// one on both engines: ffmpeg's rtp muxer and the rtp*pay elements write the same payload types,
// and their depayloaders read them back.
var rtspCarriage = Carriage{
	Video: []string{"h264", "hevc", "av1", "vp9", "vp8"},
	Audio: []string{"opus", "aac"},
}

var rtspFormats = Formats{
	Publish: map[string]Carriage{
		capabilities.EngineFfmpeg: rtspCarriage,
		capabilities.EngineGst:    rtspCarriage,
	},
	Watch: map[string]Carriage{
		capabilities.EngineFfmpeg: rtspCarriage,
		capabilities.EngineGst:    rtspCarriage,
	},
}

func (RTSP) Formats() Formats { return rtspFormats }

// draftRtpFormats are the video formats whose RTP payload format is an IETF draft.
// ffmpeg's RTP muxer writes one only with compliance loosened, and otherwise ends the publish
// on "Could not write header" before a frame goes out.
// The relay ingests both regardless, so "-strict experimental" is the whole difference between
// an AV1 or VP9 publish and none.
// Keyed by format rather than by codec, since the payload format follows the bitstream and not
// the encoder that made it.
var draftRtpFormats = map[string]bool{"vp9": true, "av1": true}

// PublishArgs muxes to RTSP, which wraps the RTP muxer, so the draft-payload flag applies here too.
func (RTSP) PublishArgs(s settings.Settings) []string {
	args := []string{"-f", "rtsp", "-rtsp_transport", s.Publish.RtspPublishProtocol}
	if c, ok := capabilities.Get(s.Publish.Codec()); ok && draftRtpFormats[c.Format] {
		args = append(args, "-strict", "experimental")
	}
	args = append(args, ffmpegTlsVerify(s)...)
	return append(args, rtspURL(s, s.PublishPath()))
}

// GstSink is one element: rtspclientsink is muxer and sink at once, payloading every attached
// parsed stream into its own RTP track of the same session.
// Carries GstMuxName, where the pipeline's audio branch attaches.
func (RTSP) GstSink(s settings.Settings) []string {
	return []string{
		"rtspclientsink", "name=" + GstMuxName,
		"protocols=" + s.Publish.RtspPublishProtocol,
		gstTlsValidation(s),
		"location=" + rtspURL(s, s.PublishPath()),
	}
}

func (RTSP) WatchURL(s settings.Settings, path string) string {
	assert.Assert(path != "", "a watch URL names the path it opens")

	return rtspURL(s, path)
}

// GstSource carries both watch-leg knobs: latency sizes the rtpjitterbuffer in milliseconds, and
// protocols is the RTP lower transport rtspsrc offers the relay.
// Neither is left at the element's default, a 2000 ms buffer and a UDP-first negotiation.
//
// A track per RTP stream means a pad per track rather than one muxed pad, and the capsfilter
// is what decides which pad the pipeline's decoder is given.
// Without it that is whichever track the relay announced first: the decoder takes any caps and
// a launch line links the first pad that fits, so a session announcing audio first would decode
// the sound and leave the picture nowhere to go.
// The track left unlinked here is decoded beside the picture (internal/receive), so an audio track
// reaches the branch that plays it.
//
// The credential rides beside the address rather than in it, rtspsrc's doing, stated with the pair
// it builds (credential.go).
func (RTSP) GstSource(s settings.Settings, path string) []string {
	assert.Assert(path != "", "a receive source names the path it decodes")

	source := []string{
		"rtspsrc",
		"location=" + rtspAddress(s, path),
		"protocols=" + s.Viewer.RtspWatchProtocol,
		gstTlsValidation(s),
		fmt.Sprintf("latency=%d", s.Viewer.RtspWatchLatencyMs),
	}
	if user, password, ok := rtspCredential(s); ok {
		source = append(source, "user-id="+user, "user-pw="+password)
	}
	// Last, the receiver linking the fragment's tail to its decoder.
	return append(source, "!", "application/x-rtp,media=video")
}

// RtspProtocols are the RTP lower transports this protocol offers, the values RtspPublishProtocol
// and RtspWatchProtocol both take.
// One list for both legs: which transports carry RTP is a fact of RTSP and not of a direction.
var RtspProtocols = []string{"tcp", "udp"}

// EncryptedRtspProtocol is the one lower transport an RTSPS session carries media on, and every
// session this app opens is RTSPS.
//
// RTSPS encrypts the control connection and nothing else: RTP over UDP travels beside it
// in the clear, so a session negotiated that way sends the picture unencrypted however the control
// channel was set up.
// Interleaving puts the RTP inside the TLS connection, which is what makes the media encrypted.
const EncryptedRtspProtocol = "tcp"

// ValidatePublishSettings refuses a lower transport RTSP does not run over, and one that would put
// the media on the wire in the clear.
//
// ffmpeg's -rtsp_transport and rtspclientsink's protocols property take a fixed set of names, so
// a value outside it fails inside the publish process, where the reason reaches the user as another
// program's error text.
//
// UDP is refused on every relay, each of them serving RTSPS and offering the interleaved transport
// alone (deploy/mediamtx-groups.yml, rtspTransports).
// Refused rather than corrected: silently interleaving a session the user asked to carry over UDP
// answers a question they did not ask, and the control that says "udp" would go on saying it while
// the stream did something else.
func (RTSP) ValidatePublishSettings(s settings.Settings) error {
	if !slices.Contains(RtspProtocols, s.Publish.RtspPublishProtocol) {
		return text.Fail(screensharev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN,
			text.ID(screensharev1.TextArgName_TEXT_ARG_NAME_RTSP_PROTOCOL, s.Publish.RtspPublishProtocol))
	}
	if s.Publish.RtspPublishProtocol != EncryptedRtspProtocol {
		return text.Fail(screensharev1.TextCode_TEXT_CODE_ENCRYPTED_RTSP_INTERLEAVES_RTP)
	}
	return nil
}

// ListenerURL is the relay's RTSPS listener, "rtsps://relay:8322".
//
// One scheme, every relay terminating TLS on this leg and binding no cleartext listener at all
// (deploy/mediamtx-groups.yml, rtspEncryption).
// Port is the settings' own, so a relay on other listeners is pointed at by it.
func (RTSP) ListenerURL(s settings.Settings) string {
	return fmt.Sprintf("rtsps://%s:%d", s.Relay.Host, s.Relay.RtspPort)
}

// rtspURL addresses one path on that listener and carries the credential,
// "rtsps://relay:8322/<path>?jwt=<token>".
//
// The credential rides as a query and not as a userinfo password, which is where MediaMTX reads
// a JWT for RTSP.
// ffmpeg and rtspclientsink both keep it there for every request of the session, and rtspsrc does
// not, which rtspCredential covers.
func rtspURL(s settings.Settings, name string) string {
	return rtspAddress(s, name) + credentialQuery(s, "?")
}

// rtspAddress is that address with no credential on it, for the reader that takes one separately.
func rtspAddress(s settings.Settings, name string) string {
	return RTSP{}.ListenerURL(s) + "/" + name
}
