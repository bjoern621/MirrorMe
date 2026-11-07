package form

import (
	"slices"
	"testing"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/platform"
)

// Lossless on the GStreamer NVIDIA H.264 encoder holds at 4:4:4 alone,
// so picking it moves the chroma and keeps the mode the user picked.
func TestLosslessOnGstNvh264encMovesTheChromaTo444(t *testing.T) {
	deps := Deps{Platform: platform.Info{OS: "linux", Display: "wayland"}}
	draft := availabilityDraft("portal", "h264_nvenc", "yuv420p", "rtsp")
	draft.Publish.Mode = capabilities.ModeLossless

	s, repaired := Repair(deps, draft)

	if s.Publish.Mode != capabilities.ModeLossless {
		t.Errorf("the mode moved to %q", s.Publish.Mode)
	}
	if s.Publish.Chroma != "yuv444p" {
		t.Errorf("the chroma stayed on %q, want yuv444p", s.Publish.Chroma)
	}
	if !slices.Contains(repaired, KeyChroma) {
		t.Errorf("the repaired list %v leaves out the chroma", repaired)
	}
}
