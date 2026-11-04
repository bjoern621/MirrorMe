package app

import (
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
