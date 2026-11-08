package publish

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// previewEncoders is a software encoder per bitstream format, launched for real below.
// Each ends the way the encoder table does, in the element ahead of the tee.
var previewEncoders = map[string]string{
	"h264": "x264enc tune=zerolatency speed-preset=ultrafast key-int-max=" + previewGop + " ! h264parse config-interval=-1",
	"hevc": "x265enc tune=zerolatency speed-preset=ultrafast key-int-max=" + previewGop + " ! h265parse config-interval=-1",
	"av1":  "svtav1enc preset=12 intra-period-length=" + previewGop + " ! av1parse",
	"vp9":  "vp9enc deadline=1 cpu-used=8 keyframe-max-dist=" + previewGop + " ! vp9parse",
	"vp8":  "vp8enc deadline=1 cpu-used=8 keyframe-max-dist=" + previewGop,
}

const (
	previewTestWidth  = 320
	previewTestHeight = 180
	previewTestFrames = 90
	// previewTestGopFrames spaces keyframes half a second apart at 30 fps.
	previewTestGopFrames = 15
)

var previewGop = strconv.Itoa(previewTestGopFrames)

func needGstElements(t *testing.T, line string) {
	t.Helper()

	if _, err := exec.LookPath(GstExe); err != nil {
		t.Skipf("%s not installed", GstExe)
	}
	for _, part := range strings.Split(line, " ! ") {
		element := strings.Fields(part)[0]
		if exec.Command("gst-inspect-1.0", "--exists", element).Run() != nil {
			t.Skipf("%s not installed", element)
		}
	}
}

// previewSender launches a publish child's shape: an encoder, the preview tap off its tee, and trunk.
func previewSender(ctx context.Context, t *testing.T, format, trunk string, port, frames int) *exec.Cmd {
	t.Helper()

	encoder := previewEncoders[format]
	needGstElements(t, encoder)
	tap, err := gstPreviewTap(previewCodecs[format], PreviewLeg{Port: port})
	if err != nil {
		t.Fatal(err)
	}

	source := fmt.Sprintf("videotestsrc is-live=true num-buffers=%d ! video/x-raw,format=I420,width=%d,height=%d,framerate=30/1 ! %s ! %s",
		frames, previewTestWidth, previewTestHeight, encoder, "")
	args := strings.Fields(strings.TrimSuffix(strings.TrimSpace(source), "!"))
	args = append(args, "!")
	args = append(args, gstTapElements([][]string{tap})...)
	args = append(args, "queue", "!")
	args = append(args, strings.Fields(trunk)...)

	cmd := exec.CommandContext(ctx, GstExe, args...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

// previewReceiver decodes the leg the way a receive chain does, into a file of raw frames.
// The child's listener comes first, since a receiver ahead of it is refused.
func previewReceiver(ctx context.Context, t *testing.T, port int, out string) *exec.Cmd {
	t.Helper()

	// Forward slashes, since the launch parser reads a backslash as an escape.
	out = filepath.ToSlash(out)
	line := PreviewSource(port) + fmt.Sprintf(
		" ! decodebin3 ! videoconvert ! video/x-raw,format=I420,width=%d,height=%d ! filesink location=%s",
		previewTestWidth, previewTestHeight, out)
	address := net.JoinHostPort(previewHost, strconv.Itoa(port))
	for {
		conn, err := net.Dial("tcp", address)
		if err == nil {
			conn.Close()
			break
		}
		if ctx.Err() != nil {
			t.Fatalf("the child never listened on %s", address)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cmd := exec.CommandContext(ctx, GstExe, append([]string{"-e"}, strings.Fields(line)...)...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func decodedFrames(t *testing.T, path string) int {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return int(info.Size()) / (previewTestWidth * previewTestHeight * 3 / 2)
}

// Every format crosses the leg through the elements the two halves name, and arrives decodable.
// The tables agree on paper alone until a real pipeline negotiates them.
//
// One whole GOP proves it.
// A reader that decodes slower than real time loses GOPs to the keyframe recovery on purpose,
// which av1 and vp9 do on a starved CI runner.
func TestEveryFormatCrossesThePreviewLeg(t *testing.T) {
	for format := range previewEncoders {
		t.Run(format, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			port, err := AllocatePreviewPort()
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "frames.yuv")
			sender := previewSender(ctx, t, format, "fakesink sync=false", port, previewTestFrames)
			receiver := previewReceiver(ctx, t, port, out)

			if err := sender.Wait(); err != nil {
				t.Fatalf("the publish failed: %v", err)
			}
			receiver.Wait()
			if got := decodedFrames(t, out); got < previewTestGopFrames {
				t.Errorf("%d of %d frames decoded, under one GOP of %d", got, previewTestFrames, previewTestGopFrames)
			}
		})
	}
}

// A receiver that dies costs the preview and leaves the stream running.
// A leg able to end the publish lets a crash in the decode process take the stream down.
func TestADeadReceiverLeavesThePublishRunning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	port, err := AllocatePreviewPort()
	if err != nil {
		t.Fatal(err)
	}
	sender := previewSender(ctx, t, "h264", "fakesink sync=false", port, previewTestFrames)
	receiver := previewReceiver(ctx, t, port, filepath.Join(t.TempDir(), "frames.yuv"))

	time.Sleep(500 * time.Millisecond)
	if err := receiver.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	receiver.Wait()

	if err := sender.Wait(); err != nil {
		t.Errorf("a dead receiver ended the publish: %v", err)
	}
}

// matroskamux takes H.264 length-prefixed and mpegtsmux takes it with start codes.
// Off one tee the two would agree on neither, so the branch converts on its own side.
// The trunk is what a publish over SRT runs.
func TestThePreviewBranchLeavesTheTrunkItsStreamFormat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	port, err := AllocatePreviewPort()
	if err != nil {
		t.Fatal(err)
	}
	needGstElements(t, "mpegtsmux")
	sender := previewSender(ctx, t, "h264", "mpegtsmux ! fakesink sync=false", port, previewTestFrames/3)
	if err := sender.Wait(); err != nil {
		t.Errorf("the preview branch broke the trunk's negotiation: %v", err)
	}
}
