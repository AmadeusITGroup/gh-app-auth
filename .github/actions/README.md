# GitHub Actions for gh-app-auth

This directory contains reusable GitHub Actions for setting up and cleaning up GitHub App authentication in CI/CD workflows.

## Available Actions

### 1. `setup-gh-app-auth`

Configures GitHub App authentication with automatic git credential helper setup.

**Features:**

- Installs GitHub CLI and gh-app-auth extension
- Configures GitHub App with encrypted keyring storage
- Automatically syncs git credential helpers
- Registers cleanup hooks (optional)

**Example Usage:**

```yaml
name: Build with GitHub App

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Setup GitHub App Auth
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          app-id: ${{ secrets.GITHUB_APP_ID }}
          private-key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
          patterns: 'github.com/myorg/*'
          app-name: 'My GitHub App'

      - name: Checkout code with submodules
        run: |
          git clone --recurse-submodules https://github.com/myorg/repo
          cd repo

      - name: Build
        run: make build
```

**Inputs:**

| Input               | Required | Default        | Description                                         |
| ------------------- | -------- | -------------- | --------------------------------------------------- |
| **Single-App Mode** |          |                |                                                     |
| `app-id`            | No*      | -              | GitHub App ID (for single app setup)                |
| `private-key`       | No*      | -              | GitHub App private key (PEM format)                 |
| `patterns`          | No*      | -              | Repository patterns (comma-separated)               |
| `app-name`          | No       | `'GitHub App'` | Friendly name for the app                           |
| **Multi-App Mode**  |          |                |                                                     |
| `apps-config`       | No*      | -              | JSON array of app configurations (see examples)     |
| **Common Options**  |          |                |                                                     |
| `cleanup-on-exit`   | No       | `'true'`       | Remove credentials on job completion                |
| `sync-git-config`   | No       | `'true'`       | Auto-sync git credential helpers                    |
| `extension-version` | No       | `''`           | Specific version to install (default: latest)       |

_*Note: Either specify `app-id`/`private-key`/`patterns` for single-app mode OR `apps-config` for multi-app mode. One mode is required._

**Outputs:**

| Output               | Description                                 |
| -------------------- | ------------------------------------------- |
| `config-path`        | Path to the gh-app-auth configuration file  |
| `cleanup-registered` | Whether cleanup hook was registered         |

---

### 2. `cleanup-gh-app-auth`

Removes GitHub App authentication configuration and credentials. **Important for non-ephemeral runners** to prevent credential leakage between jobs.

**Features:**

- Removes git credential helper configuration
- Removes credentials from system keyring
- Cleans up configuration files

**Example Usage:**

```yaml
name: Secure Build

on: [push]

jobs:
  build:
    runs-on: self-hosted  # Non-ephemeral runner
    steps:
      - name: Setup GitHub App Auth
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          app-id: ${{ secrets.GITHUB_APP_ID }}
          private-key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
          patterns: 'github.com/myorg/*'
          cleanup-on-exit: 'false'  # We'll handle cleanup manually

      - name: Build
        run: make build

      # Always run cleanup, even if build fails
      - name: Cleanup credentials
        if: always()
        uses: AmadeusITGroup/gh-app-auth/.github/actions/cleanup-gh-app-auth@main
```

**Inputs:**

| Input   | Required | Default  | Description                         |
| ------- | -------- | -------- | ----------------------------------- |
| `force` | No       | `'true'` | Force cleanup without confirmation  |

---

## Multi-Organization Examples

### Method 1: JSON Configuration (Recommended) ✨

**NEW**: Configure multiple GitHub Apps in a single action using JSON. This is the most pragmatic approach for multi-org scenarios:

```yaml
name: Multi-Org Build (JSON)

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Setup Multiple GitHub Apps
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          # Pass all app configurations as JSON
          apps-config: |
            [
              {
                "app-id": "${{ secrets.ORG1_APP_ID }}",
                "private-key": "${{ secrets.ORG1_PRIVATE_KEY }}",
                "patterns": "github.com/org1/*",
                "app-name": "Organization 1"
              },
              {
                "app-id": "${{ secrets.ORG2_APP_ID }}",
                "private-key": "${{ secrets.ORG2_PRIVATE_KEY }}",
                "patterns": "github.com/org2/*",
                "app-name": "Organization 2"
              },
              {
                "app-id": "${{ secrets.ORG3_APP_ID }}",
                "private-key": "${{ secrets.ORG3_PRIVATE_KEY }}",
                "patterns": "github.com/org3/*",
                "app-name": "Organization 3"
              }
            ]

      - name: Clone multi-org repository
        run: |
          git clone --recurse-submodules https://github.com/org1/main-repo
          cd main-repo
          # All submodules from different orgs authenticate correctly

      - name: Build
        run: make build
```

**Benefits of JSON mode:**

- ✅ **Single action call** - No need to repeat the action multiple times
- ✅ **Clear structure** - All configurations in one place
- ✅ **Maintainable** - Easy to add/remove organizations
- ✅ **Automatic sync** - Git config synced once for all apps
- ✅ **Less verbose** - Cleaner workflow files

### Method 2: Multiple Action Calls (Legacy)

Configure multiple GitHub Apps using separate action calls:

```yaml
name: Multi-Org Build (Legacy)

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      # Setup first organization
      - name: Setup Org 1 Auth
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          app-id: ${{ secrets.ORG1_APP_ID }}
          private-key: ${{ secrets.ORG1_PRIVATE_KEY }}
          patterns: 'github.com/org1/*'
          app-name: 'Org1 App'
          sync-git-config: 'false'  # Sync after all apps configured

      # Setup second organization
      - name: Setup Org 2 Auth
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          app-id: ${{ secrets.ORG2_APP_ID }}
          private-key: ${{ secrets.ORG2_PRIVATE_KEY }}
          patterns: 'github.com/org2/*'
          app-name: 'Org2 App'
          sync-git-config: 'true'  # Sync all at once

      - name: Clone multi-org repository
        run: |
          git clone --recurse-submodules https://github.com/org1/main-repo
          cd main-repo
          # Submodules from org2 will also authenticate correctly

      - name: Build
        run: make build

      # Cleanup (recommended for non-ephemeral runners)
      - name: Cleanup
        if: always()
        uses: AmadeusITGroup/gh-app-auth/.github/actions/cleanup-gh-app-auth@main
```

**Note:** While this method still works, the JSON configuration approach is recommended for better maintainability.

---

## Best Practices

### For Ephemeral Runners (GitHub-hosted)

Cleanup is less critical but still recommended:

```yaml
- uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    cleanup-on-exit: 'true'  # Default
```

### For Non-Ephemeral Runners (Self-hosted)

**Always** use explicit cleanup:

```yaml
steps:
  - uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
    with:
      cleanup-on-exit: 'false'
  
  # ... your build steps ...
  
  - uses: AmadeusITGroup/gh-app-auth/.github/actions/cleanup-gh-app-auth@main
    if: always()  # Run even if previous steps fail
```

### Long-Running Jobs

For jobs that may run longer than 1 hour (GitHub App token expiry):

```yaml
- uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    # Tokens auto-refresh when git operations are performed
    patterns: 'github.com/myorg/*'
```

The git credential helper automatically refreshes tokens before they expire.

---

## Security Considerations

1. **Secrets Management**: Always use GitHub Secrets for private keys
2. **Pattern Specificity**: Use specific patterns (e.g., `github.com/myorg/*`) rather than broad patterns
3. **Cleanup**: Always clean up credentials on non-ephemeral runners
4. **Permissions**: Configure GitHub Apps with minimal required permissions

---

## Troubleshooting

### Issue: "GitHub CLI not found"

**Solution:** The action automatically installs GitHub CLI. If it fails, ensure the runner has internet access and appropriate permissions.

### Issue: "Keyring not available"

**Solution:** The extension automatically falls back to filesystem storage if keyring is unavailable. Check logs for warnings.

### Issue: "Cleanup not working on self-hosted runners"

**Solution:** Ensure the `cleanup-gh-app-auth` action runs with `if: always()`:

```yaml
- uses: AmadeusITGroup/gh-app-auth/.github/actions/cleanup-gh-app-auth@main
  if: always()
```

### Issue: "Git still prompting for credentials"

**Solution:** Verify git configuration was synced:

```yaml
- name: Debug git config
  run: git config --global --get-regexp credential
```

---

## Related Documentation

- [Main README](../../../README.md)
- [CI/CD Guide](../../../docs/ci-cd-guide.md)
- [Architecture](../../../docs/architecture.md)
