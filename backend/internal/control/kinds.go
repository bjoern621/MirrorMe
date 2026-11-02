package control

import (
	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
)

// How each EventKind crosses the contract, one table for Read and Watch both.
//
// A kind in states stands, so Read hands it over as it is now.
// A kind in readAlone stands and is never announced: nothing tells this process when it moved.
// A kind in ticked moves at a cadence of its own, so Watch sends it only where named.
// Every other kind the broker knows happens rather than stands, and Watch alone carries it.

// states builds every kind Read answers, each as the whole Event Watch carries it in.
var states = map[screensharev1.EventKind]func(*Server) *screensharev1.Event{
	screensharev1.EventKind_EVENT_KIND_PUBLISH_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_PublishState{PublishState: s.publishState()}}
	},
	screensharev1.EventKind_EVENT_KIND_RELAY_STATUS: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_RelayStatus{RelayStatus: s.relayStatus()}}
	},
	screensharev1.EventKind_EVENT_KIND_VIEWER_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_ViewerState{ViewerState: s.viewerState()}}
	},
	screensharev1.EventKind_EVENT_KIND_TEST_STREAM_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_TestStreamState{TestStreamState: s.testStreamState()}}
	},
	screensharev1.EventKind_EVENT_KIND_CATALOG: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_Catalog{Catalog: s.catalog()}}
	},
	screensharev1.EventKind_EVENT_KIND_RECEIVE_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_ReceiveState{ReceiveState: s.receiveState()}}
	},
	screensharev1.EventKind_EVENT_KIND_MONITOR_PREVIEW_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{
			Payload: &screensharev1.Event_MonitorPreviewState{MonitorPreviewState: s.monitorPreviewState()},
		}
	},
	screensharev1.EventKind_EVENT_KIND_MEMBERS_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_MembersState{MembersState: s.membersState()}}
	},
	screensharev1.EventKind_EVENT_KIND_DISCORD_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_DiscordState{DiscordState: s.discordState()}}
	},
	screensharev1.EventKind_EVENT_KIND_UPDATE_STATE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_UpdateState{UpdateState: s.updateState()}}
	},
	screensharev1.EventKind_EVENT_KIND_STREAM_TIMELINE: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_StreamTimeline{StreamTimeline: s.streamTimeline()}}
	},
	screensharev1.EventKind_EVENT_KIND_SETTINGS: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_Settings{Settings: s.storedSettings()}}
	},
	screensharev1.EventKind_EVENT_KIND_PRESETS: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_Presets{Presets: s.presets()}}
	},
	screensharev1.EventKind_EVENT_KIND_SHARE_WINDOWS: func(s *Server) *screensharev1.Event {
		return &screensharev1.Event{Payload: &screensharev1.Event_ShareWindows{ShareWindows: s.shareWindows()}}
	},
}

var readAlone = map[screensharev1.EventKind]bool{
	screensharev1.EventKind_EVENT_KIND_SETTINGS:      true,
	screensharev1.EventKind_EVENT_KIND_PRESETS:       true,
	screensharev1.EventKind_EVENT_KIND_SHARE_WINDOWS: true,
}

var ticked = map[screensharev1.EventKind]bool{
	screensharev1.EventKind_EVENT_KIND_AUDIO_LEVELS: true,
	screensharev1.EventKind_EVENT_KIND_POINTER:      true,
}
