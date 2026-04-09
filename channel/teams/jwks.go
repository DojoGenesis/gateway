package teams

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

const (
	// defaultJWKSURL is Microsoft's Bot Framework public key endpoint.
	// https://learn.microsoft.com/en-us/azure/bot-service/rest-api/bot-framework-rest-connector-authentication
	defaultJWKSURL = "https://login.botframework.com/v1/.well-known/keys"

	// jwksCacheTTL is how long fetched keys are considered fresh.
	// Microsoft rotates keys infrequently; 24 h is safe and avoids hammering.
	jwksCacheTTL = 24 * time.Hour

	// clockSkewTolerance is the leeway applied to exp/nbf checks.
	clockSkewTolerance = 5 * time.Minute

	// maxJWKSBodyBytes caps the response body read from the JWKS endpoint.
	// Microsoft's JWKS document is ~3 KiB; 128 KiB is generous headroom.
	maxJWKSBodyBytes = 128 * 1024
)

// jwkEntry is one RSA key from a JWKS document.
type jwkEntry struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"` // base64url-encoded modulus
	E   string `json:"e"` // base64url-encoded exponent
}

// jwksResponse is the envelope returned by the JWKS endpoint.
type jwksResponse struct {
	Keys []jwkEntry `json:"keys"`
}

// jwksCache fetches and caches RSA public keys from a JWKS endpoint. It is
// safe for concurrent use. Keys are re-fetched after jwksCacheTTL expires.
type jwksCache struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
	url     string
}

func newJWKSCache(url string) *jwksCache {
	return &jwksCache{url: url}
}

// getOrRefresh returns the RSA public key for kid. If the cache is stale or
// the key is absent, it fetches fresh keys from the JWKS endpoint once.
func (c *jwksCache) getOrRefresh(ctx context.Context, client *http.Client, kid string) (*rsa.PublicKey, error) {
	if key, ok := c.getCached(kid); ok {
		return key, nil
	}
	if err := c.refresh(ctx, client); err != nil {
		return nil, err
	}
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("teams: jwks: no key for kid %q", kid)
	}
	return key, nil
}

func (c *jwksCache) getCached(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.keys == nil || time.Now().After(c.expires) {
		return nil, false
	}
	key, ok := c.keys[kid]
	return key, ok
}

func (c *jwksCache) refresh(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("teams: jwks: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("teams: jwks: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams: jwks: server returned %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJWKSBodyBytes)).Decode(&jwks); err != nil {
		return fmt.Errorf("teams: jwks: decode: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, entry := range jwks.Keys {
		// Only cache RSA signing keys. Skip entries with no kid, wrong key
		// type, or an explicit use that is not "sig" (e.g. "enc").
		if entry.Kty != "RSA" || entry.Kid == "" || (entry.Use != "" && entry.Use != "sig") {
			continue
		}
		pub, err := rsaPublicKeyFromJWK(entry)
		if err != nil {
			// Skip bad entries rather than failing the whole refresh.
			continue
		}
		keys[entry.Kid] = pub
	}

	c.mu.Lock()
	c.keys = keys
	c.expires = time.Now().Add(jwksCacheTTL)
	c.mu.Unlock()

	return nil
}

// rsaPublicKeyFromJWK builds an *rsa.PublicKey from a JWK's n and e fields.
func rsaPublicKeyFromJWK(entry jwkEntry) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(entry.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(entry.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)

	// Guard against pathological exponent lengths before shifting into int.
	// Standard RSA uses 3 bytes (e=65537=0x10001) or 1 byte (e=3).
	if len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, fmt.Errorf("teams: jwks: unexpected exponent length %d", len(eBytes))
	}
	var e int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, fmt.Errorf("teams: jwks: zero exponent")
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}
