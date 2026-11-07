package form

import (
	"testing"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// priceTestField finds one control of a resolved form by key, failing the test where the form draws none.
func priceTestField(t *testing.T, form *mirrormev1.Form, key string) *mirrormev1.Field {
	t.Helper()
	for _, g := range form.GetGroups() {
		for _, f := range g.GetFields() {
			if f.GetKey() == key {
				return f
			}
		}
	}
	t.Fatalf("the form draws no %s", key)
	return nil
}

func priceTestOption(t *testing.T, f *mirrormev1.Field, value string) *mirrormev1.FieldOption {
	t.Helper()
	for _, o := range f.GetOptions() {
		if o.GetValue() == value {
			return o
		}
	}
	t.Fatalf("%s offers no %q", f.GetKey(), value)
	return nil
}

func TestEveryFrameRateCarriesWhatPickingItCosts(t *testing.T) {
	// Detached, so the quantizer prices the rate: a preset holding a fixed bitrate prices every rate alike.
	s := estimateTestStream()
	s.Publish.Preset = ""
	form := Resolve(estimateTestDeps(), s)
	fps := priceTestField(t, form, KeyFps)

	for _, o := range fps.GetOptions() {
		if o.GetEnabled() && o.GetEstimate() == nil {
			t.Errorf("%s fps is pickable and carries no price", o.GetValue())
		}
	}

	held := priceTestOption(t, fps, "60").GetEstimate()
	if !estimateTestClose(held.GetBitrateMbps(), form.GetSummary().GetEstimate().GetBitrateMbps()) {
		t.Errorf("the rate already set prices at %v, the summary at %v",
			held.GetBitrateMbps(), form.GetSummary().GetEstimate().GetBitrateMbps())
	}
	if half := priceTestOption(t, fps, "30").GetEstimate(); half.GetBitrateMbps() >= held.GetBitrateMbps() {
		t.Errorf("30 fps prices at %v, not under 60 fps at %v", half.GetBitrateMbps(), held.GetBitrateMbps())
	}
}

func TestAControlThatMovesNoPriceCarriesNone(t *testing.T) {
	form := Resolve(estimateTestDeps(), estimateTestStream())

	for _, o := range priceTestField(t, form, KeyColorRange).GetOptions() {
		if o.GetEstimate() != nil {
			t.Errorf("colour range %q carries a price, and no range moves one", o.GetValue())
		}
	}
}

func TestARuledOutEntryCarriesNoPrice(t *testing.T) {
	form := Resolve(estimateTestDeps(), estimateTestStream())

	for _, g := range form.GetGroups() {
		for _, f := range g.GetFields() {
			for _, o := range f.GetOptions() {
				if !o.GetEnabled() && o.GetEstimate() != nil {
					t.Errorf("%s %q is ruled out and carries a price", f.GetKey(), o.GetValue())
				}
			}
		}
	}
}

func TestAReachablePresetCarriesThePriceOfWhatItProduces(t *testing.T) {
	d := estimateTestDeps()
	form := Resolve(d, estimateTestStream())

	reached := 0
	for _, p := range form.GetPresets() {
		if p.GetSettings() == nil {
			if p.GetEstimate() != nil {
				t.Errorf("%s reaches nothing here and carries a price", p.GetKey())
			}
			continue
		}
		reached++
		if p.GetEstimate() == nil {
			t.Errorf("%s is reachable and carries no price", p.GetKey())
		}
	}
	if reached == 0 {
		t.Fatal("the fixture reaches no preset, so nothing above was tested")
	}
}
