package control

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"google.golang.org/protobuf/proto"

	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/form"
	"bjoernblessin.de/screenshare/internal/publish"
	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/wire"
)

// The arms of Set and Do.
// A Set arm answers nothing: what the state became arrives on Watch (docs/ipc-api.md, "Events").
// One path into the display keeps the shell that pressed the button
// and the one that only listened from showing different things.
//
// Reading state back here for a richer answer is the same fault:
// a read after the effect is already one event out of date.

// fromBackend turns an untyped backend error into UNAVAILABLE.
//
// One helper rather than a judgement per call site.
// Backend errors carry no type, so coding them off the text would derive the error model
// from a sentence written for a person, the one input that changes without breaking the build.
//
// UNAVAILABLE is the residue.
// Every refusal the request itself earns is made above the backend call, against the arguments
// or the state: an empty name, a settings message carrying nothing, a negative count,
// a log file that is not there.
// An Entwicklungsfehler never arrives either, assert panicking in the backend.
// What is left is the world failing at something legal to ask for: a child that would not start,
// a store that could not be written, a relay that could not be reached.
//
// Cost: a settings combination no engine can build also comes back untyped from a live Set,
// where INVALID_ARGUMENT would fit better.
// Telling the two apart needs a typed refusal on Backend.
// A guess off the sentence, landing wrong, would tell a shell to blame the user for a full disk.
func fromBackend(what string, err error) error {
	assert.IsNotNil(err, "a classified failure is a failure", what)

	return unavailable("%s: %v", what, err)
}

// draftOf reads the settings off a request that needs them.
//
// An absent message and one carrying nothing are the same request:
// both INVALID_ARGUMENT, rather than a stream with no name, no relay and no codec.
// A shell that forgot to attach its draft would otherwise persist that emptiness over the settings
// and find out on the next launch.
//
// Emptiness is tested against the zero message rather than one chosen field,
// so a field added to the contract stays covered without an edit here.
func draftOf(m *screensharev1.Settings, verb string) (settings.Settings, error) {
	// Verb is this file's word for what the caller was doing, fixed at the call site,
	// so an empty one leaves a hole in the sentence a person reads.
	assert.Assert(verb != "", "a refused draft names what it was going to be used for")

	if m == nil || proto.Equal(m, &screensharev1.Settings{}) {
		return settings.Settings{}, invalidArgument("no settings to %s", verb)
	}
	return wire.ToSettings(m), nil
}

// presetOf reads one saved way of publishing off a request.
// An empty one is refused for the reason draftOf refuses an empty draft.
//
// A preset is a PublishSettings and nothing else: where the relay is belongs to a deployment,
// how this machine watches to a viewer, so neither is part of applying one.
func presetOf(m *screensharev1.PublishSettings, verb string) (settings.Publish, error) {
	assert.Assert(verb != "", "a refused preset names what it was going to be used for")

	if m == nil || proto.Equal(m, &screensharev1.PublishSettings{}) {
		return settings.Publish{}, invalidArgument("no settings to %s", verb)
	}
	return wire.ToPublish(m), nil
}

// keyOf reads the stream and leg a viewer, a decode and a page are identified by.
//
// Half a pair is INVALID_ARGUMENT for the reason StreamRef exists:
// one stream is watched over several legs at once, so half an identity names nothing that can exist.
// What names the effect in the refusal, fixed at the call site.
func keyOf(m *screensharev1.StreamRef, what string) (wire.StreamRef, error) {
	assert.Assert(what != "", "a refused stream names what it was going to be used for")

	ref := wire.StreamRefOf(m)
	if ref.StreamName == "" {
		return wire.StreamRef{}, invalidArgument("no stream named to %s", what)
	}
	if ref.Transport == "" {
		return wire.StreamRef{}, invalidArgument("no transport named to %s '%s' over", what, ref.StreamName)
	}
	return ref, nil
}

// saveSettings persists the settings the shell holds, and touches no running stream.
//
// Both engines run a child built from an argv and neither takes a value back afterwards,
// so reaching a live pipeline is setPublish's business.
// Two arms because the user's two intentions differ: keep this for next time,
// and put this on the air.
func (s *Server) saveSettings(m *screensharev1.Settings) error {
	draft, err := draftOf(m, "save")
	if err != nil {
		return err
	}

	if err := s.backend.SaveSettings(draft); err != nil {
		return fromBackend("cannot save the settings", err)
	}
	return nil
}

// setPreset stores the settings under a name, replacing a same-named preset,
// and drops the preset where the request carries no settings.
// A request with no name is INVALID_ARGUMENT.
//
// Dropping a name no preset carries succeeds: the state it names already holds.
// settings.DeletePreset rewrites the file without the name, so the store is read first
// and an absent name leaves the file untouched.
func (s *Server) setPreset(w *screensharev1.PresetWanted) error {
	name := w.GetName()
	if name == "" {
		return invalidArgument("a preset is named, and none was given")
	}

	if w.GetSettings() == nil {
		return s.dropPreset(name)
	}

	preset, err := presetOf(w.GetSettings(), "save as a preset")
	if err != nil {
		return err
	}

	if err := settings.SavePreset(name, preset); err != nil {
		return fromBackend("cannot save the preset '"+name+"'", err)
	}
	return nil
}

func (s *Server) dropPreset(name string) error {
	assert.Assert(name != "", "a dropped preset is named")

	presets, err := settings.LoadPresets()
	if err != nil {
		return fromBackend("cannot read the presets", err)
	}
	if !slices.ContainsFunc(presets, func(p settings.Preset) bool { return p.Name == name }) {
		return nil
	}

	if err := settings.DeletePreset(name); err != nil {
		return fromBackend("cannot delete the preset '"+name+"'", err)
	}
	return nil
}

// resolveFollowed turns a draft that follows a built-in preset into the configuration
// that preset produces on this machine at this moment, and leaves a detached draft alone.
//
// At the press rather than at the form:
// the machine may have changed since the form was drawn, and what was asked for is the promise,
// so the search runs against the machine as it stands.
// A preset nothing here reaches is FAILED_PRECONDITION,
// the verdict the form already blocks the start with,
// so the refusal repeats what the screen says rather than surprising it.
func (s *Server) resolveFollowed(draft settings.Settings) (settings.Settings, error) {
	if !form.Followed(draft) {
		return draft, nil
	}

	resolved, ok := form.ResolveBuiltin(s.formDeps(), draft)
	if !ok {
		return settings.Settings{}, failedPrecondition(
			"no configuration on this machine delivers the selected preset (%s). Open the stream settings to see what stands in the way",
			draft.Publish.Preset)
	}
	return resolved, nil
}

// setPublish makes a stream be on the air on the settings the request carries, or be off it.
//
// Not live ends the stream, running or waiting out a backoff, and refuses nothing:
// where nothing publishes, the state it names already holds.
//
// Live with nothing in force persists the settings and starts the encoder on them.
// Live with a stream in force on the same pipeline succeeds and does nothing.
// It is a request for a state that already holds (docs/development-principles.md,
// "Effects across a process boundary"): a shell whose answer went missing cannot tell "not done"
// from "done, answer lost", and asking again is the only move that resolves it.
// Live with a stream in force on another pipeline restarts it on the new settings,
// a pipeline waiting out a retry backoff included: the user asked for it and has not stopped it.
// publish.SamePipeline decides sameness here, in the form and at the backend,
// "these two settings are one stream" being one fact.
func (s *Server) setPublish(w *screensharev1.PublishWanted) error {
	if !w.GetLive() {
		s.backend.StopPublish()
		return nil
	}

	draft, err := draftOf(w.GetSettings(), "publish")
	if err != nil {
		return err
	}
	draft, err = s.resolveFollowed(draft)
	if err != nil {
		return err
	}

	state := s.backend.PublishState()
	if !state.Publishing() {
		if err := s.backend.StartPublish(draft); err != nil {
			return fromBackend("cannot start publishing", err)
		}
		return nil
	}

	// Brokered for the comparison alone: Discord mode's membership rides on no request,
	// so a draft holding none names a pipeline other than the one running,
	// and the repeat would restart the stream on itself (internal/app, withBrokered).
	// The draft the backend is handed stays the one that arrived,
	// a brokered copy carrying the Discord nick over the stored name.
	if same, err := publish.SamePipeline(state.Live.Settings, s.backend.Brokered(draft)); err == nil && same {
		return nil
	}

	if err := s.backend.ApplyToStream(draft); err != nil {
		return fromBackend("cannot apply the settings to the running stream", err)
	}
	return nil
}

// setViewer opens or closes an external viewer for one stream over one transport.
// A leg that cannot carry the stream's format is FAILED_PRECONDITION,
// and closing a viewer nothing has open succeeds.
//
// The transport is per viewer and independent of the publish leg,
// so one stream is watchable over any leg the relay serves it on.
func (s *Server) setViewer(w *screensharev1.ViewerWanted) error {
	if !w.GetOpen() {
		ref, err := keyOf(w.GetViewer(), "stop watching")
		if err != nil {
			return err
		}
		s.backend.StopWatch(ref)
		return nil
	}

	ref, err := keyOf(w.GetViewer(), "watch")
	if err != nil {
		return err
	}
	if err := s.backend.StartWatch(ref); err != nil {
		// Carriage refusal: the world not ready.
		// The relay re-serves a stream only on listeners whose protocol carries a payload mapping
		// for its bitstream, so an SRT viewer opened on a VP9 stream connects and receives nothing.
		// What stands in the way is the publish format at this instant.
		// Republished as H.264, the same request succeeds.
		// The backend's sentence names the format and the legs that do carry it,
		// so the reason reaches the user intact.
		//
		// A transport this build has no viewer for is INVALID_ARGUMENT under the contract's table,
		// and the backend separates the two by returning a Refused for it.
		if refused(err) {
			return invalidArgument("cannot watch '%s' over %s: %v", ref.StreamName, ref.Transport, err)
		}
		return failedPrecondition("cannot watch '%s' over %s: %v", ref.StreamName, ref.Transport, err)
	}
	return nil
}

// openInBrowser opens the relay's player page for one stream in the machine's default browser.
//
// The pair and its refusals are setViewer's, for the same reasons: the leg is per reader,
// and one the stream's format does not cross opens a page that connects and shows nothing.
//
// A second call opens a second page, the documented departure from idempotency
// (docs/development-principles.md, "Effects across a process boundary"):
// the effect lands in a program this process does not own, so there is no state to read back,
// no stop to write, and nothing here reaches the viewer state.
func (s *Server) openInBrowser(m *screensharev1.StreamRef) error {
	ref, err := keyOf(m, "open in the browser")
	if err != nil {
		return err
	}

	if err := s.backend.OpenInBrowser(ref); err != nil {
		return failedPrecondition("cannot open '%s' over %s in the browser: %v", ref.StreamName, ref.Transport, err)
	}
	return nil
}

// setReceive opens or closes a decode for one stream on one leg, inside the backend.
// A decode already open on the pair is the state an open names, and a pair nothing decodes
// the state a close names, so both succeed.
//
// The carriage refusal is setViewer's, for the same reason:
// a leg that cannot carry the stream's format is the world not being ready.
// What differs is the engine the carriage is asked about,
// a receive pipeline reaching WHEP where no player does, and the backend is what asks.
//
// Tone mapping is not checked here.
// Whether the stream carries more range than a display shows is known once the decoder negotiates
// and not before, and whether this machine can roll it down is the backend's registry,
// so a refusal here would guess at both.
func (s *Server) setReceive(w *screensharev1.ReceiveWanted) error {
	if !w.GetOpen() {
		ref, err := keyOf(w.GetStream(), "stop receiving")
		if err != nil {
			return err
		}
		s.backend.StopReceive(ref)
		return nil
	}

	ref, err := keyOf(w.GetStream(), "receive")
	if err != nil {
		return err
	}
	if err := s.backend.StartReceive(ref, w.GetToneMap()); err != nil {
		return failedPrecondition("cannot receive '%s' over %s: %v", ref.StreamName, ref.Transport, err)
	}
	return nil
}

// setMonitorPreview reads one of this machine's screens, so the wizard offers it by its picture
// rather than by its number, or stops reading it.
// A monitor already previewed, and one nothing previews, are the states the two answers name.
//
// The index is not checked here, unlike the pair the receive arms take.
// Every integer is a monitor index somewhere,
// and whether it is one of this machine's is a fact about the machine.
// The backend answers that, a Refused separating the index naming no output of this machine
// (INVALID_ARGUMENT) from a session that cannot read one screen apart from another
// (FAILED_PRECONDITION).
func (s *Server) setMonitorPreview(w *screensharev1.MonitorPreviewWanted) error {
	monitor := int(w.GetMonitor())
	if !w.GetOpen() {
		s.backend.StopMonitorPreview(monitor)
		return nil
	}

	if err := s.backend.StartMonitorPreview(monitor); err != nil {
		if refused(err) {
			return invalidArgument("cannot preview monitor %d: %v", monitor, err)
		}
		return failedPrecondition("cannot preview monitor %d: %v", monitor, err)
	}
	return nil
}

// setReceiveAudio sets how loud one decode plays and whether it plays at all.
//
// The pair is checked for the reason every receive arm checks it.
// The refusal differs: a decode that is not running is NOT_FOUND rather than a quiet success,
// being a request about something absent rather than for a state that already holds.
// A repeat of a loudness is that state, and succeeds.
//
// The volume is not range-checked here.
// The bound lives at the backend, which brings a figure past the end of the range back,
// and a refusal would turn a slider that overshot into an error a reader has to read.
func (s *Server) setReceiveAudio(a *screensharev1.ReceiveAudio) error {
	ref, err := keyOf(a.GetStream(), "set the audio of")
	if err != nil {
		return err
	}

	if err := s.backend.SetReceiveAudio(ref, a.GetVolume(), a.GetMuted()); err != nil {
		return notFound("cannot set the audio of '%s' over %s: %v", ref.StreamName, ref.Transport, err)
	}
	return nil
}

// probeEncoders test-encodes on every engine, records what this machine can really run,
// and announces the catalog that result changed.
//
// An arm of Do rather than a flag on the catalog read, for the announcement:
// every shell learns on Watch, the ones that never asked included,
// instead of one shell's read moving what a different shell's next resolve says.
//
// The catalog is announced unconditionally rather than only where the result moved.
// The probe is cached for the process lifetime, so a second call is answered from the cache
// and publishes a state its subscribers already hold.
// A duplicate whole-state event is harmless by construction, and comparing availabilities to avoid
// one would be a second definition of when two probe results are the same.
func (s *Server) probeEncoders(ctx context.Context) {
	s.backend.Encoders(ctx)
	s.events.Publish(wire.CatalogEvent(s.catalog()))
}

// setTestStreams runs count synthetic publishers, replacing a running set, and none at zero.
//
// A negative count is INVALID_ARGUMENT, naming a set of publishers that cannot exist.
// A count above MaxTestStreams is RESOURCE_EXHAUSTED: each test stream runs its own software
// encoder, so a large set saturates the machine and tests nothing further.
//
// Both refusals are made above the call, where every request-earned refusal in this file is made.
// Inferring the bound from the backend's error would put a viewer binary that could not be found
// under RESOURCE_EXHAUSTED as well, and a missing binary is not a saturated machine.
func (s *Server) setTestStreams(count int) error {
	if count < 0 {
		return invalidArgument("a test-stream count is at least zero, and %d was asked for", count)
	}
	if count == 0 {
		s.backend.StopTestStreams()
		return nil
	}

	max := s.backend.MaxTestStreams()
	assert.Assert(max > 0, "a backend that runs test streams runs at least one", max)
	if count > max {
		return resourceExhausted("this machine runs at most %d test streams at once, and %d were asked for", max, count)
	}

	if err := s.backend.StartTestStreams(count); err != nil {
		return fromBackend(fmt.Sprintf("cannot start %d test streams", count), err)
	}
	return nil
}

// forgetPortalConsent drops the stored screen-capture consent,
// so the next capture asks the compositor to pick again.
// How a share aimed at the wrong window or monitor is corrected.
func (s *Server) forgetPortalConsent() error {
	if err := s.backend.ForgetPortalConsent(); err != nil {
		return fromBackend("cannot forget the screen-capture consent", err)
	}
	return nil
}

// createGroup draws a group key at the relay named in the request and answers it.
//
// Nothing is stored: the group key goes back to the shell.
// The shell writes it to the group key field like a value the user typed,
// so the one write moving a machine between groups stays where every other settings write is.
// A relay with no group service and an unreachable one are both the caller's to fix,
// so each leaves as the backend's own sentence rather than as a group key drawn nowhere.
func (s *Server) createGroup(relay *screensharev1.RelaySettings) (*screensharev1.Group, error) {
	groupKey, groupID, err := s.backend.CreateGroup(wire.ToRelay(relay))
	if err != nil {
		return nil, fromBackend("cannot draw a group key", err)
	}
	return &screensharev1.Group{Key: groupKey, Id: groupID}, nil
}

// setDiscordLink links this install to a Discord account, or drops the link.
//
// Linking runs the consent flow and answers when the link landed or did not.
// The secret lands in the stored settings, so the answer carries nothing to show.
// An install already linked holds the state a link names, and runs no second flow.
// Unlinking drops the stored secret and reaches nothing.
func (s *Server) setDiscordLink(ctx context.Context, w *screensharev1.DiscordLinkWanted) error {
	if !w.GetLinked() {
		if err := s.backend.UnlinkDiscord(); err != nil {
			return fromBackend("cannot unlink Discord", err)
		}
		return nil
	}

	if s.backend.DiscordState().Linked {
		return nil
	}
	if err := s.backend.LinkDiscord(ctx, wire.ToRelay(w.GetRelay())); err != nil {
		return fromBackend("cannot link Discord", err)
	}
	return nil
}

// openLog opens one run log in the machine's default application.
//
// The path comes off an ExitInfo the backend handed out and a shell constructs none,
// so an empty path is INVALID_ARGUMENT, the exit having carried no log,
// and a path that is not there is NOT_FOUND.
// The second check is made here rather than left to the default handler, which cannot tell the two
// apart: run logs are rotated, so a log named by an older exit is a file that has since gone,
// and "no log file at this path" says so.
//
// A second call opens a second window, the departure openInBrowser documents.
func (s *Server) openLog(path string) error {
	if path == "" {
		return invalidArgument("no log file for this run")
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return notFound("no log file at %s. The run logs are rotated, so one named by an older exit may already be gone", path)
	}

	if err := s.backend.OpenLog(path); err != nil {
		return fromBackend("cannot open "+path, err)
	}
	return nil
}

// openLogsFolder opens the directory holding the run logs in the machine's file browser,
// and opens a second window on a second call, as openLog does.
func (s *Server) openLogsFolder() error {
	if err := s.backend.OpenLogsFolder(); err != nil {
		return fromBackend("cannot open the logs folder", err)
	}
	return nil
}

// sendReport delivers one report bundle to the group service beside the stored relay,
// and sends a second one on a second call, as openLog opens a second window.
func (s *Server) sendReport() (string, error) {
	id, err := s.backend.SendReport()
	if err != nil {
		// A well-formed request whose moment is wrong, on checkUpdate's terms:
		// a report goes to the deployment in use, and the settings may name none.
		return "", failedPrecondition("cannot send a report: %v", err)
	}
	return id, nil
}

// checkUpdate reads the published release, and fetches it where this install replaces its own files.
//
// Returns as soon as the work is under way, the download outliving the call.
// What it finds arrives on Watch, so the shell that asked and the shell that did not
// are told the same thing.
//
// Refused where the install asks the release service nothing at all,
// which the state says before a shell draws a control to ask with.
func (s *Server) checkUpdate() error {
	if err := s.backend.CheckUpdate(); err != nil {
		// A well-formed request whose moment is wrong: this install asks nothing,
		// which UpdateState.unchecked stated before any control was drawn.
		return failedPrecondition("cannot check for updates: %v", err)
	}
	return nil
}

// installUpdate starts the staged release and leaves the running app to close.
//
// It answers while the app is still up: the applier is a process of its own that waits for this
// one to exit before it replaces anything (internal/update).
// Refused where nothing is staged and verified.
func (s *Server) installUpdate() error {
	if err := s.backend.InstallUpdate(); err != nil {
		// FAILED_PRECONDITION for the reason checkUpdate answers with one:
		// nothing is staged, which the state says before the dialog offers a restart.
		return failedPrecondition("cannot install the staged release: %v", err)
	}
	return nil
}
