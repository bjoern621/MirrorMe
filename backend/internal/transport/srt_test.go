package transport

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"bjoernblessin.de/mirrorme/internal/group"
	"bjoernblessin.de/mirrorme/internal/settings"
)

func testStream() settings.Settings {
	return settings.Settings{
		Relay: settings.Relay{
			Host:     "10.0.0.5",
			SrtPort:  8890,
			GroupKey: testGroupKey.String(),
		},
		Publish: settings.Publish{
			Transport:           "srt",
			SrtPublishLatencyMs: 300,
		},
		Viewer: settings.Viewer{
			SrtWatchLatencyMs: 1200,
		},
	}
}

// mustGroupKey fails the test on a drawing failure rather than carrying it into an assertion.
func mustGroupKey(t *testing.T) group.Key {
	t.Helper()
	key, err := group.NewKey()
	if err != nil {
		t.Fatalf("drawing a group key: %v", err)
	}
	return key
}

func TestSRTRegistered(t *testing.T) {
	tr, ok := Get("srt")
	if !ok {
		t.Fatal("srt transport not registered")
	}
	if tr.Name() != "srt" {
		t.Fatalf("Name() = %q, want srt", tr.Name())
	}
	if !slices.Contains(Names(), "srt") {
		t.Errorf("Names() = %v, missing srt", Names())
	}
}

func TestSRTPublishArgs(t *testing.T) {
	args := SRT{}.PublishArgs(testStream())

	if len(args) != 3 || args[0] != "-f" || args[1] != "mpegts" {
		t.Fatalf("PublishArgs prefix = %v, want [-f mpegts URL]", args)
	}

	url := args[2]
	for _, want := range []string{
		"srt://10.0.0.5:8890",
		"streamid=publish:" + testGroupKey.Prefix() + "monitor-0",
		"latency=300000", // ffmpeg's srt protocol counts microseconds
	} {
		if !strings.Contains(url, want) {
			t.Errorf("publish URL %q missing %q", url, want)
		}
	}
}

// A keyframe sent at link speed overflows the relay's reader, and a viewer never gets a picture.
// The ceiling has to stay above the stream, which a rate setting states and crf does not.
func TestSRTPublishIsPacedAboveItsPeak(t *testing.T) {
	for _, c := range []struct {
		what  string
		p     settings.Publish
		maxbw string
	}{
		{"a modest cbr stream takes the floor", settings.Publish{Mode: "cbr", BitrateM: 8}, "maxbw=4000000"},
		{"a fast vbr stream clears its maxrate", settings.Publish{Mode: "vbr", BitrateM: 35, MaxrateM: 80}, "maxbw=12500000"},
		{"crf follows its measured rate", settings.Publish{Mode: "crf", BitrateM: 8}, "maxbw=0&oheadbw=25"},
		{"lossless follows its measured rate", settings.Publish{Mode: "lossless"}, "maxbw=0&oheadbw=25"},
	} {
		s := testStream()
		s.Publish.Mode, s.Publish.BitrateM, s.Publish.MaxrateM = c.p.Mode, c.p.BitrateM, c.p.MaxrateM

		publish := SRT{}.PublishArgs(s)[2]
		sink := strings.Join(SRT{}.GstSink(s), " ")
		if !strings.Contains(publish, "&"+c.maxbw) {
			t.Errorf("%s: publish URL %q misses %q", c.what, publish, c.maxbw)
		}
		if !strings.Contains(sink, "?"+c.maxbw+" ") {
			t.Errorf("%s: sink %q misses %q", c.what, sink, c.maxbw)
		}
	}
}

func TestSRTGstSource(t *testing.T) {
	src := SRT{}.GstSource(testStream(), "bob")

	for _, want := range []string{
		"srtsrc",
		"uri=srt://10.0.0.5:8890?rcvlatency=1200", // srtsrc counts milliseconds, unlike the ffmpeg URL
		"mode=caller",
		"streamid=read:bob",
		"latency=0", // stamps leave the source after the window, so the sink holds none of it again
	} {
		if !slices.Contains(src, want) {
			t.Errorf("GstSource = %v, missing %q", src, want)
		}
	}
}

// Every SRT leg is keyed with the passphrase the settings derive, both engines and both directions.
// A leg missing it connects to a relay that refuses the handshake, and one carrying another group's
// plays nothing.
func TestEverySRTLegCarriesTheDerivedPassphrase(t *testing.T) {
	member := testStream()
	member.Relay.GroupKey = mustGroupKey(t).String()

	for deployment, s := range map[string]settings.Settings{
		"a member's machine":      member,
		"the fixture's own group": testStream(),
	} {
		passphrase := s.Relay.SrtPassphrase()
		if passphrase == "" {
			t.Fatalf("%s derives no passphrase", deployment)
		}

		for leg, address := range map[string]string{
			"the publish command": SRT{}.PublishArgs(s)[2],
			"the watch URL":       SRT{}.WatchURL(s, "bob"),
		} {
			if !strings.Contains(address, "passphrase="+url.QueryEscape(passphrase)) {
				t.Errorf("%s of %s carries no passphrase: %q", leg, deployment, address)
			}
		}
		for leg, elements := range map[string][]string{
			"the publish sink":   SRT{}.GstSink(s),
			"the receive source": SRT{}.GstSource(s, "bob"),
		} {
			if !slices.Contains(elements, "passphrase="+passphrase) {
				t.Errorf("%s of %s carries no passphrase: %v", leg, deployment, elements)
			}
		}
	}
}

func TestSRTWatchURL(t *testing.T) {
	url := SRT{}.WatchURL(testStream(), "bob")

	for _, want := range []string{
		"srt://10.0.0.5:8890",
		"streamid=read:bob",
		"latency=1200000", // the watch knob in microseconds too
	} {
		if !strings.Contains(url, want) {
			t.Errorf("watch URL %q missing %q", url, want)
		}
	}
}

// Both engines send datagrams that cross a 1280-byte link whole: the IPv6 minimum, and the MTU of
// the relay's pod network.
// A larger one dropped there for want of fragmenting reaches the relay as a publish that connects
// and never delivers a frame.
func TestSRTDatagramsCrossTheMinimumLink(t *testing.T) {
	const link, ipv6, udp, srt, ts = 1280, 40, 8, 16, 188
	s := testStream()

	query, err := url.Parse(SRT{}.PublishArgs(s)[2])
	if err != nil {
		t.Fatal(err)
	}
	ffmpeg, err := strconv.Atoi(query.Query().Get("pkt_size"))
	if err != nil {
		t.Fatalf("publish URL %q carries no pkt_size", query)
	}
	gst := 0
	for _, arg := range (SRT{}).GstSink(s) {
		if packets, ok := strings.CutPrefix(arg, "alignment="); ok {
			n, _ := strconv.Atoi(packets)
			gst = n * ts
		}
	}

	for engine, payload := range map[string]int{"ffmpeg": ffmpeg, "gstreamer": gst} {
		if payload == 0 || payload%ts != 0 {
			t.Errorf("%s payload = %d bytes, want whole TS packets", engine, payload)
		}
		if payload+srt+udp+ipv6 > link {
			t.Errorf("%s datagram = %d bytes, past the %d-byte link", engine, payload+srt+udp+ipv6, link)
		}
	}
}
