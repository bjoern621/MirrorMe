package form

import (
	"slices"
	"testing"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// positiveTestFind is the first diagnostic under code, nil where the settings raise none.
func positiveTestFind(s settings.Settings, code mirrormev1.TextCode) *mirrormev1.Diagnostic {
	for _, w := range diagnosticsOfTheSettings(diagnosticTestDeps(), s, nil) {
		if w.GetText().GetCode() == code {
			return w
		}
	}
	return nil
}

func positiveTestNumber(w *mirrormev1.Diagnostic, name mirrormev1.TextArgName) int64 {
	for _, arg := range w.GetText().GetArgs() {
		if arg.GetName() == name {
			return arg.GetNumber()
		}
	}
	return -1
}

func TestAHardwareEncoderStatesItRunsOnTheGraphicsCard(t *testing.T) {
	gpu := diagnosticTestStream()
	gpu.Publish.UseCodec("h264_nvenc")
	w := positiveTestFind(gpu, mirrormev1.TextCode_TEXT_CODE_ENCODES_ON_GPU)
	if w == nil {
		t.Fatal("NVENC raises no statement about where it encodes")
	}
	if got := idOf(w.GetText(), mirrormev1.TextArgName_TEXT_ARG_NAME_FAMILY); got != "nvenc" {
		t.Errorf("the statement names family %q, want nvenc", got)
	}
	if w.GetFieldKey() != KeyEncoder {
		t.Errorf("anchored on %q, want %s", w.GetFieldKey(), KeyEncoder)
	}

	if positiveTestFind(diagnosticTestStream(), mirrormev1.TextCode_TEXT_CODE_ENCODES_ON_GPU) != nil {
		t.Error("x264 states it encodes on a graphics card")
	}
}

func TestAPictureEveryFamilyDecodesInHardwareSaysSo(t *testing.T) {
	if positiveTestFind(diagnosticTestStream(), mirrormev1.TextCode_TEXT_CODE_DECODES_IN_HARDWARE_EVERYWHERE) == nil {
		t.Error("H.264 4:2:0 raises no statement that every viewer decodes it in hardware")
	}

	full := diagnosticTestStream()
	full.Publish.Chroma = "yuv444p"
	if positiveTestFind(full, mirrormev1.TextCode_TEXT_CODE_DECODES_IN_HARDWARE_EVERYWHERE) != nil {
		t.Error("H.264 4:4:4 states every viewer decodes it in hardware")
	}
}

func TestARateEqualToTheRefreshSaysItMatchesTheScreen(t *testing.T) {
	if positiveTestFind(diagnosticTestStream(), mirrormev1.TextCode_TEXT_CODE_FPS_MATCHES_REFRESH) == nil {
		t.Error("60 fps on a 60 Hz screen raises no statement that it matches")
	}

	slow := diagnosticTestStream()
	slow.Publish.Fps = 30
	if positiveTestFind(slow, mirrormev1.TextCode_TEXT_CODE_FPS_MATCHES_REFRESH) != nil {
		t.Error("30 fps on a 60 Hz screen states it matches")
	}
}

// A viewer opening the stream waits for the next keyframe, at most one interval.
func TestTheKeyframeIntervalStatesHowLongAViewerWaitsToJoin(t *testing.T) {
	cases := map[string]struct {
		gop  int
		want int64
	}{
		"automatic, two seconds of frames": {gop: 0, want: 2000},
		"30 frames at 60 fps":              {gop: 30, want: 500},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s := diagnosticTestStream()
			s.Publish.Gop = c.gop
			w := positiveTestFind(s, mirrormev1.TextCode_TEXT_CODE_JOIN_WAIT)
			if w == nil {
				t.Fatal("the settings raise no join wait")
			}
			if got := positiveTestNumber(w, mirrormev1.TextArgName_TEXT_ARG_NAME_JOIN_WAIT_MS); got != c.want {
				t.Errorf("join wait %d ms, want %d", got, c.want)
			}
			if !slices.Contains(warningAnchors, w.GetFieldKey()) || w.GetFieldKey() != KeyGop {
				t.Errorf("anchored on %q, want %s", w.GetFieldKey(), KeyGop)
			}
		})
	}
}
