package app

import (
	"testing"
	"time"

	"bjoernblessin.de/screenshare/internal/events"
	"bjoernblessin.de/screenshare/internal/receive"
	"bjoernblessin.de/screenshare/internal/settings"
)

// What a test can hold the preview to, now that the pipeline itself runs in the decode host
// (internal/decode).
// Bringing one up takes a host child, and a host child is this executable spawned again, which under
// a test binary is that binary running its own tests.
// So what is asserted here is the part that answers before anything is opened: the guard against
// a second launch, and a stop that names a state already holding.

// previewApp needs no relay, no child and no control socket.
// The decode client is left nil, which every path below returns before reaching.
func previewApp() *App {
	return &App{events: events.New(), settings: settings.Defaults()}
}

// previewSettings names a codec with a local carriage; without one there is no preview to bring up.
func previewSettings() settings.Settings {
	s := settings.Defaults()
	s.Publish.UseCodec("libx264")
	return s
}

// A launch brings the preview up, and two launches can race: a retry against a manual start.
// Without the guard the second binds a second port for a child that is told about one.
func TestBringingTheLocalPreviewUpTwiceChangesNothing(t *testing.T) {
	a := previewApp()
	s := previewSettings()

	a.procMu.Lock()
	defer a.procMu.Unlock()

	// The state a second launch meets, written rather than opened:
	// what the guard answers from is the field, not the pipeline behind it.
	running := newPreviewRun(5004, receive.Stream{}, receive.Open{})
	a.preview = running

	second := a.startPreviewLocked(s)
	if second.Port != running.port {
		t.Errorf("a second start moved the port from %d to %d", running.port, second.Port)
	}
	if a.preview != running {
		t.Error("a second start replaced the pipeline the first one built")
	}
}

// A stop on a preview that is not running names a state that already holds,
// the contract StopReceive keeps.
func TestStoppingALocalPreviewThatIsNotRunning(t *testing.T) {
	a := previewApp()

	a.procMu.Lock()
	defer a.procMu.Unlock()

	a.stopPreviewLocked()
	if a.preview != nil {
		t.Error("stopping when nothing ran left a preview behind")
	}
	a.stopPreviewLocked()
}

// The child has to listen before a decode can connect, so a launching publish names the port
// and opens nothing.
// The state still reports the preview, which is what tells the shell to subscribe.
func TestALaunchNamesThePreviewAndOpensNoDecode(t *testing.T) {
	a := previewApp()
	a.procMu.Lock()
	defer a.procMu.Unlock()

	leg := a.startPreviewLocked(previewSettings())
	if !leg.Wanted() {
		t.Fatal("a launch asked for no preview leg")
	}
	if a.preview.receiver != nil {
		t.Error("a launch opened a decode ahead of the child")
	}
	snapshot := a.previewSnapshotLocked()
	if snapshot == nil || snapshot.Port != leg.Port || snapshot.Live {
		t.Errorf("the state does not report a preview waiting on its child: %+v", snapshot)
	}
}

// A subscription arriving ahead of the first frame waits, and a preview that goes releases it.
func TestASubscriptionWaitsForThePreviewAndEndsWithIt(t *testing.T) {
	a := previewApp()
	a.procMu.Lock()
	a.startPreviewLocked(previewSettings())
	a.procMu.Unlock()

	done := make(chan error)
	go func() {
		_, err := a.SubscribePreviewFrames()
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("a subscription answered ahead of the first frame: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	a.procMu.Lock()
	a.stopPreviewLocked()
	a.procMu.Unlock()

	select {
	case err := <-done:
		if err == nil {
			t.Error("a subscription to a stopped preview was served")
		}
	case <-time.After(time.Second):
		t.Fatal("a stopped preview left a subscription waiting")
	}
}

// A child that never answers costs the preview after a bounded number of decodes.
// The nil decode client refuses each open, what a host that cannot start does.
func TestAPreviewGivesUpOnAChildThatNeverAnswers(t *testing.T) {
	a := previewApp()
	a.procMu.Lock()
	a.startPreviewLocked(previewSettings())
	a.procMu.Unlock()

	for range previewAttachBudget {
		a.attachPreview()
		if a.preview == nil {
			t.Fatal("the preview gave up ahead of its budget")
		}
	}
	a.attachPreview()
	if a.preview != nil {
		t.Error("the preview outlived its budget")
	}

	a.attachPreview()
}
