package control

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/events"
	"bjoernblessin.de/screenshare/internal/text"
)

// refusalOf reads the one Refusal detail off a served status,
// failing the test where the status carries prose or no statement.
func refusalOf(t *testing.T, err error, want codes.Code) *screensharev1.Refusal {
	t.Helper()

	if err == nil {
		t.Fatal("the call succeeded")
	}
	st := status.Convert(err)
	if st.Code() != want {
		t.Errorf("code = %s, want %s", st.Code(), want)
	}
	details := st.Details()
	if len(details) != 1 {
		t.Fatalf("status carries %d details, want one Refusal", len(details))
	}
	refusal, ok := details[0].(*screensharev1.Refusal)
	if !ok {
		t.Fatalf("detail is %T, want a Refusal", details[0])
	}
	if st.Message() != refusal.GetText().GetCode().String() {
		t.Errorf("status message %q, want the code's name %q", st.Message(), refusal.GetText().GetCode())
	}
	return refusal
}

func causeOf(refusal *screensharev1.Refusal) *screensharev1.Text {
	return argOf(refusal, screensharev1.TextArgName_TEXT_ARG_NAME_CAUSE).GetText()
}

func TestARequestThisBuildCannotReadIsRefusedByCode(t *testing.T) {
	server := New(&fakeBackend{}, events.New(), "test")

	_, err := server.Set(context.Background(), &screensharev1.SetRequest{})

	refusal := refusalOf(t, err, codes.InvalidArgument)
	if got := refusal.GetText().GetCode(); got != screensharev1.TextCode_TEXT_CODE_REQUEST_NOT_IN_THIS_BUILD {
		t.Errorf("refused with %s", got)
	}
}

func TestACodedReasonNestsUnderTheRefusalAsItsCause(t *testing.T) {
	reason := text.Fail(screensharev1.TextCode_TEXT_CODE_MONITOR_NOT_ENUMERATED,
		text.Num(screensharev1.TextArgName_TEXT_ARG_NAME_MONITOR, 9))
	server := New(&fakeBackend{err: Refuse(reason)}, events.New(), "test")

	_, err := server.Set(context.Background(), &screensharev1.SetRequest{State: &screensharev1.SetRequest_MonitorPreview{
		MonitorPreview: &screensharev1.MonitorPreviewWanted{Monitor: 9, Open: true},
	}})

	refusal := refusalOf(t, err, codes.InvalidArgument)
	if got := refusal.GetText().GetCode(); got != screensharev1.TextCode_TEXT_CODE_MONITOR_PREVIEW_REFUSED {
		t.Errorf("refused with %s", got)
	}
	if got := causeOf(refusal).GetCode(); got != screensharev1.TextCode_TEXT_CODE_MONITOR_NOT_ENUMERATED {
		t.Errorf("cause = %s, want the backend's coded reason", got)
	}
	if refusal.GetMessage() != "" {
		t.Errorf("a coded reason left words %q beside it", refusal.GetMessage())
	}
}

func TestAThirdPartysWordsTravelVerbatimBesideTheStatement(t *testing.T) {
	server := New(&fakeBackend{err: errors.New("no space left on device")}, events.New(), "test")

	_, err := server.Set(context.Background(), &screensharev1.SetRequest{State: &screensharev1.SetRequest_Settings{
		Settings: &screensharev1.Settings{Publish: &screensharev1.PublishSettings{Fps: 60}},
	}})

	refusal := refusalOf(t, err, codes.Unavailable)
	if got := refusal.GetText().GetCode(); got != screensharev1.TextCode_TEXT_CODE_SETTINGS_UNSAVED {
		t.Errorf("refused with %s", got)
	}
	if causeOf(refusal) != nil {
		t.Errorf("uncoded words grew a coded cause %s", causeOf(refusal).GetCode())
	}
	if refusal.GetMessage() != "no space left on device" {
		t.Errorf("message = %q, want the words verbatim", refusal.GetMessage())
	}
}

func argOf(refusal *screensharev1.Refusal, name screensharev1.TextArgName) *screensharev1.TextArg {
	for _, arg := range refusal.GetText().GetArgs() {
		if arg.GetName() == name {
			return arg
		}
	}
	return nil
}
