// Package timeline judges one publish run the way a viewer would:
// whether it holds its frame rate, whether it is restarting, whether it stopped,
// and who came and went while it ran.
// The app feeds it what it already observes, and the shell draws the snapshot.
package timeline

import (
	"slices"
	"sync"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"
)

// Event marks one moment of the run.
type Event struct {
	Kind screensharev1.StreamEvent_Kind
	At   time.Time
	// Who joined or left.
	Who string
	// On a dip, the lowest rate reached and the target, fps.
	Fps, TargetFps float64
}

// Snapshot holds the run as judged when it was taken. Events copied.
type Snapshot struct {
	Level     screensharev1.StreamVerdict_Level
	Cause     *screensharev1.Text
	Message   string
	LogPath   string
	Events    []Event
	StartedAt time.Time
}

// Timeline holds one run's judgment. Safe for concurrent use.
type Timeline struct {
	mu  sync.Mutex
	run run
}

// One run's state, replaced whole by Begin and End.
type run struct {
	running   bool
	startedAt time.Time
	targetFps float64
	events    []Event

	slow    int
	dipping bool
	// Lowest rate of the open dip, fps. Kept apart from its event, which the cap may drop.
	dipLow float64

	restarting   bool
	restartCause *screensharev1.Text

	stopped     bool
	stopCause   *screensharev1.Text
	stopMessage string
	stopLog     string

	members     map[string]bool
	membersRead bool
}

func New() *Timeline {
	return &Timeline{}
}

// Begin starts judging a new run at targetFps, dropping everything the last run left.
func (tl *Timeline) Begin(at time.Time, targetFps float64) {
	assert.Assert(targetFps > 0, "a run is judged against a frame rate it was asked for", targetFps)

	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.run = run{running: true, startedAt: at, targetFps: targetFps}
}

// Sample takes one frame rate reading from the encoder, true where the judgment moved.
// A reading outside a run is ignored.
func (tl *Timeline) Sample(at time.Time, fps float64) bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.run.running || tl.run.stopped {
		return false
	}
	// Frames flowing again end a restart.
	moved := tl.run.restarting
	tl.run.restarting = false
	tl.run.restartCause = nil

	if fps >= RECOVER_SHARE*tl.run.targetFps {
		moved = moved || tl.run.dipping
		tl.run.slow = 0
		tl.run.dipping = false
		return moved
	}
	if fps >= DIP_SHARE*tl.run.targetFps {
		tl.run.slow = 0
		return moved
	}

	tl.run.slow++
	if tl.run.dipping {
		if fps >= tl.run.dipLow {
			return moved
		}
		tl.run.dipLow = fps
		if i := tl.openDipLocked(); i >= 0 {
			tl.run.events[i].Fps = fps
		}
		return true
	}
	if tl.run.slow < DIP_SAMPLES {
		return moved
	}
	tl.run.dipping = true
	tl.run.dipLow = fps
	tl.appendLocked(Event{Kind: screensharev1.StreamEvent_KIND_DIP, At: at, Fps: fps, TargetFps: tl.run.targetFps})
	return true
}

// Restart marks the pipeline ending with a relaunch scheduled. cause may be nil.
func (tl *Timeline) Restart(at time.Time, cause *screensharev1.Text) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.run.running {
		return
	}
	tl.run.restarting = true
	tl.run.restartCause = cause
	tl.run.slow = 0
	tl.run.dipping = false
	tl.appendLocked(Event{Kind: screensharev1.StreamEvent_KIND_RESTART, At: at})
}

// Stop marks the run ended on a failure no retry recovers.
func (tl *Timeline) Stop(at time.Time, cause *screensharev1.Text, message, logPath string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.run.running {
		return
	}
	tl.run.stopped = true
	tl.run.stopCause = cause
	tl.run.stopMessage = message
	tl.run.stopLog = logPath
}

// End closes the run without a verdict against it: a stop the user asked for.
func (tl *Timeline) End() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.run = run{}
}

// Members takes the group's display names, this machine left out, true where anyone came or went.
// The run's first reading sets the baseline; each later one marks who arrived and who left.
func (tl *Timeline) Members(at time.Time, names []string) bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.run.running {
		return false
	}
	marked := false
	now := make(map[string]bool, len(names))
	for _, name := range names {
		now[name] = true
	}
	if tl.run.membersRead {
		for _, name := range sortedKeys(now) {
			if !tl.run.members[name] {
				tl.appendLocked(Event{Kind: screensharev1.StreamEvent_KIND_JOIN, At: at, Who: name})
				marked = true
			}
		}
		for _, name := range sortedKeys(tl.run.members) {
			if !now[name] {
				tl.appendLocked(Event{Kind: screensharev1.StreamEvent_KIND_LEAVE, At: at, Who: name})
				marked = true
			}
		}
	}
	tl.run.members = now
	tl.run.membersRead = true
	return marked
}

// Snapshot judges the run as it stands.
func (tl *Timeline) Snapshot() Snapshot {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	out := Snapshot{
		Level:     screensharev1.StreamVerdict_LEVEL_CLEAN,
		Events:    slices.Clone(tl.run.events),
		StartedAt: tl.run.startedAt,
	}
	switch {
	case tl.run.stopped:
		out.Level = screensharev1.StreamVerdict_LEVEL_STOPPED
		out.Cause = tl.run.stopCause
		out.Message = tl.run.stopMessage
		out.LogPath = tl.run.stopLog
	case tl.run.restarting:
		out.Level = screensharev1.StreamVerdict_LEVEL_DISAPPOINTS
		out.Cause = tl.run.restartCause
	case tl.run.dipping:
		out.Level = screensharev1.StreamVerdict_LEVEL_DISAPPOINTS
		out.Cause = text.Of(screensharev1.TextCode_TEXT_CODE_VERDICT_DIP,
			text.Dec(screensharev1.TextArgName_TEXT_ARG_NAME_FPS, tl.run.dipLow),
			text.Dec(screensharev1.TextArgName_TEXT_ARG_NAME_TARGET_FPS, tl.run.targetFps))
	}

	assert.Assert(out.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || out.Cause == nil, "a clean run names no cause")
	assert.Assert(len(out.Events) <= MAX_EVENTS, "the run keeps at most the capped events", len(out.Events))
	return out
}

// appendLocked adds e, dropping the oldest past the cap. mu is held by the caller.
func (tl *Timeline) appendLocked(e Event) {
	tl.run.events = append(tl.run.events, e)
	if len(tl.run.events) > MAX_EVENTS {
		tl.run.events = slices.Delete(tl.run.events, 0, len(tl.run.events)-MAX_EVENTS)
	}
}

// openDipLocked finds the open dip's event, -1 once the cap dropped it. mu is held by the caller.
func (tl *Timeline) openDipLocked() int {
	for i := len(tl.run.events) - 1; i >= 0; i-- {
		if tl.run.events[i].Kind == screensharev1.StreamEvent_KIND_DIP {
			return i
		}
	}
	return -1
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
