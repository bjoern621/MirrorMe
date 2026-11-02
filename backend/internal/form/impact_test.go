package form

import (
	"slices"
	"strconv"
	"testing"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/transport"
)

func impactTestCodes(o *screensharev1.FieldOption) []screensharev1.TextCode {
	codes := make([]screensharev1.TextCode, 0, len(o.GetImpacts()))
	for _, w := range o.GetImpacts() {
		codes = append(codes, w.GetText().GetCode())
	}
	return codes
}

func TestAPixelFormatNoHardwareDecodesCarriesTheViewersCost(t *testing.T) {
	chroma := priceTestField(t, Resolve(estimateTestDeps(), estimateTestStream()), KeyChroma)

	light := impactTestCodes(priceTestOption(t, chroma, "yuv420p"))
	if !slices.Equal(light, []screensharev1.TextCode{screensharev1.TextCode_TEXT_CODE_DECODES_IN_HARDWARE_EVERYWHERE}) {
		t.Errorf("4:2:0 decodes in hardware everywhere and carries %v", light)
	}
	full := priceTestOption(t, chroma, "yuv444p")
	if !full.GetEnabled() {
		t.Fatal("the fixture rules 4:4:4 out, so nothing below was tested")
	}
	if codes := impactTestCodes(full); len(codes) == 0 {
		t.Error("H.264 4:4:4 leaves viewers decoding on their cores and carries nothing saying so")
	}
}

func TestARateAboveTheRefreshCarriesTheRepeatedFrames(t *testing.T) {
	fps := priceTestField(t, Resolve(estimateTestDeps(), estimateTestStream()), KeyFps)

	for _, o := range fps.GetOptions() {
		if rate, _ := strconv.Atoi(o.GetValue()); !o.GetEnabled() || rate <= 60 {
			continue
		}
		if !impactTestHas(o, screensharev1.TextCode_TEXT_CODE_FPS_ABOVE_REFRESH) {
			t.Errorf("%s fps sits above the 60 Hz panel and carries %v", o.GetValue(), impactTestCodes(o))
		}
	}
	if codes := impactTestCodes(priceTestOption(t, fps, "30")); len(codes) != 0 {
		t.Errorf("30 fps raises nothing the other rates do not and carries %v", codes)
	}
}

func TestADiagnosticEveryEntryRaisesSitsOnNone(t *testing.T) {
	s := estimateTestStream()
	s.Publish.UplinkMbps = 0
	form := Resolve(estimateTestDeps(), s)

	for _, o := range priceTestField(t, form, KeyFps).GetOptions() {
		if impactTestHas(o, screensharev1.TextCode_TEXT_CODE_NO_UPLINK_STATED) {
			t.Errorf("%s fps carries the missing uplink, which every rate raises alike", o.GetValue())
		}
	}
}

func TestARuledOutEntryCarriesNoImpact(t *testing.T) {
	form := Resolve(estimateTestDeps(), estimateTestStream())

	for _, g := range form.GetGroups() {
		for _, f := range g.GetFields() {
			for _, o := range f.GetOptions() {
				if !o.GetEnabled() && len(o.GetImpacts()) != 0 {
					t.Errorf("%s %q is ruled out and carries %v", f.GetKey(), o.GetValue(), impactTestCodes(o))
				}
			}
		}
	}
}

func impactTestHas(o *screensharev1.FieldOption, code screensharev1.TextCode) bool {
	for _, w := range o.GetImpacts() {
		if w.GetText().GetCode() == code {
			return true
		}
	}
	return false
}

// Every watch protocol states how far behind its publisher a stream plays on it,
// the figure the watch panel prints on its card.
func TestEachWatchProtocolStatesHowFarBehindItPlays(t *testing.T) {
	protocols := priceTestField(t, Resolve(estimateTestDeps(), estimateTestStream()), KeyTileWatchTransport)

	for _, o := range protocols.GetOptions() {
		want, ok := transport.PathDelay(o.GetValue())
		if !ok {
			t.Errorf("watch protocol %q has no measured delay", o.GetValue())
			continue
		}
		var got *screensharev1.Diagnostic
		for _, w := range o.GetImpacts() {
			if w.GetText().GetCode() == screensharev1.TextCode_TEXT_CODE_WATCH_DELAY {
				got = w
			}
		}
		if got == nil {
			t.Errorf("%s carries %v and no delay", o.GetValue(), impactTestCodes(o))
			continue
		}
		if ms := got.GetText().GetArgs()[0].GetNumber(); ms != want.Milliseconds() {
			t.Errorf("%s states %d ms behind, want %d", o.GetValue(), ms, want.Milliseconds())
		}
	}
}
