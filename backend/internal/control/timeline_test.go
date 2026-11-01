package control

import (
	"context"
	"testing"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/events"
	"bjoernblessin.de/screenshare/internal/timeline"
)

// The read answers the run as the backend judged it, the payload the event pushes.
func TestGetStreamTimelineAnswersTheJudgedRun(t *testing.T) {
	started := time.UnixMilli(1_800_000_000_000)
	backend := &fakeBackend{timeline: timeline.Snapshot{
		Level:     screensharev1.StreamVerdict_LEVEL_DISAPPOINTS,
		StartedAt: started,
		Events:    []timeline.Event{{Kind: screensharev1.StreamEvent_KIND_RESTART, At: started.Add(time.Second)}},
	}}
	server := New(backend, events.New(), "test")

	got, err := server.GetStreamTimeline(context.Background(), &screensharev1.GetStreamTimelineRequest{})
	if err != nil {
		t.Fatalf("reading the timeline answered %v, want the run", err)
	}
	if got.GetVerdict().GetLevel() != screensharev1.StreamVerdict_LEVEL_DISAPPOINTS {
		t.Errorf("verdict reads %v, want the judged level", got.GetVerdict().GetLevel())
	}
	if got.GetStartedAtUnixMillis() != started.UnixMilli() || len(got.GetEvents()) != 1 ||
		got.GetEvents()[0].GetAtUnixMillis() != started.Add(time.Second).UnixMilli() {
		t.Errorf("run reads %v, want its start and its one restart", got)
	}
}
