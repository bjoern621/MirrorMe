package control

import (
	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/go-utils/util/logger"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/form"
	"bjoernblessin.de/screenshare/internal/publish"
	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/wire"
)

// The states Read hands over and the two questions Ask answers:
// each computes and changes nothing (docs/ipc-api.md, "The rule").
//
// None states a precondition of its own.
// Each forwards to the backend and shapes the answer for the wire,
// so the contract belongs to what it forwards to.
// The one condition under all of them, the backend the service holds, is asserted in server.go.
// A state takes no argument it could refuse, so every one is safe to read at any moment.
//
// A shell resolves a form on every keystroke and reads the publish state on every mount,
// so a read that saved, started or probed would fire at a rate nobody chose.
// The form is the sharp case, and its contract states both halves: no save,
// and the probe result read rather than taken.
//
// No sentence for the screen is built here:
// labels, greyings and derived figures are form's, and the shapes on the wire are wire's.
// Here is which of the two to ask, and what to hand it.

// catalog builds every fixed fact known about this machine and the encoding model.
//
// It reads the probe result.
// Probing is Do's probe_encoders, and what it finds reaches every shell on Watch.
// One function, because the read and the event the probe announces carry the same catalog.
// Two builders would draw one table for the shell that asked for the probe,
// another for the shell that was told about it.
func (s *Server) catalog() *screensharev1.Catalog {
	catalog := wire.Catalog(wire.CatalogInput{
		Platform:     s.backend.Platform(),
		Monitors:     s.backend.Monitors(),
		Encoders:     s.backend.CachedEncoders(),
		AudioDevices: s.backend.AudioDevices(),
	})

	assert.IsNotNil(catalog, "a read of the reference set answers with one")
	return catalog
}

// storedSettings answers with the settings the backend holds:
// the draft a form starts from, and the values a fresh shell opens on.
//
// The store notice rides here rather than on the catalog.
// Why the persisted settings could not be restored is a fact about the settings,
// so it belongs beside the defaults it explains.
func (s *Server) storedSettings() *screensharev1.StoredSettings {
	return &screensharev1.StoredSettings{
		Settings:    wire.Settings(s.backend.Settings()),
		StoreNotice: s.backend.StoreNotice(),
	}
}

// resolveForm turns a settings draft into the complete description of the screen.
//
// It reads CachedEncoders.
// The contract promises a resolve cheap enough to call on every keystroke,
// and the probe takes seconds on its first call:
// taking one here would charge those seconds to the first character typed,
// with every keystroke until it returned queued behind it.
// The cached result costs the smaller and honest thing instead:
// on a machine nothing has probed, no engine is greyed for a missing encoder,
// a form that has not been told rather than a form saying there is nothing.
//
// It saves nothing either.
// A form is resolved far more often than a value is committed,
// so a resolve that wrote would make every intermediate keystroke a persisted setting.
//
// A question carrying no settings resolves the empty draft instead of being refused, unlike Set and Do.
// Repair is what a resolve is for:
// it walks a draft the tables forbid to the first legal value and names what it moved,
// and an empty draft is the far end of that same walk.
func (s *Server) resolveForm(m *screensharev1.Settings) *screensharev1.Form {
	// Brokered ahead of the resolve, so the audience diagnostic reads Discord mode's membership.
	draft := s.backend.Brokered(wire.ToSettings(m))
	resolved := form.Resolve(s.formDeps(), draft)

	// Held against the stream as the resolve answered it, repair and followed preset included:
	// those settings are what a start would run, so they are the ones an apply could repeat.
	// The answer travels as a message and Discord mode's membership rides on none,
	// so the draft's unsent half goes back on before the comparison reads a path.
	resolved.InForce = s.inForce(wire.ToSettings(resolved.GetSettings()).WithUnsent(draft))
	return resolved
}

// inForce reports whether the stream in force was built from the pipeline draft builds,
// and false with none in force.
// publish.SamePipeline decides it, as it does for a repeated live Set and for PublishState.Live.pending.
// Settings no engine renders name no running pipeline, so an unanswerable comparison is false.
func (s *Server) inForce(draft settings.Settings) bool {
	state := s.backend.PublishState()
	if !state.Publishing() {
		return false
	}

	same, err := publish.SamePipeline(state.Live.Settings, draft)
	return err == nil && same
}

// formDeps is what a resolve reads off this machine, one builder for every caller:
// the form and the live Set that resolves a followed preset (effects.go)
// answer from the same machine.
func (s *Server) formDeps() form.Deps {
	return form.Deps{
		Monitors:        s.backend.Monitors(),
		Windows:         s.backend.Windows(),
		Platform:        s.backend.Platform(),
		Device:          s.backend.Device(),
		Encoders:        s.backend.CachedEncoders(),
		AudioDevices:    s.backend.AudioDevices(),
		Portal:          s.backend.PortalCapabilities(),
		DiscordRefused:  s.backend.DiscordState().Refused,
		UpdateUnchecked: s.backend.UpdateState().Unchecked,
		TrayOff:         trayForcedOff(),
	}
}

// presets answers with the user's saved configurations.
//
// A store that could not be read is an Umgebungsfehler.
// The list comes back empty carrying the notice,
// because nothing-readable-remained and nothing-was-saved are different facts about the machine,
// and the difference is the user's to see:
// the unreadable file has been moved aside by then, and the notice names where the presets went.
// A status error would replace both facts with "the call failed".
//
// The store is reached directly rather than through Backend.
// Presets are a file, held by nothing in the running app and announced by no event,
// so a Backend method would forward and nothing else.
func (s *Server) presets() *screensharev1.PresetList {
	presets, err := settings.LoadPresets()

	var notice *screensharev1.Text
	if err != nil {
		logger.Warnf("control: presets not restored: %v", err)
		notice = settings.StoreNotice(
			screensharev1.TextCode_TEXT_CODE_PRESET_STORE_UNREADABLE, err)
	}

	return &screensharev1.PresetList{Presets: wire.Presets(presets), Notice: notice}
}

// publishState says whether a stream is in force, and whether the held settings have moved off it.
func (s *Server) publishState() *screensharev1.PublishState {
	return wire.PublishState(s.backend.PublishState())
}

// relayStatus answers with the latest relay snapshot, and always succeeds.
//
// An unreachable relay is a snapshot whose reachable is false, carrying the reason
// (errors.go, and docs/ipc-api.md, "Errors").
// "The relay is down" is a thing the screen has to say,
// and a call that failed would leave the shell nothing to say it with.
//
// The backend owns the polling,
// so several shells reading do not multiply the requests to the relay,
// and the byte-delta bitrates stay computed against one steady interval
// rather than whatever cadence each shell chose.
func (s *Server) relayStatus() *screensharev1.RelayStatus {
	return wire.RelayStatus(s.backend.RelayStatus())
}

// viewerState answers with the open external viewers,
// one entry per stream and transport it is received over.
func (s *Server) viewerState() *screensharev1.ViewerState {
	return wire.ViewerState(s.backend.Watching())
}

// receiveState answers with the streams being decoded,
// one entry per stream and leg it is received over.
func (s *Server) receiveState() *screensharev1.ReceiveState {
	return wire.ReceiveState(s.backend.ReceiveState())
}

// monitorPreviewState answers with the monitors being read,
// what a shell that has just connected converges against.
// A preview outlives the window that asked for it,
// so a shell that crashed leaves screens captured for nobody.
func (s *Server) monitorPreviewState() *screensharev1.MonitorPreviewState {
	return wire.MonitorPreviewState(s.backend.MonitorPreviewState())
}

// shareWindows answers with the windows a capture can read, as they stand.
//
// Read alone and never watched: somebody opens and closes windows while nobody is looking,
// so it is asked for when a surface is about to draw it.
// A session that enumerates none answers empty,
// and Catalog.no_window_enumeration says so beforehand.
func (s *Server) shareWindows() *screensharev1.ShareWindowList {
	return &screensharev1.ShareWindowList{Windows: wire.ShareWindows(s.backend.Windows())}
}

// testStreamState counts the synthetic publishers alive:
// one that died on its own drops out.
// The slots travel beside the count, so a set with one dead publisher names the slot.
func (s *Server) testStreamState() *screensharev1.TestStreamState {
	running, slots := s.backend.TestStreamState()
	return wire.TestStreamState(running, slots...)
}

// membersState answers with who this machine shares a group with,
// as the presence loop last read it.
//
// A read of a reading, stating no presence of its own:
// the loop polling the relay is the heartbeat,
// and a shell asking would be a second thing deciding when this machine is in its group.
func (s *Server) membersState() *screensharev1.MembersState {
	return wire.MembersState(s.backend.MembersState())
}

// discordState answers Discord mode as the last manager pass landed it.
func (s *Server) discordState() *screensharev1.DiscordState {
	return wire.DiscordState(s.backend.DiscordState())
}

// resolveLink answers the stream a link opens.
//
// The refusals are the world's rather than the caller's:
// a link into another voice channel is well formed and names a group this machine is not in,
// so it is a precondition rather than a bad argument.
// An unreadable link is the argument, and it is the one case this side can tell apart
// without asking the backend (internal/applink).
func (s *Server) resolveLink(url string) (string, error) {
	if url == "" {
		return "", invalidArgument("a link names a stream to open")
	}

	stream, err := s.backend.ResolveLink(url)
	if err != nil {
		return "", failedPrecondition("%s", err.Error())
	}
	return stream, nil
}

// updateState answers what this install knows about the release published beside it.
//
// Reaches nothing: Do's check_update is what asks the release service,
// and every step of it arrives on Watch.
func (s *Server) updateState() *screensharev1.UpdateState {
	return wire.UpdateState(s.backend.UpdateState())
}

// streamTimeline answers the publish run as the backend judged it.
func (s *Server) streamTimeline() *screensharev1.StreamTimeline {
	return wire.StreamTimeline(s.backend.StreamTimeline())
}
