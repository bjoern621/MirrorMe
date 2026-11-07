package receive

import (
	"slices"
	"testing"
)

func TestReselectionAddsBackADroppedStream(t *testing.T) {
	collection := []offeredStream{
		{id: "audio", kind: "audio", decodable: true},
		{id: "video", kind: "video", decodable: true},
	}
	cases := []struct {
		name     string
		streams  []offeredStream
		selected []string
		retried  map[string]bool
		want     []string
	}{
		{"a decodable video dropped beside audio comes back", collection, []string{"audio"}, nil, []string{"audio", "video"}},
		{"a whole selection stands", collection, []string{"audio", "video"}, nil, nil},
		{"a stream no decoder takes stays out", []offeredStream{
			{id: "audio", kind: "audio", decodable: true},
			{id: "video", kind: "video", decodable: false},
		}, []string{"audio"}, nil, nil},
		{"a stream selected again once stays out the second time", collection, []string{"audio"}, map[string]bool{"video": true}, nil},
		{"a second stream of a selected kind stays out", []offeredStream{
			{id: "video-1", kind: "video", decodable: true},
			{id: "video-2", kind: "video", decodable: true},
		}, []string{"video-1"}, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reselection(c.streams, c.selected, c.retried)
			if !slices.Equal(got, c.want) {
				t.Errorf("expected %v, got %v", c.want, got)
			}
		})
	}
}

func TestDroppedNamesAStreamStillShortOfADecoder(t *testing.T) {
	streams := []offeredStream{
		{id: "audio", kind: "audio", decodable: true},
		{id: "video", kind: "video", decodable: false},
	}
	got := dropped(streams, []string{"audio"}, nil)
	if len(got) != 1 || got[0].id != "video" {
		t.Errorf("expected the undecodable video, got %v", got)
	}
}
