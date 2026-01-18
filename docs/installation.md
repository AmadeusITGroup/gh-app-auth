# Installation & Setup Guide

This guide walks through installing the `gh-app-auth` extension, configuring GitHub Apps and Personal Access Tokens (PATs), and verifying everything works on both GitHub and Bitbucket.

## 1. Prerequisites

- [GitHub CLI](https://cli.github.com/) v2.45+ installed and authenticated (`gh auth status`)
- Git 2.30+
- Access to create/configure a GitHub App **or** a PAT
- (Optional) Bitbucket Server/Data Center PAT + username if you need non-GitHub hosts

## 2. Install the Extension

```bash
gh extension install AmadeusITGroup/gh-app-auth
```

Upgrade later with `gh extension upgrade app-auth`.

## 3. Configure Credentials

### Option A: GitHub App (recommended for automation)

```bash
# Using environment variable for key material (ideal for CI/CD)
export GH_APP_PRIVATE_KEY="$(cat ~/keys/my-app.pem)"

gh app-auth setup \
  --app-id 123456 \
  --patterns "github.com/myorg/*" \
  --name "Org Automation App" \
  --priority 5

unset GH_APP_PRIVATE_KEY  # optional cleanup
```

Alternate file-based input:

```bash
gh app-auth setup \
  --app-id 123456 \
  --key-file ~/keys/my-app.pem \
  --patterns "github.com/myorg/*"
```

### Option B: Personal Access Token (PAT)

Use PATs when you need to act as yourself or access non-GitHub providers.

```bash
# GitHub PAT (uses default username x-access-token)
gh app-auth setup \
  --pat ghp_your_token \
  --patterns "github.com/personal-org/" \
  --name "Personal Workflows" \
  --priority 15
```

```bash
# Bitbucket Server/Data Center PAT (requires real username)
gh app-auth setup \
  --pat bbpat_your_token \
  --patterns "bitbucket.example.com/" \
  --username your.bitbucket.user \
  --name "Bitbucket PAT" \
  --priority 40
```

PATs share the same pattern/priority routing and live in the encrypted keyring alongside app keys.

## 4. Sync Git Credential Helper

Automatically configure git for every pattern:

```bash
gh app-auth gitconfig --sync --global
```

**Options:**

- `--local` - Scope to the current repository only
- `--auto` - Auto-mode for CI/CD (uses `GH_APP_ID` and `GH_APP_PRIVATE_KEY_PATH` env vars)
- `--clean` - Remove all gh-app-auth git configurations

## 5. Verify Configuration

```bash
# List configured credentials and storage backend
gh app-auth list --verify-keys

# Test authentication
# (choose any repo covered by your patterns)
gh app-auth test --repo github.com/myorg/private-repo
```

For Bitbucket, pass the full HTTPS URL to `test`:

```bash
gh app-auth test --repo https://bitbucket.example.com/scm/team/repo.git
```

## 6. Common Workflows

### Add Another Organization / Host

```bash
gh app-auth setup --app-id 987654 --key-file ~/keys/second-app.pem --patterns "github.com/another-org/*"
gh app-auth gitconfig --sync
```

### Remove Credentials

```bash
gh app-auth remove --app-id 123456      # remove GitHub App
gh app-auth remove --pat-name "Bitbucket PAT"  # remove PAT entry
```

### CI/CD Quick Start

```yaml
- name: Install gh-app-auth
  run: gh extension install AmadeusITGroup/gh-app-auth

- name: Configure GitHub App
  env:
    GH_APP_PRIVATE_KEY: ${{ secrets.GH_APP_PRIVATE_KEY }}
  run: |
    gh app-auth setup \
      --app-id ${{ secrets.GH_APP_ID }} \
      --patterns "github.com/${{ github.repository_owner }}/*"
    gh app-auth gitconfig --sync --global

- name: Configure Bitbucket PAT (optional)
  if: env.BITBUCKET_PAT != ''
  run: |
    gh app-auth setup \
      --pat "$BITBUCKET_PAT" \
      --username "$BITBUCKET_USERNAME" \
      --patterns "bitbucket.example.com/"
    gh app-auth gitconfig --sync --global
```

## 7. Troubleshooting

| Issue | Fix |
|-------|-----|
| `gh app-auth gitconfig --sync` says “no GitHub Apps configured” | Run `gh app-auth setup` for at least one app or PAT first. |
| Git still prompts for username/password | Ensure pattern matches (`gh app-auth list`), re-run `gh app-auth gitconfig --sync`. |
| Bitbucket complains about username | Confirm the PAT entry uses `--username <bitbucket_user>` and `gitconfig --sync` was re-run. |

Need more detail? See:

- [Configuration Reference](configuration.md)
- [CI/CD Guide](ci-cd-guide.md)
- [Troubleshooting](troubleshooting.md)
