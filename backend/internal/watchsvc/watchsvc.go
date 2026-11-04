// Package watchsvc carries what each viewer measures of a stream back to the stream's publisher.
//
// A viewer's app states the whole set of streams it decodes, with its figures for each,
// and the publisher of a path reads that path's viewers back.
// Every figure comes off the viewer's own decode, so every leg reports the same figures,
// where the relay times a round trip on SRT alone (internal/relay, readerKinds).
//
// Credential: the relay access token groupd signs.
// Its subject names the member.
// Its read grant bounds what a statement may name, its publish grant whose viewers a member reads.
// This service holds no group key and no secret, only the key set groupd publishes.
//
// Statements live in memory under a lease (viewing.go).
// A restart forgets them, and every live viewer states its set again within a second.
//
// Every refusal answers the caller: the input arrives over HTTP from a process this one does not control.
package watchsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/serving"
	"bjoernblessin.de/screenshare/internal/token"
)

// bodyLimitBytes bounds a statement, at a few times MaxViewings rows of figures.
const bodyLimitBytes = 16 << 10

// Verifier checks a relay access token.
// token.KeySet checks against a fixed set.
// Keyring fetches a newer set on an unknown key.
type Verifier interface {
	Verify(signed string, now time.Time) (token.Claims, error)
}

// Service answers the viewers routes.
// Can be called concurrently.
type Service struct {
	keys     Verifier
	viewings *registry
	// now is read in place of time.Now, so a test moves the clock the lease and the token expiry read.
	now func() time.Time
}

// New is a service verifying tokens with keys.
func New(keys Verifier) *Service {
	assert.IsNotNil(keys, "a service verifies tokens against a key set")
	return &Service{keys: keys, viewings: newRegistry(), now: time.Now}
}

// Handler is the service's routes, answering under the version given (internal/serving).
func (s *Service) Handler(version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /viewers", s.state)
	mux.HandleFunc("GET /viewers", s.viewers)
	return serving.Naming("watchd", version, mux)
}

// state replaces everything the calling member watches with the set in the body.
// A set naming a path outside the token's read grant is refused whole and stores nothing.
func (s *Service) state(w http.ResponseWriter, r *http.Request) {
	now := s.now()
	claims, ok := s.verified(w, r, now)
	if !ok {
		return
	}

	var body struct {
		Viewing []Viewing `json:"viewing"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, bodyLimitBytes)).Decode(&body); err != nil {
		refuse(w, http.StatusBadRequest, "this request carries no viewing set this service can read")
		return
	}
	if len(body.Viewing) > MaxViewings {
		refuse(w, http.StatusBadRequest, fmt.Sprintf("a viewing set holds at most %d streams", MaxViewings))
		return
	}
	for _, v := range body.Viewing {
		if why := v.malformed(); why != "" {
			refuse(w, http.StatusBadRequest, why)
			return
		}
		if !claims.Grants(token.ActionRead, v.Path) {
			refuse(w, http.StatusForbidden, "this token grants no read of "+v.Path)
			return
		}
	}

	s.viewings.State(claims.Subject, body.Viewing, now)
	w.WriteHeader(http.StatusNoContent)
}

// viewers answers who watches the path in the query and what each of them measures.
// Only a token granting publish on that path reads it, so a member learns who watches their own streams
// and nobody else's.
func (s *Service) viewers(w http.ResponseWriter, r *http.Request) {
	now := s.now()
	claims, ok := s.verified(w, r, now)
	if !ok {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		refuse(w, http.StatusBadRequest, "a read names the path whose viewers it asks for")
		return
	}
	if !claims.Grants(token.ActionPublish, path) {
		refuse(w, http.StatusForbidden, "this token grants no publish of "+path)
		return
	}

	answer(w, map[string]any{"viewers": s.viewings.ViewersOf(path, now)})
}

// verified reads the bearer token and answers its claims, refusing the request where it does not verify.
func (s *Service) verified(w http.ResponseWriter, r *http.Request, now time.Time) (token.Claims, bool) {
	bearer, held := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !held || bearer == "" {
		refuse(w, http.StatusUnauthorized, "this request carries no relay access token")
		return token.Claims{}, false
	}
	claims, err := s.keys.Verify(bearer, now)
	if err != nil {
		reason := "this relay access token does not verify"
		if errors.Is(err, token.ErrUnknownKey) {
			reason = "this relay access token names a key the group service does not publish"
		}
		refuse(w, http.StatusUnauthorized, reason)
		return token.Claims{}, false
	}
	if claims.Subject == "" {
		refuse(w, http.StatusUnauthorized, "this relay access token names no member")
		return token.Claims{}, false
	}
	return claims, true
}

func answer(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

// refuse states why a request was not answered, as groupd states it.
func refuse(w http.ResponseWriter, status int, reason string) {
	assert.Assert(status >= 400, "a refusal carries a failing status", status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": reason})
}
