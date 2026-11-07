package control

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/receive"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// levelCadence is how often a watch naming EVENT_KIND_AUDIO_LEVELS sends.
//
// The interval the level elements post at, so one tick carries one measurement:
// a faster cadence would send a figure twice,
// a slower one would step over figures that were taken.
// One constant, read from the package that measures (docs/viewer-architecture.md).
const levelCadence = receive.LevelInterval

// levelsEvent is how loud every decode is at this instant, as one tick of a watch.
// A figure is dBFS: at most zero, silence negative infinity.
//
// Each tick is the whole set, read through to the backend.
// A shell that joined late, missed a tick or fell behind is correct again on the next one.
//
// It sends whether or not anything is decoding.
// An empty tick states that nothing is carrying audio, so a shell can take a meter away.
// Going silent instead would be indistinguishable from a backend that stopped answering.
func (s *Server) levelsEvent() *mirrormev1.Event {
	assert.Assert(levelCadence > 0, "levels tick at a positive interval", levelCadence)

	return &mirrormev1.Event{
		Payload: &mirrormev1.Event_AudioLevels{AudioLevels: wire.AudioLevels(s.backend.AudioLevels())},
	}
}
