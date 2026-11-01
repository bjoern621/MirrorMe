package form

import (
	"bjoernblessin.de/go-utils/util/assert"
	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/wire"
)

// priceOptions writes onto each enabled entry of one control what the settings cost
// with that entry picked and nothing else moved (form.proto, FieldOption.estimate).
//
// A control whose entries all price the same keeps none:
// one figure repeated down a list says nothing about the choice it sits beside.
// Which controls price is therefore read off the estimate rather than listed,
// so a setting the model starts reading prices its entries with nothing here to edit.
func priceOptions(d Deps, s settings.Settings, key string, options []*screensharev1.FieldOption) {
	prices := make([]*screensharev1.Estimate, len(options))
	var first *screensharev1.Estimate
	differ := false

	for i, o := range options {
		if !o.GetEnabled() {
			continue
		}
		prices[i] = pricePicked(d, s, key, o.GetValue())
		if prices[i] == nil {
			continue
		}
		if first == nil {
			first = prices[i]
			continue
		}
		differ = differ || prices[i].GetBitrateMbps() != first.GetBitrateMbps()
	}

	if !differ {
		return
	}
	for i, o := range options {
		o.Estimate = prices[i]
	}
}

// pricePicked is the estimate of these settings with one control set to one of its entries,
// nil where the result cannot be priced (estimate).
func pricePicked(d Deps, s settings.Settings, key, value string) *screensharev1.Estimate {
	m := wire.Settings(s)
	target, descriptor, ok := settingsField(m, key)
	assert.Assert(ok, "a priced control names a setting the message holds", key)

	v, ok := repairValue(descriptor, value)
	assert.Assert(ok, "a priced entry parses as the setting it offers", key, value)
	target.Set(descriptor, v)

	return estimate(d, walked(m, s))
}
