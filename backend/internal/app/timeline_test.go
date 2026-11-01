package app

import (
	"errors"
	"testing"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/wire"
)

// The publish paths feed the timeline what they already decide, so its verdict follows the run.

func TestAnExitWithARetryJudgesTheRunRestarting(t *testing.T) {
	a, run := deadRun(t, 0)
	a.timeline.Begin(time.Now(), 60)

	a.publishEnded(run, errors.New("the relay refused the publisher"), "", "")

	got := a.StreamTimeline()
	if got.Level != screensharev1.StreamVerdict_LEVEL_DISAPPOINTS || len(got.Events) != 1 ||
		got.Events[0].Kind != screensharev1.StreamEvent_KIND_RESTART {
		t.Fatalf("a scheduled relaunch disappoints and marks a restart, got %+v", got)
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
	a.timeline.Restart(time.Now(), nil)

	a.StopPublish()

	if got := a.StreamTimeline(); got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || !got.StartedAt.IsZero() {
		t.Fatalf("a stop the user asked for leaves no run to judge, got %+v", got)
	}
}

func TestAMembersReadingMarksWhoJoinedTheRun(t *testing.T) {
	a, _ := deadRun(t, 0)
	a.timeline.Begin(time.Now(), 60)
	self := wire.Member{MemberID: "me", DisplayName: "me", Self: true}

	a.judgeMembers(membership{Members: []wire.Member{self}})
	a.judgeMembers(membership{Members: []wire.Member{self, {MemberID: "b", DisplayName: "bo"}}})

	got := a.StreamTimeline().Events
	if len(got) != 1 || got[0].Kind != screensharev1.StreamEvent_KIND_JOIN || got[0].Who != "bo" {
		t.Fatalf("bo joining is the one mark, this machine never is, got %+v", got)
	}
}
