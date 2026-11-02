package control

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/events"
	"bjoernblessin.de/screenshare/internal/text"
)

// Read, Watch, Ask, Set and Do: each picks the named function for what the request names,
// and decides nothing beyond which one.
// Hello sits in hello.go.

// Read hands over every named state, whole, in the order asked.
//
// A kind that happens rather than stands is INVALID_ARGUMENT, and so is an empty list:
// a request that failed to say what it wanted asked for nothing.
func (s *Server) Read(ctx context.Context, req *screensharev1.ReadRequest) (*screensharev1.ReadResponse, error) {
	kinds := req.GetKinds()
	if len(kinds) == 0 {
		return nil, invalidArgument(text.Fail(readNamesNothing))
	}

	answer := &screensharev1.ReadResponse{States: make([]*screensharev1.Event, 0, len(kinds))}
	for _, kind := range kinds {
		build, ok := states[kind]
		if !ok {
			return nil, invalidArgument(text.Fail(kindNotReadable, argKind(kind)))
		}
		answer.States = append(answer.States, build(s))
	}

	assert.Assert(len(answer.GetStates()) == len(kinds), "a read answers every state it names", len(kinds))
	return answer, nil
}

// Watch carries what changed, for as long as the shell holds the call.
//
// The stream the other methods lean on:
// a Set answers empty because the resulting state arrives here,
// and Read exists so a shell that has just mounted starts from the picture this keeps current.
// Every event carries a whole state, so a duplicate is harmless,
// and a dropped connection is recovered from by reading again (docs/ipc-api.md, "Events").
//
// An unknown kind is refused rather than ignored,
// with INVALID_ARGUMENT naming the kind and listing the ones this build carries.
// Ignored, it would leave the shell holding an open stream that never delivers,
// reading as a backend where nothing is happening rather than as a name got wrong.
// A kind Read alone serves is refused the same way.
//
// The broker's events carry the sequence it stamps per subscription.
// A tick carries none: each is the present, and nothing about the previous one is worth counting.
func (s *Server) Watch(req *screensharev1.WatchRequest, out grpc.ServerStreamingServer[screensharev1.Event]) error {
	assert.IsNotNil(out, "a watch writes to the client's stream")

	announced, levels, pointer, err := splitKinds(req.GetKinds())
	if err != nil {
		return err
	}

	var feed <-chan *screensharev1.Event
	if len(req.GetKinds()) == 0 || len(announced) > 0 {
		subscribed, cancel, err := s.events.Subscribe(announced)
		if err != nil {
			return refusedKind(err)
		}
		// The broker sends on a channel it holds until the cancel removes it,
		// so a call that returned without cancelling would leave a subscriber nothing reads.
		defer cancel()
		feed = subscribed
	}

	var levelTick <-chan time.Time
	if levels {
		ticker := time.NewTicker(levelCadence)
		defer ticker.Stop()
		levelTick = ticker.C
	}

	var pointerTick <-chan time.Time
	var spot *pointerFeed
	if pointer {
		spot = s.pointerOf(req.GetPointerOf())
		ticker := time.NewTicker(spot.cadence)
		defer ticker.Stop()
		pointerTick = ticker.C
	}

	// A ticker's channel drops the ticks a blocked Send ran past,
	// so a slow reader receives the present rather than a queue of the past.
	done := out.Context().Done()
	for {
		var event *screensharev1.Event
		select {
		case <-done:
			// A client gone or one that stopped listening: a window closed, a process quit.
			// A watch ends that way normally, so no failure is reported.
			return nil
		case announcement, open := <-feed:
			if !open {
				// The channel closes when the subscription is released:
				// on this path, the backend shutting the broker down under an open call.
				return nil
			}
			event = announcement
		case <-levelTick:
			event = s.levelsEvent()
		case <-pointerTick:
			moved, ok := spot.next()
			if !ok {
				continue
			}
			event = moved
		}

		if err := out.Send(event); err != nil {
			// A failed send is the transport reporting the client gone.
			// The error carries a status already, so it goes back unwrapped rather than reclassified.
			return err
		}
	}
}

// splitKinds sorts a watch's kinds into the ones the broker announces and the two ticks.
// Empty asks for every announced kind and no tick.
func splitKinds(kinds []screensharev1.EventKind) (announced []screensharev1.EventKind, levels, pointer bool, err error) {
	for _, kind := range kinds {
		switch {
		case readAlone[kind]:
			return nil, false, false, invalidArgument(text.Fail(kindNotWatchable, argKind(kind)))
		case kind == screensharev1.EventKind_EVENT_KIND_AUDIO_LEVELS:
			levels = true
		case kind == screensharev1.EventKind_EVENT_KIND_POINTER:
			pointer = true
		default:
			announced = append(announced, kind)
		}
	}
	return announced, levels, pointer, nil
}

// refusedKind words the broker's one refusal for the caller.
func refusedKind(err error) error {
	var unknown *events.UnknownKindError
	if !errors.As(err, &unknown) {
		// The broker refuses a subscription for one reason, in one type.
		// A second reason reaching here is a change to the broker this method was never told about.
		assert.Never("a subscription is refused only for a kind this build has none of", err)
	}
	return invalidArgument(text.Fail(kindUnknown, argKind(unknown.Kind),
		text.IDs(screensharev1.TextArgName_TEXT_ARG_NAME_KINDS, events.KindNames())))
}

// Ask answers one question about its arguments and changes nothing.
func (s *Server) Ask(ctx context.Context, req *screensharev1.AskRequest) (*screensharev1.AskResponse, error) {
	switch question := req.GetQuestion().(type) {
	case *screensharev1.AskRequest_Form:
		return &screensharev1.AskResponse{Answer: &screensharev1.AskResponse_Form{Form: s.resolveForm(question.Form)}}, nil
	case *screensharev1.AskRequest_Link:
		stream, err := s.resolveLink(question.Link)
		if err != nil {
			return nil, err
		}
		return &screensharev1.AskResponse{Answer: &screensharev1.AskResponse_StreamName{StreamName: stream}}, nil
	default:
		// Off the wire, so possibly from a shell built against a later minor: the caller's to fix.
		return nil, invalidArgument(text.Fail(requestNotInThisBuild))
	}
}

// Set makes the named state true, and succeeds where it already holds.
func (s *Server) Set(ctx context.Context, req *screensharev1.SetRequest) (*emptypb.Empty, error) {
	var err error
	switch state := req.GetState().(type) {
	case *screensharev1.SetRequest_Settings:
		err = s.saveSettings(state.Settings)
	case *screensharev1.SetRequest_Preset:
		err = s.setPreset(state.Preset)
	case *screensharev1.SetRequest_Publish:
		err = s.setPublish(state.Publish)
	case *screensharev1.SetRequest_Viewer:
		err = s.setViewer(state.Viewer)
	case *screensharev1.SetRequest_Receive:
		err = s.setReceive(state.Receive)
	case *screensharev1.SetRequest_ReceiveAudio:
		err = s.setReceiveAudio(state.ReceiveAudio)
	case *screensharev1.SetRequest_MonitorPreview:
		err = s.setMonitorPreview(state.MonitorPreview)
	case *screensharev1.SetRequest_TestStreams:
		err = s.setTestStreams(int(state.TestStreams))
	case *screensharev1.SetRequest_DiscordLink:
		err = s.setDiscordLink(ctx, state.DiscordLink)
	default:
		err = invalidArgument(text.Fail(requestNotInThisBuild))
	}
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Do runs one effect and answers what it leaves with its caller.
func (s *Server) Do(ctx context.Context, req *screensharev1.DoRequest) (*screensharev1.DoResponse, error) {
	switch action := req.GetAction().(type) {
	case *screensharev1.DoRequest_OpenInBrowser:
		return done(s.openInBrowser(action.OpenInBrowser))
	case *screensharev1.DoRequest_ProbeEncoders:
		s.probeEncoders(ctx)
		return &screensharev1.DoResponse{}, nil
	case *screensharev1.DoRequest_MeasureUplink:
		mbps, err := s.measureUplink(ctx)
		if err != nil {
			return nil, err
		}
		return &screensharev1.DoResponse{Outcome: &screensharev1.DoResponse_UplinkMbps{UplinkMbps: mbps}}, nil
	case *screensharev1.DoRequest_MeasureEncodeRate:
		rate, err := s.measureEncodeRate(ctx, action.MeasureEncodeRate)
		if err != nil {
			return nil, err
		}
		return &screensharev1.DoResponse{Outcome: &screensharev1.DoResponse_EncodeRate{EncodeRate: rate}}, nil
	case *screensharev1.DoRequest_CheckRelay:
		check, err := s.checkRelay(ctx, action.CheckRelay)
		if err != nil {
			return nil, err
		}
		return &screensharev1.DoResponse{Outcome: &screensharev1.DoResponse_RelayCheck{RelayCheck: check}}, nil
	case *screensharev1.DoRequest_CheckUpdate:
		return done(s.checkUpdate())
	case *screensharev1.DoRequest_InstallUpdate:
		return done(s.installUpdate())
	case *screensharev1.DoRequest_ForgetPortalConsent:
		return done(s.forgetPortalConsent())
	case *screensharev1.DoRequest_CreateGroup:
		group, err := s.createGroup(action.CreateGroup)
		if err != nil {
			return nil, err
		}
		return &screensharev1.DoResponse{Outcome: &screensharev1.DoResponse_Group{Group: group}}, nil
	case *screensharev1.DoRequest_OpenLog:
		return done(s.openLog(action.OpenLog))
	case *screensharev1.DoRequest_OpenLogsFolder:
		return done(s.openLogsFolder())
	case *screensharev1.DoRequest_SendReport:
		id, err := s.sendReport()
		if err != nil {
			return nil, err
		}
		return &screensharev1.DoResponse{Outcome: &screensharev1.DoResponse_ReportId{ReportId: id}}, nil
	default:
		return nil, invalidArgument(text.Fail(requestNotInThisBuild))
	}
}

// done answers an effect that leaves its caller nothing.
func done(err error) (*screensharev1.DoResponse, error) {
	if err != nil {
		return nil, err
	}
	return &screensharev1.DoResponse{}, nil
}
