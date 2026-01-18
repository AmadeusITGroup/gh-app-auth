# Token Caching and Security

## Overview

`gh-app-auth` uses two types of tokens for authentication:

### 1. JWT Tokens (Short-lived)

- **Purpose**: Authenticate as the GitHub App to request installation tokens
- **Validity**: ~10 minutes (GitHub default)
- **Storage**: Generated on-demand, **NOT cached**
- **Security**: Generated fresh each time, minimal exposure window

### 2. Installation Tokens (Long-lived)

- **Purpose**: Authenticate git operations and API calls
- **Validity**: 1 hour (GitHub default)
- **Storage**: **In-memory cache scoped to the running process only** (55-minute TTL with 5-minute safety buffer)
- **Security**: Memory-only, zeroed on cleanup
- **Important**: Each gh-app-auth invocation starts with an empty cache. Git's credential helper protocol launches a fresh process per request, so caching only helps commands that make multiple token requests inside the same process (e.g., `gh app-auth test`, `gh app-auth debug`).

## Current Implementation

### Token Cache Location

Installation tokens are cached **only in memory** using `pkg/cache/cache.go`:

```go
// In-memory cache structure
type TokenCache struct {
    mu    sync.RWMutex
    cache map[string]*CachedToken
}

type CachedToken struct {
    Token     string
    ExpiresAt time.Time
    CreatedAt time.Time
}
```

### Cache Key Format

```go
cacheKey := fmt.Sprintf("app_%d_inst_%d", appID, installationID)
```

### Expiration Check

Tokens are automatically checked for expiration on every `Get()` call:

```go
func (c *TokenCache) Get(key string) (string, bool) {
    cached, exists := c.cache[key]
    if !exists || time.Now().After(cached.ExpiresAt) {
        return "", false  // Expired or not found
    }
    return cached.Token, true
}
```

### Automatic Cleanup

Background goroutine runs every minute to remove expired tokens:

```go
func (c *TokenCache) startCleanupWorker() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        c.cleanup()  // Removes expired tokens
    }
}
```

### Memory Security

Tokens are zeroed out on deletion (best-effort):

```go
func (c *TokenCache) zeroToken(token string) {
    tokenBytes := []byte(token)
    for i := range tokenBytes {
        tokenBytes[i] = 0
    }
    runtime.GC()  // Encourage garbage collection
}
```

**Note**: Go strings are immutable, so this only clears our local copy. Original string may remain in memory until GC.

## Security Considerations

### ⚠️ Current Limitations

1. **Memory-Only Storage**
   - Tokens are NOT persisted to disk
   - Lost on process restart/crash
   - Requires re-authentication after restart

2. **No Encryption at Rest**
   - Unlike private keys (stored in OS keyring), tokens are in plain memory
   - Vulnerable to memory dumps/debugging tools
   - No encryption layer

3. **Process Memory Exposure**
   - Tokens exist in process memory space
   - Accessible via debuggers or memory inspection
   - Cannot be fully protected in user-space

### ✅ Security Measures in Place

1. **Short Cache TTL**
   - 55-minute cache (vs 60-minute validity)
   - 5-minute safety buffer reduces risk of using expired tokens
   - Limits exposure window

2. **Automatic Expiration**
   - Tokens checked on every access
   - Expired tokens rejected immediately
   - Background cleanup every minute

3. **Memory Zeroing**
   - Best-effort token clearing on deletion
   - Reduces (but doesn't eliminate) memory exposure

4. **Thread-Safe**
   - Mutex-protected cache access
   - Prevents race conditions in concurrent scenarios

## Usage Flow

### First Request (Cache Miss)

```
1. Check cache → Not found
2. Load private key from secure storage (keyring/filesystem)
3. Generate JWT token (10-min validity)
4. Request installation token from GitHub API
5. Cache installation token (55-min TTL)
6. Return token for git operation
```

### Subsequent Requests (Cache Hit)

```
1. Check cache → Found and valid
2. Return cached token immediately
3. No API calls to GitHub
```

### Token Expiration

```
1. Check cache → Found but expired (time.Now() > ExpiresAt)
2. Return "not found"
3. Trigger new authentication flow (as above)
```

## Performance Impact

### Cache Benefits

- **Reduced API Calls**: One call per 55 minutes instead of per operation
- **Faster Operations**: No JWT generation or API roundtrip on cache hit
- **Lower Rate Limits**: Fewer API requests preserves quota

### Typical Latency

- **Cache Hit**: <1ms (memory lookup)
- **Cache Miss**: 200-500ms (JWT gen + API call + keyring access)

## Future Enhancements

### Potential Improvements

1. **Persistent Secure Cache**
   - Store tokens in OS keyring (like private keys)
   - Persist across process restarts
   - Encrypted at rest
   - **Tradeoff**: Longer token exposure window if keyring compromised

2. **Token Refresh**
   - Proactive token renewal before expiration
   - Reduce "cache miss" latency
   - Background refresh for active tokens

3. **Metrics & Monitoring**
   - Cache hit/miss rates
   - Token generation frequency
   - API call reduction statistics

### Why NOT Implemented Yet

**Persistent Token Storage Risks:**

- Installation tokens are powerful (1-hour validity, full repo access)
- Storing in keyring increases attack surface
- If keyring compromised, attacker has hour-long access window
- Current approach: tokens ephemeral, limited to process lifetime

**Security vs. Convenience Tradeoff:**

- Memory-only = More secure, less convenient
- Persistent = More convenient, less secure
- Current design prioritizes security

## Debugging

### Check Cache Status

```bash
# No built-in command yet, but you can observe behavior:

# First operation (cache miss) - slower
time git clone https://github.com/org/repo1.git

# Second operation within 55 minutes (cache hit) - faster  
time git clone https://github.com/org/repo2.git
```

### Cache Statistics

Cache size and stats available programmatically:

```go
auth := auth.NewAuthenticator()
stats := auth.tokenCache.GetStats()
// stats.TotalTokens, stats.ValidTokens, stats.ExpiredTokens
```

## Comparison with gh CLI

The official `gh` CLI stores OAuth tokens differently:

### gh CLI Approach

- **Token Type**: Personal OAuth tokens (indefinite validity until revoked)
- **Storage**: OS keyring (persistent)
- **Security**: Encrypted at rest, persistent across sessions
- **Use Case**: User authentication, not programmatic GitHub Apps

### gh-app-auth Approach

- **Token Type**: GitHub App installation tokens (1-hour validity)
- **Storage**: Memory only (ephemeral)
- **Security**: Process-lifetime only, no persistent storage
- **Use Case**: Automated/programmatic access via GitHub Apps

### Why Different?

1. **Validity Duration**
   - OAuth tokens: Indefinite (safe to persist)
   - Installation tokens: 1 hour (less critical to persist)

2. **Risk Profile**
   - OAuth: User-level access (all repos user can access)
   - Installation: App-level access (only configured repos)

3. **Refresh Complexity**
   - OAuth: Requires user interaction to refresh
   - Installation: Can auto-refresh without user (we have private key)

## Recommendations

### For Development

- Memory-only cache is sufficient
- Process restarts are infrequent
- Re-authentication overhead is acceptable (200-500ms)

### For CI/CD

- Memory-only cache is ideal
- Ephemeral environments (containers) restart frequently anyway
- Tokens don't need to persist beyond job execution

### For Long-Running Daemons

- Consider implementing persistent cache
- Balance security vs. convenience
- Monitor for suspicious keyring access

## FAQ

**Q: Why aren't tokens cached to disk like private keys?**  
A: Private keys have indefinite validity and are required for operation. Installation tokens expire in 1 hour and can be regenerated. The security risk of persistent token storage outweighs the convenience benefit.

**Q: What happens if the process crashes?**  
A: All cached tokens are lost. Next git operation will regenerate tokens automatically. Typical overhead: 200-500ms.

**Q: Can I clear the token cache?**  
A: Cache is automatically cleared on process exit. To force refresh, restart the credential helper or wait for natural expiration (55 minutes).

**Q: How many API calls does caching save?**  
A: Without caching: 1 API call per git operation. With caching: ~1 API call per 55 minutes. The savings depend on how many git operations you perform within the cache window.

**Q: Is the cache secure?**  
A: Memory-only cache is reasonably secure for process lifetime. Tokens are zeroed on cleanup (best-effort). For maximum security, tokens are never written to disk unencrypted.

## Related Documentation

- [Encrypted Key Storage](../README.md#encrypted-storage)
- [Security Best Practices](SECURITY.md)
- [Architecture Overview](ARCHITECTURE.md)
