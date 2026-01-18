# Git Configuration Management

The `gitconfig` command automates the setup and cleanup of git credential helpers for gh-app-auth, eliminating manual configuration steps.

## Overview

Previously, after configuring a GitHub App (or PAT), you needed to manually run git commands:

```bash
# Old manual approach ❌
gh app-auth setup --app-id 123456 --key-file app.pem --patterns "github.com/org/*"
git config --global credential."https://github.com/org".helper "!gh app-auth git-credential --pattern 'github.com/org/*'"
```

Now, you can automate this for every GitHub App **and Personal Access Token**:

```bash
# New automated approach ✅
gh app-auth setup --app-id 123456 --key-file app.pem --patterns "github.com/org/*"
gh app-auth gitconfig --sync
```

## Commands

### Sync Configuration

Automatically configures git credential helpers for all configured GitHub Apps **and PATs**:

```bash
gh app-auth gitconfig --sync
```

**What it does:**

1. Reads your gh-app-auth configuration
2. Extracts all configured patterns (Apps + PATs)
3. Configures git credential helpers for each pattern
4. Sets up automatic token authentication

**Output example:**

```
Configuring git credential helpers (--global)...

✅ Configured: https://github.com/org1
   Credential: Org1 App (ID: 123456)
   Pattern: github.com/org1/*

✅ Configured: https://github.com/org2
   Credential: Org2 App (ID: 789012)
   Pattern: github.com/org2/*

✨ Successfully configured 2 credential helper(s)

You can now use git commands and they will authenticate using GitHub Apps or PATs (with automatic username overrides for Bitbucket if configured):
  git clone https://github.com/org/repo
  git submodule update --init --recursive
```

### Clean Configuration

Removes all gh-app-auth git credential helper configurations:

```bash
gh app-auth gitconfig --clean
```

**What it does:**

1. Scans git configuration for gh-app-auth helpers
2. Removes each configured helper
3. Leaves other git configurations intact

**Output example:**

```
Cleaning gh-app-auth git configurations (--global)...

🗑️  Removed: https://github.com/org1
🗑️  Removed: https://github.com/org2

✨ Successfully removed 2 credential helper(s)
```

**Use cases:**

- Switching between different authentication methods
- Troubleshooting authentication issues
- Cleaning up after testing
- Preparing for removal of gh-app-auth

## Scope Options

### Global Configuration (Default)

Applies to all git repositories on your system:

```bash
gh app-auth gitconfig --sync --global
gh app-auth gitconfig --clean --global
```

**Equivalent to:**

```bash
git config --global credential.*.helper ...
```

### Local Configuration

Applies only to the current git repository:

```bash
cd /path/to/repo
gh app-auth gitconfig --sync --local
```

**Use cases:**

- Repository-specific authentication
- Testing configuration without affecting other repos
- Overriding global configuration for specific projects

### Auto Mode

Configures a single global credential helper that dynamically handles all repositories:

```bash
gh app-auth gitconfig --sync --auto
```

**Requirements:**

- `GH_APP_PRIVATE_KEY_PATH` environment variable must be set (path to private key file)
- `GH_APP_ID` environment variable must be set

**How it works:**

1. Sets up a single global git credential helper
2. The helper automatically authenticates for any repository using the configured GitHub App
3. No need to configure patterns for each organization

**Use cases:**

- CI/CD environments where a single GitHub App has access to all needed repositories
- Simplified setup when one app covers all authentication needs
- Dynamic environments where repository patterns aren't known in advance

**Example:**

```bash
export GH_APP_PRIVATE_KEY_PATH="/path/to/app.pem"
export GH_APP_ID="123456"
gh app-auth gitconfig --sync --auto

# Now any git clone will use the GitHub App
git clone https://github.com/any-org/any-repo
```

## How Pattern Matching Works

The `gitconfig` command intelligently extracts credential contexts from your patterns:

| Pattern | Git Credential Context | Notes |
|---------|----------------------|-------|
| `github.com/org/*` | `https://github.com/org` | Organization-level |
| `github.com/org/repo` | `https://github.com/org` | Same as above |
| `github.enterprise.com/*/*` | `https://github.enterprise.com` | Host-level only |
| `github.com/*/*` | `https://github.com` | All repositories |
| `bitbucket.example.com/` | `https://bitbucket.example.com` | Host-level PAT with custom username |

### Examples

**Single Organization:**

```bash
# Configuration
gh app-auth setup --app-id 123 --patterns "github.com/myorg/*"
gh app-auth gitconfig --sync

# Git will use GitHub App auth for:
git clone https://github.com/myorg/repo1
git clone https://github.com/myorg/repo2
# But NOT for:
git clone https://github.com/other-org/repo  # Uses default git auth
```

**Multiple Organizations:**

```bash
# Configure multiple apps
gh app-auth setup --app-id 123 --patterns "github.com/org1/*"
gh app-auth setup --app-id 456 --patterns "github.com/org2/*"
gh app-auth gitconfig --sync

# Each organization uses its own GitHub App
git clone https://github.com/org1/repo  # Uses App 123
git clone https://github.com/org2/repo  # Uses App 456
```

**Enterprise GitHub:**

```bash
gh app-auth setup --app-id 789 --patterns "github.enterprise.com/*/*"
gh app-auth gitconfig --sync

# All repos on enterprise GitHub use this app
git clone https://github.enterprise.com/any-org/any-repo
```

## Workflow Integration

### Initial Setup Workflow

```bash
# 1. Install extension
gh extension install AmadeusITGroup/gh-app-auth

# 2. Configure GitHub App(s)
gh app-auth setup --app-id 123456 --key-file app.pem --patterns "github.com/myorg/*"

# 3. Sync git configuration
gh app-auth gitconfig --sync

# 4. Verify setup
gh app-auth list
git config --global --get-regexp credential

# 5. Test it works (GitHub App)
git clone https://github.com/myorg/private-repo

# Optional: configure Bitbucket PAT with username override
gh app-auth setup --pat bbpat_xxx --patterns "bitbucket.example.com/" --username bitbucket_user --name "Bitbucket PAT"
gh app-auth gitconfig --sync
git clone https://bitbucket.example.com/scm/team/repo.git
```

### Adding New Organization

```bash
# 1. Add new app
gh app-auth setup --app-id 789012 --key-file app2.pem --patterns "github.com/neworg/*"

# 2. Re-sync (automatically includes new app)
gh app-auth gitconfig --sync

# 3. Test
git clone https://github.com/neworg/private-repo
```

### Troubleshooting Workflow

```bash
# 1. Check current configuration
gh app-auth list
git config --global --get-regexp credential

# 2. Clean everything
gh app-auth gitconfig --clean

# 3. Re-sync from scratch
gh app-auth gitconfig --sync

# 4. Test again
git clone https://github.com/myorg/private-repo
```

### Removal Workflow

```bash
# 1. Clean git configuration
gh app-auth gitconfig --clean

# 2. Remove app configurations
gh app-auth remove --all

# 3. Uninstall extension (optional)
gh extension remove app-auth
```

## CI/CD Integration

### Automatic Sync in CI

```yaml
name: Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Setup GitHub App Auth
        env:
          GH_APP_PRIVATE_KEY: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
        run: |
          gh extension install AmadeusITGroup/gh-app-auth
          gh app-auth setup \
            --app-id ${{ secrets.GITHUB_APP_ID }} \
            --patterns "github.com/${{ github.repository_owner }}/*"
          
          # Automatically sync git config
          gh app-auth gitconfig --sync --global

      - name: Checkout with submodules
        run: |
          git clone --recurse-submodules https://github.com/myorg/repo
          cd repo

      - name: Build
        run: make build
```

### Cleanup After CI

```yaml
      - name: Cleanup
        if: always()
        run: |
          gh app-auth gitconfig --clean --global
          gh app-auth remove --all --force
```

## Advanced Usage

### Mixed Authentication

Use GitHub App auth for some orgs, PATs for others:

```bash
# Configure GitHub App for specific org
gh app-auth setup --app-id 123 --patterns "github.com/work-org/*"

# Configure PAT for personal org
gh app-auth setup --pat ghp_personal --patterns "github.com/personal-org/" --priority 15

gh app-auth gitconfig --sync

git clone https://github.com/work-org/repo     # Uses GitHub App
git clone https://github.com/personal-org/repo  # Uses PAT stored in keyring
```

### Per-Repository Override

Override global config for specific repository:

```bash
# Global: GitHub App auth for work org
gh app-auth gitconfig --sync --global

# Repository-specific: Use different auth
cd /path/to/special-repo
git config --local credential.helper ""  # Clear helpers
git config --local credential.helper "your-custom-helper"
```

### Testing Without Global Impact

Test configuration in a single repository:

```bash
cd /path/to/test-repo

# Configure locally only
gh app-auth gitconfig --sync --local

# Test
git fetch

# If it works, apply globally
gh app-auth gitconfig --sync --global
```

## Comparison with Manual Configuration

### Manual Approach

```bash
# Setup
gh app-auth setup --app-id 123 --patterns "github.com/org1/*"
git config --global credential."https://github.com/org1".helper \
  "!gh app-auth git-credential --pattern 'github.com/org1/*'"

gh app-auth setup --app-id 456 --patterns "github.com/org2/*"
git config --global credential."https://github.com/org2".helper \
  "!gh app-auth git-credential --pattern 'github.com/org2/*'"

# Cleanup
git config --global --unset-all credential."https://github.com/org1".helper
git config --global --unset-all credential."https://github.com/org2".helper
```

**Problems:**

- ❌ Error-prone (typos, wrong patterns)
- ❌ Tedious for multiple apps
- ❌ Hard to remember exact commands
- ❌ Manual cleanup is incomplete

### Automated Approach

```bash
# Setup
gh app-auth setup --app-id 123 --patterns "github.com/org1/*"
gh app-auth setup --app-id 456 --patterns "github.com/org2/*"
gh app-auth gitconfig --sync

# Cleanup
gh app-auth gitconfig --clean
```

**Benefits:**

- ✅ One command configures everything
- ✅ Automatically handles all configured apps
- ✅ No typos or manual errors
- ✅ Complete cleanup guaranteed

## Troubleshooting

### Issue: "no GitHub Apps configured"

```bash
$ gh app-auth gitconfig --sync
Error: no GitHub Apps configured. Run 'gh app-auth setup' first
```

**Solution:** Configure at least one GitHub App first:

```bash
gh app-auth setup --app-id 123456 --key-file app.pem --patterns "github.com/org/*"
gh app-auth gitconfig --sync
```

### Issue: "gh-app-auth executable not found"

```bash
$ gh app-auth gitconfig --sync
Error: gh-app-auth executable not found in PATH or extension directory
```

**Solution:** Ensure gh-app-auth is properly installed:

```bash
gh extension list | grep app-auth
gh extension install AmadeusITGroup/gh-app-auth
```

### Issue: Git still prompting for credentials

```bash
$ git clone https://github.com/org/repo
Username:  # Should not prompt!
```

**Possible causes:**

1. Pattern mismatch
2. Git config not synced
3. Wrong scope (local vs global)

**Debug:**

```bash
# Check what's configured
gh app-auth list
git config --global --get-regexp credential

# Re-sync
gh app-auth gitconfig --clean
gh app-auth gitconfig --sync

# Test with verbose output
GIT_CURL_VERBOSE=1 GIT_TRACE=1 git clone https://github.com/org/repo
```

### Issue: "Skipping invalid pattern"

```bash
$ gh app-auth gitconfig --sync
⚠️  Skipping invalid pattern: invalid-pattern
```

**Solution:** Check your patterns are correctly formatted:

- ✅ Good: `github.com/org/*`
- ✅ Good: `github.enterprise.com/*/*`
- ❌ Bad: `org/*` (missing host)
- ❌ Bad: `github.com` (too broad, missing pattern)

## See Also

- [CI/CD Integration Guide](./ci-cd-guide.md)
- [Pattern Routing Documentation](./PATTERN_ROUTING.md)
- [Main README](../README.md)
