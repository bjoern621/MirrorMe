package control

import (
	"testing"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/events"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// The read is the presence loop's last reading and states nothing of its own,
// so a shell that mounts and one that listened are told the same thing.
// There is no membership call beside it:
// the group key and the display name in the settings are what put this machine in a group.
func TestReadingTheMembersStateAnswersWithTheGroupAsItWasRead(t *testing.T) {
	backend := &fakeBackend{members: wire.MembersSnapshot{
		Joined:  true,
		Members: []wire.Member{{MemberID: "MFZWIZLTOQ2DGNBV", DisplayName: "Björn", Self: true}},
	}}
	server := New(backend, events.New(), "test")

	got := readState(t, server, mirrormev1.EventKind_EVENT_KIND_MEMBERS_STATE).GetMembersState()
	if !got.GetJoined() {
		t.Error("a machine in a group reads as out of it")
	}
	if len(got.GetMembers()) != 1 || !got.GetMembers()[0].GetSelf() {
		t.Errorf("the group reads as %v, want this machine's own row", got.GetMembers())
	}
}
