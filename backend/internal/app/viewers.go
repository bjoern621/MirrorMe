package app

import (
	"slices"

	"bjoernblessin.de/go-utils/util/logger"

	"bjoernblessin.de/mirrorme/internal/relay"
	"bjoernblessin.de/mirrorme/internal/watchsvc"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// withViewers puts watchd's answer on the path this machine publishes, and hands status back unchanged
// where nothing is published or watchd does not answer.
// Rides the relay poll, so the table moves at the snapshot's cadence.
func (a *App) withViewers(status relay.Status) relay.Status {
	a.procMu.Lock()
	published, path := "", ""
	if a.run != nil {
		published, path = a.run.settings.StreamName(), a.run.settings.PublishPath()
	}
	a.procMu.Unlock()
	if published == "" {
		return status
	}

	a.settingsMu.Lock()
	s := a.settings
	a.settingsMu.Unlock()

	base, ok := s.Relay.ViewerService()
	if !ok {
		return status
	}
	s, err := a.settingsForCommand(s)
	if err != nil || s.Relay.Token == "" {
		logger.Debugf("no token to read who watches %s: %v", published, err)
		return status
	}
	viewers, err := a.viewerSvc.Viewers(base, s.Relay.Token, path)
	if err != nil {
		logger.Debugf("watchd did not say who watches %s: %v", published, err)
		return status
	}
	return attachViewers(status, published, viewers, a.membership().Members)
}

// attachViewers answers a copy of status whose path named published carries viewers,
// each under the name members lists it by and the empty name where members lists no such member.
func attachViewers(status relay.Status, published string, viewers []watchsvc.Viewer, members []wire.Member) relay.Status {
	names := make(map[string]string, len(members))
	for _, m := range members {
		names[m.MemberID] = m.DisplayName
	}

	status.Paths = slices.Clone(status.Paths)
	for i, path := range status.Paths {
		if path.Name != published {
			continue
		}
		named := make([]relay.Viewer, 0, len(viewers))
		for _, v := range viewers {
			named = append(named, relay.Viewer{DisplayName: names[v.Member], Viewer: v})
		}
		status.Paths[i].Viewers = named
	}
	return status
}
