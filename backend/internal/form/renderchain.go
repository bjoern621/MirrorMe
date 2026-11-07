package form

import (
	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/receive"
)

// renderChainReason states why this machine cannot render through a chain, nil for one it can.
//
// The one availability rule that asks the machine rather than a table:
// whether a chain runs is whether this GStreamer registers the element factories it is built from,
// which no compiled-in list can answer.
// The receive package asks, and this turns its answer into the chain
// and the first element it needs and does not have.
//
// A name no chain carries is not greyed.
// It is a settings file naming a chain this build dropped,
// which the repair moves off on the same resolve.
// Refusing it here would grey an entry no list offers.
func (av availability) renderChainReason(name string) *mirrormev1.Text {
	for _, c := range receive.Chains() {
		if c.Name != name || c.Available {
			continue
		}
		return say(renderChainElementMissing, argValue(name), argElement(c.MissingElement))
	}
	return nil
}
