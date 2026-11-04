package capabilities

import (
	"strconv"
	"testing"
)

// radeonsiPreset names the encoder preset Mesa's radeonsi decodes a target-usage value into
// (src/gallium/frontends/va/encode.c): bits 1 and 2 pick the preset,
// and the value 1 alone turns on pre-encoding and VBAQ beside balance.
func radeonsiPreset(value string) string {
	v, err := strconv.Atoi(value)
	if err != nil {
		return "not a number"
	}
	if v == 1 {
		return "balance with pre-encode"
	}
	return [...]string{"speed", "balance", "quality", "high quality"}[(v>>1)&3]
}

// oneVPL counts 1 at quality and 7 at speed, and radeonsi reads the figure as a bitfield.
// Written as counted, the default step 4 lands on the slower quality preset (radeonsiTargetUsages).
func TestRadeonsiReadsEachStepAsTheMatchingPreset(t *testing.T) {
	radeonsi := Device{Driver: "radeonsi"}
	want := map[string]string{
		"1": "high quality",
		"2": "quality", "3": "quality",
		"4": "balance", "5": "balance", "6": "balance", "7": "balance",
	}
	for _, name := range []string{"h264_vaapi", "hevc_vaapi", "av1_vaapi"} {
		c := mustGet(t, name)
		for _, step := range c.Effort.Steps {
			spelled := c.Effort.SpelledFor(radeonsi, step)
			if got := radeonsiPreset(spelled); got != want[step] {
				t.Errorf("%s step %s reaches radeonsi as %s, which picks %s, want %s",
					name, step, spelled, got, want[step])
			}
		}
	}
}

// A driver the table names no spelling for reads the step as oneVPL counts it.
func TestOtherDriversReadTheStepAsWritten(t *testing.T) {
	c := mustGet(t, "h264_vaapi")
	for _, d := range []Device{{}, {Driver: "iHD"}} {
		for _, step := range c.Effort.Steps {
			if got := c.Effort.SpelledFor(d, step); got != step {
				t.Errorf("driver %q reads step %s as %s", d.Driver, step, got)
			}
		}
	}
}
