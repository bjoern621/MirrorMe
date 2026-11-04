package wire

import (
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/timeline"
)

// StreamTimeline puts the judged run in the contract's shape, for the read and the event alike.
func StreamTimeline(s timeline.Snapshot) *screensharev1.StreamTimeline {
	return &screensharev1.StreamTimeline{
		Verdict: &screensharev1.StreamVerdict{
			Level:   s.Level,
			Cause:   s.Cause,
			Message: s.Message,
			LogPath: s.LogPath,
		},
		StartedAtUnixMillis: unixMillisOrZero(s.StartedAt),
	}
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
