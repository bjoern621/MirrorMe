package control

import (
	"context"
	"testing"

	"google.golang.org/protobuf/types/known/emptypb"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
)

// readState reads one state the way a shell does, failing the test where the read is refused.
func readState(t *testing.T, server *Server, kind screensharev1.EventKind) *screensharev1.Event {
	t.Helper()

	answer, err := server.Read(context.Background(), &screensharev1.ReadRequest{Kinds: []screensharev1.EventKind{kind}})
	if err != nil {
		t.Fatalf("reading %s answered %v, want the state", kind, err)
	}
	return answer.GetStates()[0]
}

func set(server *Server, req *screensharev1.SetRequest) error {
	_, err := server.Set(context.Background(), req)
	return err
}

func empty() *emptypb.Empty { return &emptypb.Empty{} }
