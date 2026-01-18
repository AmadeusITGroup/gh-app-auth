# JSON-Based Multi-Organization Configuration

**NEW**: The pragmatic approach to configuring multiple GitHub Apps in CI/CD workflows.

## Overview

Instead of calling the `setup-gh-app-auth` action multiple times for different organizations, you can now pass all configurations as a single JSON array. This simplifies workflows and makes multi-org setups more maintainable.

## Why JSON Configuration?

### The Old Way (Multiple Action Calls)

```yaml
steps:
  - uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
    with:
      app-id: ${{ secrets.ORG1_APP_ID }}
      private-key: ${{ secrets.ORG1_PRIVATE_KEY }}
      patterns: 'github.com/org1/*'
      sync-git-config: 'false'

  - uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
    with:
      app-id: ${{ secrets.ORG2_APP_ID }}
      private-key: ${{ secrets.ORG2_PRIVATE_KEY }}
      patterns: 'github.com/org2/*'
      sync-git-config: 'false'

  - uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
    with:
      app-id: ${{ secrets.ORG3_APP_ID }}
      private-key: ${{ secrets.ORG3_PRIVATE_KEY }}
      patterns: 'github.com/org3/*'
      sync-git-config: 'true'  # Only sync after all are configured
```

**Problems:**

- ❌ Repetitive - same action called multiple times
- ❌ Verbose - lots of boilerplate
- ❌ Error-prone - easy to forget `sync-git-config` coordination
- ❌ Hard to maintain - adding/removing orgs requires editing multiple blocks

### The New Way (JSON Configuration)

```yaml
steps:
  - uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
    with:
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
```

**Benefits:**

- ✅ Single action call
- ✅ Clear structure - all configs in one place
- ✅ Easy to maintain - add/remove orgs by editing JSON
- ✅ Automatic sync - git config synced once for all apps
- ✅ Less verbose - ~40% fewer lines of code

## JSON Format

The `apps-config` input accepts a JSON array of app configuration objects:

```json
[
  {
    "app-id": "123456",
    "private-key": "-----BEGIN RSA PRIVATE KEY-----\n...",
    "patterns": "github.com/org/*",
    "app-name": "Optional Friendly Name"
  }
]
```

### Fields

| Field | Required | Description | Example |
|-------|----------|-------------|---------|
| `app-id` | ✅ Yes | GitHub App ID | `"123456"` |
| `private-key` | ✅ Yes | Private key in PEM format | `"-----BEGIN RSA..."` |
| `patterns` | ✅ Yes | Repository patterns | `"github.com/org/*"` |
| `app-name` | ❌ No | Friendly name (default: "GitHub App") | `"Org1 App"` |

## Usage Examples

### Basic Multi-Org Setup

```yaml
name: Build with Multi-Org Submodules

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Setup GitHub Apps
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          apps-config: |
            [
              {
                "app-id": "${{ secrets.ORG1_APP_ID }}",
                "private-key": "${{ secrets.ORG1_PRIVATE_KEY }}",
                "patterns": "github.com/org1/*"
              },
              {
                "app-id": "${{ secrets.ORG2_APP_ID }}",
                "private-key": "${{ secrets.ORG2_PRIVATE_KEY }}",
                "patterns": "github.com/org2/*"
              }
            ]

      - name: Clone repository
        run: git clone --recurse-submodules https://github.com/org1/main-repo
```

### With App Names

```yaml
- name: Setup GitHub Apps
  uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    apps-config: |
      [
        {
          "app-id": "${{ secrets.FRONTEND_APP_ID }}",
          "private-key": "${{ secrets.FRONTEND_PRIVATE_KEY }}",
          "patterns": "github.com/frontend-team/*",
          "app-name": "Frontend Team App"
        },
        {
          "app-id": "${{ secrets.BACKEND_APP_ID }}",
          "private-key": "${{ secrets.BACKEND_PRIVATE_KEY }}",
          "patterns": "github.com/backend-team/*",
          "app-name": "Backend Team App"
        },
        {
          "app-id": "${{ secrets.INFRA_APP_ID }}",
          "private-key": "${{ secrets.INFRA_PRIVATE_KEY }}",
          "patterns": "github.com/infrastructure/*",
          "app-name": "Infrastructure App"
        }
      ]
```

### Enterprise GitHub + GitHub.com

```yaml
- name: Setup GitHub Apps (Enterprise + Cloud)
  uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    apps-config: |
      [
        {
          "app-id": "${{ secrets.ENTERPRISE_APP_ID }}",
          "private-key": "${{ secrets.ENTERPRISE_PRIVATE_KEY }}",
          "patterns": "github.enterprise.com/*/*",
          "app-name": "Enterprise GitHub"
        },
        {
          "app-id": "${{ secrets.CLOUD_APP_ID }}",
          "private-key": "${{ secrets.CLOUD_PRIVATE_KEY }}",
          "patterns": "github.com/myorg/*",
          "app-name": "GitHub.com"
        }
      ]
```

### Self-Hosted Runners with Cleanup

```yaml
jobs:
  build:
    runs-on: self-hosted
    steps:
      - name: Setup GitHub Apps
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          apps-config: |
            [
              {"app-id": "${{ secrets.ORG1_APP_ID }}", "private-key": "${{ secrets.ORG1_KEY }}", "patterns": "github.com/org1/*"},
              {"app-id": "${{ secrets.ORG2_APP_ID }}", "private-key": "${{ secrets.ORG2_KEY }}", "patterns": "github.com/org2/*"}
            ]
          cleanup-on-exit: 'false'  # Manual cleanup for self-hosted

      - name: Build
        run: make build

      # Always cleanup, even on failure
      - name: Cleanup
        if: always()
        uses: AmadeusITGroup/gh-app-auth/.github/actions/cleanup-gh-app-auth@main
```

## Advanced Patterns

### Dynamic Configuration from Secrets

Store the entire JSON configuration in a single secret:

```yaml
steps:
  - name: Setup GitHub Apps
    uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
    with:
      # Single secret contains the full JSON array
      apps-config: ${{ secrets.GITHUB_APPS_CONFIG }}
```

Then in your repository secrets, create `GITHUB_APPS_CONFIG`:

```json
[
  {
    "app-id": "123456",
    "private-key": "-----BEGIN RSA PRIVATE KEY-----\n...",
    "patterns": "github.com/org1/*",
    "app-name": "Org1"
  },
  {
    "app-id": "789012",
    "private-key": "-----BEGIN RSA PRIVATE KEY-----\n...",
    "patterns": "github.com/org2/*",
    "app-name": "Org2"
  }
]
```

**Benefits:**

- ✅ Centralized configuration
- ✅ Easy to update all at once
- ✅ Can be reused across workflows
- ✅ Version controlled in secrets manager

### Compact JSON (Minified)

For readability, you can also use compact JSON:

```yaml
apps-config: '[{"app-id":"${{secrets.A}}","private-key":"${{secrets.B}}","patterns":"github.com/o1/*"},{"app-id":"${{secrets.C}}","private-key":"${{secrets.D}}","patterns":"github.com/o2/*"}]'
```

### Conditional Organizations

Use GitHub Actions expressions to conditionally include organizations:

```yaml
- name: Setup GitHub Apps
  uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    apps-config: |
      [
        {
          "app-id": "${{ secrets.ORG1_APP_ID }}",
          "private-key": "${{ secrets.ORG1_PRIVATE_KEY }}",
          "patterns": "github.com/org1/*"
        }
        ${{ github.ref == 'refs/heads/main' && format(', {{"app-id": "{0}", "private-key": "{1}", "patterns": "github.com/prod/*"}}', secrets.PROD_APP_ID, secrets.PROD_KEY) || '' }}
      ]
```

## Comparison: JSON vs Multiple Actions

| Aspect | JSON Config | Multiple Actions |
|--------|------------|------------------|
| Lines of code | ~15 lines | ~25 lines |
| Action calls | 1 | 3+ (one per org) |
| Git sync coordination | Automatic | Manual |
| Maintainability | Easy (single block) | Harder (scattered) |
| Readability | Clear structure | Repetitive |
| Error potential | Low | Medium (sync timing) |
| Performance | Slightly faster | Slightly slower |

## Migration Guide

### From Multiple Actions to JSON

**Before:**

```yaml
- uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    app-id: ${{ secrets.ORG1_APP_ID }}
    private-key: ${{ secrets.ORG1_PRIVATE_KEY }}
    patterns: 'github.com/org1/*'
    sync-git-config: 'false'

- uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    app-id: ${{ secrets.ORG2_APP_ID }}
    private-key: ${{ secrets.ORG2_PRIVATE_KEY }}
    patterns: 'github.com/org2/*'
    sync-git-config: 'true'
```

**After:**

```yaml
- uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    apps-config: |
      [
        {
          "app-id": "${{ secrets.ORG1_APP_ID }}",
          "private-key": "${{ secrets.ORG1_PRIVATE_KEY }}",
          "patterns": "github.com/org1/*"
        },
        {
          "app-id": "${{ secrets.ORG2_APP_ID }}",
          "private-key": "${{ secrets.ORG2_PRIVATE_KEY }}",
          "patterns": "github.com/org2/*"
        }
      ]
```

## Troubleshooting

### Issue: "Failed to parse JSON"

**Cause:** Invalid JSON syntax or missing commas.

**Solution:** Validate your JSON using a tool like jq:

```bash
echo '[{"app-id":"123","private-key":"...","patterns":"github.com/org/*"}]' | jq .
```

### Issue: "app-id is required"

**Cause:** Missing required field in one of the objects.

**Solution:** Ensure each object has `app-id`, `private-key`, and `patterns`:

```json
{
  "app-id": "123456",        // ✅ Required
  "private-key": "...",      // ✅ Required
  "patterns": "github.com/*" // ✅ Required
}
```

### Issue: "Error: jq not found"

**Cause:** The action automatically installs jq, but installation might have failed.

**Solution:** Manually install jq before the action:

```yaml
- name: Install jq
  run: sudo apt-get update && sudo apt-get install -y jq

- name: Setup GitHub Apps
  uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
  with:
    apps-config: |
      [...]
```

## Best Practices

1. **Use app-name**: Always specify friendly names for better logs

   ```json
   {"app-id": "123", "app-name": "Frontend Team", ...}
   ```

2. **Store in secrets**: For complex setups, store the entire JSON in a secret

   ```yaml
   apps-config: ${{ secrets.APPS_CONFIG }}
   ```

3. **Format for readability**: Use multi-line YAML with proper indentation

   ```yaml
   apps-config: |
     [
       {...},
       {...}
     ]
   ```

4. **Document patterns**: Add comments about what each app covers (using YAML comments)

   ```yaml
   apps-config: |
     [
       # Frontend team repositories
       {"app-id": "123", ...},
       # Backend team repositories
       {"app-id": "456", ...}
     ]
   ```

5. **Test locally**: Validate JSON before committing:

   ```bash
   cat config.json | jq -e . > /dev/null && echo "Valid JSON"
   ```

## See Also

- [GitHub Actions Documentation](../actions/README.md)
- [Example Workflows](../.github/workflows/example-usage.yml.example)
- [Main README](../README.md)
