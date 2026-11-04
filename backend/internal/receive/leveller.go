package receive

import (
	"time"

	"bjoernblessin.de/go-utils/util/assert"
)

// The constant-volume target and its limits, in dB against full scale.
const (
	// levelTargetDB is the power average every levelled stream is brought to.
	// The same for every decode, so two streams at one slider position play at about one loudness.
	levelTargetDB = -20
	// levelGateDB is the power average below which a measurement counts as silence and moves nothing,
	// so a pause is never turned up into a blast on the next sound.
	levelGateDB = -50
	// levelMaxBoostDB and levelMaxCutDB bound the gain, a near-silent stream staying near-silent.
	levelMaxBoostDB = 15
	levelMaxCutDB   = -30
	// levelPeakCeilingDB is the highest a boosted peak may land, short of full scale and its clipping.
	levelPeakCeilingDB = -1
	// levelPeakDecayDBPerSecond is how fast the held peak falls back after a loud moment.
	levelPeakDecayDBPerSecond = 3
)

// How fast the measured loudness follows the stream: quickly when it gets louder,
// so a loud scene is turned down before it hurts, and slowly when it gets quieter,
// so a soft passage is not pumped up between two loud ones.
const (
	levelAttackDuration  = time.Second
	levelReleaseDuration = 4 * time.Second
)

// leveller turns one stream's measured loudness into the gain that brings it to levelTargetDB.
// It is fed the level element's readings at LevelInterval and holds no reference to the pipeline.
type leveller struct {
	// loudnessDB is the smoothed power average of the measurements above the gate.
	loudnessDB float64
	// peakDB follows the loudest samples, rising over levelAttackDuration and falling back at
	// levelPeakDecayDBPerSecond, so one sharp sample moves the gain as little as one loud reading does.
	peakDB float64
	// primed is whether a measurement above the gate has arrived, unity gain applying until one has.
	primed bool
}

// Measure takes one reading of the level element: the loudest sample and the power average, in dBFS.
func (l *leveller) Measure(peakDB, rmsDB float64) {
	if rmsDB < levelGateDB {
		return
	}
	if !l.primed {
		l.loudnessDB, l.peakDB, l.primed = rmsDB, peakDB, true
		return
	}

	attack := LevelInterval.Seconds() / levelAttackDuration.Seconds()
	if peakDB > l.peakDB {
		l.peakDB += attack * (peakDB - l.peakDB)
	} else {
		l.peakDB = max(peakDB, l.peakDB-levelPeakDecayDBPerSecond*LevelInterval.Seconds())
	}

	window := levelReleaseDuration
	if rmsDB > l.loudnessDB {
		window = levelAttackDuration
	}
	share := LevelInterval.Seconds() / window.Seconds()
	assert.Assert(share > 0 && share <= 1, "one reading moves the loudness by a share of the gap", share)

	l.loudnessDB += share * (rmsDB - l.loudnessDB)
}

// GainDB is the gain that brings the stream to the target, 0 until a sound has been measured.
func (l *leveller) GainDB() float64 {
	if !l.primed {
		return 0
	}

	gain := min(max(levelTargetDB-l.loudnessDB, levelMaxCutDB), levelMaxBoostDB)
	gain = min(gain, levelPeakCeilingDB-l.peakDB)

	assert.Assert(gain <= levelMaxBoostDB, "the gain never boosts past its ceiling", gain)
	return gain
}
