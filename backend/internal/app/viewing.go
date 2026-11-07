package app

import (
	"time"

	"bjoernblessin.de/go-utils/util/logger"

	"bjoernblessin.de/mirrorme/internal/watchsvc"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// viewingInterval spaces this machine's statements to watchd at the relay poll's cadence.
// Each statement carries a token, and in Discord mode each token costs a trade with the manager.
// watchsvc.Lease covers several missed ones.
const viewingInterval = relayPollInterval

// viewerService is this app's side of watchd (internal/viewerclient).
type viewerService interface {
	State(base, token string, viewing []watchsvc.Viewing) error
	Viewers(base, token, path string) ([]watchsvc.Viewer, error)
}

// wayHereOf is encoder to decoder, the buffering row included.
// Absent on a stream carrying no stamp.
func wayHereOf(budget wire.DelayBudget) *float64 {
	if budget.Path == nil || budget.Arrive == nil {
		return budget.Path
	}
	way := *budget.Path + *budget.Arrive
	return &way
}

// viewingOf turns one receive sample into the set watchd takes, one viewing per running decode.
// pathOf puts the group's prefix back on a stream's name.
// An empty sample states an empty set, which clears this member at watchd.
func viewingOf(sample []wire.ReceiveStreamStats, pathOf func(streamName string) string) []watchsvc.Viewing {
	viewing := make([]watchsvc.Viewing, 0, len(sample))
	for _, s := range sample {
		viewing = append(viewing, watchsvc.Viewing{
			Path:       pathOf(s.Stream.StreamName),
			Transport:  s.Stream.Transport,
			PathMs:     wayHereOf(s.Delay),
			TotalMs:    s.Delay.Total,
			ShownFPS:   s.RenderFPS,
			DroppedFPS: s.DiscardedFPS,
		})
	}
	return viewing
}

// shouldStateViewing reports whether a sample goes to watchd:
// an empty one always, since it ends what this machine watched, and any other once per viewingInterval.
func shouldStateViewing(sample []wire.ReceiveStreamStats, lastStated, now time.Time) bool {
	return len(sample) == 0 || now.Sub(lastStated) >= viewingInterval
}

// stateViewing tells watchd what this machine decodes, off the stats loop.
//
// A statement still in flight drops this one.
// The next restates the whole set, so nothing is lost but a sample.
// Every failure goes to the debug log: a relay running no watchd costs its publishers the table and nothing else.
func (a *App) stateViewing(sample []wire.ReceiveStreamStats) {
	if !a.viewingInFlight.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer a.viewingInFlight.Store(false)

		a.settingsMu.Lock()
		s := a.settings
		a.settingsMu.Unlock()

		base, ok := s.Relay.ViewerService()
		if !ok {
			return
		}
		s, err := a.settingsForCommand(s)
		if err != nil || s.Relay.Token == "" {
			logger.Debugf("no token to state what this machine watches: %v", err)
			return
		}
		if err := a.viewerSvc.State(base, s.Relay.Token, viewingOf(sample, s.WatchPath)); err != nil {
			logger.Debugf("watchd did not take what this machine watches: %v", err)
		}
	}()
}
