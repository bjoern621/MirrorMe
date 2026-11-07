package control

import (
	"context"
	"testing"

	"google.golang.org/protobuf/types/known/emptypb"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// readState reads one state the way a shell does, failing the test where the read is refused.
func readState(t *testing.T, server *Server, kind mirrormev1.EventKind) *mirrormev1.Event {
	t.Helper()

	answer, err := server.Read(context.Background(), &mirrormev1.ReadRequest{Kinds: []mirrormev1.EventKind{kind}})
	if err != nil {
		t.Fatalf("reading %s answered %v, want the state", kind, err)
	}
	return answer.GetStates()[0]
}

func set(server *Server, req *mirrormev1.SetRequest) error {
	_, err := server.Set(context.Background(), req)
	return err
}

func empty() *emptypb.Empty { return &emptypb.Empty{} }
