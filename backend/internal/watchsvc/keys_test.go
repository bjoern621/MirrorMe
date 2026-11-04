package watchsvc

import (
	"errors"
	"testing"
	"time"

	"bjoernblessin.de/screenshare/internal/token"
)

// rotatingGroupd answers whichever key set its signer currently publishes, and counts the fetches.
type rotatingGroupd struct {
	signer  *token.Signer
	fetches int
	down    bool
}

func (g *rotatingGroupd) fetch() (token.KeySet, error) {
	g.fetches++
	if g.down {
		return nil, errors.New("connection refused")
	}
	return token.ParseKeySet(g.signer.JWKS())
}

func mustSigner(t *testing.T) *token.Signer {
	t.Helper()
	s, err := token.NewSigner()
	if err != nil {
		t.Fatalf("drawing a signing key: %v", err)
	}
	return s
}

func TestKeyringFetchesAgainOnAKeyItLacks(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	groupd := &rotatingGroupd{signer: mustSigner(t)}
	keys := NewKeyring(groupd.fetch)

	groupd.signer = mustSigner(t)
	signed, err := groupd.signer.Sign("alice", token.GroupPermissions("G/", "G/"), now, time.Minute)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	if _, err := keys.Verify(signed, now); err != nil {
		t.Errorf("a token under the rotated key gave %v, want it verified after a fetch", err)
	}
	if groupd.fetches != 2 {
		t.Errorf("%d fetches, want the first and one on the unknown key", groupd.fetches)
	}
}

func TestKeyringBoundsHowOftenAnUnknownKeyFetches(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	groupd := &rotatingGroupd{signer: mustSigner(t)}
	keys := NewKeyring(groupd.fetch)

	stranger := mustSigner(t)
	signed, err := stranger.Sign("mallory", token.GroupPermissions("G/", "G/"), now, time.Minute)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	for range 5 {
		if _, err := keys.Verify(signed, now); !errors.Is(err, token.ErrUnknownKey) {
			t.Fatalf("a token under a key groupd never published gave %v, want ErrUnknownKey", err)
		}
	}
	if groupd.fetches != 2 {
		t.Errorf("five unknown-key tokens inside one interval made %d fetches, want 2", groupd.fetches)
	}

	if _, err := keys.Verify(signed, now.Add(RefetchAfter)); !errors.Is(err, token.ErrUnknownKey) {
		t.Fatalf("gave %v, want ErrUnknownKey", err)
	}
	if groupd.fetches != 3 {
		t.Errorf("an unknown key after the interval made %d fetches in all, want 3", groupd.fetches)
	}
}

func TestKeyringStartsEmptyWhileGroupdIsDown(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	groupd := &rotatingGroupd{signer: mustSigner(t), down: true}
	keys := NewKeyring(groupd.fetch)

	signed, err := groupd.signer.Sign("alice", token.GroupPermissions("G/", "G/"), now, time.Minute)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}
	if _, err := keys.Verify(signed, now); !errors.Is(err, token.ErrUnknownKey) {
		t.Errorf("with groupd down, gave %v, want ErrUnknownKey", err)
	}

	groupd.down = false
	if _, err := keys.Verify(signed, now.Add(RefetchAfter)); err != nil {
		t.Errorf("once groupd answers, gave %v, want the token verified", err)
	}
}
