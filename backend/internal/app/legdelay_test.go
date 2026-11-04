package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"bjoernblessin.de/screenshare/internal/gstrun"
	"bjoernblessin.de/screenshare/internal/receive"
	"bjoernblessin.de/screenshare/internal/settings"
	"bjoernblessin.de/screenshare/internal/transport"
	"bjoernblessin.de/screenshare/internal/wire"
)

// Every native leg, a real relay in the middle: deploy/mediamtx-groups.yml with its token check and
// its hooks swapped out, a publisher stamping frames and a receive pipeline reading them.
// Needs mediamtx on the PATH, which the flake's dev shell carries, and takes about a minute.

const (
	legOpenTimeout = 30 * time.Second
	// legSettleDuration lets a jitter buffer fill before the interval that counts.
	legSettleDuration = 3 * time.Second
	legSampleDuration = 3 * time.Second
	// windowRaiseMs is what the window test adds to a leg's window.
	windowRaiseMs = 400
	// windowToleranceMs covers what two runs differ by on a loaded machine.
	windowToleranceMs = 150.0
)

var everyLeg = []string{"rtsp", "webrtc", "srt", "hls", "rtmp"}

func TestEveryLegMeasuresItsDelayThroughARelay(t *testing.T) {
	s := servedRelay(t)

	for _, leg := range everyLeg {
		t.Run(leg, func(t *testing.T) { checkLegBudget(t, s, leg) })
	}
	for _, c := range []struct {
		leg    string
		window func(*settings.Settings) *int
	}{
		{"rtsp", func(s *settings.Settings) *int { return &s.Viewer.RtspWatchLatencyMs }},
		{"srt", func(s *settings.Settings) *int { return &s.Viewer.SrtWatchLatencyMs }},
	} {
		t.Run(c.leg+" window", func(t *testing.T) { checkWindowHeldOnce(t, s, c.leg, c.window) })
	}
}

func checkLegBudget(t *testing.T, s settings.Settings, leg string) {
	budget := measureLeg(t, s, leg)

	for name, stage := range map[string]*float64{
		"path": budget.Path, "decode": budget.Decode, "present": budget.Present, "total": budget.Total,
	} {
		if stage == nil {
			t.Errorf("%s is absent", name)
		}
	}
	if t.Failed() {
		return
	}
	way := wayHereOf(budget)
	if *budget.Total < *way {
		t.Errorf("total %.0f ms is shorter than the way here %.0f ms", *budget.Total, *way)
	}

	stated, ok := transport.PathDelay(leg)
	if !ok {
		t.Fatalf("the leg table states no delay for %s", leg)
	}
	low, high := float64(stated.Milliseconds())/3, float64(stated.Milliseconds())*2+150
	if *way < low || *way > high {
		t.Errorf("the way here took %.0f ms, the leg table states %v", *way, stated)
	}
	t.Logf("way here %.0f ms, total %.0f ms", *way, *budget.Total)
}

// checkWindowHeldOnce raises a leg's window, which moves the delay by exactly that much when it is
// held once.
func checkWindowHeldOnce(t *testing.T, s settings.Settings, leg string, window func(*settings.Settings) *int) {
	before := measureLeg(t, s, leg)

	raised := s
	*window(&raised) += windowRaiseMs
	after := measureLeg(t, raised, leg)

	if before.Total == nil || after.Total == nil {
		t.Fatalf("a total is absent: %v and %v", before.Total, after.Total)
	}
	moved := *after.Total - *before.Total
	if moved < windowRaiseMs-windowToleranceMs || moved > windowRaiseMs+windowToleranceMs {
		t.Errorf("raising the window by %d ms moved the total by %.0f ms", windowRaiseMs, moved)
	}
}

// measureLeg opens one decode over leg and reads its budget over one interval.
func measureLeg(t *testing.T, s settings.Settings, leg string) wire.DelayBudget {
	t.Helper()

	ended := make(chan string, 1)
	r, err := receive.New(receive.Stream{Name: s.StreamName(), Transport: leg, Source: openableSource(t, s, leg)},
		receive.Open{Chain: "cpu"}, receive.Events{
			OnLive:  func() {},
			OnAudio: func() {},
			OnEnd:   func(message string) { ended <- message },
		})
	if err != nil {
		t.Fatalf("opening %s: %v", leg, err)
	}
	defer r.Stop()

	deadline := time.After(legOpenTimeout)
	for r.Stats().PathFrames == 0 {
		select {
		case message := <-ended:
			t.Fatalf("%s ended: %s", leg, message)
		case <-deadline:
			t.Fatalf("%s carried no stamped frame", leg)
		case <-time.After(100 * time.Millisecond):
		}
	}

	time.Sleep(legSettleDuration)
	last := r.Stats()
	time.Sleep(legSampleDuration)
	return receiveDelayOf(r.Stats(), last, true, publishDelay{})
}

// openableSource waits for the relay to serve leg, HLS holding no playlist until its first segments.
func openableSource(t *testing.T, s settings.Settings, leg string) string {
	t.Helper()

	deadline := time.Now().Add(legOpenTimeout)
	for {
		source, err := transport.ReceiveSource(leg, s, s.StreamName())
		if err == nil {
			return strings.Join(source, " ")
		}
		if time.Now().After(deadline) {
			t.Fatalf("resolving %s: %v", leg, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// servedRelay starts a relay and publishes one stamped stream to it,
// and is the settings that reach both.
func servedRelay(t *testing.T) settings.Settings {
	t.Helper()

	if testing.Short() {
		t.Skip("runs real pipelines for about a minute")
	}
	binary, err := exec.LookPath("mediamtx")
	if err != nil {
		t.Skip("needs mediamtx on the PATH")
	}

	s := settings.Defaults()
	s.Relay.Host = "127.0.0.1"
	s.Relay.RtspPort, s.Relay.RtmpPort, s.Relay.HlsPort = freePort(t), freePort(t), freePort(t)
	s.Relay.WebrtcPort, s.Relay.SrtPort = freePort(t), freePort(t)
	s.Publish.Transport = "rtsp"

	startRelay(t, binary, s)
	publishStamped(t, s)
	// The relay holds no playlist before the stream reaches it, whichever leg opens first.
	openableSource(t, s, "hls")
	return s
}

func startRelay(t *testing.T, binary string, s settings.Settings) {
	t.Helper()

	dir := t.TempDir()
	cert, key := drawCertificate(t, dir)
	config, err := filepath.Abs("../../../deploy/mediamtx-groups.yml")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binary, config)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"MTX_AUTHMETHOD=internal",
		"MTX_AUTHINTERNALUSERS_0_USER=any",
		"MTX_AUTHINTERNALUSERS_0_PERMISSIONS_0_ACTION=publish",
		"MTX_AUTHINTERNALUSERS_0_PERMISSIONS_1_ACTION=read",
		"MTX_PATHDEFAULTS_RUNONREAD=",
		"MTX_PATHDEFAULTS_RUNONAVAILABLE=",
		"MTX_API=no", "MTX_METRICS=no", "MTX_MOQ=no",
		"MTX_RTSPSERVERCERT="+cert, "MTX_RTSPSERVERKEY="+key,
		"MTX_RTMPSERVERCERT="+cert, "MTX_RTMPSERVERKEY="+key,
		fmt.Sprintf("MTX_RTSPSADDRESS=:%d", s.Relay.RtspPort),
		fmt.Sprintf("MTX_RTMPSADDRESS=:%d", s.Relay.RtmpPort),
		fmt.Sprintf("MTX_HLSADDRESS=127.0.0.1:%d", s.Relay.HlsPort),
		fmt.Sprintf("MTX_WEBRTCADDRESS=127.0.0.1:%d", s.Relay.WebrtcPort),
		fmt.Sprintf("MTX_WEBRTCLOCALUDPADDRESS=:%d", freePort(t)),
		fmt.Sprintf("MTX_SRTADDRESS=:%d", s.Relay.SrtPort),
	)
	log, err := os.Create(filepath.Join(dir, "relay.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = log.Close()
	})
	waitForListener(t, s.Relay.RtspPort, filepath.Join(dir, "relay.log"))
}

// publishStamped runs a publisher in this process until the test ends.
func publishStamped(t *testing.T, s settings.Settings) {
	t.Helper()

	sink, ok := transport.GstSink(s)
	if !ok {
		t.Fatalf("%s terminates no GStreamer pipeline", s.Publish.Transport)
	}
	description := "videotestsrc is-live=true pattern=ball ! video/x-raw,width=640,height=360,framerate=30/1 ! " +
		"x264enc speed-preset=ultrafast tune=zerolatency key-int-max=30 bitrate=1000 ! " +
		"h264parse config-interval=-1 ! identity name=stats ! " + strings.Join(sink, " ") + " " +
		"audiotestsrc is-live=true wave=ticks ! audio/x-raw,rate=48000,channels=2 ! " +
		"opusenc bitrate=64000 ! opusparse ! mux."

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gstrun.RunWithOptions(ctx, description, gstrun.Options{Delay: "stats"}, io.Discard) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func waitForListener(t *testing.T, port int, log string) {
	t.Helper()

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(legOpenTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	written, _ := os.ReadFile(log)
	t.Fatalf("the relay never listened on %s:\n%s", address, written)
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// drawCertificate writes a self-signed pair, RTSP and RTMP terminating TLS on every relay.
func drawCertificate(t *testing.T, dir string) (cert, key string) {
	t.Helper()

	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotAfter: time.Now().Add(time.Hour)}
	signed, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalECPrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}

	cert, key = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	writePEM(t, cert, "CERTIFICATE", signed)
	writePEM(t, key, "EC PRIVATE KEY", encoded)
	return cert, key
}

func writePEM(t *testing.T, path, kind string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: data}), 0o600); err != nil {
		t.Fatal(err)
	}
}
