package receive

import (
	"github.com/go-gst/go-gst/pkg/gst"
	"github.com/go-gst/go-gst/pkg/gstsdp"

	"bjoernblessin.de/go-utils/util/assert"
)

// An RTSPS relay describes every track as secure RTP keyed by MIKEY,
// and turns down the MIKEY key rtspsrc answers with (a TEK with its salt, which gortsplib does
// not parse), ending the read at SETUP.
// Every RTSP session here runs over TCP inside TLS, where the relay takes plain RTP,
// so rtspsrc reads each track as plain RTP before it sets one up.

const rtspSourceFactory = "rtspsrc"

// plainProfiles maps each secure RTP profile to the same profile without SRTP.
var plainProfiles = map[string]string{
	"RTP/SAVP":  "RTP/AVP",
	"RTP/SAVPF": "RTP/AVPF",
}

// readPlainSDP rewrites every description rtspsrc receives before it configures the tracks.
func readPlainSDP(src gst.Element) {
	src.Connect("on-sdp", func(_ gst.Element, msg *gstsdp.SDPMessage) {
		readPlainRTP(msg)
	})
}

// readPlainRTP leaves a plain track as it stands, so a second pass changes nothing.
func readPlainRTP(msg *gstsdp.SDPMessage) {
	assert.Assert(msg != nil, "rtspsrc hands over the description it read")

	for _, media := range msg.Medias() {
		if plain, ok := plainProfiles[media.GetProto()]; ok {
			media.SetProto(plain)
		}
	}
}
