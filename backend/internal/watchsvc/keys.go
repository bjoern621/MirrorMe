package watchsvc

import (
	"errors"
	"sync"
	"time"

	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/go-utils/util/logger"

	"bjoernblessin.de/mirrorme/internal/token"
)

// RefetchAfter bounds how often a token under an unknown key fetches the key set again.
// A forged key id costs one fetch an interval, however many tokens carry it.
const RefetchAfter = 30 * time.Second

// Keyring holds the key set groupd publishes and fetches it again when a token names a key it lacks.
// groupd draws a new key only on a fresh start without its key file, so a fetch on demand covers it.
// Can be called concurrently.
type Keyring struct {
	fetch func() (token.KeySet, error)

	mu        sync.Mutex
	keys      token.KeySet
	fetchedAt time.Time
}

// NewKeyring fetches the key set once.
// A fetch that fails leaves the ring empty, and the first token to arrive asks again.
func NewKeyring(fetch func() (token.KeySet, error)) *Keyring {
	assert.IsNotNil(fetch, "a keyring fetches its key set from somewhere")

	k := &Keyring{fetch: fetch, keys: token.KeySet{}}
	k.refreshLocked(time.Now())
	return k
}

// Verify checks a token against the held set, and against a fetched one where it names a key the set lacks.
func (k *Keyring) Verify(signed string, now time.Time) (token.Claims, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	claims, err := k.keys.Verify(signed, now)
	if !errors.Is(err, token.ErrUnknownKey) || now.Sub(k.fetchedAt) < RefetchAfter {
		return claims, err
	}
	k.refreshLocked(now)
	return k.keys.Verify(signed, now)
}

// refreshLocked fetches the key set, keeping the held one where groupd does not answer.
// Assumes the caller holds mu.
func (k *Keyring) refreshLocked(now time.Time) {
	k.fetchedAt = now
	keys, err := k.fetch()
	if err != nil {
		logger.Warnf("the group service's key set would not load, so tokens under a new key stay refused: %v", err)
		return
	}
	k.keys = keys
}
