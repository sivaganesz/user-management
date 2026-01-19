package utils

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key type (RSA)
	Use string `json:"use"` // Key use (sig for signature)
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm (RS256)
	N   string `json:"n"`   // RSA modulus (base64url encoded)
	E   string `json:"e"`   // RSA exponent (base64url encoded)
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKSCache handles fetching and caching of JWKS
type JWKSCache struct {
	endpoint      string
	cacheDuration time.Duration
	keys          map[string]*rsa.PublicKey // kid -> public key
	lastFetch     time.Time
	mu            sync.RWMutex
}

// NewJWKSCache creates a new JWKS cache
func NewJWKSCache(endpoint string, cacheDuration time.Duration) *JWKSCache {
	return &JWKSCache{
		endpoint:      endpoint,
		cacheDuration: cacheDuration,
		keys:          make(map[string]*rsa.PublicKey),
	}
}

// GetPublicKey retrieves a public key by key ID (kid)
// Fetches from cache if available and not expired, otherwise refetches from endpoint
func (c *JWKSCache) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()

	// Check if key exists in cache and cache is not expired
	if key, exists := c.keys[kid]; exists && time.Since(c.lastFetch) < c.cacheDuration {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	// Cache miss or expired - fetch from endpoint
	if err := c.fetchKeys(); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Try again after fetch
	c.mu.RLock()
	defer c.mu.RUnlock()

	if key, exists := c.keys[kid]; exists {
		return key, nil
	}

	return nil, fmt.Errorf("key not found in JWKS: kid=%s", kid)
}

// fetchKeys fetches the JWKS from the endpoint and updates the cache
func (c *JWKSCache) fetchKeys() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if another goroutine already fetched while we were waiting
	if time.Since(c.lastFetch) < 1*time.Second {
		return nil
	}

	// Fetch JWKS from endpoint
	resp, err := http.Get(c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS from %s: %w", c.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	// Parse JWKS
	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKs to RSA public keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue // Skip non-RSA keys
		}

		publicKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			// Log error but continue processing other keys
			fmt.Printf("Warning: failed to convert JWK to RSA public key (kid=%s): %v\n", jwk.Kid, err)
			continue
		}

		newKeys[jwk.Kid] = publicKey
	}

	if len(newKeys) == 0 {
		return fmt.Errorf("no valid RSA keys found in JWKS")
	}

	// Update cache
	c.keys = newKeys
	c.lastFetch = time.Now()

	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode base64url-encoded modulus (N)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url-encoded exponent (E)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return publicKey, nil
}

// ValidateRS256Token validates a token signed with RS256 using JWKS
func (c *JWKSCache) ValidateRS256Token(tokenString string) (jwt.MapClaims, error) {
	// Parse token without validation to get the kid from header
	unverifiedToken, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get kid from token header
	kidInterface, exists := unverifiedToken.Header["kid"]
	if !exists {
		return nil, fmt.Errorf("token missing kid in header")
	}

	kid, ok := kidInterface.(string)
	if !ok {
		return nil, fmt.Errorf("invalid kid type in token header")
	}

	// Get public key from cache
	publicKey, err := c.GetPublicKey(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Validate token with public key
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// RefreshCache forces a refresh of the JWKS cache
func (c *JWKSCache) RefreshCache() error {
	return c.fetchKeys()
}

// GetCachedKeyCount returns the number of keys currently in cache
func (c *JWKSCache) GetCachedKeyCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.keys)
}

// GetLastFetchTime returns the time of the last successful fetch
func (c *JWKSCache) GetLastFetchTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastFetch
}
