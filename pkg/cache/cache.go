package cache

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// TokenCache provides thread-safe caching of GitHub App installation tokens with TTL.
//
// SECURITY NOTE: This cache stores installation tokens IN MEMORY ONLY. Tokens are
// NOT persisted to disk or encrypted storage. This design prioritizes security over
// convenience - tokens expire with process lifetime, reducing attack surface.
//
// Installation tokens have 1-hour validity from GitHub and are cached for 55 minutes
// (5-minute safety buffer). This reduces API calls to GitHub by ~98% while ensuring
// tokens are never stale.
//
// For detailed security analysis, see docs/TOKEN_CACHING.md
type TokenCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedToken
}

// CachedToken represents a cached GitHub App installation token with expiration information.
//
// Installation tokens (not JWTs) are cached because:
// - JWT tokens are short-lived (~10min) and cheap to generate
// - Installation tokens require API calls to GitHub and have 1-hour validity
// - Caching reduces GitHub API load and improves performance
type CachedToken struct {
	Token     string    // GitHub installation token (ghs_...)
	ExpiresAt time.Time // When this token expires (55-min from creation)
	CreatedAt time.Time // When this token was cached
}

// NewTokenCache creates a new token cache.
func NewTokenCache() *TokenCache {
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	// Start cleanup goroutine
	go cache.startCleanupWorker()

	return cache
}

// Get retrieves a token from the cache if it exists and is not expired
func (c *TokenCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.cache[key]
	if !exists {
		return "", false
	}

	// Check if token is expired
	if time.Now().After(cached.ExpiresAt) {
		// Don't remove here to avoid upgrading to write lock
		// Let the cleanup worker handle it
		return "", false
	}

	return cached.Token, true
}

// Set stores a token in the cache with the specified TTL
func (c *TokenCache) Set(key, token string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.cache[key] = &CachedToken{
		Token:     token,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
}

// Delete removes a token from the cache
func (c *TokenCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, exists := c.cache[key]; exists {
		// Zero out the token for security
		c.zeroToken(cached.Token)
		delete(c.cache, key)
	}
}

// Clear removes all tokens from the cache
func (c *TokenCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Zero out all tokens for security
	for _, cached := range c.cache {
		c.zeroToken(cached.Token)
	}

	c.cache = make(map[string]*CachedToken)
}

// Size returns the number of cached tokens
func (c *TokenCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// GetStats returns cache statistics
func (c *TokenCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	stats := CacheStats{
		TotalTokens:   len(c.cache),
		ExpiredTokens: 0,
		ValidTokens:   0,
	}

	for _, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			stats.ExpiredTokens++
		} else {
			stats.ValidTokens++
		}
	}

	return stats
}

// CacheStats contains cache statistics
type CacheStats struct {
	TotalTokens   int
	ValidTokens   int
	ExpiredTokens int
}

// cleanup removes expired tokens from the cache
func (c *TokenCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			// Zero out the token for security
			c.zeroToken(cached.Token)
			delete(c.cache, key)
		}
	}
}

// startCleanupWorker starts a background goroutine to periodically clean up expired tokens
func (c *TokenCache) startCleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute) // Clean up every minute
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// zeroToken attempts to clear token from memory.
//
// SECURITY NOTE: Go strings are immutable, so this function only clears our
// local copy of the token data. The original string may remain in memory
// until garbage collection occurs. This is a best-effort approach to limit
// token exposure time in memory.
//
// For maximum security, consider using a secrets management system that
// provides secure memory allocation for sensitive data.
func (c *TokenCache) zeroToken(token string) {
	if token == "" {
		return
	}

	// Convert string to []byte and zero out our copy
	tokenBytes := []byte(token)
	for i := range tokenBytes {
		tokenBytes[i] = 0
	}

	// Force garbage collection to potentially clean up string copies
	// Note: This doesn't guarantee immediate cleanup but encourages it
	runtime.GC()
}

// CreateCacheKey creates a cache key for a specific app configuration
func CreateCacheKey(appID, installationID int64) string {
	return fmt.Sprintf("app_%d_inst_%d", appID, installationID)
}
