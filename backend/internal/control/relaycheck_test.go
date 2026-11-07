package control

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
	"bjoernblessin.de/mirrorme/internal/reach"
	"bjoernblessin.de/mirrorme/internal/settings"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// The draft says which relay is being asked about, so a request carrying none is refused:
// checking the relay nobody named would answer every leg unaddressed,
// which reads as news about the relay on screen.
func TestCheckRelayRefusesARequestWithNoDraft(t *testing.T) {
	server := New(&fakeBackend{}, events.New(), "test")

	_, err := server.Do(context.Background(), &mirrormev1.DoRequest{Action: &mirrormev1.DoRequest_CheckRelay{}})
	if err == nil {
		t.Fatal("a check with no settings was answered, want it refused")
	}
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Errorf("code = %s, want %s", got, codes.InvalidArgument)
	}
}

// A relay that answers nothing still answers the call:
// it succeeds and every leg carries its own verdict (docs/ipc-api.md, "Errors").
func TestARelayAnsweringNothingIsStillAnAnswer(t *testing.T) {
	backend := &fakeBackend{legs: []reach.Result{
		{Leg: "rtsp", Address: "rtsps://relay:8322", Verdict: reach.Unreachable, Detail: "i/o timeout"},
		{Leg: "groups", Verdict: reach.Unaddressed, Unused: reach.ReasonNoRelay},
	}}
	server := New(backend, events.New(), "test")

	outcome, err := server.Do(context.Background(), &mirrormev1.DoRequest{
		Action: &mirrormev1.DoRequest_CheckRelay{CheckRelay: wire.Settings(settings.Defaults())},
	})
	answer := outcome.GetRelayCheck()
	if err != nil {
		t.Fatalf("a relay nothing answered on failed the call: %v", err)
	}
	if len(answer.GetLegs()) != 2 {
		t.Fatalf("%d legs, want one per checked leg", len(answer.GetLegs()))
	}
	if got := answer.GetLegs()[0].GetDetail(); got != "i/o timeout" {
		t.Errorf("the unreachable leg says %q, want the dial's own words", got)
	}
	if answer.GetLegs()[1].GetUnused() == nil {
		t.Error("the undialled leg says nothing about why")
	}
}
