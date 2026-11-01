package form

import (
	"slices"
	"strconv"

	"bjoernblessin.de/go-utils/util/assert"
	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/settings"
)

// optionDetailLadder holds the quantizer's detail steps on the anchor scale of 51 points, least detail first.
// Clean sits on the quantizer that the bitrate prediction is anchored at.
var optionDetailLadder = []struct {
	step string
	cq51 int
}{
	{"lighter", 28},
	{"clean", estimateAnchorCq},
	{"sharper", 18},
}

// optionCqSteps offers each detail step on the selected codec's scale (presetCq),
// plus the held quantizer wherever it sits off them, ordered least detail first.
//
// Each step names itself in its note:
// off the 51-point scale the number alone does not say which step it is.
// A scale too short to part two steps offers the first of them.
func optionCqSteps(_ Deps, s settings.Settings) []*screensharev1.FieldOption {
	codec, engine := s.Publish.Codec(), optionEngineOf(s)

	steps := make([]int, 0, len(optionDetailLadder)+1)
	notes := map[int]*screensharev1.Text{}
	for _, d := range optionDetailLadder {
		cq := presetCq(d.cq51, codec, engine)
		if _, taken := notes[cq]; taken {
			continue
		}
		steps = append(steps, cq)
		notes[cq] = say(detailStep, argDetailStep(d.step))
	}
	if !slices.Contains(steps, s.Publish.Cq) {
		steps = append(steps, s.Publish.Cq)
	}
	// A higher quantizer keeps less detail.
	slices.SortFunc(steps, func(a, b int) int { return b - a })

	out := make([]*screensharev1.FieldOption, 0, len(steps))
	for _, cq := range steps {
		out = append(out, optionEntry(strconv.Itoa(cq), notes[cq], false))
	}
	assert.Assert(slices.ContainsFunc(out, func(o *screensharev1.FieldOption) bool {
		return o.GetValue() == strconv.Itoa(s.Publish.Cq)
	}), "the ladder offers the quantizer the settings hold", s.Publish.Cq)
	return out
}
