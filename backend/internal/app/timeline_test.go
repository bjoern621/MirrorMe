package app

import (
	"errors"
	"testing"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
)

// The publish paths feed the timeline what they already decide, so its verdict follows the run.

func TestAnExitWithARetryJudgesTheRunRestarting(t *testing.T) {
	a, run := deadRun(t, 0)
	a.timeline.Begin(time.Now(), 60)

	a.publishEnded(run, errors.New("the relay refused the publisher"), "", "")

	got := a.StreamTimeline()
	if got.Level != screensharev1.StreamVerdict_LEVEL_DISAPPOINTS {
		t.Fatalf("a scheduled relaunch disappoints, got %+v", got)
	}
}

func TestAnExitNoRetryRecoversJudgesTheRunStopped(t *testing.T) {
	a, run := deadRun(t, len(publishBackoff))
	a.timeline.Begin(time.Now(), 60)

	a.publishEnded(run, errors.New("the relay refused the publisher"), "", "/tmp/publish.log")

	got := a.StreamTimeline()
	if got.Level != screensharev1.StreamVerdict_LEVEL_STOPPED || got.LogPath != "/tmp/publish.log" {
		t.Fatalf("a spent budget stops the run with its log, got %+v", got)
	}
}

func TestStoppingThePublishClearsTheJudgment(t *testing.T) {
	a, _ := deadRun(t, 0)
	a.timeline.Begin(time.Now(), 60)
	a.timeline.Restart(nil)

	a.StopPublish()

	if got := a.StreamTimeline(); got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || !got.StartedAt.IsZero() {
		t.Fatalf("a stop the user asked for leaves no run to judge, got %+v", got)
	}
}
