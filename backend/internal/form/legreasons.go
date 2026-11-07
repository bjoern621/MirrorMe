package form

import (
	"slices"
	"strings"

	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/publish"
	"bjoernblessin.de/mirrorme/internal/transport"
)

// transportReason states why the selected capture
// backend's engine cannot carry a publish transport.
// The map from backend to carriable transports is publish's, so a transport known to some backend
// and absent from this one is one whose sink this engine has no serialization for.
func (av availability) transportReason(name string) *mirrormev1.Text {
	allowed, err := publish.TransportsFor(av.s.Publish.Capture)
	if err != nil {
		return nil
	}
	assert.Assert(av.engine != "", "a capture backend the app runs names a publish engine", av.s.Publish.Capture)
	if slices.Contains(allowed, name) {
		return av.legElementReason(name)
	}
	return say(engineHasNoPublishSink,
		argCapture(av.s.Publish.Capture), argEngine(av.engine), argTransport(name))
}

// legElementReason states that this install carries none of the elements the leg's sink is made of,
// nil where it does or where nothing asked.
//
// The other half of what the encoder probe answers: an encoder that runs says nothing
// about the sink after it, and an install carrying an older WHIP element than the one this app
// builds passes every codec probe there is and dies at launch on the sink.
// The GStreamer engine alone, which is the engine whose sink is a named element rather than a muxer
// inside the ffmpeg build (encoders.Availability.Legs).
func (av availability) legElementReason(name string) *mirrormev1.Text {
	if av.engine != capabilities.EngineGst {
		return nil
	}
	if ran, tested := av.deps.Encoders.Legs[name]; !tested || ran {
		return nil
	}
	elements := transport.GstSinkElements(name)
	if len(elements) == 0 {
		return nil
	}
	return say(publishSinkElementMissing,
		argEngine(av.engine), argTransport(name), argElement(strings.Join(elements, " ")))
}

// rtspProtocolReason states why an RTSPS session carries RTP one way only.
//
// RTSPS encrypts the control connection, and RTP over UDP is a second flow beside it that no part
// of that handshake covers.
// Interleaving puts the media inside the TLS connection, so it is the encrypted session's
// only lower transport rather than its faster one (internal/transport, EncryptedRtspProtocol).
//
// Every relay serves RTSPS alone, so the answer does not follow the address
// (deploy/mediamtx-groups.yml).
// Serves both legs, the session being encrypted the same way whichever end negotiates it.
// The publish leg refuses the same value in transport.RTSP ValidatePublishSettings,
// so a draft reaching it with "udp" was greyed here first.
func (av availability) rtspProtocolReason(value string) *mirrormev1.Text {
	if value == transport.EncryptedRtspProtocol {
		return nil
	}
	return say(encryptedRtspInterleavesRtp)
}

// watchesOver reports whether anything on this machine receives over the named transport: the tile,
// set to one leg, or a player, openable on any leg this machine has a receiver for.
//
// The two receivers are asked different questions, which is the difference
// between what each of their settings decides.
// A tile receives over the leg TileWatchTransport names and over no other.
// A player is opened per press on whichever leg the reader picked and stores none,
// so every leg this machine has a player for is one a press can reach.
// Asking a stored leg here hid knobs that were in force: a player opened over RTSP reads
// RtspWatchProtocol whatever a setting says.
// The browser is the same press, so every leg the relay serves a page for is one an address gets
// built on: MoQ reaches this machine through that reader and no other.
//
// Hidden and greyed are both answers about a knob that does nothing here.
// One that does something is shown (docs/field-availability.md).
func (av availability) watchesOver(name string) bool {
	return av.s.Viewer.TileWatchTransport == name ||
		transport.CanWatch(name, capabilities.EngineFfmpeg) ||
		transport.CanWatch(name, transport.EngineBrowser)
}

// reachesOver reports whether anything on this machine builds an address on the relay's listener
// for the named transport, publishing or watching.
//
// The relay serves one listener per protocol and both directions are addressed on it,
// so the port belongs to neither leg alone: SRT.WatchURL and rtspAddress read the same setting
// the publish leg does (internal/transport).
// Asked of the publish leg by itself, the number a viewer still dials disappears from the screen
// the moment publishing moves elsewhere, which leaves a setting in force that nothing shows
// and nothing can change (docs/field-availability.md, "The rule").
func (av availability) reachesOver(name string) bool {
	return av.s.Publish.Transport == name || av.watchesOver(name)
}
