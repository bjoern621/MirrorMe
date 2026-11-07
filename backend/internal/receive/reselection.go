package receive

import (
	"slices"

	"bjoernblessin.de/go-utils/util/assert"
)

// offeredStream holds one stream of a collection, as the reselection weighs it.
type offeredStream struct {
	id        string
	kind      string // The stream type's nick: "video", "audio".
	decodable bool   // Whether a decoder takes the stream's caps whole.
}

// dropped answers the first stream of each kind the selection holds none of,
// leaving out a stream selected again once already:
// a decoder that fails for a reason of its own drops the stream again and would loop.
// One stream per kind, the way decodebin3 picks by default.
func dropped(streams []offeredStream, selected []string, retried map[string]bool) []offeredStream {
	var kinds []string
	for _, s := range streams {
		if slices.Contains(selected, s.id) {
			kinds = append(kinds, s.kind)
		}
	}

	var out []offeredStream
	for _, s := range streams {
		if slices.Contains(kinds, s.kind) || retried[s.id] {
			continue
		}
		kinds = append(kinds, s.kind)
		out = append(out, s)
	}
	return out
}

// reselection answers the IDs a select-streams event names to let the decodable dropped streams back
// in, or nil where the selection stands.
func reselection(streams []offeredStream, selected []string, retried map[string]bool) []string {
	var added []string
	for _, s := range dropped(streams, selected, retried) {
		if s.decodable {
			added = append(added, s.id)
		}
	}
	if len(added) == 0 {
		return nil
	}
	return append(slices.Clone(selected), added...)
}

// withOffer answers streams with the entry of fresh's ID replaced by fresh.
func withOffer(streams []offeredStream, fresh offeredStream) []offeredStream {
	out := slices.Clone(streams)
	i := slices.IndexFunc(out, func(s offeredStream) bool { return s.id == fresh.id })
	assert.Assert(i >= 0, "a stream whose caps changed belongs to the collection it was watched from")
	out[i] = fresh
	return out
}
