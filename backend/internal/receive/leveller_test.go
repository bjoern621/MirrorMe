package receive

import (
	"math"
	"testing"
	"time"
)

// feed hands a leveller the same measurement for a stretch of time, at the level element's rate.
func feed(l *leveller, peakDB, rmsDB float64, stretch time.Duration) {
	for range int(stretch / LevelInterval) {
		l.Measure(peakDB, rmsDB)
	}
}

// A loud stream and a quiet one leave at one loudness, the way a music service levels its tracks.
func TestALoudAndAQuietStreamLeaveAtOneLoudness(t *testing.T) {
	var loud, quiet leveller
	feed(&loud, -6, -14, 10*time.Second)
	feed(&quiet, -18, -30, 10*time.Second)

	loudOut := -14 + loud.GainDB()
	quietOut := -30 + quiet.GainDB()
	if math.Abs(loudOut-quietOut) > 1 {
		t.Errorf("the loud stream leaves at %.1f dBFS and the quiet one at %.1f, want them within 1 dB", loudOut, quietOut)
	}
}

func TestAFreshLevellerPlaysAtUnity(t *testing.T) {
	var l leveller
	if l.GainDB() != 0 {
		t.Errorf("a leveller that measured nothing applies %.1f dB, want 0", l.GainDB())
	}
}

// A pause in the talk is not a quiet stream: turning it up would blast the next word.
func TestSilenceMovesNothing(t *testing.T) {
	var l leveller
	feed(&l, -18, -30, 10*time.Second)
	before := l.GainDB()
	feed(&l, -80, -90, 10*time.Second)

	if math.Abs(l.GainDB()-before) > 0.1 {
		t.Errorf("ten seconds of silence moved the gain from %.1f to %.1f dB", before, l.GainDB())
	}
}

func TestABoostStopsAtItsCeiling(t *testing.T) {
	var l leveller
	feed(&l, -40, -48, 10*time.Second)

	if l.GainDB() > levelMaxBoostDB {
		t.Errorf("a near-silent stream is boosted %.1f dB, want at most %v", l.GainDB(), float64(levelMaxBoostDB))
	}
}

// A stream with sharp peaks over a quiet body is boosted only as far as its peaks stay under full scale.
func TestABoostKeepsThePeaksUnderFullScale(t *testing.T) {
	var l leveller
	feed(&l, -3, -30, 10*time.Second)

	if peak := -3 + l.GainDB(); peak > levelPeakCeilingDB+0.01 {
		t.Errorf("the boosted peak lands at %.1f dBFS, want at most %v", peak, float64(levelPeakCeilingDB))
	}
}

// The gain follows a change in the stream over seconds, so a loud scene never jumps the volume.
func TestTheGainMovesOverSecondsAndNotInOneStep(t *testing.T) {
	var l leveller
	feed(&l, -18, -30, 10*time.Second)
	before := l.GainDB()
	l.Measure(-1, -6)

	if step := before - l.GainDB(); step > 3 {
		t.Errorf("one loud measurement cut the gain by %.1f dB at once, want a gradual move", step)
	}
}
