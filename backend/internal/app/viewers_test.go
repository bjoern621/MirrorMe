package app

import (
	"testing"
	"time"

	"bjoernblessin.de/screenshare/internal/relay"
	"bjoernblessin.de/screenshare/internal/watchsvc"
	"bjoernblessin.de/screenshare/internal/wire"
)

func TestViewingStatesEveryRunningDecodeWithItsFigures(t *testing.T) {
	path, arrive, total, shown, shed := 41.5, 20.0, 88.0, 59.8, 0.2
	sample := []wire.ReceiveStreamStats{
		{
			Stream:       wire.StreamRef{StreamName: "Bob/screen", Transport: "rtsp"},
			RenderFPS:    &shown,
			DiscardedFPS: &shed,
			Delay:        wire.DelayBudget{Path: &path, Arrive: &arrive, Total: &total},
		},
		{Stream: wire.StreamRef{StreamName: "Carol/game", Transport: "srt"}},
	}

	viewing := viewingOf(sample, func(name string) string { return "G/" + name })

	if len(viewing) != 2 {
		t.Fatalf("%d viewings stated, want one per decode", len(viewing))
	}
	bob := viewing[0]
	if bob.Path != "G/Bob/screen" || bob.Transport != "rtsp" {
		t.Errorf("first viewing names %s over %s, want G/Bob/screen over rtsp", bob.Path, bob.Transport)
	}
	if *bob.PathMs != path+arrive || *bob.TotalMs != total || *bob.ShownFPS != shown || *bob.DroppedFPS != shed {
		t.Errorf("figures stated as %+v", bob)
	}
	carol := viewing[1]
	if carol.PathMs != nil || carol.TotalMs != nil || carol.ShownFPS != nil || carol.DroppedFPS != nil {
		t.Errorf("a decode that measured nothing stated %+v, want every figure absent", carol)
	}
}

func TestViewingOfNothingIsAnEmptySet(t *testing.T) {
	if viewing := viewingOf(nil, func(name string) string { return name }); viewing == nil || len(viewing) != 0 {
		t.Errorf("no decode stated %v, want an empty set that clears this member", viewing)
	}
}

func TestViewersLandOnThePublishedPathUnderTheirNames(t *testing.T) {
	status := relay.Status{Reachable: true, Paths: []relay.Path{
		{Name: "Bob/screen", Ready: true, Readers: 2},
		{Name: "Carol/game", Ready: true, Readers: 1},
	}}
	viewers := []watchsvc.Viewer{
		{Member: "m-alice", Viewing: watchsvc.Viewing{Path: "G/Bob/screen", Transport: "rtsp"}},
		{Member: "m-gone", Viewing: watchsvc.Viewing{Path: "G/Bob/screen", Transport: "hls"}},
	}
	members := []wire.Member{{MemberID: "m-alice", DisplayName: "Alice"}, {MemberID: "m-bob", DisplayName: "Bob"}}

	landed := attachViewers(status, "Bob/screen", viewers, members)

	got := landed.Paths[0].Viewers
	if len(got) != 2 || got[0].DisplayName != "Alice" || got[1].DisplayName != "" || got[1].Member != "m-gone" {
		t.Errorf("Bob's path carries %+v, want Alice named and m-gone unnamed", got)
	}
	if len(landed.Paths[1].Viewers) != 0 {
		t.Errorf("Carol's path carries %+v, want nothing: watchd answers the publisher of one path", landed.Paths[1].Viewers)
	}
	if len(status.Paths[0].Viewers) != 0 {
		t.Error("attaching viewers wrote into the status it was handed")
	}
}

func TestViewersOfAPathTheRelayDoesNotListGoNowhere(t *testing.T) {
	status := relay.Status{Reachable: true, Paths: []relay.Path{{Name: "Carol/game"}}}
	viewers := []watchsvc.Viewer{{Member: "m-alice", Viewing: watchsvc.Viewing{Path: "G/Bob/screen", Transport: "rtsp"}}}

	landed := attachViewers(status, "Bob/screen", viewers, nil)

	if len(landed.Paths) != 1 || len(landed.Paths[0].Viewers) != 0 {
		t.Errorf("viewers of an unlisted path landed as %+v", landed.Paths)
	}
}

func TestAStatementWaitsOutTheIntervalUnlessItEndsTheViewing(t *testing.T) {
	stated := time.Unix(1_800_000_000, 0)
	decoding := []wire.ReceiveStreamStats{{Stream: wire.StreamRef{StreamName: "Bob/screen", Transport: "rtsp"}}}

	if shouldStateViewing(decoding, stated, stated.Add(viewingInterval/2)) {
		t.Error("a sample inside the interval went out")
	}
	if !shouldStateViewing(decoding, stated, stated.Add(viewingInterval)) {
		t.Error("a sample a whole interval later stayed back")
	}
	if !shouldStateViewing(nil, stated, stated.Add(viewingInterval/2)) {
		t.Error("the empty sample ending the viewing waited for the interval")
	}
}

func TestAWayHereWithNoBufferingReadingIsThePathAlone(t *testing.T) {
	path := 41.5
	if way := wayHereOf(wire.DelayBudget{Path: &path}); way == nil || *way != path {
		t.Errorf("way here = %v, want the path", way)
	}
	if way := wayHereOf(wire.DelayBudget{}); way != nil {
		t.Errorf("way here of an unstamped stream = %v, want it absent", *way)
	}
}
