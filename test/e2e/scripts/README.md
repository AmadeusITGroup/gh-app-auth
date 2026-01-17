# E2E Test Scripts

Automated scripts for setting up and validating gh-app-auth end-to-end testing environment.

## Quick Start

```bash
# Interactive setup (recommended for first-time users)
./setup-wizard.sh

# Or manual step-by-step
./01-verify-prerequisites.sh
export TEST_ORG="your-org-name"
./02-create-test-repos.sh
# ... follow tutorial for GitHub App setup ...
export APP_ID="123456"
export INSTALLATION_ID="12345678"
./03-validate-basic-functionality.sh
./04-run-advanced-tests.sh
```

## Scripts Overview

### Setup Scripts

| Script | Purpose | When to Use |
|--------|---------|-------------|
| `setup-wizard.sh` | **Interactive setup wizard** | First time setup, guides through all steps |
| `01-verify-prerequisites.sh` | Check system requirements | Before any setup, troubleshooting |
| `02-create-test-repos.sh` | Create test repositories | After creating organization |

### Testing Scripts

| Script | Purpose | When to Use |
|--------|---------|-------------|
| `03-validate-basic-functionality.sh` | Basic validation tests | After configuration, verify core features |
| `04-run-advanced-tests.sh` | Advanced scenario tests | After basic tests pass, comprehensive validation |

### Cleanup Scripts

| Script | Purpose | When to Use |
|--------|---------|-------------|
| `99-cleanup.sh` | Complete cleanup | When done testing, remove all test resources |

## Detailed Usage

### setup-wizard.sh

**Interactive Setup Wizard** - Complete guided setup from start to finish.

```bash
./setup-wizard.sh
```

Features:
- Checks prerequisites automatically
- Guides through organization setup
- Creates test repositories
- Provides step-by-step GitHub App instructions
- Configures gh-app-auth
- Runs validation tests
- Saves environment variables

**Time**: 20-30 minutes  
**Recommended for**: First-time users, complete setup

---

### 01-verify-prerequisites.sh

**Prerequisites Verification** - Checks that all required tools are installed.

```bash
./01-verify-prerequisites.sh
```

Checks:
- ✅ GitHub CLI installed and authenticated
- ✅ Git installed
- ✅ jq (JSON processor) installed
- ✅ gh-app-auth extension installed
- ✅ Network connectivity to GitHub
- ✅ GitHub API permissions
- ✅ OS keyring availability

**Exit codes:**

- `0` - All prerequisites met
- `1` - One or more prerequisites missing

---

### 02-create-test-repos.sh

**Test Repository Creation** - Creates 3 test repositories in your organization.

```bash
export TEST_ORG="your-organization-name"
./02-create-test-repos.sh
```

Creates:

1. **public-test-repo** (public) - Basic functionality testing
2. **private-test-repo** (private) - Authentication testing
3. **submodule-parent** (private) - Submodule testing

Requirements:

- `TEST_ORG` environment variable set
- Owner/admin access to the organization
- Confirmed to delete existing repos if found

**Exit codes:**

- `0` - All repositories created successfully
- `1` - Organization not found or creation failed

---

### 03-validate-basic-functionality.sh

**Basic Validation Suite** - Tests core gh-app-auth functionality.

```bash
export TEST_ORG="your-organization"
export APP_ID="123456"  # Optional but recommended
./03-validate-basic-functionality.sh
```

Tests:

- ✅ gh-app-auth configuration
- ✅ JWT token generation
- ✅ Installation token retrieval
- ✅ Git credential helper integration
- ✅ Public repository access
- ✅ Private repository access (with auth)
- ✅ Git operations (clone, commit, push)
- ✅ Scope detection
- ✅ Token caching

Requirements:

- gh-app-auth configured
- Git credential helper synced
- Test repositories accessible

**Exit codes:**

- `0` - All tests passed
- `1` - One or more tests failed

---

### 04-run-advanced-tests.sh

**Advanced Test Suite** - Tests complex scenarios and edge cases.

```bash
export TEST_ORG="your-organization"
./04-run-advanced-tests.sh
```

Tests:

- ✅ Git submodule support
- ✅ URL prefix pattern matching
- ✅ Concurrent git operations
- ✅ Large file operations
- ✅ Token cache performance
- ✅ Error handling

Requirements:

- Basic validation tests passed
- Test repositories configured
- Submodule parent repository available

**Exit codes:**

- `0` - All advanced tests passed
- `1` - One or more tests failed

---

### 99-cleanup.sh

**Complete Cleanup** - Removes all test resources and configurations.

```bash
export TEST_ORG="your-organization"  # Optional for repo cleanup
./99-cleanup.sh
```

Cleans:

- gh-app-auth configurations
- Git credential helper settings
- Test repositories (with confirmation)
- Local temporary files
- Private key files
- Extension config directory

Provides instructions for:

- Uninstalling GitHub App
- Deleting GitHub App
- Deleting test organization

**Interactive**: Prompts for confirmation before each cleanup step.

---

## Environment Variables

### Required

```bash
export TEST_ORG="your-organization-name"    # Your test organization
```

### Recommended

```bash
export APP_ID="123456"                      # GitHub App ID
export INSTALLATION_ID="12345678"           # Installation ID
```

### Optional

```bash
export GH_APP_AUTH_DEBUG=1                  # Enable debug logging
```

### Persistence

Save environment for reuse:

```bash
# The setup wizard saves to:
source /tmp/gh-app-auth-e2e-env.sh

# Or create your own:
cat > ~/.gh-app-auth-e2e-env << 'EOF'
export TEST_ORG="your-org"
export APP_ID="123456"
export INSTALLATION_ID="12345678"
EOF

source ~/.gh-app-auth-e2e-env
```

## Workflow Examples

### First-Time Setup

```bash
# Use the wizard for guided setup
./setup-wizard.sh

# Wizard handles everything:
# - Prerequisites check
# - Repository creation
# - GitHub App guidance
# - Configuration
# - Validation
```

### Manual Step-by-Step

```bash
# 1. Verify prerequisites
./01-verify-prerequisites.sh

# 2. Set organization
export TEST_ORG="gh-app-auth-testing"

# 3. Create repositories
./02-create-test-repos.sh

# 4. Create GitHub App (manual - see tutorial)
# - Go to: https://github.com/organizations/$TEST_ORG/settings/apps
# - Follow: docs/E2E_TESTING_TUTORIAL.md#step-3

# 5. Configure gh-app-auth (manual - see tutorial)
export APP_ID="123456"
export INSTALLATION_ID="12345678"
# See: docs/E2E_TESTING_TUTORIAL.md#step-6

# 6. Validate
./03-validate-basic-functionality.sh

# 7. Advanced tests
./04-run-advanced-tests.sh
```

### Quick Validation

```bash
# Just run validation on existing setup
source /tmp/gh-app-auth-e2e-env.sh
./03-validate-basic-functionality.sh
```

### Complete Cleanup

```bash
# Clean everything
export TEST_ORG="gh-app-auth-testing"
./99-cleanup.sh

# Follow prompts to selectively clean resources
```

## Troubleshooting

### Script Fails Immediately

```bash
# Check script is executable
chmod +x test/e2e/scripts/*.sh

# Check bash is available
which bash

# Run with explicit bash
bash ./01-verify-prerequisites.sh
```

### "TEST_ORG not set"

```bash
# Set the environment variable
export TEST_ORG="your-organization-name"

# Verify it's set
echo $TEST_ORG
```

### "Organization not found"

```bash
# Verify you have access
gh api /orgs/$TEST_ORG

# List your organizations
gh api user/orgs --jq '.[].login'
```

### Tests Failing

```bash
# Enable debug mode
export GH_APP_AUTH_DEBUG=1

# Re-run with verbose output
./03-validate-basic-functionality.sh

# Check configuration
gh app-auth list --verify-keys
git config --get-regexp credential
```

### Permission Errors

```bash
# Check GitHub CLI authentication
gh auth status

# Re-authenticate if needed
gh auth login

# Check app installation
gh api /orgs/$TEST_ORG/installation
```

## Exit Codes Reference

| Exit Code | Meaning |
| --------- | ------- |
| `0`       | Success - all checks/tests passed |
| `1`       | Failure - one or more checks/tests failed |

## Script Dependencies

```
setup-wizard.sh
├── 01-verify-prerequisites.sh
├── 02-create-test-repos.sh
└── 03-validate-basic-functionality.sh

03-validate-basic-functionality.sh
└── (requires gh-app-auth configured)

04-run-advanced-tests.sh
└── (requires basic validation passed)

99-cleanup.sh
└── (no dependencies)
```

## Contributing

When adding new scripts:

1. Follow naming convention: `NN-description.sh`
2. Include shebang: `#!/usr/bin/env bash`
3. Use `set -e` for error handling
4. Add color codes for output
5. Include usage documentation
6. Make executable: `chmod +x script.sh`
7. Update this README

## Support

- **Tutorial**: [`docs/E2E_TESTING_TUTORIAL.md`](../../../docs/E2E_TESTING_TUTORIAL.md)
- **Issues**: <https://github.com/AmadeusITGroup/gh-app-auth/issues>
- **Discussions**: <https://github.com/AmadeusITGroup/gh-app-auth/discussions>
