package settings

import "fmt"

// ViewerServicePort is where watchd answers on a relay this network reaches directly,
// its own default (cmd/watchd, -listen).
// Behind the proxy its /viewers route sits on the one name (deploy/Caddyfile).
const ViewerServicePort = 9445

// ViewerService is where viewers state what they watch and publishers read who watches them,
// ok=false where no relay is named.
// The deployment decides the address the way GroupService's does.
func (r Relay) ViewerService() (base string, ok bool) {
	if r.Host == "" {
		return "", false
	}
	if r.OnTrustedNetwork() {
		return fmt.Sprintf("http://%s:%d", r.Host, ViewerServicePort), true
	}
	return "https://" + r.Host, true
}
