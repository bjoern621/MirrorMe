package token

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"bjoernblessin.de/go-utils/util/assert"
)

// ErrUnverified refuses a token whose shape, algorithm, signature or expiry fails.
var ErrUnverified = errors.New("token does not verify")

// ErrUnknownKey refuses a token naming a key the set does not hold.
// Apart from ErrUnverified because a caller may fetch the set again on it:
// groupd drew a new key, and the set in hand predates it.
var ErrUnknownKey = errors.New("token names a key the key set does not hold")

// KeySet maps a key id to the public key a JWKS publishes under it.
type KeySet map[string]*ecdsa.PublicKey

// Claims are what a verified token states.
type Claims struct {
	Subject     string
	Expires     time.Time
	Permissions []Permission
}

// ParseKeySet reads a JWKS document.
// A key of another type or curve verifies no token signed here, so it is left out.
// Errors if the document does not parse or a P-256 key is no point on the curve.
func ParseKeySet(document []byte) (KeySet, error) {
	var parsed struct {
		Keys []struct {
			Kty, Crv, Kid, X, Y string
		} `json:"keys"`
	}
	if err := json.Unmarshal(document, &parsed); err != nil {
		return nil, fmt.Errorf("reading a key set: %w", err)
	}

	keys := KeySet{}
	for _, key := range parsed.Keys {
		if key.Kty != "EC" || key.Crv != Curve {
			continue
		}
		x, errX := raw.DecodeString(key.X)
		y, errY := raw.DecodeString(key.Y)
		if errX != nil || errY != nil || len(x) != CoordinateBytes || len(y) != CoordinateBytes {
			return nil, fmt.Errorf("key %q carries coordinates that are not two %d-byte numbers", key.Kid, CoordinateBytes)
		}
		point := append([]byte{0x04}, append(x, y...)...)
		public, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key.Kid, err)
		}
		keys[key.Kid] = public
	}
	return keys, nil
}

// Verify checks a token's signature against the key its header names and its expiry against now.
// Errors with ErrUnknownKey if the set lacks that key, and with ErrUnverified on every other refusal.
func (k KeySet) Verify(signed string, now time.Time) (Claims, error) {
	parts := strings.Split(signed, ".")
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("%w: %d segments", ErrUnverified, len(parts))
	}

	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeSegment(parts[0], &header); err != nil {
		return Claims{}, err
	}
	if header.Alg != Algorithm {
		return Claims{}, fmt.Errorf("%w: algorithm %q, want %s", ErrUnverified, header.Alg, Algorithm)
	}
	public, held := k[header.Kid]
	if !held {
		return Claims{}, fmt.Errorf("%w: %q", ErrUnknownKey, header.Kid)
	}

	signature, err := raw.DecodeString(parts[2])
	if err != nil || len(signature) != 2*CoordinateBytes {
		return Claims{}, fmt.Errorf("%w: signature is not two %d-byte numbers", ErrUnverified, CoordinateBytes)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:CoordinateBytes])
	s := new(big.Int).SetBytes(signature[CoordinateBytes:])
	if !ecdsa.Verify(public, digest[:], r, s) {
		return Claims{}, fmt.Errorf("%w: signature does not match key %q", ErrUnverified, header.Kid)
	}

	var payload struct {
		Sub         string       `json:"sub"`
		Exp         int64        `json:"exp"`
		Permissions []Permission `json:"mediamtx_permissions"`
	}
	if err := decodeSegment(parts[1], &payload); err != nil {
		return Claims{}, err
	}
	expires := time.Unix(payload.Exp, 0)
	if !now.Before(expires) {
		return Claims{}, fmt.Errorf("%w: expired at %s", ErrUnverified, expires.UTC().Format(time.RFC3339))
	}

	return Claims{Subject: payload.Sub, Expires: expires, Permissions: payload.Permissions}, nil
}

// decodeSegment reads one base64 JSON segment of a token.
func decodeSegment(encoded string, into any) error {
	decoded, err := raw.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnverified, err)
	}
	if err := json.Unmarshal(decoded, into); err != nil {
		return fmt.Errorf("%w: %v", ErrUnverified, err)
	}
	return nil
}

// Grants reports whether a permission allows action on path, matched the way the relay matches it:
// a path opening with a tilde reads as a regular expression, an empty one covers every path,
// and any other has to equal path.
func (c Claims) Grants(action, path string) bool {
	assert.Assert(action != "", "a grant is asked about an action")

	for _, p := range c.Permissions {
		if p.Action == action && pathMatches(p.Path, path) {
			return true
		}
	}
	return false
}

func pathMatches(granted, path string) bool {
	if granted == "" {
		return true
	}
	expression, isExpression := strings.CutPrefix(granted, "~")
	if !isExpression {
		return granted == path
	}
	compiled, err := regexp.Compile(expression)
	if err != nil {
		return false
	}
	return compiled.MatchString(path)
}
