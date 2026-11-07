package control

import (
	"time"

	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/pointer"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// pointerCadence is how often this machine's own pointer is read for a watch naming EVENT_KIND_POINTER.
//
// The reader's own interval, so one tick is one reading:
// faster would send the same position twice, slower would step over positions that were taken.
// Not the frame rate: a position costs no frame,
// so a 240 Hz pointer over a 30 fps stream is the win over drawing it into the picture.
const pointerCadence = pointer.Interval

// streamPointerCadence is how often a watched stream's pointer is read.
//
// Slower than the capture's own, because the reading is: the position rides in the frames
// (internal/framestamp), so it changes once a frame however often it is asked for.
// 16 ms is under the frame period of any stream this publishes, so nothing a reader could have
// seen is stepped over, and it keeps a marker off one read of every decode's counters per 4 ms.
const streamPointerCadence = 16 * time.Millisecond

// pointerFeed reads where a publishing machine's pointer is, for one watch.
//
// One feed for this machine's own capture and for a stream it is watching,
// the watch naming which: the two are one reading over different legs,
// and a shell drawing a marker over a picture asks the same question of both.
//
// A publish reporting no position sends nothing and the watch stays open.
// The cursor mode can change under a watch,
// and a shell made to watch again on every change would hold a pointer from the mode before.
type pointerFeed struct {
	read    func() (pointer.Spot, bool)
	cadence time.Duration

	// The last position sent, so an unmoved pointer sends nothing:
	// the watch carries what moves, and an unchanged position every reader already holds.
	//
	// Three fields rather than the message, a message carrying a lock that copying it would copy.
	// The moment of the reading is left out: what is compared is what a viewer draws from,
	// and a pointer that has not moved has not moved however often it was read.
	lastX, lastY      float32
	lastVisible, sent bool
}

// pointerOf opens a feed on the stream ref names, nil being this machine's own publish.
func (s *Server) pointerOf(ref *mirrormev1.StreamRef) *pointerFeed {
	assert.Assert(pointerCadence > 0, "the pointer ticks at a positive interval", pointerCadence)

	if ref == nil {
		return &pointerFeed{read: s.backend.Pointer, cadence: pointerCadence}
	}
	stream := wire.StreamRefOf(ref)
	return &pointerFeed{
		read:    func() (pointer.Spot, bool) { return s.backend.StreamPointer(stream) },
		cadence: streamPointerCadence,
	}
}

// next is the event one tick sends, and false where the pointer reports nothing or has not moved.
func (p *pointerFeed) next() (*mirrormev1.Event, bool) {
	spot, reporting := p.read()
	if !reporting {
		return nil, false
	}

	message := wire.PointerPosition(spot)
	if p.sent && message.GetX() == p.lastX && message.GetY() == p.lastY && message.GetVisible() == p.lastVisible {
		return nil, false
	}
	p.lastX, p.lastY, p.lastVisible, p.sent = message.GetX(), message.GetY(), message.GetVisible(), true

	return &mirrormev1.Event{Payload: &mirrormev1.Event_Pointer{Pointer: message}}, true
}
