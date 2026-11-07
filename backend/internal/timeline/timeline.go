// Package timeline judges one publish run the way a viewer would:
// whether it holds its frame rate, whether it is restarting, whether it stopped.
// The app feeds it what it already observes, and the shell draws the snapshot.
package timeline

import (
	"sync"
	"time"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
	"bjoernblessin.de/mirrorme/internal/text"

	"bjoernblessin.de/go-utils/util/assert"
)

// Snapshot holds the run as judged when it was taken.
type Snapshot struct {
	Level     mirrormev1.StreamVerdict_Level
	Cause     *mirrormev1.Text
	Message   string
	LogPath   string
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

	slow    int
	dipping bool
	// Lowest rate of the open dip, fps.
	dipLow float64

	restarting   bool
	restartCause *mirrormev1.Text

	stopped     bool
	stopCause   *mirrormev1.Text
	stopMessage string
	stopLog     string
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
func (tl *Timeline) Sample(fps float64) bool {
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
		return true
	}
	if tl.run.slow < DIP_SAMPLES {
		return moved
	}
	tl.run.dipping = true
	tl.run.dipLow = fps
	return true
}

// Restart marks the pipeline ending with a relaunch scheduled. cause may be nil.
func (tl *Timeline) Restart(cause *mirrormev1.Text) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.run.running {
		return
	}
	tl.run.restarting = true
	tl.run.restartCause = cause
	tl.run.slow = 0
	tl.run.dipping = false
}

// Stop marks the run ended on a failure no retry recovers.
func (tl *Timeline) Stop(cause *mirrormev1.Text, message, logPath string) {
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

// Snapshot judges the run as it stands.
func (tl *Timeline) Snapshot() Snapshot {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	out := Snapshot{
		Level:     mirrormev1.StreamVerdict_LEVEL_CLEAN,
		StartedAt: tl.run.startedAt,
	}
	switch {
	case tl.run.stopped:
		out.Level = mirrormev1.StreamVerdict_LEVEL_STOPPED
		out.Cause = tl.run.stopCause
		out.Message = tl.run.stopMessage
		out.LogPath = tl.run.stopLog
	case tl.run.restarting:
		out.Level = mirrormev1.StreamVerdict_LEVEL_DISAPPOINTS
		out.Cause = tl.run.restartCause
	case tl.run.dipping:
		out.Level = mirrormev1.StreamVerdict_LEVEL_DISAPPOINTS
		out.Cause = text.Of(mirrormev1.TextCode_TEXT_CODE_VERDICT_DIP,
			text.Dec(mirrormev1.TextArgName_TEXT_ARG_NAME_FPS, tl.run.dipLow),
			text.Dec(mirrormev1.TextArgName_TEXT_ARG_NAME_TARGET_FPS, tl.run.targetFps))
	}

	assert.Assert(out.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN || out.Cause == nil, "a clean run names no cause")
	return out
}
