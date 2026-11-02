package receive

import (
	"testing"

	"github.com/go-gst/go-gst/pkg/gstsdp"
)

// rtspsDescription carries one track per profile, the first two as an RTSPS relay writes them.
func rtspsDescription(t *testing.T, protos ...string) *gstsdp.SDPMessage {
	t.Helper()
	msg, res := gstsdp.NewSDPMessage()
	if res != gstsdp.SdpOK {
		t.Fatalf("NewSDPMessage = %v", res)
	}
	for _, proto := range protos {
		media, res := gstsdp.NewSDPMedia()
		if res != gstsdp.SdpOK {
			t.Fatalf("NewSDPMedia = %v", res)
		}
		media.SetMedia("video")
		media.SetProto(proto)
		media.AddFormat("96")
		msg.AddMedia(media)
	}
	return msg
}

func TestAnRTSPSDescriptionIsReadAsPlainRTP(t *testing.T) {
	msg := rtspsDescription(t, "RTP/SAVP", "RTP/SAVPF", "RTP/AVP")

	readPlainRTP(msg)
	readPlainRTP(msg)

	want := []string{"RTP/AVP", "RTP/AVPF", "RTP/AVP"}
	if got := msg.MediasLen(); got != uint(len(want)) {
		t.Fatalf("description holds %d tracks, want %d", got, len(want))
	}
	for i, media := range msg.Medias() {
		if got := media.GetProto(); got != want[i] {
			t.Errorf("track %d proto = %q, want %q", i, got, want[i])
		}
	}
}
