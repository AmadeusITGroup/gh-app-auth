---
mode: ask
description: "Perform a security review of gh-app-auth code"
---

# Security Review for gh-app-auth

Perform a security assessment of the specified code focusing on authentication and credential handling.

## Security Checklist

### 1. Secret Handling

- [ ] No tokens, keys, or passwords logged in plain text
- [ ] Secrets use `secrets.HashToken()` for debug logging
- [ ] Sensitive data zeroed from memory after use
- [ ] No hardcoded credentials or test secrets

### 2. Input Validation

- [ ] Path traversal prevention (`../` rejected)
- [ ] URL validation and normalization
- [ ] App ID validation (numeric only)
- [ ] Pattern validation for credential routing

### 3. File Security

- [ ] Private key files require 600/400 permissions
- [ ] Config files created with 600 permissions
- [ ] No world-readable sensitive files

### 4. Token Security

- [ ] Installation tokens cached in memory only
- [ ] Token TTL enforced (55 minutes max)
- [ ] Automatic expiration on access
- [ ] No persistent token storage

### 5. Keyring Usage

- [ ] OS-native keyring preferred over filesystem
- [ ] Graceful degradation if keyring unavailable
- [ ] Timeout protection on keyring operations (3s max)
- [ ] User informed of storage method used

### 6. Error Handling

- [ ] Errors don't expose sensitive information
- [ ] Failed authentication doesn't leak valid usernames
- [ ] Rate limiting considered for repeated failures

## Review Output Format

Provide findings in this format:

### Critical Issues

_Issues that must be fixed before release_

### High Priority

_Security improvements that should be addressed_

### Recommendations

_Best practice suggestions_

### Passed Checks

_Security measures correctly implemented_

## Files to Review

Focus on these security-critical paths:

- `pkg/auth/` - Authentication logic
- `pkg/secrets/` - Key and token storage
- `pkg/jwt/` - JWT generation
- `cmd/git-credential.go` - Credential helper
- `cmd/setup.go` - Initial configuration
