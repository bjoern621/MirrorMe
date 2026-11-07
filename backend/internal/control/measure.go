package control

import (
	"context"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/text"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// The measurements.
// Both run the real thing instead of predicting it, both take seconds,
// and both are refused while a stream is publishing.
//
// That refusal is the whole of this file.
// A measurement earns its seconds only where what it measures is idle:
// an uplink probe beside a live stream measures the line minus the stream,
// and an encoder timing beside a live encode measures the silicon minus the encode.
// Either answers with a figure that reads as a property of the machine and is one of the moment.
// A bitrate or a frame rate is then set from it.
// So the competition is the reason given,
// and the code is FAILED_PRECONDITION, in the terms of the contract's table:
// the request is well formed and the world is not ready for it.
//
// The publish state decides, which leaves a pipeline waiting out a retry backoff
// on the refused side.
// That is still the stream the user asked for and it returns by itself,
// so a measurement started in the gap would be running when it does.

// measureUplink probes this machine's real upload throughput,
// so a guessed uplink figure can be replaced with a measured one.
func (s *Server) measureUplink(ctx context.Context) (float64, error) {
	if s.backend.PublishState().Publishing() {
		return 0, failedPrecondition(text.Fail(uplinkMeasureWhileLive))
	}

	mbps, err := s.backend.MeasureUplink(ctx)
	if err != nil {
		return 0, fromBackend(err, uplinkUnmeasured)
	}
	return mbps, nil
}

// measureEncodeRate times the configured encoder on generated frames of the captured monitor's size,
// so a target frame rate can be held against what this machine encodes at these settings.
//
// The settings decide what is timed,
// so a request carrying none is refused by the gate Set and Do use (draftOf, in effects.go):
// the empty draft would time an encoder nobody chose,
// and the figure would read as an answer about the one they did.
func (s *Server) measureEncodeRate(ctx context.Context, m *mirrormev1.Settings) (*mirrormev1.EncodeRate, error) {
	draft, err := draftOf(m)
	if err != nil {
		return nil, err
	}

	if s.backend.PublishState().Publishing() {
		return nil, failedPrecondition(text.Fail(encodeRateMeasureWhileLive))
	}

	rate, err := s.backend.MeasureEncodeRate(ctx, draft)
	if err != nil {
		return nil, fromBackend(err, encodeRateUnmeasured)
	}
	return wire.EncodeRate(rate), nil
}
