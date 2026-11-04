package receive

import "testing"

// TestADecodeOpensMuted holds the loudness a decode starts at: muted, with the slider at half.
// A stream somebody opened has not been asked to play yet,
// and unmuting plays it at the level a fresh slider shows.
func TestADecodeOpensMuted(t *testing.T) {
	r, err := New(Stream{Name: "opens-muted", Source: "videotestsrc is-live=true", Raw: true}, Open{}, Events{})
	if err != nil {
		t.Fatalf("the test pipeline does not build: %v", err)
	}
	defer r.Stop()

	volume, muted, _ := r.Audio()
	if !muted || volume != 0.5 {
		t.Errorf("a fresh decode plays at %v muted=%v, expected 0.5 muted=true", volume, muted)
	}
}

// TestTheVolumeSliderIsCubic holds the curve between a slider position and the gain it plays at.
func TestTheVolumeSliderIsCubic(t *testing.T) {
	for _, c := range []struct{ position, gain float64 }{
		{0, 0},
		{0.5, 0.125},
		{1, 1},
	} {
		if got := gainAt(c.position); got != c.gain {
			t.Errorf("slider at %v plays at gain %v, expected %v", c.position, got, c.gain)
		}
	}
}
