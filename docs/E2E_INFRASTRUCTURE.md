# E2E Test Infrastructure

This document describes the **one-time manual setup** required to run the E2E test
suite. The infrastructure is intentionally stable (not ephemeral) to keep costs low and
setup reproducible.

## Overview

The E2E tests authenticate against **real private GitHub repositories** spread across
**two test organizations**. This validates the core enterprise use-case: cross-organization
repository access via GitHub App credentials, including git submodules.

```
gh-app-auth-test-1 (org)
└── main-repo (private)
    ├── README.md
    ├── data/main-marker.txt         ← contains "gh-app-auth-e2e-main-marker"
    └── .gitmodules                  ← submodule → gh-app-auth-test-2/submodule-repo

gh-app-auth-test-2 (org)
└── submodule-repo (private)
    ├── README.md
    └── data/submodule-marker.txt   ← contains "gh-app-auth-e2e-submodule-marker"
```

## Step 1 — Create Test Organizations

Create two GitHub organizations. Names are configurable via CI secrets but default to:

```
gh-app-auth-test-1
gh-app-auth-test-2
```

```bash
# Verify you have org creation rights (requires GitHub account)
gh api user/orgs --jq '.[].login'
```

Go to <https://github.com/organizations/plan> and create both organizations using the
**Free** plan. They do not need members beyond the owner account.

## Step 2 — Create Test Repositories

### In `gh-app-auth-test-1`

```bash
export ORG1="gh-app-auth-test-1"

# Create main-repo (MUST be private)
gh repo create "${ORG1}/main-repo" --private --description "E2E test repo — do not use"

# Populate with required content
TMPDIR=$(mktemp -d)
gh repo clone "${ORG1}/main-repo" "${TMPDIR}/main-repo"
cd "${TMPDIR}/main-repo"

mkdir -p data
echo "gh-app-auth-e2e-main-marker" > data/main-marker.txt

git add data/main-marker.txt
git commit -m "chore: add E2E test marker file"
git push
cd - && rm -rf "${TMPDIR}"
```

### In `gh-app-auth-test-2`

```bash
export ORG2="gh-app-auth-test-2"

# Create submodule-repo (MUST be private)
gh repo create "${ORG2}/submodule-repo" --private --description "E2E submodule repo — do not use"

TMPDIR=$(mktemp -d)
gh repo clone "${ORG2}/submodule-repo" "${TMPDIR}/submodule-repo"
cd "${TMPDIR}/submodule-repo"

mkdir -p data
echo "gh-app-auth-e2e-submodule-marker" > data/submodule-marker.txt

git add data/submodule-marker.txt
git commit -m "chore: add E2E test marker file"
git push
cd - && rm -rf "${TMPDIR}"
```

### Add submodule from org2 into org1's main-repo

```bash
TMPDIR=$(mktemp -d)
gh repo clone "${ORG1}/main-repo" "${TMPDIR}/main-repo"
cd "${TMPDIR}/main-repo"

git submodule add "https://github.com/${ORG2}/submodule-repo" submodule-repo

git add .gitmodules submodule-repo
git commit -m "chore: add cross-org submodule for E2E testing"
git push
cd - && rm -rf "${TMPDIR}"
```

## Step 3 — Create the GitHub App

Create a **single GitHub App** that will be installed in **both organizations**.

```bash
# Navigate to your GitHub account or organization settings
open "https://github.com/settings/apps/new"
```

App configuration:

| Field | Value |
|-------|-------|
| **Name** | `gh-app-auth-e2e` |
| **Homepage URL** | `https://github.com/AmadeusITGroup/gh-app-auth` |
| **Webhook** | Disabled (uncheck "Active") |
| **Repository permissions** | `Contents: Read-only` |
| **Where can it be installed** | Any account |

After creation:
1. Note the **App ID** (shown on the app settings page)
2. Generate and download a **private key** (`.pem` file)

## Step 4 — Install App in Both Test Organizations

Install the app in both test organizations. The E2E tests will discover the installation IDs dynamically via the GitHub API:

```bash
# Install in org1
open "https://github.com/apps/gh-app-auth-e2e/installations/new"
# Select gh-app-auth-test-1 → All repositories

# Install in org2
# Select gh-app-auth-test-2 → All repositories
```

Each organization will have its own installation. The E2E tests automatically discover these IDs using the GitHub App credentials.

## Step 5 — Encode the Private Key for CI

The private key is stored as a base64-encoded repository secret:

```bash
# Linux
base64 -w0 < /path/to/gh-app-auth-e2e.YYYY-MM-DD.private-key.pem

# macOS
base64 -i /path/to/gh-app-auth-e2e.YYYY-MM-DD.private-key.pem | tr -d '\n'
```

## Step 6 — Set Repository Secrets

In <https://github.com/AmadeusITGroup/gh-app-auth/settings/secrets/actions>, create:

| Secret Name | Value | Description |
|-------------|-------|-------------|
| `E2E_APP_ID` | `123456` | GitHub App ID (numeric) |
| `E2E_PRIVATE_KEY_B64` | `<base64 output>` | Base64-encoded private key PEM |

```bash
# Set secrets via gh CLI
gh secret set E2E_APP_ID --body "123456" --repo AmadeusITGroup/gh-app-auth
gh secret set E2E_PRIVATE_KEY_B64 --body "$(base64 -w0 < key.pem)" --repo AmadeusITGroup/gh-app-auth
```

> **Security**: The E2E tests generate short-lived installation tokens using the App ID and private key. Installation IDs are discovered dynamically via the GitHub API. No long-lived tokens are required.

## Step 7 — Verify the Infrastructure

Run the pre-flight check locally before pushing a release:

```bash
export E2E_APP_ID="123456"
export E2E_PRIVATE_KEY_B64="$(base64 -w0 < key.pem)"

go test -v -tags=e2e -run TestPreflight -timeout=2m ./test/e2e/...
```

All four sub-tests must pass:
- `binary_accessible` ✓
- `app_token_valid` ✓
- `org1_main_repo_is_private` ✓
- `org2_submodule_repo_is_private` ✓

> **Note**: Installation IDs are discovered dynamically via the GitHub API. No manual configuration needed.

## Repository Maintenance

The test repositories are **static fixtures** — they are created once and not modified
by CI runs. There is no ephemeral cleanup required.

**Do NOT:**
- Make the test repositories public (E2E tests will refuse to run and fail with a clear error)
- Add real code or sensitive data to the test repositories
- Grant external collaborators access

**If repositories are accidentally made public:**
The `TestPreflight/org1_main_repo_is_private` test will fail with:
```
SECURITY GATE FAILED: gh-app-auth-test-1/main-repo is PUBLIC
```
Set them back to private immediately.

## Custom Organization Names

If you use different organization names, set them as repository variables (not secrets):

```bash
gh variable set E2E_TEST_ORG_1 --body "your-org-1" --repo AmadeusITGroup/gh-app-auth
gh variable set E2E_TEST_ORG_2 --body "your-org-2" --repo AmadeusITGroup/gh-app-auth
```

Then update the `test_org_1` / `test_org_2` inputs in `.github/workflows/release.yml`.

## Running E2E Tests Locally

```bash
# Set required environment variables
export E2E_APP_ID="123456"
export E2E_PRIVATE_KEY_B64="$(base64 -w0 < /path/to/key.pem)"

# Run all E2E tests (builds binary from source)
make test-e2e

# Or run a specific test
go test -v -tags=e2e -run TestAuthentication -timeout=5m ./test/e2e/...

# Run against a locally built binary
make test-e2e-local

# Run against a specific binary (e.g. a downloaded release artifact)
export E2E_BINARY_PATH=/path/to/gh-app-auth
make test-e2e
```

> **Note**: The E2E tests automatically discover installation IDs for each organization using the GitHub API. Each organization may have a different installation ID.

## Cost Estimate

| Resource | Runs | Approximate cost |
|----------|------|-----------------|
| ubuntu-latest (E2E linux) | Per release tag | ~5 min × $0.008/min |
| Fedora container | Per release tag | ~8 min × $0.008/min |
| macos-latest | Per release tag | ~8 min × $0.08/min |
| windows-latest | Per release tag | ~8 min × $0.016/min |

Total per release: **~$1–2** (paid plans). Free for public repositories.
