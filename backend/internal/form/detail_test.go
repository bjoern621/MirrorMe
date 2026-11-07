package form

import (
	"strconv"
	"testing"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/capabilities"
)

// Each step sits where the anchor's step sits on the codec's own scale,
// so "clean" names the same picture on a 51-point scale and on a 63-point one.
func TestTheDetailLadderPlacesEachStepOnTheCodecsScale(t *testing.T) {
	anchor := map[string]int{"lighter": 28, "clean": 23, "sharper": 18}

	for _, codec := range []string{"libx264", "libvpx-vp9"} {
		s := estimateTestStream()
		s.Publish.UseCodec(codec)
		c, _ := capabilities.Get(codec)
		scale := c.CqMaxOn(capabilities.EngineFfmpeg)

		steps := map[string]int{}
		for _, o := range optionCqSteps(fieldTestDeps(), s) {
			if o.GetNote().GetCode() != mirrormev1.TextCode_TEXT_CODE_DETAIL_STEP {
				continue
			}
			cq, err := strconv.Atoi(o.GetValue())
			if err != nil {
				t.Fatalf("%s detail ladder offers %q, which is not a number", codec, o.GetValue())
			}
			steps[o.GetNote().GetArgs()[0].GetId()] = cq
		}

		for step, cq51 := range anchor {
			if want := cq51 * scale / 51; steps[step] != want {
				t.Errorf("%s puts %s at %d, %d on its %d-point scale", codec, step, steps[step], want, scale)
			}
		}
	}
}

// A quantizer typed off the ladder is offered all the same, so the closed control shows what the encode runs at.
func TestTheDetailLadderCarriesASavedQuantizerOffIt(t *testing.T) {
	s := estimateTestStream()
	s.Publish.Cq = 31

	for _, o := range optionCqSteps(fieldTestDeps(), s) {
		if o.GetValue() == "31" {
			return
		}
	}
	t.Error("the detail ladder drops a saved quantizer that is not one of its steps")
}

func TestEveryDetailStepCarriesWhatItCosts(t *testing.T) {
	s := estimateTestStream()
	s.Publish.Preset = ""
	cq := priceTestField(t, Resolve(estimateTestDeps(), s), KeyCq)

	if cq.GetControl() != mirrormev1.ControlKind_CONTROL_KIND_NUMBER_SELECT {
		t.Fatalf("the quantizer is a %v, so its steps are no shortcut beside a typed value", cq.GetControl())
	}
	for _, o := range cq.GetOptions() {
		if o.GetEnabled() && o.GetEstimate() == nil {
			t.Errorf("detail step %s is pickable and carries no price", o.GetValue())
		}
	}
}
