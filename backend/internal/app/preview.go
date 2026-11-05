package app

import (
	"sync"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/go-utils/util/logger"

	"bjoernblessin.de/screenshare/internal/decode"
	"bjoernblessin.de/screenshare/internal/publish"
	"bjoernblessin.de/screenshare/internal/receive"
	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/wire"
)

// The local preview: this machine decoding the stream it sends, the relay no party to it
// (docs/viewer-architecture.md, "What the insights preview draws").
//
// It belongs to the publish and to nothing else.
// The child serves the copy on a loopback port, so the port is allocated as part of launching it,
// and the decode reading that port goes down with it.
// No effect on the contract opens one, and none could name it:
// PublishState.live is singular,
// so "the preview" is a complete identity while one exists and no identity otherwise.
//
// A field of its own rather than a receiver beside the relay decodes,
// which are keyed by StreamRef, a stream and the protocol it crossed the relay over.
// This stream crossed no relay,
// so an entry there would need a transport name for a leg no transport carries,
// and every consumer of that table would read a protocol that does not exist.

// previewLeg is what the log calls the carriage between the child and this machine's decode.
// No transport entry carries it.
const previewLeg = "loopback Matroska"

// previewAttachBudget is how many decodes one preview opens before it gives up on the child.
// The first report can beat the listener, which costs one.
const previewAttachBudget = 5

// previewRun is the local preview in force:
// the port the child serves the copy on, and the decode reading it once the child answers.
//
// Replaced whole rather than mutated, as publishRun is:
// a superseded pipeline's callbacks still fire,
// and one reporting against the preview that replaced it would say the running one had ended.
type previewRun struct {
	port   int
	stream receive.Stream
	open   receive.Open
	// receiver is nil until the child listens, and again after a decode that drew nothing.
	receiver *decode.Handle
	// attempts counts the decodes opened, up to previewAttachBudget.
	attempts int
	// settled closes on the first frame or when the preview goes, whichever comes first.
	settled chan struct{}
	settle  sync.Once
}

func newPreviewRun(port int, stream receive.Stream, open receive.Open) *previewRun {
	return &previewRun{port: port, stream: stream, open: open, settled: make(chan struct{})}
}

func (r *previewRun) markSettled() {
	r.settle.Do(func() { close(r.settled) })
}

func (r *previewRun) drewFrame() bool {
	select {
	case <-r.settled:
		return true
	default:
		return false
	}
}

// startPreviewLocked names the preview for a launching publish,
// and returns the leg the child is told to serve.
// procMu is held by the caller.
//
// Idempotent, guarded by a read of what is running rather than by a flag:
// a preview already named is the state asked for,
// and a second port would go to a child told about one.
//
// Every failure here is an Umgebungsfehler costing the preview and not the stream:
// the publish goes ahead with no second output and the state reports no preview,
// which the insights screen says rather than drawing a picture that would never arrive.
func (a *App) startPreviewLocked(s settings.Settings) publish.PreviewLeg {
	if a.preview != nil {
		return publish.PreviewLeg{Port: a.preview.port}
	}

	format, carried := publish.PreviewCarried(s.Publish.Codec())
	if !carried {
		logger.Warnf("no local preview for '%s': %s produces %s, which has no local carriage",
			s.Publish.Name(), s.Publish.Codec(), format)
		return publish.PreviewLeg{}
	}

	port, err := publish.AllocatePreviewPort()
	if err != nil {
		logger.Warnf("no local preview for '%s': %v", s.Publish.Name(), err)
		return publish.PreviewLeg{}
	}

	a.preview = newPreviewRun(port, receive.Stream{
		Name:      s.Publish.Name(),
		Transport: previewLeg,
		Source:    publish.PreviewSource(port),
	}, receive.Open{Chain: s.Viewer.RenderChain})
	logger.Infof("previewing '%s' locally on 127.0.0.1:%d", s.Publish.Name(), port)
	return publish.PreviewLeg{Port: port}
}

// attachPreview opens the decode of the copy that the child serves, and does nothing where
// no preview waits for one.
// A report from the child calls it, because a child that reports frames has opened its outputs.
// Can be called concurrently.
func (a *App) attachPreview() {
	a.procMu.Lock()
	run := a.preview
	if run == nil || run.receiver != nil {
		a.procMu.Unlock()
		return
	}
	if run.attempts >= previewAttachBudget {
		a.dropPreviewLocked(run)
		a.procMu.Unlock()
		logger.Warnf("the local preview on 127.0.0.1:%d never drew a frame", run.port)
		a.emitPublishState()
		return
	}
	run.attempts++

	receiver, err := a.decodes.Open(decode.PreviewID(), run.stream, run.open, decode.Events{
		// The first frame settles what the state reports: the chain that ran,
		// the memory the pads negotiated.
		// It rides the publish state, the preview being part of the publish.
		OnLive: func() {
			run.markSettled()
			a.emitPublishState()
		},
		OnEnd: func(message string) {
			a.previewEnded(run, message)
		},
	})
	if err != nil {
		a.procMu.Unlock()
		logger.Warnf("no local preview for '%s': %v", run.stream.Name, err)
		return
	}
	run.receiver = receiver
	a.procMu.Unlock()
}

// stopPreviewLocked takes the preview down, and succeeds where none is running:
// the stop names a state that already holds, as StopReceive does on a closed decode.
// procMu is held by the caller.
//
// The field is cleared before the receiver is stopped,
// so an exit reported during the teardown finds no preview to end.
func (a *App) stopPreviewLocked() {
	run := a.preview
	if run == nil {
		return
	}
	a.dropPreviewLocked(run)

	if run.receiver != nil {
		run.receiver.Stop()
	}
	logger.Infof("stopped previewing on 127.0.0.1:%d", run.port)
}

// dropPreviewLocked takes run out of force and releases a subscription waiting on it.
// procMu is held by the caller.
func (a *App) dropPreviewLocked(run *previewRun) {
	assert.Assert(a.preview == run, "a dropped preview is the one in force", run.port)

	a.preview = nil
	run.markSettled()
}

// previewEnded handles a preview pipeline that ended on its own.
//
// A decode that drew nothing reached the child too early, so the next report opens another.
// A decode that drew frames ended for good and the card says so:
// the publish state carries no preview and the stream is untouched, the leg being a copy.
func (a *App) previewEnded(run *previewRun, message string) {
	assert.IsNotNil(run, "an exit belongs to the preview that produced it")

	a.procMu.Lock()
	if a.preview != run {
		// The exit names a preview the app has already moved off:
		// a stop was asked for, or the publish behind it was replaced.
		a.procMu.Unlock()
		return
	}
	// The host keeps a pipeline that ended, carrying the reason, until it is stopped.
	a.decodes.Stop(decode.PreviewID())
	run.receiver = nil
	if !run.drewFrame() {
		a.procMu.Unlock()
		logger.Debugf("the local preview on 127.0.0.1:%d drew nothing: %s", run.port, message)
		return
	}
	a.dropPreviewLocked(run)
	a.procMu.Unlock()

	logger.Warnf("the local preview on 127.0.0.1:%d ended: %s", run.port, message)
	a.emitPublishState()
}

// previewSnapshotLocked is what the running preview turned out to be, read off the pipeline,
// and nil while none runs.
// procMu is held by the caller.
//
// Nothing is cached, the rule ReceiveState follows:
// a chain falls back at build time and the memory features settle when the pads negotiate,
// so a state assembled from what a caller believed it started would report the chain asked for,
// not the one that ran.
func (a *App) previewSnapshotLocked() *wire.PreviewSnapshot {
	if a.preview == nil {
		return nil
	}
	if a.preview.receiver == nil {
		return &wire.PreviewSnapshot{Port: a.preview.port}
	}

	stats := a.decodes.Snapshot()[decode.PreviewID()].Stats
	return &wire.PreviewSnapshot{
		Port:         a.preview.port,
		Live:         stats.Frames > 0,
		Chain:        stats.Chain,
		DecodeMemory: stats.DecodeMemory,
		RenderMemory: stats.RenderMemory,
		Decoder:      stats.Decoder,
		Hardware:     stats.Hardware,
	}
}

// SubscribePreviewFrames opens one consumer's view of the local preview's frames.
//
// It opens no pipeline, as SubscribeFrames opens no decode:
// the publish brings the preview up,
// and a subscription starting one would be the frame channel deciding a stream should be published.
// Nothing publishing is a refusal rather than a wait,
// and the publish state is what a shell reads to know whether to ask.
//
// A preview still waiting on its child holds the call until the first frame or until the preview
// goes, since the shell asks once for a preview that the state reports.
func (a *App) SubscribePreviewFrames() (decode.Subscription, error) {
	a.procMu.Lock()
	run := a.preview
	a.procMu.Unlock()

	if run == nil {
		return nil, text.Fail(screensharev1.TextCode_TEXT_CODE_PREVIEW_NOT_RUNNING)
	}
	<-run.settled

	a.procMu.Lock()
	receiver := run.receiver
	current := a.preview == run
	a.procMu.Unlock()

	if !current || receiver == nil {
		return nil, text.Fail(screensharev1.TextCode_TEXT_CODE_PREVIEW_NOT_RUNNING)
	}
	return receiver.Subscribe()
}
