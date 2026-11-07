package control

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
	"bjoernblessin.de/mirrorme/internal/text"
)

// refusalOf reads the one Refusal detail off a served status,
// failing the test where the status carries prose or no statement.
func refusalOf(t *testing.T, err error, want codes.Code) *mirrormev1.Refusal {
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
	refusal, ok := details[0].(*mirrormev1.Refusal)
	if !ok {
		t.Fatalf("detail is %T, want a Refusal", details[0])
	}
	if st.Message() != refusal.GetText().GetCode().String() {
		t.Errorf("status message %q, want the code's name %q", st.Message(), refusal.GetText().GetCode())
	}
	return refusal
}

func causeOf(refusal *mirrormev1.Refusal) *mirrormev1.Text {
	return argOf(refusal, mirrormev1.TextArgName_TEXT_ARG_NAME_CAUSE).GetText()
}

func TestARequestThisBuildCannotReadIsRefusedByCode(t *testing.T) {
	server := New(&fakeBackend{}, events.New(), "test")

	_, err := server.Set(context.Background(), &mirrormev1.SetRequest{})

	refusal := refusalOf(t, err, codes.InvalidArgument)
	if got := refusal.GetText().GetCode(); got != mirrormev1.TextCode_TEXT_CODE_REQUEST_NOT_IN_THIS_BUILD {
		t.Errorf("refused with %s", got)
	}
}

func TestACodedReasonNestsUnderTheRefusalAsItsCause(t *testing.T) {
	reason := text.Fail(mirrormev1.TextCode_TEXT_CODE_MONITOR_NOT_ENUMERATED,
		text.Num(mirrormev1.TextArgName_TEXT_ARG_NAME_MONITOR, 9))
	server := New(&fakeBackend{err: Refuse(reason)}, events.New(), "test")

	_, err := server.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_MonitorPreview{
		MonitorPreview: &mirrormev1.MonitorPreviewWanted{Monitor: 9, Open: true},
	}})

	refusal := refusalOf(t, err, codes.InvalidArgument)
	if got := refusal.GetText().GetCode(); got != mirrormev1.TextCode_TEXT_CODE_MONITOR_PREVIEW_REFUSED {
		t.Errorf("refused with %s", got)
	}
	if got := causeOf(refusal).GetCode(); got != mirrormev1.TextCode_TEXT_CODE_MONITOR_NOT_ENUMERATED {
		t.Errorf("cause = %s, want the backend's coded reason", got)
	}
	if refusal.GetMessage() != "" {
		t.Errorf("a coded reason left words %q beside it", refusal.GetMessage())
	}
}

func TestAThirdPartysWordsTravelVerbatimBesideTheStatement(t *testing.T) {
	server := New(&fakeBackend{err: errors.New("no space left on device")}, events.New(), "test")

	_, err := server.Set(context.Background(), &mirrormev1.SetRequest{State: &mirrormev1.SetRequest_Settings{
		Settings: &mirrormev1.Settings{Publish: &mirrormev1.PublishSettings{Fps: 60}},
	}})

	refusal := refusalOf(t, err, codes.Unavailable)
	if got := refusal.GetText().GetCode(); got != mirrormev1.TextCode_TEXT_CODE_SETTINGS_UNSAVED {
		t.Errorf("refused with %s", got)
	}
	if causeOf(refusal) != nil {
		t.Errorf("uncoded words grew a coded cause %s", causeOf(refusal).GetCode())
	}
	if refusal.GetMessage() != "no space left on device" {
		t.Errorf("message = %q, want the words verbatim", refusal.GetMessage())
	}
}

func argOf(refusal *mirrormev1.Refusal, name mirrormev1.TextArgName) *mirrormev1.TextArg {
	for _, arg := range refusal.GetText().GetArgs() {
		if arg.GetName() == name {
			return arg
		}
	}
	return nil
}
