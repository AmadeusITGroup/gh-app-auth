# Configuration Reference

Comprehensive reference for the `~/.config/gh/extensions/gh-app-auth/config.yml` file. Describes every field for GitHub Apps and Personal Access Tokens (PATs), including Bitbucket-specific settings.

> Use `gh app-auth list --json` or open the YAML file directly to inspect your current configuration.

---

## Top-Level Structure

```yaml
version: "1"
github_apps:
  - ...
pats:
  - ...
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | ✅ | Schema version. Currently `"1"`. |
| `github_apps` | array | ✅ (unless `pats` present) | List of GitHub App entries. |
| `pats` | array | ✅ (unless `github_apps` present) | List of Personal Access Token entries. |

At least one GitHub App or PAT must be present.

---

## GitHub App Entry

```yaml
- name: Org Automation App
  app_id: 123456
  installation_id: 987654        # optional (auto-detect)
  private_key_source: keyring    # keyring | filesystem | inline (legacy)
  private_key_path: ~/.keys/app.pem  # only used when source=filesystem
  patterns:
    - github.com/myorg/
    - github.enterprise.com/team/
  priority: 5                    # deprecated (see PAT priority guidance)
  scope:                         # optional cache of installation scope
    repository_selection: selected
    account_login: myorg
    account_type: Organization
    repositories: []
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Friendly label shown in `gh app-auth list`. |
| `app_id` | int | ✅ | GitHub App ID. |
| `installation_id` | int | ➖ | Optional override. If omitted, auto-detection is attempted during `setup`. |
| `private_key_source` | enum | ✅ | `keyring`, `filesystem`, or `inline` (legacy). Indicates where the key material lives after setup. |
| `private_key_path` | string | ➖ | Populated when `private_key_source=filesystem`. |
| `patterns` | array | ✅ | URL prefixes matched during credential lookup (e.g., `github.com/org/`). |
| `priority` | int | ➖ | Legacy field (matching now prefers the **longest prefix**, then priority). |
| `scope` | object | ➖ | Cached metadata from scope discovery. Used internally by diagnostics. |

---

## Personal Access Token Entry

```yaml
- name: Bitbucket PAT
  token_source: keyring           # stored in OS keyring
  patterns:
    - bitbucket.example.com/
  priority: 40                    # higher beats Apps if prefixes tie
  username: bitbucket.user        # optional, defaults to x-access-token
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Friendly label (also used as secret storage key). |
| `token_source` | enum | ✅ | `keyring` or `filesystem`. `filesystem` used only if keyring unavailable. |
| `patterns` | array | ✅ | URL prefixes that should use this PAT. Applies to GitHub or Bitbucket hosts. |
| `priority` | int | ✅ | Higher priority wins when pattern lengths tie. Useful for overriding App auth with PATs. |
| `username` | string | ➖ | Optional real username for providers that require it (Bitbucket Server/Data Center). Defaults to `x-access-token` for GitHub. |

### Username Guidance

| Scenario | Username value |
|----------|----------------|
| GitHub.com / GitHub Enterprise | Leave blank (defaults to `x-access-token`). |
| Bitbucket Server/Data Center | Set to your Bitbucket username (e.g., `jsmith`). |
| Other HTTPS Git providers | Use whatever username the provider expects; PAT is sent as password. |

---

## Pattern Matching Logic

1. Normalize URL input (protocol + host + optional path).
2. Compare against every `patterns` entry (Apps + PATs) using longest-prefix match.
3. If multiple entries share the same prefix length, use the highest `priority`.
4. If still tied, the most recently configured credential wins.

Examples:

| URL | Matches | Result |
|-----|---------|--------|
| `github.com/org/repo` | App: `github.com/org/` (len 16) vs PAT: `github.com/` (len 11) | GitHub App wins (longer prefix). |
| `bitbucket.example.com/scm/team/repo` | PAT pattern `bitbucket.example.com/` | PAT wins (only match). |
| `github.enterprise.com/team/repo` | App pattern `github.enterprise.com/team/` | GitHub App wins. |

---

## Secret Storage

Secrets are stored separately from config:

- **Keyring (default):** macOS Keychain, Windows Credential Manager, Linux Secret Service.
- **Filesystem fallback:** `~/.config/gh/extensions/gh-app-auth/secrets/` (used only if keyring unavailable).
- Deleting a GitHub App or PAT via `gh app-auth remove` automatically wipes the corresponding secret.

---

## Editing Configuration

Use `gh app-auth setup` for most workflows. To edit manually:

```bash
$EDITOR ~/.config/gh/extensions/gh-app-auth/config.yml
```

After editing, run `gh app-auth list` to ensure the file still validates. Invalid entries (e.g., missing patterns) cause `gh app-auth` commands to exit with an error until fixed.

---

## Exporting / Importing

```bash
# Export JSON copy
gh app-auth list --json > gh-app-auth-config.json

# Restore (manual edit or custom script required)
cp gh-app-auth-config.json ~/.config/gh/extensions/gh-app-auth/config.yml
```

> ⚠️ Secrets are **not** included in the YAML/JSON file. When migrating machines, re-run `gh app-auth setup` with the original keys/PATs.

---

## Advanced Pattern Routing

### Git Credential Helper Configuration

The `gitconfig` command automatically configures git credential helpers for your patterns:

```bash
# Auto-configure for all apps and PATs
gh app-auth gitconfig --sync --global

# Apply to current repository only
gh app-auth gitconfig --sync --local

# Remove all gh-app-auth git configurations
gh app-auth gitconfig --clean --global
```

**How pattern contexts are extracted:**

| Pattern | Git Credential Context |
|---------|----------------------|
| `github.com/org/*` | `https://github.com/org` |
| `github.com/org/repo` | `https://github.com/org` |
| `github.enterprise.com/*/*` | `https://github.enterprise.com` |
| `bitbucket.example.com/` | `https://bitbucket.example.com` |

### Manual Git Configuration

If you prefer manual control over git credentials:

```bash
# Organization-specific routing
git config --global credential.'https://github.com/myorg'.helper \
  '!gh app-auth git-credential --pattern "github.com/myorg/*"'

# Host-level routing (all repos on host)
git config --global credential.'https://github.enterprise.com'.helper \
  '!gh app-auth git-credential --pattern "github.enterprise.com/*/*"'

# Check which credential would be used
gh app-auth scope --repo github.com/myorg/repo
```

---

## Multi-Organization Setup Examples

### Multiple Organizations (Same Host)

```yaml
# config.yml
version: "1"
github_apps:
  - name: Frontend Team
    app_id: 111111
    private_key_source: keyring
    patterns:
      - github.com/frontend-org/
    priority: 10

  - name: Backend Team
    app_id: 222222
    private_key_source: keyring
    patterns:
      - github.com/backend-org/
    priority: 10
```

### Enterprise + GitHub.com

```yaml
version: "1"
github_apps:
  - name: Enterprise App
    app_id: 333333
    private_key_source: keyring
    patterns:
      - github.enterprise.com/
    priority: 10

  - name: Cloud App
    app_id: 444444
    private_key_source: keyring
    patterns:
      - github.com/myorg/
    priority: 10
```

### GitHub Apps + Bitbucket PAT

```yaml
version: "1"
github_apps:
  - name: GitHub App
    app_id: 555555
    private_key_source: keyring
    patterns:
      - github.com/myorg/
    priority: 10

pats:
  - name: Bitbucket PAT
    token_source: keyring
    patterns:
      - bitbucket.example.com/
    priority: 40
    username: jsmith  # Required for Bitbucket
```

---

## Related Guides

- [Installation Guide](installation.md)
- [CI/CD Integration Guide](ci-cd-guide.md)
- [Security Considerations](security.md)
- [Troubleshooting](troubleshooting.md)
