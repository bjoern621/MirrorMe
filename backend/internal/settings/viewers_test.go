package settings

import "testing"

func TestViewerServiceFollowsTheDeployment(t *testing.T) {
	base, ok := (Relay{Host: "192.168.1.9"}).ViewerService()
	if !ok || base != "http://192.168.1.9:9445" {
		t.Fatalf("a relay on this network is asked on watchd's own port, got %q ok=%v", base, ok)
	}

	base, ok = (Relay{Host: "streamrelay.example.com"}).ViewerService()
	if !ok || base != "https://streamrelay.example.com" {
		t.Fatalf("a public relay is asked through the proxy on its one name, got %q ok=%v", base, ok)
	}

	if _, ok := (Relay{}).ViewerService(); ok {
		t.Fatal("no relay is no watchd to ask")
	}
}
