package control

import (
	"slices"

	mirrormev1"bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// How each EventKind crosses the contract, one table for Read and Watch both.
//
// A kind in states stands, so Read hands it over as it is now and Watch opens on it.
// A kind in readAlone stands and is never announced: nothing tells this process when it moved.
// AUDIO_LEVELS and POINTER move at a cadence of their own, so Watch sends them only where named (splitKinds).
// Every other kind the broker knows happens rather than stands, and Watch alone carries it.

// states builds every kind Read answers, each as the whole Event Watch carries it in.
var states = map[mirrormev1.EventKind]func(*Server) *mirrormev1.Event{
	mirrormev1.EventKind_EVENT_KIND_PUBLISH_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_PublishState{PublishState: s.publishState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_RELAY_STATUS: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_RelayStatus{RelayStatus: s.relayStatus()}}
	},
	mirrormev1.EventKind_EVENT_KIND_VIEWER_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_ViewerState{ViewerState: s.viewerState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_TEST_STREAM_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_TestStreamState{TestStreamState: s.testStreamState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_CATALOG: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_Catalog{Catalog: s.catalog()}}
	},
	mirrormev1.EventKind_EVENT_KIND_RECEIVE_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_ReceiveState{ReceiveState: s.receiveState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_MONITOR_PREVIEW_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{
			Payload: &mirrormev1.Event_MonitorPreviewState{MonitorPreviewState: s.monitorPreviewState()},
		}
	},
	mirrormev1.EventKind_EVENT_KIND_MEMBERS_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_MembersState{MembersState: s.membersState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_DISCORD_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_DiscordState{DiscordState: s.discordState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_UPDATE_STATE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_UpdateState{UpdateState: s.updateState()}}
	},
	mirrormev1.EventKind_EVENT_KIND_STREAM_TIMELINE: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_StreamTimeline{StreamTimeline: s.streamTimeline()}}
	},
	mirrormev1.EventKind_EVENT_KIND_SETTINGS: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_Settings{Settings: s.storedSettings()}}
	},
	mirrormev1.EventKind_EVENT_KIND_PRESETS: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_Presets{Presets: s.presets()}}
	},
	mirrormev1.EventKind_EVENT_KIND_SHARE_WINDOWS: func(s *Server) *mirrormev1.Event {
		return &mirrormev1.Event{Payload: &mirrormev1.Event_ShareWindows{ShareWindows: s.shareWindows()}}
	},
}

var readAlone = map[mirrormev1.EventKind]bool{
	mirrormev1.EventKind_EVENT_KIND_SETTINGS:      true,
	mirrormev1.EventKind_EVENT_KIND_PRESETS:       true,
	mirrormev1.EventKind_EVENT_KIND_SHARE_WINDOWS: true,
}

// standingKinds lists the kinds a watch opens on, in enum order:
// each announced kind that stands, or every one that stands where announced is empty.
func standingKinds(announced []mirrormev1.EventKind) []mirrormev1.EventKind {
	var kinds []mirrormev1.EventKind
	for kind := range states {
		if !readAlone[kind] && (len(announced) == 0 || slices.Contains(announced, kind)) {
			kinds = append(kinds, kind)
		}
	}
	slices.Sort(kinds)
	return kinds
}
