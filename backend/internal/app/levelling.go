package app

import (
	"maps"
	"slices"

	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/go-utils/util/logger"

	"bjoernblessin.de/mirrorme/internal/decode"
)

// levelReceivers writes the constant-volume setting onto every open decode.
// Repeatable: each decode is handed the state wanted, whatever it held before.
func (a *App) levelReceivers(on bool) {
	a.procMu.Lock()
	receivers := slices.Collect(maps.Values(a.receivers))
	a.procMu.Unlock()

	for _, receiver := range receivers {
		a.levelReceiver(receiver, on)
	}
}

// levelReceiver writes the setting onto one decode.
// A decode that ended meanwhile refuses, and the next start levels its replacement.
func (a *App) levelReceiver(receiver *decode.Handle, on bool) {
	assert.IsNotNil(receiver, "a decode to level")

	if err := receiver.SetLevelling(on); err != nil {
		logger.Warnf("constant volume not set on a decode: %v", err)
	}
}
