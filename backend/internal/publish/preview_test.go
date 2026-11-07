package publish

import (
	"net"
	"strings"
	"testing"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/ffmpeg"
	"bjoernblessin.de/mirrorme/internal/settings"
)

// previewCodecs is one codec per bitstream format that the app publishes.
// RTSP is the transport in these tests because it carries every format.
var previewCodecs = map[string]string{
	"h264": "libx264",
	"hevc": "libx265",
	"av1":  "libsvtav1",
	"vp9":  "libvpx-vp9",
	"vp8":  "libvpx",
}

// previewStream is settings that publish the codec over the one transport carrying every format,
// through the engine the capture backend runs on.
//
// The rate-control mode is asked of the capability table rather than named, for the reason every
// consumer of that table asks it: the defaults are a lossless encode at 150 Mbit/s that most rows
// gap one way or another, where what these tests are about is the shape of the command rather than
// which qualities a family can code.
// A mode written here would be a second statement of the table's own answer.
func previewStream(t *testing.T, capture, codec string) settings.Settings {
	t.Helper()

	engine, err := EngineFor(capture)
	if err != nil {
		t.Fatal(err)
	}

	s := baseStream()
	s.Publish.Capture, s.Publish.Transport = capture, "rtsp"
	s.Publish.UseCodec(codec)
	s.Publish.Chroma = "yuv420p"
	s.Publish.BitrateM, s.Publish.ColorRange = 20, "tv"
	for _, mode := range capabilities.Modes {
		s.Publish.Mode = mode
		if capabilities.Validate(engine, codec, s.Publish.CapabilityOptions(), s.Publish.Cq, s.Publish.BitrateM, s.Publish.Gop, capabilities.Device{}) == nil {
			return s
		}
	}
	t.Fatalf("%s codes nothing on the %s engine, so no command can be rendered for it", codec, engine)
	return s
}

// Every format that the app publishes has a local preview leg.
// A format with no row publishes with no preview, and nothing else reports the gap.
func TestEveryPublishableFormatHasAPreviewLeg(t *testing.T) {
	for _, format := range capabilities.Formats() {
		codec, ok := previewCodecs[format]
		if !ok {
			t.Fatalf("format %s has no codec in this test's table, so its preview leg is untested", format)
		}
		if got, carried := PreviewCarried(codec); !carried {
			t.Fatalf("%s produces %s, which carries no local preview", codec, got)
		}
		if _, err := gstPreviewTap(codec, PreviewLeg{Port: 5000}); err != nil {
			t.Fatalf("%s: %v", codec, err)
		}
	}
}

// Both halves name one port and one container.
func TestThePreviewSourceReadsWhatTheTapServes(t *testing.T) {
	source := PreviewSource(45678)
	if !strings.Contains(source, "tcpclientsrc host="+previewHost+" port=45678") {
		t.Errorf("the source reads no loopback port: %s", source)
	}
	if !strings.Contains(source, "matroskademux") {
		t.Errorf("the source demuxes no Matroska: %s", source)
	}
}

// The preview hangs off the tee that the meter hangs on.
// A second tee would be a second copy of the encoded stream.
func TestTheGstPipelineTeesThePreviewOffTheEncodedStream(t *testing.T) {
	for format, codec := range previewCodecs {
		s := previewStream(t, "portal", codec)

		plain, err := buildPipeline(s, []string{"videotestsrc"}, "", PreviewLeg{})
		if err != nil {
			t.Fatalf("%s: %v", codec, err)
		}
		if line := strings.Join(plain, " "); strings.Contains(line, "tcpserversink") {
			t.Errorf("%s: a pipeline built without a preview port carries a preview branch: %s", format, line)
		}

		preview, err := buildPipeline(s, []string{"videotestsrc"}, "54321", PreviewLeg{Port: 45678})
		if err != nil {
			t.Fatalf("%s: %v", codec, err)
		}
		line := strings.Join(preview, " ")
		if strings.Count(line, "tee name=") != 1 {
			t.Errorf("%s: the preview and the meter do not share one tee: %s", format, line)
		}
		if !strings.Contains(line, "tcpserversink host="+previewHost+" port=45678") {
			t.Errorf("%s: no preview sink on the allocated loopback port: %s", format, line)
		}
		if !strings.Contains(line, "tcpclientsink host="+previewHost+" port=54321") {
			t.Errorf("%s: the preview branch displaced the meter's: %s", format, line)
		}
		// A preview able to hold up the encode path can stall the stream that it previews.
		muxer := strings.Index(line, "matroskamux")
		leaky := strings.LastIndex(line[:muxer], "leaky=downstream")
		if leaky < 0 {
			t.Errorf("%s: the preview branch can backpressure the encoder: %s", format, line)
		}
		// A frozen receiver drops back to a keyframe, since a gap inside a GOP corrupts every frame
		// up to the next one.
		if !strings.Contains(line, "recover-policy=keyframe") {
			t.Errorf("%s: a receiver falling behind resumes mid-GOP: %s", format, line)
		}
		// A tee's request pad and a muxer's do not link to each other directly.
		last := strings.LastIndex(line, gstTeeName+". !")
		if mux := strings.Index(line, "name="+"mux"); last < 0 || mux < last || !strings.Contains(line[last:], "queue") {
			t.Errorf("%s: the muxer is not the last branch off the tee: %s", format, line)
		}
	}
}

// The port belongs to one launch, and SamePipeline compares this string.
func TestTheRenderedCommandCarriesNoPreviewLeg(t *testing.T) {
	for _, capture := range []string{"portal", "gdigrab"} {
		for _, codec := range previewCodecs {
			s := previewStream(t, capture, codec)
			engine, err := For(capture)
			if err != nil {
				t.Fatal(err)
			}
			line, err := engine.Command(s)
			if err != nil {
				// A capture backend this machine cannot render for is not what this test is about.
				// The shape of the command it would render is.
				continue
			}
			if strings.Contains(line, "tcpserversink") || strings.Contains(line, "tcp://"+previewHost) ||
				strings.Contains(line, "-f tee") {
				t.Errorf("%s/%s: the rendered command carries a run's own preview leg: %s", capture, codec, line)
			}
		}
	}
}

// The ffmpeg child writes one encoder's packets to two outputs through the tee muxer.
func TestTheFfmpegCommandTeesThePreviewBesideTheRelayLeg(t *testing.T) {
	for format, codec := range previewCodecs {
		s := previewStream(t, "gdigrab", codec)

		tap, ok := ffmpegPreviewTap(codec, PreviewLeg{Port: 45678})
		if !ok {
			t.Fatalf("%s: %s has no ffmpeg preview output", format, codec)
		}
		args, err := ffmpeg.BuildPublishArgs(s, []ffmpeg.Tap{tap})
		if err != nil {
			t.Fatalf("%s: %v", codec, err)
		}

		line := strings.Join(args, " ")
		if strings.Count(line, "-f tee") != 1 {
			t.Errorf("%s: not exactly one tee: %s", format, line)
		}
		if !strings.Contains(line, "[f=rtsp:rtsp_transport=tcp") {
			t.Errorf("%s: the relay leg is not the transport's own arguments as a slave: %s", format, line)
		}
		if !strings.Contains(line, "tcp://"+previewHost+":45678?listen=1") {
			t.Errorf("%s: no preview slave listening on the allocated loopback port: %s", format, line)
		}
		if !strings.Contains(line, "select=v") {
			t.Errorf("%s: the preview slave takes more than the video: %s", format, line)
		}
		if !strings.Contains(line, "onfail=ignore") {
			t.Errorf("%s: a preview slave that cannot open would end the stream: %s", format, line)
		}
		// The streams are mapped by hand: automatic stream selection does not apply to a tee,
		// and an unmapped tee writes no stream at all.
		if !strings.Contains(line, "-map 0:v") {
			t.Errorf("%s: the video is not mapped into the tee: %s", format, line)
		}
		// A slave waiting on its reader must not hold up the relay's.
		if !strings.Contains(line, "use_fifo=1") || !strings.Contains(line, "drop_pkts_on_overflow=1") {
			t.Errorf("%s: a stalled preview reader can stall the relay leg: %s", format, line)
		}
		// Matroska states the parameter sets in its header, and the flag takes them out of the
		// bitstream that the relay leg reads them from.
		if !strings.Contains(line, "-flags +global_header") || !strings.Contains(line, "dump_extra=freq=keyframe") {
			t.Errorf("%s: the header or the relay leg goes without parameter sets: %s", format, line)
		}
		if strings.Index(line, "-flags +global_header") > strings.Index(line, "-f tee") {
			t.Errorf("%s: the encoder options land behind the output: %s", format, line)
		}
	}
}

// A filter source has no input to map,
// so its chain's output is labelled and the map names that label.
// Without it the tee writes nothing,
// and the failure is a stream that never starts rather than a preview that does not.
func TestAFilterSourceIsMappedIntoTheTeeByLabel(t *testing.T) {
	s := previewStream(t, "ddagrab", "libx264")
	tap, ok := ffmpegPreviewTap(s.Publish.Codec(), PreviewLeg{Port: 45678})
	if !ok {
		t.Fatal("libx264 has no ffmpeg preview output")
	}

	args, err := ffmpeg.BuildPublishArgs(s, []ffmpeg.Tap{tap})
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Join(args, " ")
	if !strings.Contains(line, "-map [out]") {
		t.Errorf("a filter source is not mapped by the label its chain ends in: %s", line)
	}
	if !strings.Contains(line, "-filter_complex ") || !strings.Contains(line, "[out] -map") {
		t.Errorf("the filter chain does not end in the label the map names: %s", line)
	}
	if strings.Contains(line, "-map 0:v") {
		t.Errorf("a filter source has no input 0 to map: %s", line)
	}
}

// Two allocations in a row do not collide.
func TestPreviewPortsAreAllocatedAndBindable(t *testing.T) {
	first, err := AllocatePreviewPort()
	if err != nil {
		t.Fatal(err)
	}
	second, err := AllocatePreviewPort()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Errorf("two allocations landed on one port: %d", first)
	}
	for _, port := range []int{first, second} {
		if port <= 0 || port > 65535 {
			t.Errorf("port %d is not a port", port)
		}
		// The child binds it, so the allocation has to let go of it.
		conn, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(previewHost), Port: port})
		if err != nil {
			t.Errorf("port %d was not released by the allocation that reported it: %v", port, err)
			continue
		}
		conn.Close()
	}
}

// A codec whose format has no local carriage publishes without a preview
// rather than failing to publish.
// A complete table leaves that branch unreachable, so a row is taken out to reach it:
// it is the branch that decides whether a format added later costs a stream.
func TestAFormatWithNoLocalCarriagePublishesWithoutAPreview(t *testing.T) {
	const format = "h264"
	codec := previewCodecs[format]

	carriage := previewCarriages[format]
	delete(previewCarriages, format)
	defer func() { previewCarriages[format] = carriage }()

	if _, carried := PreviewCarried(codec); carried {
		t.Fatal("a format taken out of the table still reports a preview leg")
	}
	if _, err := ffmpeg.BuildPublishArgs(previewStream(t, "gdigrab", codec), nil); err != nil {
		t.Fatalf("the ffmpeg command must still build: %v", err)
	}
	if _, ok := ffmpegPreviewTap(codec, PreviewLeg{Port: 45678}); ok {
		t.Error("a format with no carriage yielded an ffmpeg preview output")
	}
	if _, err := gstPreviewTap(codec, PreviewLeg{Port: 45678}); err == nil {
		t.Error("a format with no carriage yielded a GStreamer preview branch")
	}

	pipeline, err := buildPipeline(previewStream(t, "portal", codec), []string{"videotestsrc"}, "", PreviewLeg{Port: 45678})
	if err != nil {
		t.Fatalf("the GStreamer pipeline must still build: %v", err)
	}
	if line := strings.Join(pipeline, " "); strings.Contains(line, "tcpserversink") {
		t.Errorf("a format with no carriage still grew a preview branch: %s", line)
	}
}
