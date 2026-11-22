---
title: End-to-end test infrastructure
description: The maintainer recipe for the end-to-end test infrastructure, its workflows and the credentials they need.
sidebar:
  order: 5
---

This document describes the **one-time manual setup** required to run the E2E test
suite. The infrastructure is intentionally stable (not ephemeral) to keep costs low and
setup reproducible.

The default fixture is a pair of **existing, dedicated, non-production** organizations and
repositories. Creating organizations or repositories, pushing fixture content, installing the
GitHub App and setting repository secrets all change shared systems and need explicit authorization
from a maintainer or the organization administrators. The commands below describe what an authorized
maintainer does once; they are not routine steps and this documentation never runs them for you.

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

The fixture needs two GitHub organizations. Names are configurable but default to:

```text
gh-app-auth-test-1
gh-app-auth-test-2
```

Normally these already exist and you only need access to them. Creating organizations is an
administrative action that depends on your company's GitHub policy, so ask the administrator who owns
that policy to approve and, where required, perform it. `gh api user/orgs` only lists organizations you
already belong to; it does not show whether you may create one. The organizations can use the
**Free** plan and need no members beyond an owner.

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

## Step 5 — Prepare the Private Key for CI

The private key is stored as a base64-encoded repository secret. Keep the downloaded `.pem` file in a
location approved for secrets, readable only by you. Do not print the encoded key to the terminal, and do
not pass it as a command-line argument, where it could appear in the process list or shell history.
Encode it straight into the secret in the next step instead.

## Step 6 — Set Repository Secrets

In <https://github.com/AmadeusITGroup/gh-app-auth/settings/secrets/actions>, create:

| Secret Name | Value | Description |
|-------------|-------|-------------|
| `E2E_APP_ID` | `123456` | GitHub App ID (numeric) |
| `E2E_PRIVATE_KEY_B64` | `<base64 output>` | Base64-encoded private key PEM |

An authorized maintainer sets the secrets from the approved key file, with the value coming from
standard input rather than an argument:

```bash
printf '%s' "$APP_ID" | gh secret set E2E_APP_ID --repo AmadeusITGroup/gh-app-auth
base64 < "$KEY_FILE" | tr -d '\n' | gh secret set E2E_PRIVATE_KEY_B64 --repo AmadeusITGroup/gh-app-auth
```

`APP_ID` and `KEY_FILE` are variables you set to your App's ID and the path of the approved key file.
This is a manual maintainer action and this documentation does not execute it.

> **Security**: The E2E tests generate short-lived installation tokens using the App ID and private key. Installation IDs are discovered dynamically via the GitHub API. No long-lived tokens are required.

## Step 7 — Verify the Infrastructure

Run the pre-flight check locally before pushing a release:

```bash
export E2E_APP_ID="123456"
export E2E_PRIVATE_KEY_B64="$(base64 < "$KEY_FILE" | tr -d '\n')"

go test -v -tags=e2e -run TestPreflight -timeout=2m ./test/e2e/...
```

`KEY_FILE` is the path of the approved test key file. If `E2E_PRIVATE_KEY` (the raw PEM) is also set, it
takes precedence over `E2E_PRIVATE_KEY_B64`. Run this only against the dedicated fixture, and never
print the encoded key.

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
export E2E_PRIVATE_KEY_B64="$(base64 < "$KEY_FILE" | tr -d '\n')"

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

## Measuring E2E Coverage

`go test -cover` cannot measure this suite — the tests exec the binary as a
subprocess, so counters live outside the test process. Use the coverage
wrapper instead, which builds an instrumented binary (`go build -cover
-coverpkg=./...`) and collects counters from every spawned process via
`GOCOVERDIR`:

```bash
# Requires the same E2E_* env vars as the suite itself
source .env.e2e
make test-e2e-cover
```

Outputs:

| File | Contents |
|------|----------|
| `.e2e-covdata/` | Raw `covdata` counters (`go tool covdata -i=.e2e-covdata …`) |
| `e2e-coverage.out` | textfmt coverprofile (`go tool cover -func=e2e-coverage.out`) |
| `e2e-coverage-summary.md` | Per-package markdown table — appended to `$GITHUB_STEP_SUMMARY` in CI |

All paths are overridable (`E2E_COV_DIR`, `E2E_COV_BINARY`, `E2E_COV_PROFILE`,
`E2E_COV_SUMMARY`) — see `scripts/e2e-coverage.sh`.

In CI, `.github/workflows/e2e.yml` runs this on every PR/push, appends the
summary table to the workflow-run summary, and uploads the covdata + profile
as the `e2e-coverage-<os>` artifact.

## Cost drivers

This document does not quote prices or run times, because both change. What drives the cost of the
end-to-end workflow is:

- the number of runs: the workflow runs on pull requests, pushes to `main` and manual dispatch, and the
  release pipeline gates on it;
- the operating systems in the matrix, where macOS and Windows runners are billed at higher multipliers
  than Linux for private repositories;
- the container job for Fedora, and the time each job spends on retries and network access;
- the runners and secrets being available at all: when repository credentials are unavailable (for
  example on a fork), the secrets check skips the three operating-system jobs.

Public repositories use standard GitHub-hosted runners at no charge. Check the current GitHub Actions
pricing for your plan before estimating.
