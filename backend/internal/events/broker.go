// Package events is the one place a change to the running state is announced.
//
// Every producer in the backend publishes here and every surface in front of it subscribes here,
// so a window that pressed a button and a window that did not are told the same thing
// (docs/ipc-api.md, "Events").
//
// The events are the contract's own messages rather than a shape of this package's own.
// A producer builds a mirrormev1.Event and this package decides who receives it and in what order:
// a second state vocabulary between the producer and the wire is the drift the contract removes.
//
// Two rules of the contract are structural here.
// An event carries a whole state and never a delta,
// so a dropped one costs a subscriber the interval it was stale for and nothing else.
// A subscriber that acted still waits for the event,
// so an effect never answers with the state it produced.
package events

import (
	"strconv"
	"sync"

	"google.golang.org/protobuf/proto"

	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/go-utils/util/logger"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// Kinds lists every event kind, in the order events.proto declares them.
//
// A kind is the contract's own enum value, not a string this package spells,
// so a subscription names something the compiler checked rather than something waiting on a typo.
var Kinds = []mirrormev1.EventKind{
	mirrormev1.EventKind_EVENT_KIND_PUBLISH_STATE,
	mirrormev1.EventKind_EVENT_KIND_PUBLISH_STATS,
	mirrormev1.EventKind_EVENT_KIND_PUBLISH_EXIT,
	mirrormev1.EventKind_EVENT_KIND_RELAY_STATUS,
	mirrormev1.EventKind_EVENT_KIND_VIEWER_STATE,
	mirrormev1.EventKind_EVENT_KIND_VIEWER_EXIT,
	mirrormev1.EventKind_EVENT_KIND_TEST_STREAM_STATE,
	mirrormev1.EventKind_EVENT_KIND_TEST_STREAM_EXIT,
	mirrormev1.EventKind_EVENT_KIND_CATALOG,
	mirrormev1.EventKind_EVENT_KIND_SETTINGS_CHANGED,
	mirrormev1.EventKind_EVENT_KIND_RECEIVE_STATE,
	mirrormev1.EventKind_EVENT_KIND_RECEIVE_EXIT,
	mirrormev1.EventKind_EVENT_KIND_RECEIVE_STATS,
	mirrormev1.EventKind_EVENT_KIND_MONITOR_PREVIEW_STATE,
	mirrormev1.EventKind_EVENT_KIND_MEMBERS_STATE,
	mirrormev1.EventKind_EVENT_KIND_DISCORD_STATE,
	mirrormev1.EventKind_EVENT_KIND_UPDATE_STATE,
	mirrormev1.EventKind_EVENT_KIND_STREAM_TIMELINE,
	mirrormev1.EventKind_EVENT_KIND_SETTINGS,
	mirrormev1.EventKind_EVENT_KIND_PRESETS,
	mirrormev1.EventKind_EVENT_KIND_SHARE_WINDOWS,
	mirrormev1.EventKind_EVENT_KIND_AUDIO_LEVELS,
	mirrormev1.EventKind_EVENT_KIND_POINTER,
}

// queueDepth is how many events one subscriber may fall behind by before the oldest is dropped.
//
// A depth exists because the per-second statistics are the high-rate kind,
// and a subscriber that stalls must not stall the encoder publishing them.
// The oldest goes because every event is a whole state: the newest is the one worth having,
// and the sequence number is what says a gap happened.
const queueDepth = 64

// Broker fans one event out to every subscriber.
type Broker struct {
	// mu guards the map, nextID, and every subscriber's sequence and dropped counters.
	mu sync.Mutex
	// subscribers is keyed by an id handed out at subscribe time,
	// so a cancel removes the one it belongs to and never a same-shaped neighbour.
	subscribers map[uint64]*subscriber
	nextID      uint64
}

// subscriber is one open Subscribe call.
type subscriber struct {
	// kinds is the filter, nil for a subscriber that asked for everything.
	kinds map[mirrormev1.EventKind]bool
	// out is buffered to queueDepth and closed by the cancel function alone,
	// so a send never races a close.
	out chan *mirrormev1.Event
	// sequence numbers the events this subscription was sent, from one.
	//
	// Per subscriber and not per broker, which makes the number mean what the contract says.
	// A shared counter skips a number for every event a filter drops,
	// leaving a subscriber that named kinds a permanent gap it cannot tell from falling behind.
	sequence uint64
	// dropped counts what this subscriber fell behind by.
	// Written and read under the broker's lock,
	// warned about at the first loss and totalled where the subscription ends.
	dropped uint64
}

func New() *Broker {
	return &Broker{subscribers: map[uint64]*subscriber{}}
}

func Known(kind mirrormev1.EventKind) bool {
	for _, k := range Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Subscribe opens a stream of events,
// narrowed to kinds where any are named and carrying every kind where none are.
//
// cancel is the caller's obligation, whether the call ends by return or by the client going away:
// until it runs, the broker still holds a channel it sends on.
// A second cancel is a success, which lets it be both deferred and called on an error path.
//
// An unknown kind is refused rather than answered with an empty stream,
// so a shell asking for a kind this build has none of learns it at the subscribe.
// EVENT_KIND_UNSPECIFIED is one of the unknown ones:
// a request that failed to say what it wanted is not a request for all of it.
func (b *Broker) Subscribe(kinds []mirrormev1.EventKind) (<-chan *mirrormev1.Event, func(), error) {
	var filter map[mirrormev1.EventKind]bool
	if len(kinds) > 0 {
		filter = make(map[mirrormev1.EventKind]bool, len(kinds))
		for _, kind := range kinds {
			if !Known(kind) {
				return nil, nil, &UnknownKindError{Kind: kind}
			}
			filter[kind] = true
		}
	}

	sub := &subscriber{kinds: filter, out: make(chan *mirrormev1.Event, queueDepth)}

	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.subscribers[id] = sub
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, id)
			// Read under the lock, which is the only place Publish writes it.
			dropped := sub.dropped
			b.mu.Unlock()
			// Out of the map before the close, so no Publish can still be sending on the channel,
			// and the receiver's range ends.
			close(sub.out)

			// What the subscription cost, once, where the number is final.
			// A reader that lost events saw a state it never caught up on,
			// and the gap in the sequence numbers it did receive is the only other trace.
			if dropped > 0 {
				logger.Warnf("event subscriber %d ended having lost %d events", id, dropped)
			}
		})
	}

	return sub.out, cancel, nil
}

// Publish hands one event to every subscriber that asked for its kind,
// each a copy of its own carrying its own sequence number.
//
// It never blocks.
// A subscriber whose queue is full loses its oldest queued event to make room:
// every event is a whole state and the newest is the one worth keeping.
// The number is spent before the enqueue rather than after it,
// so a dropped event leaves the gap that is the point of numbering them.
//
// The clone per subscriber is what keeps the number per reader:
// one shared message would have every stream writing its own sequence into the same field.
func (b *Broker) Publish(e *mirrormev1.Event) {
	assert.IsNotNil(e, "an announced change is an event")
	kind := KindOf(e)
	assert.Assert(Known(kind), "an announced change names a declared kind", kind.String())

	b.mu.Lock()
	defer b.mu.Unlock()

	for id, sub := range b.subscribers {
		if sub.kinds != nil && !sub.kinds[kind] {
			continue
		}

		sub.sequence++
		event := proto.Clone(e).(*mirrormev1.Event)
		event.Sequence = sub.sequence
		assert.Assert(event.Sequence > 0, "a delivered event is numbered from one", event.Sequence)

		select {
		case sub.out <- event:
		default:
			// Full, so the oldest goes to make room for this one.
			// The receive cannot block: this is the only sender and the buffer is known full.
			before := sub.dropped
			select {
			case <-sub.out:
				sub.dropped++
			default:
			}
			select {
			case sub.out <- event:
			default:
				sub.dropped++
			}
			// Once per subscriber, at the first loss.
			// A consumer that falls behind stays behind,
			// so a line per dropped event would bury the stuck subscription under the merely busy one,
			// and the fact worth having is that this subscriber lost anything at all.
			// The total goes out where the subscription ends.
			if before == 0 && sub.dropped > 0 {
				logger.Warnf("event subscriber %d fell behind and is losing events", id)
			}
		}
	}
}

// KindOf reads the kind off the payload the event carries,
// so no producer states one that disagrees with what it sent.
func KindOf(e *mirrormev1.Event) mirrormev1.EventKind {
	assert.IsNotNil(e, "an event kind belongs to an event")

	switch e.GetPayload().(type) {
	case *mirrormev1.Event_PublishState:
		return mirrormev1.EventKind_EVENT_KIND_PUBLISH_STATE
	case *mirrormev1.Event_PublishStats:
		return mirrormev1.EventKind_EVENT_KIND_PUBLISH_STATS
	case *mirrormev1.Event_PublishExit:
		return mirrormev1.EventKind_EVENT_KIND_PUBLISH_EXIT
	case *mirrormev1.Event_RelayStatus:
		return mirrormev1.EventKind_EVENT_KIND_RELAY_STATUS
	case *mirrormev1.Event_ViewerState:
		return mirrormev1.EventKind_EVENT_KIND_VIEWER_STATE
	case *mirrormev1.Event_ViewerExit:
		return mirrormev1.EventKind_EVENT_KIND_VIEWER_EXIT
	case *mirrormev1.Event_TestStreamState:
		return mirrormev1.EventKind_EVENT_KIND_TEST_STREAM_STATE
	case *mirrormev1.Event_TestStreamExit:
		return mirrormev1.EventKind_EVENT_KIND_TEST_STREAM_EXIT
	case *mirrormev1.Event_Catalog:
		return mirrormev1.EventKind_EVENT_KIND_CATALOG
	case *mirrormev1.Event_SettingsChanged:
		return mirrormev1.EventKind_EVENT_KIND_SETTINGS_CHANGED
	case *mirrormev1.Event_ReceiveState:
		return mirrormev1.EventKind_EVENT_KIND_RECEIVE_STATE
	case *mirrormev1.Event_ReceiveExit:
		return mirrormev1.EventKind_EVENT_KIND_RECEIVE_EXIT
	case *mirrormev1.Event_ReceiveStats:
		return mirrormev1.EventKind_EVENT_KIND_RECEIVE_STATS
	case *mirrormev1.Event_MonitorPreviewState:
		return mirrormev1.EventKind_EVENT_KIND_MONITOR_PREVIEW_STATE
	case *mirrormev1.Event_MembersState:
		return mirrormev1.EventKind_EVENT_KIND_MEMBERS_STATE
	case *mirrormev1.Event_DiscordState:
		return mirrormev1.EventKind_EVENT_KIND_DISCORD_STATE
	case *mirrormev1.Event_UpdateState:
		return mirrormev1.EventKind_EVENT_KIND_UPDATE_STATE
	case *mirrormev1.Event_StreamTimeline:
		return mirrormev1.EventKind_EVENT_KIND_STREAM_TIMELINE
	case *mirrormev1.Event_Settings:
		return mirrormev1.EventKind_EVENT_KIND_SETTINGS
	case *mirrormev1.Event_Presets:
		return mirrormev1.EventKind_EVENT_KIND_PRESETS
	case *mirrormev1.Event_ShareWindows:
		return mirrormev1.EventKind_EVENT_KIND_SHARE_WINDOWS
	case *mirrormev1.Event_AudioLevels:
		return mirrormev1.EventKind_EVENT_KIND_AUDIO_LEVELS
	case *mirrormev1.Event_Pointer:
		return mirrormev1.EventKind_EVENT_KIND_POINTER
	default:
		assert.Never("unexpected event payload", e.GetPayload())
		return mirrormev1.EventKind_EVENT_KIND_UNSPECIFIED
	}
}

// KindNames is the declared kinds as their enum names, for the sentence that refuses an unknown one.
func KindNames() []string {
	out := make([]string, 0, len(Kinds))
	for _, k := range Kinds {
		out = append(out, k.String())
	}

	assert.Assert(len(out) == len(Kinds), "every declared kind is named", len(out), len(Kinds))
	return out
}

// UnknownKindError is a subscriber naming an event kind this build has none of.
// An Umgebungsfehler rather than a bug in this code:
// the value came off the wire from a shell that may have been built against another minor version.
type UnknownKindError struct {
	Kind mirrormev1.EventKind
}

func (e *UnknownKindError) Error() string {
	name := e.Kind.String()
	if name == "" {
		name = strconv.Itoa(int(e.Kind))
	}
	return "no event kind named " + name
}
