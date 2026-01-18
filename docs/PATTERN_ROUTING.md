# Pattern-Based Routing for Git Credentials

This document describes how to use the `--pattern` flag to route different repositories to different GitHub Apps.

## Overview

The `--pattern` flag allows you to explicitly specify which GitHub App should handle authentication for specific repository patterns. This is useful when you have multiple GitHub Apps configured and want fine-grained control over which app is used for which repositories.

## Why Use Pattern-Based Routing?

### Use Cases

1. **Multiple Organizations**: Different GitHub Apps for different organizations
2. **Enterprise + Cloud**: Separate apps for GitHub Enterprise and GitHub.com
3. **Access Control**: Different permission scopes for different repository groups
4. **Testing**: Use a test app for specific repos while keeping production app for others

### Traditional Approach vs Pattern Routing

**Without `--pattern` (Auto-matching):**

```bash
# All repos use first matching pattern from config
git config --global credential.helper '!/path/to/gh-app-auth git-credential'
```

- Relies on pattern priority in config file
- First match wins
- Less explicit about which app handles which repo

**With `--pattern` (Explicit routing):**

```bash
# Explicitly route specific org to specific app
git config credential.'https://github.com/myorg'.helper '!/path/to/gh-app-auth git-credential --pattern "github.com/myorg/*"'
```

- Git's native URL scoping
- Explicit app selection
- Multiple apps can coexist

## Configuration

### Step 1: Configure Your GitHub Apps

First, set up your apps in `~/.config/gh/extensions/gh-app-auth/config.yml`:

```yaml
github_apps:
  - app_id: 111111
    name: "AmadeusITGroup App"
    private_key_path: ~/.config/gh/extensions/gh-app-auth/keys/amadeus.pem
    patterns:
      - "github.com/AmadeusITGroup/*"
    
  - app_id: 222222
    name: "Personal Projects App"
    private_key_path: ~/.config/gh/extensions/gh-app-auth/keys/personal.pem
    patterns:
      - "github.com/myusername/*"
      - "github.com/myorg/*"
    
  - app_id: 333333
    name: "Enterprise App"
    private_key_path: ~/.config/gh/extensions/gh-app-auth/keys/enterprise.pem
    patterns:
      - "github.enterprise.com/*/*"
```

### Step 2: Configure Git Credential Helpers

#### Option A: Organization-Specific Routing

```bash
# AmadeusITGroup repositories
git config --global credential.'https://github.com/AmadeusITGroup'.helper ""
git config --global --add credential.'https://github.com/AmadeusITGroup'.helper \
  '!/home/youruser/.local/share/gh/extensions/gh-app-auth/gh-app-auth git-credential --pattern "github.com/AmadeusITGroup/*"'

# Personal repositories
git config --global credential.'https://github.com/myorg'.helper ""
git config --global --add credential.'https://github.com/myorg'.helper \
  '!/home/youruser/.local/share/gh/extensions/gh-app-auth/gh-app-auth git-credential --pattern "github.com/myorg/*"'

# Enterprise repositories
git config --global credential.'https://github.enterprise.com'.helper ""
git config --global --add credential.'https://github.enterprise.com'.helper \
  '!/home/youruser/.local/share/gh/extensions/gh-app-auth/gh-app-auth git-credential --pattern "github.enterprise.com/*/*"'
```

#### Option B: Repository-Specific Routing

For even finer control, configure at the repository level:

```bash
cd /path/to/your/repo

# Use specific app for this repo only
git config credential.helper ""
git config --add credential.helper \
  '!/home/youruser/.local/share/gh/extensions/gh-app-auth/gh-app-auth git-credential --pattern "github.com/AmadeusITGroup/*"'
```

#### Option C: Mixed Approach

Combine specific and fallback configurations:

```bash
# Specific organization
git config --global credential.'https://github.com/AmadeusITGroup'.helper \
  '!/path/to/gh-app-auth git-credential --pattern "github.com/AmadeusITGroup/*"'

# Global fallback (no pattern - uses auto-matching)
git config --global credential.helper \
  '!/path/to/gh-app-auth git-credential'
```

## How It Works

### Pattern Matching Process

1. **Git calls credential helper** with repository URL
2. **Helper reads `--pattern` flag** (if provided)
3. **Pattern matching:**
   - **With `--pattern`**: Searches config for app with exactly this pattern
   - **Without `--pattern`**: Uses URL-based matching (original behavior)
4. **App found**: Generates credentials and returns to git
5. **No app found**: Silently exits (git tries next helper or prompts)

### Pattern Format

Patterns must match **exactly** as configured in your config file:

```yaml
patterns:
  - "github.com/AmadeusITGroup/*"    # ✓ Correct
  - "github.com/org/*"                # ✓ Correct
  - "github.enterprise.com/*/*"      # ✓ Correct
```

The pattern you pass to `--pattern` must be identical:

```bash
# ✓ Correct - exact match
--pattern "github.com/AmadeusITGroup/*"

# ✗ Wrong - won't match
--pattern "github.com/AmadeusITGroup"
--pattern "AmadeusITGroup/*"
```

## Examples

### Example 1: Multiple Organizations

**Scenario**: You work with two organizations, each with its own GitHub App.

**Setup**:

```bash
# Organization A
git config --global credential.'https://github.com/orgA'.helper \
  '!gh-app-auth git-credential --pattern "github.com/orgA/*"'

# Organization B
git config --global credential.'https://github.com/orgB'.helper \
  '!gh-app-auth git-credential --pattern "github.com/orgB/*"'
```

**Usage**:

```bash
# Automatically uses orgA's app
git clone https://github.com/orgA/private-repo

# Automatically uses orgB's app
git clone https://github.com/orgB/private-repo
```

### Example 2: Enterprise and Cloud

**Scenario**: You have repos on both GitHub Enterprise and GitHub.com.

**Setup**:

```bash
# Enterprise
git config --global credential.'https://github.enterprise.com'.helper \
  '!gh-app-auth git-credential --pattern "github.enterprise.com/*/*"'

# Cloud
git config --global credential.'https://github.com'.helper \
  '!gh-app-auth git-credential --pattern "github.com/*/*"'
```

### Example 3: Testing New App

**Scenario**: Test a new GitHub App on one repository before rolling out.

**Setup**:

```bash
cd /path/to/test-repo

# Use test app for this repo only
git config credential.helper \
  '!gh-app-auth git-credential --pattern "github.com/org/test-repo"'
```

## Troubleshooting

### Enable Debug Logging

```bash
export GH_APP_AUTH_DEBUG_LOG=1
# or
export GH_APP_AUTH_DEBUG_LOG=/tmp/gh-app-auth-debug.log

# Run your git command
git clone https://github.com/org/repo

# View logs
tail -f ~/.config/gh/extensions/gh-app-auth/debug.log
```

### Common Issues

#### 1. Pattern Not Found

**Log shows**: `no_pattern_match`

**Cause**: The pattern doesn't exist in any configured app.

**Solution**: Check your config file and ensure the pattern matches exactly:

```bash
gh app-auth list  # View configured patterns
```

#### 2. Git Still Prompts for Credentials

**Cause**: Git isn't calling your credential helper.

**Solution**: Verify git configuration:

```bash
git config --get-all credential.helper
git config --get-all credential."https://github.com/org".helper
```

#### 3. Wrong App Being Used

**Log shows**: `app_matched_by_pattern` with unexpected app_id

**Cause**: Multiple apps might have the same pattern.

**Solution**: Make patterns unique per app, or use more specific URL scoping in git config.

### View Active Configuration

```bash
# Global credential helpers
git config --global --get-regexp credential

# Repository-specific helpers
cd /path/to/repo
git config --get-regexp credential

# Test which helper git would use
GIT_TRACE=1 git ls-remote https://github.com/org/repo 2>&1 | grep credential
```

## Performance

- **Minimal overhead**: Pattern matching is a simple string comparison
- **No network calls**: Matching happens locally before authentication
- **Caching**: Tokens are still cached per repository URL
- **Same speed**: Identical performance to auto-matching mode

## Security Considerations

### Pattern Exposure

Patterns are visible in git configuration:

```bash
git config --list | grep credential
```

This is safe because patterns are **public information** (repository URLs), not secrets.

### Private Keys

Private keys remain protected:

- Stored separately from patterns
- File permissions: `0600`
- Never logged or exposed

### Token Handling

Tokens generated via pattern routing have the same security as auto-matched tokens:

- Short-lived (1 hour expiration)
- Scoped to repository
- Cached securely

## Migration Guide

### From Auto-Matching to Pattern Routing

**Before** (auto-matching):

```bash
git config --global credential.helper '!gh-app-auth git-credential'
```

**After** (pattern routing):

```bash
# Clear old config
git config --global --unset credential.helper

# Add pattern-specific configs
git config --global credential.'https://github.com/orgA'.helper \
  '!gh-app-auth git-credential --pattern "github.com/orgA/*"'

git config --global credential.'https://github.com/orgB'.helper \
  '!gh-app-auth git-credential --pattern "github.com/orgB/*"'

# Optional: Add fallback for unmatched repos
git config --global credential.helper '!gh-app-auth git-credential'
```

### Gradual Rollout

1. **Keep existing config** (auto-matching)
2. **Add one pattern-specific config** for testing
3. **Verify it works** with debug logging
4. **Add more patterns** incrementally
5. **Remove fallback** once all repos covered

## Advanced Usage

### Combining with Other Helpers

```bash
# Try gh-app-auth first, fall back to gh CLI
git config --global credential.helper \
  '!gh-app-auth git-credential --pattern "github.com/myorg/*"'
git config --global --add credential.helper \
  '!gh auth git-credential'
```

### Per-Repository Override

Repository config takes precedence over global:

```bash
# Global: use App A
git config --global credential.'https://github.com/org'.helper \
  '!gh-app-auth git-credential --pattern "github.com/org/*"'

# This repo: use App B instead
cd /path/to/special-repo
git config credential.helper \
  '!gh-app-auth git-credential --pattern "github.com/specialorg/*"'
```

## See Also

- [Git Credential Helper Documentation](https://git-scm.com/docs/gitcredentials)
- [Diagnostic Logging](./DIAGNOSTIC_LOGGING.md)
- [GitHub App Setup Guide](../README.md#setup)
