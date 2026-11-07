package control

import (
	"testing"
	"time"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
	"bjoernblessin.de/mirrorme/internal/timeline"
)

// The read answers the run as the backend judged it, the payload the event pushes.
func TestReadingTheStreamTimelineAnswersTheJudgedRun(t *testing.T) {
	started := time.UnixMilli(1_800_000_000_000)
	backend := &fakeBackend{timeline: timeline.Snapshot{
		Level:     mirrormev1.StreamVerdict_LEVEL_DISAPPOINTS,
		StartedAt: started,
	}}
	server := New(backend, events.New(), "test")

	got := readState(t, server, mirrormev1.EventKind_EVENT_KIND_STREAM_TIMELINE).GetStreamTimeline()
	if got.GetVerdict().GetLevel() != mirrormev1.StreamVerdict_LEVEL_DISAPPOINTS {
		t.Errorf("verdict reads %v, want the judged level", got.GetVerdict().GetLevel())
	}
	if got.GetStartedAtUnixMillis() != started.UnixMilli() {
		t.Errorf("run reads %v, want its start", got)
	}
}
