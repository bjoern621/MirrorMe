package control

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
	"bjoernblessin.de/mirrorme/internal/update"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// fakeStream stands in for the client's half of a server-streaming call.
//
// grpc.ServerStream is embedded and left nil: Watch uses Context and Send,
// and a call to any other method surfaces as a panic rather than being quietly satisfied.
// The buffer on sent keeps Send from blocking once a test has stopped reading,
// so a failing test still lets the call under test return instead of deadlocking on its way out.
type fakeStream struct {
	grpc.ServerStream

	ctx  context.Context
	sent chan *mirrormev1.Event
}

func newFakeStream(ctx context.Context) *fakeStream {
	return &fakeStream{ctx: ctx, sent: make(chan *mirrormev1.Event, 1024)}
}

func (f *fakeStream) Context() context.Context { return f.ctx }

func (f *fakeStream) Send(event *mirrormev1.Event) error {
	f.sent <- event
	return nil
}

// unknownKind is an EventKind value no build declares,
// what a shell generated against a later minor version sends.
// A number, the kinds being an enum:
// the mistake left to a shell is naming a kind that exists somewhere and not here.
const unknownKind = mirrormev1.EventKind(9999)

// Ignored, a kind this build has none of would leave the shell holding an open stream that never
// delivers, which reads as a backend where nothing is happening rather than as a name got wrong.
// The refusal is INVALID_ARGUMENT and names the kind:
// a request may carry several and only one of them is the mistake.
func TestAnUnknownEventKindIsRefusedRatherThanIgnored(t *testing.T) {
	server := New(&fakeBackend{}, events.New(), "test")

	err := server.Watch(
		&mirrormev1.WatchRequest{Kinds: []mirrormev1.EventKind{mirrormev1.EventKind_EVENT_KIND_PUBLISH_STATE, unknownKind}},
		newFakeStream(context.Background()))
	refusal := refusalOf(t, err, codes.InvalidArgument)
	if got := argOf(refusal, mirrormev1.TextArgName_TEXT_ARG_NAME_KIND).GetId(); got != unknownKind.String() {
		t.Errorf("refusal names kind %q, want the one that was wrong, %q", got, unknownKind.String())
	}
}

// Narrowing exists for a surface that wants the state changes without the per-second statistics,
// so a filter that leaked would hand that surface the traffic it asked not to have.
func TestANarrowedWatchReceivesOnlyItsKinds(t *testing.T) {
	broker := events.New()
	server := New(&fakeBackend{}, broker, "test")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := newFakeStream(ctx)

	go func() {
		// What the call returns is the next test's subject; here it only has to be open.
		_ = server.Watch(&mirrormev1.WatchRequest{Kinds: []mirrormev1.EventKind{mirrormev1.EventKind_EVENT_KIND_VIEWER_STATE}}, out)
	}()

	wanted := &mirrormev1.Event{
		Payload: &mirrormev1.Event_ViewerState{ViewerState: &mirrormev1.ViewerState{}},
	}
	unwanted := &mirrormev1.Event{
		Payload: &mirrormev1.Event_PublishStats{PublishStats: &mirrormev1.PublishStats{}},
	}

	// The subscription opens on another goroutine and the broker signals nothing about being registered,
	// so both kinds go out every round until something arrives, rather than once behind a sleep.
	// Publishing both each round is what makes the arrival conclusive:
	// a filter leaking the wrong kind would have delivered it by the round the right one arrives on.
	var received *mirrormev1.Event
	for attempt := 0; attempt < 200 && received == nil; attempt++ {
		broker.Publish(unwanted)
		broker.Publish(wanted)

		select {
		case event := <-out.sent:
			received = event
		case <-time.After(10 * time.Millisecond):
		}
	}
	if received == nil {
		t.Fatal("a narrowed subscription received nothing, want the kind it asked for")
	}

	for {
		if kind := events.KindOf(received); kind != mirrormev1.EventKind_EVENT_KIND_VIEWER_STATE {
			t.Fatalf("received a %s event, want only %s", kind, mirrormev1.EventKind_EVENT_KIND_VIEWER_STATE)
		}
		select {
		case received = <-out.sent:
		default:
			return
		}
	}
}

// A check can settle after a shell's read and before its watch, while the broker holds no subscriber for it.
// Nothing announces that state again, so the shell keeps the checking state it read.
func TestAWatchOpensOnAnUpdateStateThatSettledBeforeIt(t *testing.T) {
	requests := map[string]*mirrormev1.WatchRequest{
		"every kind": {},
		"named":      {Kinds: []mirrormev1.EventKind{mirrormev1.EventKind_EVENT_KIND_UPDATE_STATE}},
	}

	for name, request := range requests {
		t.Run(name, func(t *testing.T) {
			backend := &fakeBackend{}
			broker := events.New()
			server := New(backend, broker, "test")

			backend.updates = update.State{Stage: update.StageCurrent, Running: "0.15.0", Latest: "v0.15.0"}
			broker.Publish(wire.UpdateStateEvent(backend.updates))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := newFakeStream(ctx)
			go func() { _ = server.Watch(request, out) }()

			event := nextOfKind(t, out, mirrormev1.EventKind_EVENT_KIND_UPDATE_STATE)
			if got := event.GetUpdateState().GetStage(); got != mirrormev1.UpdateStage_UPDATE_STAGE_CURRENT {
				t.Errorf("the watch opened on stage %s, want the settled stage", got)
			}
		})
	}
}

// A presence reading can land after a shell's read and before its watch.
// The broker holds no subscriber for that reading, so a shell keeps the older group until a later reading lands.
func TestAWatchOpensOnTheGroupThatMovedBeforeIt(t *testing.T) {
	requests := map[string]*mirrormev1.WatchRequest{
		"every kind": {},
		"named":      {Kinds: []mirrormev1.EventKind{mirrormev1.EventKind_EVENT_KIND_MEMBERS_STATE}},
	}

	for name, request := range requests {
		t.Run(name, func(t *testing.T) {
			backend := &fakeBackend{}
			broker := events.New()
			server := New(backend, broker, "test")

			backend.members = wire.MembersSnapshot{Joined: true}
			broker.Publish(wire.MembersStateEvent(backend.members))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := newFakeStream(ctx)
			go func() { _ = server.Watch(request, out) }()

			event := nextOfKind(t, out, mirrormev1.EventKind_EVENT_KIND_MEMBERS_STATE)
			if !event.GetMembersState().GetJoined() {
				t.Error("the watch opened on a group this machine is not in, want the group it joined before the watch")
			}
		})
	}
}

// A shell reads each of these once before it watches, and any of them can move in between.
func TestAWatchOpensOnEveryStandingState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := newFakeStream(ctx)
	server := New(&fakeBackend{}, events.New(), "test")
	go func() { _ = server.Watch(&mirrormev1.WatchRequest{}, out) }()

	unseen := map[mirrormev1.EventKind]bool{}
	for kind := range states {
		if !readAlone[kind] {
			unseen[kind] = true
		}
	}

	for count := len(unseen); count > 0; count-- {
		select {
		case event := <-out.sent:
			kind := events.KindOf(event)
			if !unseen[kind] {
				t.Fatalf("the watch opened on %s, want one event for each standing state and nothing else", kind)
			}
			delete(unseen, kind)
		case <-time.After(2 * time.Second):
			t.Fatalf("the watch opened on no state for %v", slices.Collect(maps.Keys(unseen)))
		}
	}
}

// A level watch stands open beside the state watch.
// A snapshot here would land on a reader that asserts every event is a level.
func TestAWatchOfATickOpensOnNoStandingState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := newFakeStream(ctx)
	server := New(&fakeBackend{}, events.New(), "test")
	request := &mirrormev1.WatchRequest{Kinds: []mirrormev1.EventKind{mirrormev1.EventKind_EVENT_KIND_AUDIO_LEVELS}}
	go func() { _ = server.Watch(request, out) }()

	select {
	case event := <-out.sent:
		if kind := events.KindOf(event); kind != mirrormev1.EventKind_EVENT_KIND_AUDIO_LEVELS {
			t.Errorf("a level watch opened on %s, want levels alone", kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a level watch delivered nothing")
	}
}

// nextOfKind reads a watch up to its first event of kind, passing over the opening states of other kinds.
func nextOfKind(t *testing.T, out *fakeStream, kind mirrormev1.EventKind) *mirrormev1.Event {
	t.Helper()

	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-out.sent:
			if events.KindOf(event) == kind {
				return event
			}
		case <-timeout:
			t.Fatalf("the watch delivered no %s, so a shell holds what it read before the watch", kind)
		}
	}
}

// The broker sends on a channel it holds until the cancel removes it,
// so a call outliving its client would leave a subscriber nothing reads,
// and every publish still walks past.
// The call returning is the observable half of that release,
// the only way out of the loop being the deferred cancel,
// and it is what a shell closing its window produces.
func TestAWatchEndsWithTheClientThatOpenedIt(t *testing.T) {
	server := New(&fakeBackend{}, events.New(), "test")

	ctx, cancel := context.WithCancel(context.Background())
	out := newFakeStream(ctx)

	ended := make(chan error, 1)
	go func() {
		ended <- server.Watch(&mirrormev1.WatchRequest{}, out)
	}()

	cancel()

	select {
	case err := <-ended:
		if err != nil {
			t.Errorf("a client that went away ended the call with %v, want no failure", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the call did not return after its client's context ended, so the subscription is still held")
	}
}
