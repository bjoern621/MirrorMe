package receive

import "testing"

// TestADecodeOpensMuted holds the loudness a decode starts at: muted, at unity gain.
// A stream somebody opened has not been asked to play yet,
// and unmuting returns it to the level it arrived with.
func TestADecodeOpensMuted(t *testing.T) {
	r, err := New(Stream{Name: "opens-muted", Source: "videotestsrc is-live=true", Raw: true}, Open{}, Events{})
	if err != nil {
		t.Fatalf("the test pipeline does not build: %v", err)
	}
	defer r.Stop()

	volume, muted, _ := r.Audio()
	if !muted || volume != 1 {
		t.Errorf("a fresh decode plays at %v muted=%v, expected 1 muted=true", volume, muted)
	}
}
