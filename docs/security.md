# Security Considerations

This document covers security best practices for using `gh-app-auth` in production environments.

## Private Key Security

### File Permissions

The extension enforces strict file permissions for private key files:

- **Required**: `600` (owner read/write) or `400` (owner read-only)
- **Rejected**: World-readable or group-readable permissions

```bash
# Set correct permissions
chmod 600 ~/.ssh/my-app.private-key.pem
```

### Storage Options

| Method | Security Level | Use Case |
|--------|---------------|----------|
| **OS Keyring** (default) | ✅ Highest | Local development, persistent workstations |
| **Environment Variable** | ✅ High | CI/CD pipelines with secrets management |
| **Filesystem** (fallback) | ⚠️ Medium | Headless servers without keyring |

#### OS Keyring (Recommended)

Keys are encrypted using platform-native security:

- **macOS**: Keychain (AES-256)
- **Windows**: Credential Manager (DPAPI)
- **Linux**: Secret Service API (GNOME Keyring, KWallet)

```bash
# Store via environment variable (recommended)
export GH_APP_PRIVATE_KEY="$(cat ~/my-key.pem)"
gh app-auth setup --app-id 123456 --patterns "github.com/myorg/*"
unset GH_APP_PRIVATE_KEY
```

#### Environment Variables (CI/CD)

For automated pipelines, use your CI system's secrets management:

```yaml
# GitHub Actions example
- name: Setup GitHub App
  env:
    GH_APP_PRIVATE_KEY: ${{ secrets.APP_PRIVATE_KEY }}
  run: gh app-auth setup --app-id ${{ secrets.APP_ID }} --patterns "github.com/myorg/*"
```

## Token Security

### In-Memory Only

Installation tokens are:

- **Never written to disk**
- **Cached in memory** for 55 minutes (GitHub provides 60-minute validity)
- **Zeroed from memory** on cleanup (best-effort)
- **Lost on process restart** (re-authentication required)

### Why Not Persistent Caching?

Installation tokens provide powerful repository access. We prioritize security:

| Approach | Risk | Benefit |
|----------|------|---------|
| Memory-only | Re-auth on restart (~500ms) | No persistent tokens to compromise |
| Disk cache | Potential token theft | Slightly faster cold starts |

## Configuration Security

### Config File Permissions

Configuration files are stored with restricted permissions:

```
~/.config/gh/extensions/gh-app-auth/config.yml  # Mode 600
```

### What's Stored

| Data | Storage Location | Encrypted |
|------|------------------|-----------|
| App IDs | Config file | No (not sensitive) |
| Installation IDs | Config file | No (not sensitive) |
| Patterns | Config file | No (not sensitive) |
| Private keys | OS Keyring | Yes (platform-native) |
| PATs | OS Keyring | Yes (platform-native) |

### What's NOT Stored

- Installation tokens (memory only)
- JWT tokens (generated on demand)
- API responses

## Input Validation

All user inputs are validated:

- **Path traversal prevention**: `../` and absolute paths are rejected for key files
- **Pattern validation**: URL patterns are normalized and validated
- **App ID validation**: Numeric values only
- **Command injection prevention**: Inputs are never passed to shell commands

## Logging Security

The extension follows secure logging practices:

- ✅ Debug logs include flow steps and timing
- ✅ Token hashes (SHA-256) are logged for debugging
- ❌ **Never logged**: Full tokens, private keys, or secrets

### Debug Log Example

```
[2024-10-13T20:02:03.400Z] FLOW_STEP step=generate_jwt app_id=123456
[2024-10-13T20:02:03.450Z] FLOW_STEP step=exchange_token token_hash=sha256:abc123...
```

## CI/CD Best Practices

### GitHub Actions

```yaml
- name: Setup with cleanup
  uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    app-id: ${{ secrets.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    patterns: 'github.com/myorg/*'
    cleanup-on-exit: 'true'  # Removes credentials after job
```

### Jenkins

```groovy
post {
    always {
        sh 'rm -f /tmp/*.pem'  // Clean up key files
        sh 'gh app-auth remove --all --force || true'  // Remove config
    }
}
```

### Self-Hosted Runners

For non-ephemeral runners, always clean up credentials:

```bash
# After job completion
gh app-auth gitconfig --clean
gh app-auth remove --all --force
```

## Threat Model

### In Scope

| Threat | Mitigation |
|--------|------------|
| Private key theft from disk | OS keyring encryption |
| Token exposure in logs | Hash-only logging |
| Config file tampering | Strict file permissions |
| Path traversal attacks | Input validation |
| Credential persistence on runners | Cleanup actions |

### Out of Scope

| Threat | Reason |
|--------|--------|
| Compromised OS keyring | OS-level security |
| Memory dumps on compromised systems | Requires root access |
| GitHub API vulnerabilities | GitHub's responsibility |

## Reporting Security Issues

Please report security vulnerabilities through:

1. **GitHub Security Advisories** (preferred): [Report a vulnerability](https://github.com/AmadeusITGroup/gh-app-auth/security)
2. **Email**: See [SECURITY.md](../.github/SECURITY.md) for contact information

**Do not** report security issues through public GitHub issues.

## Security Checklist

Before deploying to production:

- [ ] Private keys have correct permissions (`600` or `400`)
- [ ] Keys are stored in OS keyring, not filesystem
- [ ] CI/CD pipelines use secrets management
- [ ] Cleanup steps configured for non-ephemeral runners
- [ ] Debug logging disabled in production
- [ ] Patterns are as specific as possible
