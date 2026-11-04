package viewerclient

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bjoernblessin.de/screenshare/internal/token"
	"bjoernblessin.de/screenshare/internal/watchsvc"
)

func serve(t *testing.T) (*httptest.Server, *token.Signer) {
	t.Helper()
	signer, err := token.NewSigner()
	if err != nil {
		t.Fatalf("drawing a signing key: %v", err)
	}
	keys, err := token.ParseKeySet(signer.JWKS())
	if err != nil {
		t.Fatalf("reading the key set: %v", err)
	}
	server := httptest.NewServer(watchsvc.New(keys).Handler("test"))
	t.Cleanup(server.Close)
	return server, signer
}

func sign(t *testing.T, signer *token.Signer, subject, publishPrefix string) string {
	t.Helper()
	signed, err := signer.Sign(subject, token.GroupPermissions("G/", publishPrefix), time.Now(), time.Minute)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}
	return signed
}

func TestAStatedViewingReadsBackForThePublisher(t *testing.T) {
	server, signer := serve(t)
	client := New()

	delay := 41.5
	viewing := []watchsvc.Viewing{{Path: "G/Bob/screen", Transport: "rtsp", PathMs: &delay}}
	if err := client.State(server.URL, sign(t, signer, "alice", "G/Alice/"), viewing); err != nil {
		t.Fatalf("stating: %v", err)
	}

	viewers, err := client.Viewers(server.URL, sign(t, signer, "bob", "G/Bob/"), "G/Bob/screen")
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if len(viewers) != 1 || viewers[0].Member != "alice" || viewers[0].PathMs == nil || *viewers[0].PathMs != delay {
		t.Errorf("read back %+v, want alice at %v ms", viewers, delay)
	}
}

func TestARefusalCarriesTheServicesReason(t *testing.T) {
	server, signer := serve(t)

	_, err := New().Viewers(server.URL, sign(t, signer, "carol", "G/Carol/"), "G/Bob/screen")
	if err == nil {
		t.Fatal("reading another member's viewers succeeded")
	}
	if want := "this token grants no publish of G/Bob/screen"; !strings.Contains(err.Error(), want) {
		t.Errorf("refusal %q, want it to carry %q", err, want)
	}
}

func TestAnUnreachableServiceIsAnError(t *testing.T) {
	server, signer := serve(t)
	server.Close()

	if err := New().State(server.URL, sign(t, signer, "alice", "G/Alice/"), nil); err == nil {
		t.Error("stating to a closed service succeeded")
	}
}

