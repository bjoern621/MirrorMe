package control

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/events"
	"bjoernblessin.de/screenshare/internal/group"
	"bjoernblessin.de/screenshare/internal/wire"
)

// A live Set naming the pipeline already publishing succeeds and neither starts nor restarts,
// in Discord mode as in any other.
// Membership is brokered rather than sent (internal/wire, RelaySettings),
// so a draft read straight off the request names a pipeline that is not the one running,
// and the repeat a shell makes after a lost answer would restart the stream on itself.
func TestALiveSetRepeatsADiscordStream(t *testing.T) {
	backend := &probedBackend{}
	server := New(backend, events.New(), "test")

	idle, err := server.Ask(context.Background(), &screensharev1.AskRequest{Question: &screensharev1.AskRequest_Form{}})
	if err != nil {
		t.Fatalf("resolving a form answered %v, want an answer", err)
	}
	draft := idle.GetForm().GetSettings()
	draft.Relay.GroupSource = group.SourceDiscord

	running := backend.Brokered(wire.ToSettings(draft))
	backend.publish = wire.PublishSnapshot{Live: &wire.LiveSnapshot{Settings: running}}

	if _, err := server.Set(context.Background(), &screensharev1.SetRequest{State: &screensharev1.SetRequest_Publish{
		Publish: &screensharev1.PublishWanted{Live: true, Settings: draft},
	}}); err != nil {
		t.Errorf("asking for the stream already publishing answered %v, want the repeat to succeed", err)
	}
	if backend.starts != 0 || backend.applies != 0 {
		t.Errorf("the repeat started %d and restarted %d streams, want neither", backend.starts, backend.applies)
	}
}

// A live Set is the one way onto the air, so what it does follows from what is in force:
// nothing starts a stream, another pipeline restarts the running one on the new settings.
func TestALiveSetStartsOrRestartsByWhatIsInForce(t *testing.T) {
	backend := &probedBackend{}
	server := New(backend, events.New(), "test")

	idle, err := server.Ask(context.Background(), &screensharev1.AskRequest{Question: &screensharev1.AskRequest_Form{}})
	if err != nil {
		t.Fatalf("resolving a form answered %v, want an answer", err)
	}
	draft := idle.GetForm().GetSettings()
	// The empty draft repairs to no frame rate here, which no engine renders and SamePipeline refuses to compare.
	draft.Publish.Fps = 30
	live := &screensharev1.SetRequest{State: &screensharev1.SetRequest_Publish{
		Publish: &screensharev1.PublishWanted{Live: true, Settings: draft},
	}}

	if err := set(server, live); err != nil {
		t.Fatalf("going live with nothing in force answered %v, want a start", err)
	}
	if backend.starts != 1 || backend.applies != 0 {
		t.Errorf("nothing in force started %d and restarted %d streams, want one start", backend.starts, backend.applies)
	}

	running := wire.ToSettings(draft)
	running.Publish.Fps++
	backend.publish = wire.PublishSnapshot{Live: &wire.LiveSnapshot{Settings: running}}

	if err := set(server, live); err != nil {
		t.Fatalf("going live over another pipeline answered %v, want a restart", err)
	}
	if backend.starts != 1 || backend.applies != 1 {
		t.Errorf("another pipeline in force started %d and restarted %d streams, want one restart", backend.starts, backend.applies)
	}
}

// The name the service stored the report under is what a reader quotes,
// so it crosses the contract rather than being logged where the press cannot see it.
func TestSendReportAnswersTheStoredName(t *testing.T) {
	backend := &probedBackend{}
	backend.reportID = "7f3a91c2"
	server := New(backend, events.New(), "test")

	sent, err := server.Do(context.Background(), &screensharev1.DoRequest{Action: &screensharev1.DoRequest_SendReport{SendReport: empty()}})
	if err != nil {
		t.Fatalf("sending a report answered %v, want the send to succeed", err)
	}
	if got := sent.GetReportId(); got != backend.reportID {
		t.Errorf("report id = %q, want %q", got, backend.reportID)
	}
}

// A report goes to the group service of the deployment in use,
// and settings naming no relay are a moment that is wrong rather than a request that is
// (docs/ipc-api.md).
func TestSendReportRefusesWithNoRelay(t *testing.T) {
	backend := &probedBackend{}
	backend.err = errors.New("the settings name no relay")
	server := New(backend, events.New(), "test")

	if _, err := server.Do(context.Background(), &screensharev1.DoRequest{Action: &screensharev1.DoRequest_SendReport{SendReport: empty()}}); err == nil {
		t.Fatal("sending a report to no relay succeeded, want a refusal")
	} else if got := status.Code(err); got != codes.FailedPrecondition {
		t.Errorf("code = %s, want %s", got, codes.FailedPrecondition)
	}
}
