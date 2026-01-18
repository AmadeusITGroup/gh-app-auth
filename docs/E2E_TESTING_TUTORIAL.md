# End-to-End Testing Tutorial for gh-app-auth

Complete guide to setting up a real-world testing environment with GitHub organizations, apps, and repositories.

## Quick Start

```bash
# 1. Set your organization name
export TEST_ORG="gh-app-auth-testing"

# 2. Run automated setup
./test/e2e/scripts/setup-wizard.sh

# 3. Follow the interactive prompts
```

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Step-by-Step Setup](#step-by-step-setup)
4. [Validation](#validation)
5. [Advanced Testing](#advanced-testing)
6. [Cleanup](#cleanup)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Tools

- **GitHub account** (free tier works)
- **GitHub CLI** (`gh`) v2.0+
- **Git** v2.0+
- **jq** (JSON processor)
- **gh-app-auth extension**

### Verification

Run the prerequisite checker:

```bash
./test/e2e/scripts/01-verify-prerequisites.sh
```

Or manually verify:

```bash
gh --version && gh auth status
git --version
jq --version
gh extension list | grep app-auth
```

### Installation

```bash
# macOS
brew install gh git jq

# Ubuntu/Debian
sudo apt-get install gh git jq

# Install gh-app-auth
gh extension install AmadeusITGroup/gh-app-auth
```

---

## Architecture Overview

```
GitHub Cloud
├── Organization (test-org)
│   ├── Repository 1 (public)
│   ├── Repository 2 (private)
│   └── Repository 3 (submodules)
└── GitHub App (test-app)
    ├── App ID: 123456
    ├── Installation ID: 12345678
    └── Private Key: RSA 2048-bit

Local Machine
├── gh-app-auth Extension
│   ├── Config: ~/.config/gh/extensions/gh-app-auth/
│   └── Private Key: OS Keyring (encrypted)
└── Git Credential Helper
    └── Pattern: github.com/test-org/*
```

---

## Step-by-Step Setup

### Step 1: Create Test Organization

**Web UI (Required):**

1. Go to <https://github.com/settings/organizations>
2. Click "New organization" → "Create a free organization"
3. Name: `gh-app-auth-testing` (or your choice)
4. Complete setup

**Verify:**

```bash
export TEST_ORG="gh-app-auth-testing"
gh api /orgs/$TEST_ORG --jq '.login'
```

### Step 2: Create Test Repositories

**Automated:**

```bash
./test/e2e/scripts/02-create-test-repos.sh
```

**Manual:**

```bash
# Public repository
gh repo create $TEST_ORG/public-test-repo --public --clone
cd public-test-repo && echo "# Public Test" > README.md
git add README.md && git commit -m "Init" && git push && cd ..

# Private repository
gh repo create $TEST_ORG/private-test-repo --private --clone
cd private-test-repo && echo "# Private Test" > README.md
git add README.md && git commit -m "Init" && git push && cd ..

# Submodule parent
gh repo create $TEST_ORG/submodule-parent --private --clone
cd submodule-parent && echo "# Submodule Parent" > README.md
git add README.md && git commit -m "Init" && git push && cd ..
```

### Step 3: Create GitHub App

**Web UI (Required):**

1. Go to `https://github.com/organizations/$TEST_ORG/settings/apps`
2. Click "New GitHub App"
3. Configure:
   - **Name**: `gh-app-auth-test-app-[username]` (globally unique)
   - **Homepage**: `https://github.com/AmadeusITGroup/gh-app-auth`
   - **Webhook**: Uncheck "Active"
   - **Repository permissions**:
     - Contents: **Read and write**
   - **Where can this app be installed?**: "Only on this account"
4. Click "Create GitHub App"
5. **Note the App ID** (e.g., 123456)

**Save App ID:**

```bash
export APP_ID="123456"  # Replace with your App ID
```

### Step 4: Generate Private Key

**Web UI:**

1. On app settings page, scroll to "Private keys"
2. Click "Generate a private key"
3. File downloads: `your-app-name.YYYY-MM-DD.private-key.pem`

**Secure Storage:**

```bash
# Move to secure location
mkdir -p ~/.ssh/github-apps
chmod 700 ~/.ssh/github-apps
mv ~/Downloads/*.private-key.pem ~/.ssh/github-apps/test-app.pem
chmod 600 ~/.ssh/github-apps/test-app.pem

# Verify
ls -la ~/.ssh/github-apps/test-app.pem
# Expected: -rw------- (600 permissions)
```

### Step 5: Install GitHub App

**Web UI:**

1. On app settings, click "Install App" → Install next to your org
2. Select "All repositories" or specific repos
3. Click "Install"
4. **Note the Installation ID** from URL: `.../installations/12345678`

**Via CLI:**

```bash
export INSTALLATION_ID=$(gh api /orgs/$TEST_ORG/installation --jq '.id')
echo "Installation ID: $INSTALLATION_ID"
```

### Step 6: Configure gh-app-auth

**Encrypted Storage (Recommended):**

```bash
# Load private key
export GH_APP_PRIVATE_KEY="$(cat ~/.ssh/github-apps/test-app.pem)"

# Setup
gh app-auth setup \
  --app-id $APP_ID \
  --installation-id $INSTALLATION_ID \
  --patterns "github.com/$TEST_ORG/*" \
  --name "test-app"

# Clear sensitive variable
unset GH_APP_PRIVATE_KEY

# Verify
gh app-auth list --verify-keys
```

### Step 7: Configure Git Credential Helper

**Automatic:**

```bash
gh app-auth gitconfig --sync --global
```

**Verify:**

```bash
git config --global --get-regexp credential
# Expected: credential.https://github.com/[org].helper !gh app-auth git-credential
```

---

## Validation

### Basic Tests

```bash
# Test JWT generation
gh app-auth test --app-id $APP_ID

# Test installation token
gh app-auth test --repo github.com/$TEST_ORG/private-test-repo

# Test git clone (public)
cd /tmp && rm -rf public-test-repo
git clone https://github.com/$TEST_ORG/public-test-repo

# Test git clone (private - requires auth)
cd /tmp && rm -rf private-test-repo
git clone https://github.com/$TEST_ORG/private-test-repo

# Test git push
cd /tmp/private-test-repo
echo "Test $(date)" >> test.txt
git add test.txt && git commit -m "Test" && git push
```

### Automated Validation

```bash
./test/e2e/scripts/03-validate-basic-functionality.sh
```

Expected output: All tests pass ✅

---

## Advanced Testing

### Token Caching

```bash
# First clone (cache miss ~2-5s)
time git clone https://github.com/$TEST_ORG/private-test-repo /tmp/test1

# Second clone (cache hit ~1-2s, faster)
time git clone https://github.com/$TEST_ORG/private-test-repo /tmp/test2
```

### Submodules

```bash
cd /tmp/submodule-parent
git submodule add https://github.com/$TEST_ORG/private-test-repo sub
git commit -m "Add submodule" && git push

# Test recursive clone
cd /tmp && rm -rf submodule-parent
git clone --recurse-submodules https://github.com/$TEST_ORG/submodule-parent
ls submodule-parent/sub/  # Should contain submodule files
```

### Scope Detection

```bash
gh app-auth scope github.com/$TEST_ORG/any-repo
# Shows which app will handle this repository
```

### Run All Advanced Tests

```bash
./test/e2e/scripts/04-run-advanced-tests.sh
```

---

## Cleanup

### Quick Cleanup

```bash
./test/e2e/scripts/99-cleanup.sh
```

### Manual Cleanup

```bash
# Remove gh-app-auth config
gh app-auth list --json | jq -r '.[].name' | \
  xargs -I {} gh app-auth remove --name {}
gh app-auth gitconfig --clean --global

# Uninstall app (Web UI)
# 1. Go to org settings → Installations
# 2. Configure → Uninstall

# Delete app (Web UI)
# 1. Go to org settings → GitHub Apps → Your app
# 2. Scroll down → Delete GitHub App

# Delete repositories
gh repo delete $TEST_ORG/public-test-repo --yes
gh repo delete $TEST_ORG/private-test-repo --yes
gh repo delete $TEST_ORG/submodule-parent --yes

# Delete organization (Optional)
# Web UI: Org settings → Delete this organization

# Clean local files
rm -f ~/.ssh/github-apps/test-app.pem
rm -rf /tmp/{public,private,submodule}*
```

---

## Troubleshooting

### "App not found"

```bash
# Check configuration
gh app-auth list

# Check pattern matching
gh app-auth scope github.com/$TEST_ORG/repo

# Verify pattern has no trailing /
```

### "Failed to get installation token"

```bash
# Verify installation
gh api /orgs/$TEST_ORG/installation

# Check app permissions (Web UI)
# Must have: Contents - Read and write

# Verify installation ID
echo $INSTALLATION_ID
```

### "Git not using gh-app-auth"

```bash
# Verify git config
git config --get-regexp credential

# Re-sync
gh app-auth gitconfig --sync --global

# Test credential helper
echo -e "protocol=https\nhost=github.com\npath=$TEST_ORG/repo" | \
  gh app-auth git-credential get
```

### Debug Mode

```bash
export GH_APP_AUTH_DEBUG=1
GIT_TRACE=1 git clone https://github.com/$TEST_ORG/private-test-repo
```

### Reset Everything

```bash
gh app-auth gitconfig --clean --global
rm -rf ~/.config/gh/extensions/gh-app-auth/
gh extension remove app-auth
gh extension install AmadeusITGroup/gh-app-auth
# Start over from Step 6
```

---

## Scripts Reference

| Script | Purpose |
|--------|---------|
| `setup-wizard.sh` | Interactive setup (all steps) |
| `01-verify-prerequisites.sh` | Check system requirements |
| `02-create-test-repos.sh` | Create test repositories |
| `03-validate-basic-functionality.sh` | Basic validation suite |
| `04-run-advanced-tests.sh` | Advanced scenario tests |
| `99-cleanup.sh` | Complete cleanup |

---

## Environment Variables

```bash
# Required
export TEST_ORG="gh-app-auth-testing"
export APP_ID="123456"
export INSTALLATION_ID="12345678"

# Optional (for multi-org tests)
export TEST_ORG_2="another-org"
export APP_ID_2="234567"
export INSTALLATION_ID_2="23456789"

# Debug
export GH_APP_AUTH_DEBUG=1
```

---

## Success Checklist

- [ ] Prerequisites verified
- [ ] Organization created
- [ ] 3 test repositories created
- [ ] GitHub App created
- [ ] Private key downloaded and secured
- [ ] App installed to organization
- [ ] gh-app-auth configured
- [ ] Git credential helper synced
- [ ] Basic validation passed
- [ ] Advanced tests passed

---

## Next Steps

After successful E2E validation:

1. **Documentation**: Update your project docs with working examples
2. **CI/CD**: Integrate into GitHub Actions workflows
3. **Production**: Deploy with confidence knowing it works end-to-end
4. **Monitoring**: Set up alerts for authentication failures

For production deployments, see:

- [CI/CD Integration Guide](../README.md#cicd-integration)
- [Security Best Practices](SECURITY.md)
- [GitHub Actions Examples](../.github/workflows/)

---

## Support

- **Issues**: <https://github.com/AmadeusITGroup/gh-app-auth/issues>
- **Discussions**: <https://github.com/AmadeusITGroup/gh-app-auth/discussions>
- **Documentation**: <https://github.com/AmadeusITGroup/gh-app-auth/tree/main/docs>
