package control

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// A start the backend refuses with a plain error is FAILED_PRECONDITION:
// the request is well formed, every integer being a monitor index somewhere,
// and what refuses it is a fact about this machine.
// The line docs/ipc-api.md draws between the two codes.
// The reason behind it travels beside the statement instead of being replaced.
func TestARefusedMonitorPreviewIsAPrecondition(t *testing.T) {
	server := New(&fakeBackend{err: errors.New("this session cannot read one monitor apart from another")},
		events.New(), "test")

	_, err := server.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_MonitorPreview{
		MonitorPreview: &mirrormev1.MonitorPreviewWanted{Monitor: 2, Open: true},
	}})

	refusal := refusalOf(t, err, codes.FailedPrecondition)
	if refusal.GetMessage() != "this session cannot read one monitor apart from another" {
		t.Errorf("refusal words %q drop the backend's own reason", refusal.GetMessage())
	}
	if got := argOf(refusal, mirrormev1.TextArgName_TEXT_ARG_NAME_MONITOR).GetNumber(); got != 2 {
		t.Errorf("refusal names monitor %d, want the screen it is about", got)
	}
}

// An index no output is enumerated under is INVALID_ARGUMENT,
// which the contract states and which only a typed refusal can carry:
// both cases arrive as an error from one method,
// and a code read off the sentence would be the contract deriving itself from prose (refusal.go).
func TestAPreviewOfAScreenThatDoesNotExistIsARequestFault(t *testing.T) {
	server := New(&fakeBackend{err: Refuse(errors.New("monitor 9 is not one of this machine's outputs"))},
		events.New(), "test")

	_, err := server.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_MonitorPreview{
		MonitorPreview: &mirrormev1.MonitorPreviewWanted{Monitor: 9, Open: true},
	}})
	refusalOf(t, err, codes.InvalidArgument)
}

// The same line on the watch pair:
// a leg this build has no viewer for is the request naming something that does not exist,
// while a leg that cannot carry the stream's format is the world not being ready.
func TestAWatchOverALegThisBuildHasNoViewerForIsARequestFault(t *testing.T) {
	viewer := &mirrormev1.StreamRef{StreamName: "desk", Transport: "moq"}

	refused := New(&fakeBackend{err: Refuse(errors.New(`no viewer implements transport "moq"`))}, events.New(), "test")
	_, err := refused.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_Viewer{
		Viewer: &mirrormev1.ViewerWanted{Viewer: viewer, Open: true},
	}})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Errorf("a leg with no viewer answered %s, want %s", got, codes.InvalidArgument)
	}

	unready := New(&fakeBackend{err: errors.New("desk is av1, which srt cannot carry")}, events.New(), "test")
	_, err = unready.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_Viewer{
		Viewer: &mirrormev1.ViewerWanted{Viewer: viewer, Open: true},
	}})
	if got := status.Code(err); got != codes.FailedPrecondition {
		t.Errorf("a leg that cannot carry the format answered %s, want %s", got, codes.FailedPrecondition)
	}
}

// A stop names the state the caller wants, and for a screen nothing is reading that state holds,
// so the call succeeds: the idempotency the whole contract is built on.
// The backend's error field is set to prove the stop never consults it.
func TestStoppingAScreenNobodyIsReadingSucceeds(t *testing.T) {
	server := New(&fakeBackend{err: errors.New("would refuse anything that asked")}, events.New(), "test")

	if _, err := server.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_MonitorPreview{
		MonitorPreview: &mirrormev1.MonitorPreviewWanted{Monitor: 7},
	}}); err != nil {
		t.Errorf("stopping a screen nothing is reading was refused: %v", err)
	}
}

// The three subscription arms reach three different backend methods,
// and one that named no arm is refused rather than served the first.
// Two arms carry no key,
// so a discriminator read off a missing key would send an empty request to whichever method it fell
// through to.
func TestAFrameSubscriptionNamesOneOfTheThreePictures(t *testing.T) {
	for name, tc := range map[string]struct {
		subscribe *mirrormev1.FrameSubscribe
		want      wire.FrameSourceKind
		named     bool
	}{
		"a relay decode carries the pair that identifies it": {
			subscribe: &mirrormev1.FrameSubscribe{
				Source: &mirrormev1.FrameSubscribe_Stream{
					Stream: &mirrormev1.StreamRef{StreamName: "desk", Transport: "srt"},
				},
			},
			want:  wire.FrameSourceRelay,
			named: true,
		},
		"the publish preview carries nothing and needs nothing": {
			subscribe: &mirrormev1.FrameSubscribe{
				Source: &mirrormev1.FrameSubscribe_PublishPreview{
					PublishPreview: &mirrormev1.PublishPreview{},
				},
			},
			want:  wire.FrameSourcePublishPreview,
			named: true,
		},
		"a screen carries the index its output is enumerated under": {
			subscribe: &mirrormev1.FrameSubscribe{
				Source: &mirrormev1.FrameSubscribe_MonitorPreview{
					MonitorPreview: &mirrormev1.MonitorPreview{Monitor: 3},
				},
			},
			want:  wire.FrameSourceMonitorPreview,
			named: true,
		},
		"a subscription that filled no arm names none of them": {
			subscribe: &mirrormev1.FrameSubscribe{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, named := wire.FrameSourceOf(tc.subscribe)

			if named != tc.named {
				t.Fatalf("named = %v, want %v", named, tc.named)
			}
			if !named {
				return
			}
			if source.Kind != tc.want {
				t.Errorf("kind = %v, want %v", source.Kind, tc.want)
			}
			if tc.want == wire.FrameSourceMonitorPreview && source.Monitor != 3 {
				t.Errorf("monitor = %d, want 3", source.Monitor)
			}
		})
	}
}
