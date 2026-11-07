package form

import (
	"slices"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/share"
	"bjoernblessin.de/mirrorme/internal/window"
)

// availabilityRules is one row per field the form declares, each deciding that control's state
// from the draft and the machine.
//
// A table (docs/development-principles.md): a control added to keys.go and left out here
// fails the lookup in fieldStateOf, where a switch would render it as a plain enabled widget.
// A row that decides nothing says so, where a switch would fall through a default nobody reads.
var availabilityRules = map[string]func(availability) state{
	// The connection group.
	// A listener port is a knob of one protocol and hides with it, the hidden treatment's own case:
	// a user on SRT has no reason to read what the RTMP listener's port means.
	KeyRelayHost: func(availability) state { return availabilityLive() },
	// Encryption follows the relay's address and is stored nowhere (settings.Relay.Tls),
	// so there is nothing here to set: the box says what the connection does,
	// and the field above it is what changes the answer.
	KeyRelayTls: func(availability) state {
		return availabilityDisabled(say(encryptionFollowsTheAddress))
	},
	// The choice stays live under both sources, being what moves between them.
	KeyGroupSource: func(availability) state { return availabilityLive() },
	// Live under both sources as the choice above it is,
	// turning it on being what moves the group onto Discord (internal/app, SaveSettings).
	// Greying it under the key would leave the one control that reaches that source
	// behind the source itself.
	KeyDiscordRichPresence: func(availability) state { return availabilityLive() },
	// Both manual group controls grey while the group comes from the voice channel:
	// the key is unread there and the name comes off the Discord account,
	// so either would be a control that changes nothing (docs/discord-mode.md).
	KeyGroupKey: func(av availability) state {
		if av.s.Relay.FollowsDiscord() {
			return availabilityDisabled(say(groupFollowsDiscord))
		}
		return availabilityLive()
	},
	// A name is claimed per group, so no capture backend, encoder
	// or leg on this screen rules one out.
	// An empty name greys nothing either: it is what leaves this machine outside the group
	// the key names, and this control is where that is answered.
	KeyDisplayName: func(av availability) state {
		if av.s.Relay.FollowsDiscord() {
			return availabilityDisabled(say(groupFollowsDiscord))
		}
		return availabilityLive()
	},
	// Every listener port follows both legs, because the relay serves one listener per protocol
	// in both directions and the number here is what each of them is addressed on (reachesOver).
	// HLS reaches the same rule from the other side: the relay serves it and ingests nothing over it,
	// so the publish half of the question is never the one that answers.
	KeySrtPort:    func(av availability) state { return availabilityShownFor(av.reachesOver(availabilitySrt)) },
	KeyRtspPort:   func(av availability) state { return availabilityShownFor(av.reachesOver(availabilityRtsp)) },
	KeyWebrtcPort: func(av availability) state { return availabilityShownFor(av.reachesOver(availabilityWebrtc)) },
	KeyRtmpPort:   func(av availability) state { return availabilityShownFor(av.reachesOver(availabilityRtmp)) },
	KeyHlsPort:    func(av availability) state { return availabilityShownFor(av.reachesOver(availabilityHls)) },
	KeyMoqPort:    func(av availability) state { return availabilityShownFor(av.reachesOver(availabilityMoq)) },

	// The app group, which no capture backend, codec or leg reaches.
	// Crash reporting stays live on every machine: what a send runs into is an Umgebungsfehler
	// the run reports where it happens, needing no verdict here.
	KeySendCrashReports: func(availability) state { return availabilityLive() },
	// Greyed where the update channel already refuses every check, the same fact the band's own
	// version control states (control.proto, GetUpdateState): a preference over a check that never
	// runs is a control with nothing behind it.
	// Carrying the channel's own statement, so the two surfaces cannot disagree about which fact
	// turned the checks off.
	KeyCheckUpdatesOnStart: func(av availability) state {
		if av.deps.UpdateUnchecked != nil {
			return availabilityDisabled(av.deps.UpdateUnchecked)
		}
		return availabilityLive()
	},
	// The icon is the shell's, drawn against whatever tray the desktop serves,
	// and a desktop serving none leaves the toggle live with nothing to draw.
	// Greyed for one fact: a run holding the icon out whatever the setting holds
	// is a toggle nothing follows.
	KeyTrayIcon: func(av availability) state {
		if av.deps.TrayOff {
			return availabilityDisabled(say(trayIconOff))
		}
		return availabilityLive()
	},
	// The synthetic set publishes to the relay like any other stream and is refused outside a group
	// the same way, which the start reports where it happens rather than greying the toggle here.
	// Hidden in a released build: the set serves development, and a user has no use for the toggle.
	KeyTestStreams: func(av availability) state {
		return availabilityShownFor(!av.deps.TestStreamsOff)
	},

	// Greyed per entry rather than per control: a greyed entry
	// and its reason name the thing to change.
	KeyTransport: func(availability) state { return availabilityLive() },
	KeyFormat:    func(availability) state { return availabilityLive() },
	KeyEncoder:   func(availability) state { return availabilityLive() },
	KeyMode:      func(availability) state { return availabilityLive() },
	KeyCapture:   func(availability) state { return availabilityLive() },

	KeyFps: func(availability) state { return availabilityLive() },
	// The pointer greys per entry, the control itself staying live: every backend serves at least one mode.
	KeyPointer: func(availability) state { return availabilityLive() },

	// The pixel format carries a note about the viewer's machine (decodeNote)
	// and greys for one fact only: on the device path that converts nothing the encoder picks
	// the layout, so the control would otherwise show a value the stream does not carry.
	KeyChroma: func(av availability) state {
		if reason := av.encoderColourReason(); reason != nil {
			return availabilityDisabled(reason)
		}
		return availabilityNoted(av.decodeNote())
	},
	// Two facts block the colour range, and the encoder-colour path outranks the format's own:
	// under it neither colour field reaches the stream, so naming RGB would send the user changing
	// a control that changes nothing.
	KeyColorRange: func(av availability) state {
		if reason := av.encoderColourReason(); reason != nil {
			return availabilityDisabled(reason)
		}
		if slices.Contains(availabilityFullRangeChromas, av.s.Publish.Chroma) {
			return availabilityDisabled(say(rgbIsFullRange))
		}
		return availabilityLive()
	},

	// The rate-control knobs.
	// Three facts decide each, weighed by the knob helper
	// in the order docs/field-availability.md states.
	KeyCq: func(av availability) state {
		return av.knob(KeyCq, av.mode().usesCq, say(cqOnlyInConstantQuality))
	},
	KeyBitrateM: func(av availability) state {
		return av.knob(KeyBitrateM, av.mode().usesBitrate, say(bitrateNotInMode, argMode(av.s.Publish.Mode)))
	},
	// The ceiling and the buffer holding it answer together.
	// A bitrate mode is bounded by its target, so both come with the mode alone;
	// a quality target is free of the rate,
	// so both need an encoder that holds it in a VBV and a ceiling to hold.
	KeyMaxrateM: func(av availability) state {
		st := av.knob(KeyMaxrateM, av.mode().usesMaxrate && av.encoderBounds(), av.ceilingReason())
		if st.enabled {
			st.note = av.ceilingNote()
		}
		return st
	},
	KeyVbvMs: func(av availability) state {
		uses := av.mode().usesVbv && av.encoderBounds() && av.ceilingStated()
		return av.knob(KeyVbvMs, uses, av.windowReason())
	},
	// The keyframe interval is no rate-control concept, so no mode withholds it.
	// Only an encoder element with no property for it does.
	KeyGop: func(av availability) state { return av.knob(KeyGop, true, nil) },
	// Two independent facts block B-frames, so the reason names the one that applies rather
	// than always blaming the mode.
	// Which families take a count is the family table's, so the statement lists them from it.
	KeyBframes: func(av availability) state {
		usesBframes := av.mode().usesBframes
		reason := say(bframesOffInMode, argMode(av.s.Publish.Mode))
		if usesBframes {
			reason = say(bframesOnlyOnFamilies,
				argFamilies(availabilityFamiliesWith(func(f availabilityFamily) bool { return f.takesBframes })))
		}
		return av.knob(KeyBframes, usesBframes && av.family().takesBframes, reason)
	},
	// The effort step follows the codec's own row rather than its family's, because the ladder does:
	// the steps are the encoder's identifiers, so two codecs of one family can offer different ones.
	// The entries this control lists and the step a build spends come off that same row,
	// which keeps the greying and the encode in step.
	KeyEffort: func(av availability) state {
		ladder := av.codec.Effort
		pinned := ladder.PinsIn(av.s.Publish.Mode)
		reason := say(codecTakesNoEffortLadder, argCodec(av.s.Publish.Codec()))
		// A mode that pins the step carries the one it pins to, so the statement names the declared value
		// rather than a restated one.
		if pinned {
			step, _ := ladder.StepFor(av.s.Publish.Mode)
			reason = say(effortPinnedByMode, argMode(av.s.Publish.Mode), argEffort(step))
		}
		return av.knob(KeyEffort, len(ladder.Steps) > 0 && !pinned, reason)
	},
	// The tune reads the same row, being the other half of one decision: how hard the encoder works,
	// and what it works towards.
	// Either ladder can be declared without the other, the Vulkan rows tuning
	// and taking no effort step and the VAAPI ones declaring neither,
	// so each control asks about its own.
	KeyTune: func(av availability) state {
		ladder := av.codec.Tune
		pinned := ladder.PinsIn(av.s.Publish.Mode)
		reason := say(codecTakesNoTuneLadder, argCodec(av.s.Publish.Codec()))
		if pinned {
			step, _ := ladder.StepFor(av.s.Publish.Mode)
			reason = say(tunePinnedByMode, argMode(av.s.Publish.Mode), argTune(step))
		}
		return av.knob(KeyTune, len(ladder.Steps) > 0 && !pinned, reason)
	},

	// The audio codec is read only where the stream has a track to code,
	// so with no source the control is inert rather than wrong.
	// Greyed rather than hidden: the codec is a general concept,
	// so the statement says why it does not apply here.
	KeyAudioCodec: func(av availability) state {
		if len(av.s.Publish.Recorded()) == 0 {
			return availabilityDisabled(say(audioCodecNeedsSource))
		}
		return availabilityLive()
	},

	// One entry of the source list.
	// The kind is always editable, because it is what an entry is:
	// setting it to none is how an entry is taken off,
	// and greying it would leave a source nobody can remove.
	KeyAudioSource: func(availability) state { return availabilityLive() },

	// What is inside the kind, which only some kinds have more than one of.
	// A kind with one device leaves nothing to choose, so the control greys naming the kind rather
	// than offering a list of one.
	KeyAudioSourceDevice: func(av availability) state {
		if !av.source.Records() {
			return availabilityDisabled(say(audioDeviceNeedsSource))
		}
		if len(audioDevicesOf(av.deps, av.source.Source)) < 2 {
			return availabilityDisabled(say(audioSourceHasOneDevice, argAudio(av.source.Source)))
		}
		return availabilityLive()
	},

	// The level and the silence mean nothing until the entry names a source.
	// Both reach a pipeline that is already running, so nothing else greys them.
	KeyAudioSourceGain: func(av availability) state {
		if !av.source.Records() {
			return availabilityDisabled(say(audioDeviceNeedsSource))
		}
		return availabilityLive()
	},
	KeyAudioSourceMute: func(av availability) state {
		if !av.source.Records() {
			return availabilityDisabled(say(audioDeviceNeedsSource))
		}
		return availabilityLive()
	},

	// The DRM download strategy is the hidden treatment's own example:
	// a knob of the kmsgrab scanout path and of nothing else, whose help would teach a user
	// on any other backend nothing.
	// Under kmsgrab it greys rather than hiding a second time, which would make it appear
	// and vanish while the user changes codecs.
	KeyDrmMap: func(av availability) state {
		if av.s.Publish.Capture != availabilityKmsgrab {
			return availabilityHidden()
		}
		if av.staysOnDevice() {
			return availabilityDisabled(say(drmMapUnusedOnDevice))
		}
		return availabilityLive()
	},
	// What is shared: one control for the kind, and one per target the kinds name (share.go).
	KeyShareKind: shareKindState,
	// ddagrab selects an output by index
	// and the X backends crop the X screen to the monitor's geometry.
	// A backend that takes no index names itself,
	// so the surface can state what that one captures instead.
	KeyMonitor: func(av availability) state {
		if st := shareTargetState(av, share.Monitor); !st.visible {
			return st
		}
		if slices.Contains(availabilityMonitorless, av.s.Publish.Capture) {
			return availabilityDisabled(say(captureTakesNoMonitor, argCapture(av.s.Publish.Capture)))
		}
		return availabilityLive()
	},
	KeyShareWindow: func(av availability) state {
		if st := shareTargetState(av, share.Window); !st.visible {
			return st
		}
		if gap := window.Statement(av.deps.Platform); gap != nil {
			return availabilityDisabled(gap)
		}
		return availabilityLive()
	},
	KeyShareRegion: func(av availability) state { return shareTargetState(av, share.Region) },
	// The frame memory never greys as a whole: auto and the system copy are values every pair
	// satisfies, so no combination leaves a dead control.
	KeyCaptureMemory: func(av availability) state { return availabilityNoted(av.frameMemoryNote()) },

	// The per-protocol knobs, hidden with their protocol like the listener ports:
	// each names a mechanism of one leg of one transport.
	KeySrtPublishLatencyMs: func(av availability) state {
		return availabilityShownFor(av.s.Publish.Transport == availabilitySrt)
	},
	// Greyed per entry on an encrypted relay rather than as a control,
	// so the value that works stays selectable: a disabled dropdown still showing "udp" would name
	// the refusal and offer no way out of it (rtspProtocolReason).
	KeyRtspPublishProtocol: func(av availability) state {
		return availabilityShownFor(av.s.Publish.Transport == availabilityRtsp)
	},
	// Hidden on the ffmpeg engine, which holds the encoder on every run and offers no other answer.
	KeyCongestion: func(av availability) state {
		return availabilityShownFor(av.engine == capabilities.EngineGst)
	},
	// The SRT window and the RTP lower transport belong to the link rather than to one reader,
	// so they follow either receiver (watchesOver).
	KeySrtWatchLatencyMs: func(av availability) state {
		return availabilityShownFor(av.watchesOver(availabilitySrt))
	},
	KeyRtspWatchProtocol: func(av availability) state {
		return availabilityShownFor(av.watchesOver(availabilityRtsp))
	},
	// The jitter buffer is the tile receiver's alone:
	// a player buffers by reorder queue rather than by time.
	KeyRtspWatchLatencyMs: func(av availability) state {
		return availabilityShownFor(av.s.Viewer.TileWatchTransport == availabilityRtsp)
	},

	KeyUplinkMbps:         func(availability) state { return availabilityLive() },
	KeyTileWatchTransport: func(availability) state { return availabilityLive() },

	// The render chain is live whatever leg is chosen: what a decode converts its frames
	// with is a property of this machine's GStreamer rather than of the protocol they arrived over.
	KeyRenderChain:    func(availability) state { return availabilityLive() },
	KeyConstantVolume: func(availability) state { return availabilityLive() },

	// What refuses an output resolution is one entry rather than the control,
	// so a frame path that resizes nothing leaves the source size standing (outputResolutionReason).
	KeyOutputResolution: func(availability) state { return availabilityLive() },
}

// availabilityOptionRules is the option half of the same table:
// one row per field whose entries grey individually,
// each answering why this combination rules a value out.
// A row returns the statement shown in place of the value it withholds,
// nil for a value it leaves selectable.
// A field with no row here greys no entry.
var availabilityOptionRules = map[string]func(availability, string) *mirrormev1.Text{
	KeyShareKind:         shareKindReason,
	KeyCapture:           availability.captureReason,
	KeyTransport:         availability.transportReason,
	KeyFormat:            availability.formatReason,
	KeyEncoder:           availability.encoderReason,
	KeyChroma:            availability.chromaReason,
	KeyMode:              availability.modeReason,
	KeyColorRange:        availability.colorRangeReason,
	KeyTune:              availability.tuneReason,
	KeyAudioSource:       availability.audioReason,
	KeyAudioSourceDevice: availability.audioDeviceReason,
	KeyAudioCodec:        availability.audioCodecReason,
	KeyCaptureMemory:     availability.frameMemoryReason,
	KeyPointer:           availability.pointerReason,
	KeyOutputResolution:  availability.outputResolutionReason,
	// Both legs, one rule: RTSPS covers the control connection
	// and neither direction's RTP travels inside it over UDP.
	KeyRtspPublishProtocol: availability.rtspProtocolReason,
	KeyRtspWatchProtocol:   availability.rtspProtocolReason,
	// The tile's leg has no row.
	// Its roster is the legs a GStreamer receiver can open (optionTileWatchTransports),
	// so every entry is one, and the format is answered per stream as the decode opens
	// (internal/app, carriesStream).
	KeyRenderChain: availability.renderChainReason,
}

// availabilityKmsgrab is the one capture backend a field is gated on by name:
// the DRM download strategy is a knob of its scanout path and of nothing else.
const availabilityKmsgrab = "kmsgrab"

// The transport names the per-protocol knobs are gated on,
// spelled as the transport registry keys them.
// A name the registry does not carry would gate a control on a protocol nothing publishes
// (TestEveryGatedTransportIsRegistered).
const (
	availabilitySrt    = "srt"
	availabilityRtsp   = "rtsp"
	availabilityWebrtc = "webrtc"
	availabilityRtmp   = "rtmp"
	availabilityHls    = "hls"
	availabilityMoq    = "moq"
)

// availabilityMonitorless is the capture backends that take no monitor index.
// What each captures instead follows from the backend, so the statement names the backend
// and the surface says what that one grabs,
// the same sentence its entry in the capture dropdown already writes.
var availabilityMonitorless = []string{availabilityKmsgrab, "gdigrab", "portal"}

// availabilityFullRangeChromas are the pixel formats carrying no quantization range choice at all.
// RGB is full range by construction, leaving the colour range control nothing to decide under it.
var availabilityFullRangeChromas = []string{"gbrp"}
