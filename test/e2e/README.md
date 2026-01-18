# End-to-End Tests

This directory contains end-to-end tests that validate complete workflows with real or near-real integrations.

## Overview

E2E tests differ from unit tests in that they:

- Build and execute the actual binary
- May interact with real external services (GitHub API)
- Test complete user workflows from start to finish
- Validate integration between components

## Running E2E Tests

### Basic E2E Tests (No External Dependencies)

These tests run with mocked or stubbed external services:

```bash
# Run all e2e tests
go test -v ./test/e2e/...

# Run specific test
go test -v ./test/e2e/... -run TestCompleteSetupWorkflow
```

### Real GitHub API Tests (Requires Credentials)

These tests use real GitHub Apps and require credentials:

```bash
# Set required environment variables
export GITHUB_APP_ID="123456"
export GITHUB_APP_PRIVATE_KEY="$(cat ~/.ssh/test-app.pem)"
export GITHUB_TEST_REPO="org/private-test-repo"

# Run with e2e tag
go test -v -tags=e2e ./test/e2e/...
```

## Test Structure

```
test/e2e/
├── README.md                    # This file
├── workflows_test.go            # Complete workflow tests (mocked)
├── github_api_test.go           # Real GitHub API tests (requires creds)
├── actions_test.go              # GitHub Actions composite action tests
├── multi_org_test.go            # Multi-organization scenarios
└── testutil/
    ├── mock_github.go           # Mock GitHub API server
    ├── test_helpers.go          # Common test utilities
    └── test_config.go           # Test configuration helpers
```

## Test Categories

### 1. Workflow Tests (`workflows_test.go`)

Test complete user workflows with mocked external dependencies:

- **Setup → GitConfig → Clone**: Complete authentication flow
- **Multi-Org Setup**: Configure multiple organizations
- **Migration**: Migrate from filesystem to keyring
- **Cleanup**: Verify credential cleanup

**Example:**

```go
func TestCompleteSetupWorkflow(t *testing.T) {
    // 1. Setup GitHub App
    // 2. Run gitconfig sync
    // 3. Attempt git clone (mocked)
    // 4. Verify success
    // 5. Cleanup
}
```

### 2. GitHub API Tests (`github_api_test.go`)

Test with real GitHub API (optional, requires test app):

```go
// +build e2e

func TestRealGitHubAuthentication(t *testing.T) {
    if os.Getenv("GITHUB_APP_ID") == "" {
        t.Skip("Skipping: GITHUB_APP_ID not set")
    }
    // Test real JWT generation and token fetch
}
```

### 3. Multi-Org Tests (`multi_org_test.go`)

Test multi-organization scenarios:

- **Multiple Apps**: Configure apps for different orgs
- **Pattern Matching**: Verify correct app selection
- **Git Config Sync**: Validate all orgs configured
- **JSON Config**: Test JSON-based multi-app setup

### 4. Actions Tests (`actions_test.go`)

Test GitHub Actions composite actions:

- **Setup Action**: Validate action inputs and outputs
- **Cleanup Action**: Verify credential removal
- **JSON Multi-Org**: Test apps-config JSON input

## Required Test Apps

For real API testing, create test GitHub Apps with minimal permissions:

### Test App Setup

1. Go to GitHub Settings → Developer settings → GitHub Apps
2. Create a new GitHub App:
   - **Name**: `gh-app-auth-e2e-test`
   - **Homepage URL**: `https://github.com/AmadeusITGroup/gh-app-auth`
   - **Repository permissions**:
     - Contents: Read-only (for clone testing)
   - **Where can this app be installed**: Only on this account

3. Generate a private key and download it

4. Install the app to a test organization or repository

5. Set environment variables:

   ```bash
   export GITHUB_APP_ID="<your-app-id>"
   export GITHUB_APP_PRIVATE_KEY="$(cat ~/Downloads/test-app.pem)"
   export GITHUB_TEST_REPO="yourorg/test-private-repo"
   ```

### Security Note

⚠️ **Never commit test credentials to the repository!**

- Use environment variables for credentials
- Add `*.pem` to `.gitignore`
- Consider using GitHub Actions secrets for CI/CD

## Writing New E2E Tests

### Best Practices

1. **Use Build Tags for Real API Tests**

   ```go
   // +build e2e
   
   package e2e
   ```

2. **Skip When Credentials Missing**

   ```go
   if os.Getenv("GITHUB_APP_ID") == "" {
       t.Skip("Skipping: credentials not provided")
   }
   ```

3. **Clean Up Resources**

   ```go
   t.Cleanup(func() {
       // Remove test configurations
       // Clean up git config
       // Delete test files
   })
   ```

4. **Use Subtests for Scenarios**

   ```go
   func TestWorkflow(t *testing.T) {
       t.Run("setup", func(t *testing.T) { ... })
       t.Run("configure", func(t *testing.T) { ... })
       t.Run("cleanup", func(t *testing.T) { ... })
   }
   ```

5. **Test Helpers in `testutil/`**
   - Mock GitHub API servers
   - Test configuration builders
   - Common assertion helpers

### Example Test Template

```go
package e2e

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestNewWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping e2e test in short mode")
    }

    // Setup
    tempDir := t.TempDir()
    binaryPath := buildBinary(t)
    
    t.Cleanup(func() {
        // Cleanup resources
    })

    // Test steps
    t.Run("step1", func(t *testing.T) {
        // Test logic
    })

    t.Run("step2", func(t *testing.T) {
        // Test logic
    })
}

func buildBinary(t *testing.T) string {
    t.Helper()
    binaryPath := filepath.Join(t.TempDir(), "gh-app-auth")
    cmd := exec.Command("go", "build", "-o", binaryPath, "../..")
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to build binary: %v", err)
    }
    return binaryPath
}
```

## CI/CD Integration

### Makefile Targets

```makefile
# Run e2e tests (no credentials)
test-e2e:
    go test -v ./test/e2e/...

# Run e2e tests with real API (requires credentials)
test-e2e-real:
    go test -v -tags=e2e ./test/e2e/...
```

### GitHub Actions

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-basic:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-e2e

  e2e-real-api:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-e2e-real
        env:
          GITHUB_APP_ID: ${{ secrets.TEST_APP_ID }}
          GITHUB_APP_PRIVATE_KEY: ${{ secrets.TEST_PRIVATE_KEY }}
```

## Troubleshooting

### Tests Timeout

E2E tests may take longer than unit tests. Increase timeout:

```bash
go test -v -timeout 5m ./test/e2e/...
```

### Binary Build Fails

Ensure you're in the project root and dependencies are downloaded:

```bash
cd /path/to/gh-app-auth
go mod download
```

### Git Commands Fail

Some tests require git to be available:

```bash
which git  # Verify git is installed
git --version  # Verify git works
```

### GitHub API Rate Limiting

If real API tests hit rate limits:

1. Use authenticated requests (they have higher limits)
2. Add delays between tests
3. Cache results when possible
4. Run less frequently (only on main branch)

## Future Improvements

- [ ] Add mock GitHub API server in `testutil/`
- [ ] Add Docker-based testing environment
- [ ] Add cross-platform E2E tests (Windows, macOS)
- [ ] Add performance/load testing scenarios
- [ ] Add chaos testing (network failures, API errors)
- [ ] Integrate with GitHub Actions workflow tests

## Questions?

See the main [CONTRIBUTING.md](../../CONTRIBUTING.md) or open an issue.
