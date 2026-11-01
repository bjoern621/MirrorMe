package app

import (
	"time"

	"bjoernblessin.de/screenshare/internal/timeline"
	"bjoernblessin.de/screenshare/internal/wire"
)

// StreamTimeline reports this machine's publish run as a viewer would judge it (internal/timeline).
func (a *App) StreamTimeline() timeline.Snapshot {
	return a.timeline.Snapshot()
}

func (a *App) emitTimeline() {
	a.emit(wire.StreamTimelineEvent(a.timeline.Snapshot()))
}

// judgeMembers hands the timeline who else is in the group. Announced on a join or a leave alone:
// the read lands every pass of the presence loop.
func (a *App) judgeMembers(m membership) {
	names := make([]string, 0, len(m.Members))
	for _, member := range m.Members {
		if !member.Self {
			names = append(names, member.DisplayName)
		}
	}
	if a.timeline.Members(time.Now(), names) {
		a.emitTimeline()
	}
}
