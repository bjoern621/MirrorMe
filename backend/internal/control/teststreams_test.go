package control

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
)

// A released build runs no synthetic set, so a client asking for one learns why
// where a count above the bound would blame the machine.
func TestAReleasedBuildRefusesTestStreams(t *testing.T) {
	server := New(&fakeBackend{released: true}, events.New(), "test")

	_, err := server.Set(context.Background(), &mirrormev1.SetRequest{
		State: &mirrormev1.SetRequest_TestStreams{TestStreams: 3},
	})

	refusal := refusalOf(t, err, codes.FailedPrecondition)
	if got := refusal.GetText().GetCode(); got != mirrormev1.TextCode_TEXT_CODE_TEST_STREAMS_RELEASE_BUILD {
		t.Errorf("refusal code %v, want the released build's own", got)
	}
}

// The form reads the same bound the refusal does, so a released build draws no toggle.
func TestAReleasedBuildHidesTheTestStreamToggle(t *testing.T) {
	for _, released := range []bool{false, true} {
		server := New(&fakeBackend{released: released}, events.New(), "test")
		if got := server.formDeps().TestStreamsOff; got != released {
			t.Errorf("released=%v: the form reads TestStreamsOff=%v", released, got)
		}
	}
}
