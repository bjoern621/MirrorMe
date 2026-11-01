package wire

import (
	"time"

	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/timeline"
)

// StreamTimeline puts the judged run in the contract's shape, for the read and the event alike.
func StreamTimeline(s timeline.Snapshot) *screensharev1.StreamTimeline {
	events := make([]*screensharev1.StreamEvent, 0, len(s.Events))
	for _, e := range s.Events {
		events = append(events, &screensharev1.StreamEvent{
			Kind:         e.Kind,
			AtUnixMillis: e.At.UnixMilli(),
			Who:          e.Who,
			Fps:          e.Fps,
			TargetFps:    e.TargetFps,
		})
	}

	out := &screensharev1.StreamTimeline{
		Verdict: &screensharev1.StreamVerdict{
			Level:   s.Level,
			Cause:   s.Cause,
			Message: s.Message,
			LogPath: s.LogPath,
		},
		Events:              events,
		StartedAtUnixMillis: unixMillisOrZero(s.StartedAt),
	}

	assert.Assert(len(out.Events) == len(s.Events), "an event on the wire per event of the run", len(out.Events), len(s.Events))
	return out
}

// StreamTimelineEvent announces the judged run on every change.
func StreamTimelineEvent(s timeline.Snapshot) *screensharev1.Event {
	return &screensharev1.Event{
		Payload: &screensharev1.Event_StreamTimeline{StreamTimeline: StreamTimeline(s)},
	}
}

// Zero for a run that never started, where time.Time{}.UnixMilli would be far negative.
func unixMillisOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
