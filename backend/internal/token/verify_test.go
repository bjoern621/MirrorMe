package token

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func mustKeySet(t *testing.T, s *Signer) KeySet {
	t.Helper()
	keys, err := ParseKeySet(s.JWKS())
	if err != nil {
		t.Fatalf("reading the signer's own JWKS: %v", err)
	}
	return keys
}

func mustSign(t *testing.T, s *Signer, subject string, permissions []Permission, now time.Time, window time.Duration) string {
	t.Helper()
	signed, err := s.Sign(subject, permissions, now, window)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}
	return signed
}

func TestVerifyReadsBackWhatWasSigned(t *testing.T) {
	s := mustSigner(t)
	now := time.Unix(1_800_000_000, 0)
	signed := mustSign(t, s, "member-1", GroupPermissions("G/", "G/Bob/"), now, 5*time.Minute)

	claims, err := mustKeySet(t, s).Verify(signed, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("a token verifies against the key set its signer publishes: %v", err)
	}
	if claims.Subject != "member-1" {
		t.Errorf("subject %q, want member-1", claims.Subject)
	}
	if len(claims.Permissions) != 2 {
		t.Errorf("%d permissions, want the two GroupPermissions grants", len(claims.Permissions))
	}
}

func TestVerifyRefusesAnExpiredToken(t *testing.T) {
	s := mustSigner(t)
	now := time.Unix(1_800_000_000, 0)
	signed := mustSign(t, s, "member-1", GroupPermissions("G/", "G/"), now, time.Minute)

	_, err := mustKeySet(t, s).Verify(signed, now.Add(2*time.Minute))
	if !errors.Is(err, ErrUnverified) {
		t.Errorf("a token past its expiry gave %v, want ErrUnverified", err)
	}
}

func TestVerifyRefusesAnAlteredPayload(t *testing.T) {
	s := mustSigner(t)
	now := time.Unix(1_800_000_000, 0)
	signed := mustSign(t, s, "member-1", GroupPermissions("G/", "G/"), now, time.Minute)

	parts := strings.Split(signed, ".")
	forged, err := segment(map[string]any{
		"sub": "member-2", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
		PermissionClaim: GroupPermissions("G/", "G/"),
	})
	if err != nil {
		t.Fatalf("rendering the forged payload: %v", err)
	}

	_, err = mustKeySet(t, s).Verify(parts[0]+"."+forged+"."+parts[2], now)
	if !errors.Is(err, ErrUnverified) {
		t.Errorf("a payload the signature does not cover gave %v, want ErrUnverified", err)
	}
}

func TestVerifyRefusesATokenFromAnotherKey(t *testing.T) {
	ours, theirs := mustSigner(t), mustSigner(t)
	now := time.Unix(1_800_000_000, 0)
	signed := mustSign(t, theirs, "member-1", GroupPermissions("G/", "G/"), now, time.Minute)

	_, err := mustKeySet(t, ours).Verify(signed, now)
	if !errors.Is(err, ErrUnknownKey) {
		t.Errorf("a token naming a key the set lacks gave %v, want ErrUnknownKey", err)
	}
}

func TestVerifyRefusesAnotherAlgorithm(t *testing.T) {
	s := mustSigner(t)
	now := time.Unix(1_800_000_000, 0)
	signed := mustSign(t, s, "member-1", GroupPermissions("G/", "G/"), now, time.Minute)

	parts := strings.Split(signed, ".")
	header, err := segment(map[string]string{"alg": "none", "kid": s.KeyID()})
	if err != nil {
		t.Fatalf("rendering the header: %v", err)
	}

	_, err = mustKeySet(t, s).Verify(header+"."+parts[1]+"."+parts[2], now)
	if !errors.Is(err, ErrUnverified) {
		t.Errorf("a header naming alg none gave %v, want ErrUnverified", err)
	}
}

func TestVerifyRefusesGarbage(t *testing.T) {
	s := mustSigner(t)
	for _, signed := range []string{"", "a.b", "a.b.c", "...", "x.y.z.w"} {
		if _, err := mustKeySet(t, s).Verify(signed, time.Now()); !errors.Is(err, ErrUnverified) {
			t.Errorf("%q gave %v, want ErrUnverified", signed, err)
		}
	}
}

func TestGrantsMatchesTheAnchoredPrefix(t *testing.T) {
	claims := Claims{Permissions: GroupPermissions("G/", "G/Bob/")}

	cases := []struct {
		action, path string
		want         bool
	}{
		{ActionRead, "G/Bob/screen", true},
		{ActionRead, "G/Alice/screen", true},
		{ActionRead, "H/Bob/screen", false},
		{ActionRead, "xG/Bob/screen", false},
		{ActionPublish, "G/Bob/screen", true},
		{ActionPublish, "G/Alice/screen", false},
		{ActionAPI, "G/Bob/screen", false},
	}
	for _, c := range cases {
		if got := claims.Grants(c.action, c.path); got != c.want {
			t.Errorf("Grants(%s, %s) = %v, want %v", c.action, c.path, got, c.want)
		}
	}
}

func TestParseKeySetRefusesAKeyOffTheCurve(t *testing.T) {
	document := `{"keys":[{"kty":"EC","crv":"P-256","alg":"ES256","kid":"k","x":"AQ","y":"AQ"}]}`
	if _, err := ParseKeySet([]byte(document)); err == nil {
		t.Error("a point that is not on P-256 parsed into a key set")
	}
}
