package control

import (
	"testing"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/events"
	"bjoernblessin.de/screenshare/internal/timeline"
)

// The read answers the run as the backend judged it, the payload the event pushes.
func TestReadingTheStreamTimelineAnswersTheJudgedRun(t *testing.T) {
	started := time.UnixMilli(1_800_000_000_000)
	backend := &fakeBackend{timeline: timeline.Snapshot{
		Level:     screensharev1.StreamVerdict_LEVEL_DISAPPOINTS,
		StartedAt: started,
	}}
	server := New(backend, events.New(), "test")

	got := readState(t, server, screensharev1.EventKind_EVENT_KIND_STREAM_TIMELINE).GetStreamTimeline()
	if got.GetVerdict().GetLevel() != screensharev1.StreamVerdict_LEVEL_DISAPPOINTS {
		t.Errorf("verdict reads %v, want the judged level", got.GetVerdict().GetLevel())
	}
	if got.GetStartedAtUnixMillis() != started.UnixMilli() {
		t.Errorf("run reads %v, want its start", got)
	}
}
