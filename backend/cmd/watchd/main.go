// Command watchd carries what each viewer measures of a stream back to the stream's publisher
// (internal/watchsvc).
//
// A binary of its own beside groupd, since it holds no secret:
// it verifies the tokens groupd signs against the key set groupd publishes, and reads nothing else.
// Viewer statements live in memory alone, and every live viewer restates its set within a second of a restart.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"bjoernblessin.de/go-utils/util/logger"

	"bjoernblessin.de/screenshare/internal/token"
	"bjoernblessin.de/screenshare/internal/watchsvc"
)

// version is the build stamp every answer names (internal/serving).
// In main because the linker writes into it: -ldflags "-X main.version=...".
var version = "dev"

// fetchTimeout bounds one read of groupd's key set, a small document over a local hop.
const fetchTimeout = 10 * time.Second

// main serves the viewers routes on the address given.
// An address that will not serve ends the process through logger.Errorf, since nothing else is left to do.
func main() {
	listen := flag.String("listen", "127.0.0.1:9445", "address to serve on")
	groupService := flag.String("group-service", "http://127.0.0.1:9443", "where groupd answers, for its key set")
	flag.Parse()

	client := &http.Client{Timeout: fetchTimeout}
	keys := watchsvc.NewKeyring(func() (token.KeySet, error) { return fetchKeySet(client, *groupService) })
	service := watchsvc.New(keys)
	logger.Infof("serving viewers on %s, verifying against %s/jwks.json", *listen, *groupService)

	// Bounded in every phase like groupd: the routes are small JSON over a local hop,
	// so a request outlasting these bounds comes from a caller that stopped.
	server := &http.Server{
		Addr:              *listen,
		Handler:           service.Handler(version),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Errorf("serving: %v", err)
	}
}

// fetchKeySet reads the key set groupd publishes.
// Errors if groupd does not answer, answers with a failing status or answers a document that does not parse.
func fetchKeySet(client *http.Client, base string) (token.KeySet, error) {
	address := base + "/jwks.json"
	resp, err := client.Get(address)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", address, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: status %d", address, resp.StatusCode)
	}
	document, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", address, err)
	}
	return token.ParseKeySet(document)
}
