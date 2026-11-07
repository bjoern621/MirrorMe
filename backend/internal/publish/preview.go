package publish

import (
	"fmt"
	"net"
	"strconv"
	"time"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
	"bjoernblessin.de/mirrorme/internal/text"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/capabilities"
	"bjoernblessin.de/mirrorme/internal/ffmpeg"
)

// The local preview leg: a publish child serves a copy of its encoded stream on a loopback port,
// and the decode host reads it from there (docs/viewer-architecture.md,
// "What the insights preview draws").
//
// The leg carries Matroska over TCP.
// A keyframe arrives as one burst, and TCP makes a burst wait for the reader where a datagram
// socket drops whatever overflows its buffer.
// The child listens and the reader connects, because a tcpclientsink ends the pipeline once its peer
// dies.
//
// Both halves of the wire format live here.

// previewHost is the address the child listens on.
// Loopback alone, since what travels here is the user's screen.
const previewHost = "127.0.0.1"

// A reader that falls behind stays attached up to the soft limit, then drops to the next keyframe.
// The hard limit disconnects it.
const (
	previewBacklogSoft = 500 * time.Millisecond
	previewBacklogMax  = 10 * time.Second
)

// previewFifoPackets is how many packets the ffmpeg slave queues ahead of its reader.
// The queue also covers the wait for the reader to connect.
const previewFifoPackets = 120

// previewCarriages is the parser that the preview branch runs, per bitstream format.
// The keys are capabilities.Codec.Format values, and a format with no row has no local preview.
//
// The parser sits inside the branch because matroskamux takes H.26x length-prefixed.
// Upstream of the tee that demand reaches the trunk, and a trunk ending in mpegtsmux fails to
// negotiate.
// VP8 has no parser element, and matroskamux demands nothing of it.
var previewCarriages = map[string][]string{
	"h264": {"h264parse"},
	"hevc": {"h265parse"},
	"av1":  {"av1parse"},
	"vp9":  {"vp9parse"},
	"vp8":  nil,
}

// PreviewLeg is where a publish child serves its already-encoded video for the local preview.
// The zero value is a run with no preview, what Command renders.
type PreviewLeg struct {
	// Port is the loopback TCP port the child listens on (AllocatePreviewPort).
	Port int
}

func (p PreviewLeg) Wanted() bool { return p.Port > 0 }

// PreviewCarried reports whether the codec's bitstream format has a local preview leg,
// and names the format it asked about so a caller can say which one has none.
func PreviewCarried(codec string) (format string, carried bool) {
	c, ok := capabilities.Get(codec)
	if !ok {
		return "", false
	}
	_, carried = previewCarriages[c.Format]
	return c.Format, carried
}

// AllocatePreviewPort returns a loopback TCP port nothing else holds.
//
// The socket closes before the port is handed out, because the child binds it.
// Something else binding the port in that window makes the child fail to start, and the publish
// retries on a port of its own.
func AllocatePreviewPort() (int, error) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(previewHost), Port: 0})
	if err != nil {
		return 0, text.Because(err, mirrormev1.TextCode_TEXT_CODE_LOOPBACK_PORT_UNAVAILABLE)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	assert.Assert(ok, "a tcp listener has a tcp address", listener.Addr().String())
	assert.Assert(addr.Port > 0, "the kernel answers an allocation with a port", addr.Port)
	return addr.Port, nil
}

// PreviewSource returns the launch-line fragment a receiving pipeline reads the local preview from,
// up to the encoded stream, in the shape of transport.GstSource.
// The decodebin3 that ends the chain autoplugs the format out of the Matroska header.
func PreviewSource(port int) string {
	assert.Assert(port > 0, "a preview source names the port it reads", port)

	return fmt.Sprintf("tcpclientsrc host=%s port=%d ! matroskademux", previewHost, port)
}

// gstPreviewTap returns the branch a GStreamer publish pipeline copies its encoded video to.
// A format with no local preview leg is an error.
//
// tcpserversink never blocks its branch, and the queue leaks on top of that, so a stalled reader
// cannot hold up the encode.
// A reader joining mid-stream starts at a keyframe.
func gstPreviewTap(codec string, preview PreviewLeg) ([]string, error) {
	assert.Assert(preview.Wanted(), "a preview branch is built for a run that wants one", codec)

	c, ok := capabilities.Get(codec)
	if !ok {
		return nil, text.Fail(mirrormev1.TextCode_TEXT_CODE_IDENTIFIER_UNKNOWN, text.Codec(codec))
	}
	parser, ok := previewCarriages[c.Format]
	if !ok {
		return nil, text.Fail(mirrormev1.TextCode_TEXT_CODE_PREVIEW_HAS_NO_LEG, text.Codec(codec), text.Format(c.Format))
	}

	tap := []string{"queue", "max-size-buffers=8", "leaky=downstream", "!"}
	if len(parser) > 0 {
		tap = append(append(tap, parser...), "!")
	}
	return append(tap,
		"matroskamux", "streamable=true",
		"!", "tcpserversink",
		"host="+previewHost,
		"port="+strconv.Itoa(preview.Port),
		"sync=false", "async=false",
		"sync-method=latest-keyframe",
		"recover-policy=keyframe",
		"unit-format=time",
		"units-soft-max="+strconv.FormatInt(previewBacklogSoft.Nanoseconds(), 10),
		"units-max="+strconv.FormatInt(previewBacklogMax.Nanoseconds(), 10),
	), nil
}

// ffmpegPreviewTap returns the second output an ffmpeg publish command copies its already-encoded
// video to, and false for a format with no local preview leg.
//
// select=v drops the audio, since the preview draws pictures alone.
// onfail=ignore keeps a slave that cannot open from ending the stream.
// use_fifo moves the slave onto a thread of its own, which the listener's wait for a reader needs:
// it would otherwise hold up the relay's slave.
// A fifo that overflows drops packets, and a reader that dies leaves the output closed until
// the next publish.
//
// global_header puts the parameter sets where the Matroska header reads them, and takes them out of
// the bitstream.
// dump_extra writes them back at each keyframe for the relay leg.
func ffmpegPreviewTap(codec string, preview PreviewLeg) (ffmpeg.Tap, bool) {
	assert.Assert(preview.Wanted(), "a preview output is built for a run that wants one", codec)

	c, ok := capabilities.Get(codec)
	if !ok {
		return ffmpeg.Tap{}, false
	}
	if _, ok := previewCarriages[c.Format]; !ok {
		return ffmpeg.Tap{}, false
	}

	return ffmpeg.Tap{
		Options: []string{
			"select=v",
			"f=matroska",
			"onfail=ignore",
			"use_fifo=1",
			fmt.Sprintf(`fifo_options=drop_pkts_on_overflow=1\\:attempt_recovery=0\\:queue_size=%d`, previewFifoPackets),
		},
		URL:         fmt.Sprintf("tcp://%s:%d?listen=1", previewHost, preview.Port),
		EncoderArgs: []string{"-flags", "+global_header", "-bsf:v", "dump_extra=freq=keyframe"},
	}, true
}
