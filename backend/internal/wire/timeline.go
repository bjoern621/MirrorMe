package wire

import (
	"time"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
	"bjoernblessin.de/mirrorme/internal/timeline"
)

// StreamTimeline puts the judged run in the contract's shape, for the read and the event alike.
func StreamTimeline(s timeline.Snapshot) *mirrormev1.StreamTimeline {
	return &mirrormev1.StreamTimeline{
		Verdict: &mirrormev1.StreamVerdict{
			Level:   s.Level,
			Cause:   s.Cause,
			Message: s.Message,
			LogPath: s.LogPath,
		},
		StartedAtUnixMillis: unixMillisOrZero(s.StartedAt),
	}
}

// StreamTimelineEvent announces the judged run on every change.
func StreamTimelineEvent(s timeline.Snapshot) *mirrormev1.Event {
	return &mirrormev1.Event{
		Payload: &mirrormev1.Event_StreamTimeline{StreamTimeline: StreamTimeline(s)},
	}
}

// Zero for a run that never started, where time.Time{}.UnixMilli would be far negative.
func unixMillisOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
