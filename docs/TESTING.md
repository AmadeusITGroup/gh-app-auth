# Testing Guide for gh-app-auth

This document describes the comprehensive test suite for the gh-app-auth git credential helper.

## Test Coverage Overview

The test suite provides three levels of testing:

1. **Unit Tests** - Test individual functions in isolation
2. **Integration Tests** - Test the full command flow with simulated git input
3. **Manual Tests** - Interactive script for real-world verification

## Running Tests

### Run All Tests

```bash
make test
```

Or directly:

```bash
go test ./...
```

### Run Specific Test Suites

```bash
# Unit tests only
go test ./cmd/...

# Integration tests only
go test ./test/integration/...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v ./cmd/... -run TestReadCredentialInput
```

## 1. Unit Tests

**Location:** `cmd/git-credential_test.go`

**Coverage:** 20 test cases

### TestReadCredentialInput

Tests parsing of git's credential input format.

**Test Cases:**

- Complete input (protocol, host, path)
- Host only (no path)
- With username field
- Empty input
- Extra whitespace handling

**Example:**

```go
input := `protocol=https
host=github.com
path=myorg/myrepo

`
result, err := readCredentialInput(strings.NewReader(input))
// result = {"protocol": "https", "host": "github.com", "path": "myorg/myrepo"}
```

### TestBuildRepositoryURL

Tests URL construction from git credential input.

**Test Cases:**

- Complete HTTPS URL
- Host only (no path)
- No protocol (defaults to https)
- Path with .git suffix
- Path with leading/trailing slashes
- Enterprise GitHub instances
- Missing host
- Empty input
- HTTP protocol

**Example:**

```go
input := map[string]string{
    "protocol": "https",
    "host":     "github.com",
    "path":     "myorg/myrepo.git",
}
result := buildRepositoryURL(input)
// result = "https://github.com/myorg/myrepo"
```

### TestBuildRepositoryURL_RealWorldExamples

Tests real-world git scenarios.

**Test Cases:**

- `git clone` with HTTPS
- `git fetch` operation
- Initial connection (host only)

## 2. Integration Tests

**Location:** `test/integration/git_credential_test.go`

**Coverage:** 8 integration test scenarios

### Test Scenarios

#### TestGitCredentialHelper_NoConfig

Verifies silent exit when no configuration file exists.

**Expected:** Exit code 0, no output (allows fallback to other helpers)

#### TestGitCredentialHelper_NoMatchingApp

Verifies silent exit when no app matches the repository pattern.

**Expected:** Exit code 0, no output

#### TestGitCredentialHelper_HostOnly

Verifies handling of host-only requests (no repository path).

**Expected:** Exit code 0, no output (git will call again with full path)

#### TestGitCredentialHelper_Store

Tests the `store` operation (git storing credentials).

**Expected:** Exit code 0 (we don't actually store anything)

#### TestGitCredentialHelper_Erase

Tests the `erase` operation (git clearing credentials).

**Expected:** Exit code 0

#### TestGitCredentialHelper_InvalidOperation

Tests rejection of unsupported operations.

**Expected:** Exit code != 0, error message

#### TestGitCredentialProtocol_MultiStage

Simulates git's two-stage credential request protocol.

**Stage 1:** Host only → Silent exit
**Stage 2:** Full path → Attempt authentication

#### TestGitCredentialHelper_OutputFormat

Verifies the output format matches git's expectations.

**Expected Format:**

```
username=app-name[bot]
password=ghs_token123
```

### Running Integration Tests

```bash
# Run all integration tests
go test -v ./test/integration/...

# Run with timeout
go test -v ./test/integration/... -timeout 30s

# Run specific test
go test -v ./test/integration/... -run TestGitCredentialHelper_NoConfig
```

## 3. Manual Test Script

**Location:** `test/manual/test-git-credential.sh`

**Purpose:** Interactive testing with real binary

### Usage

```bash
# Build binary first
go build -o gh-app-auth

# Run manual tests
./test/manual/test-git-credential.sh ./gh-app-auth
```

### Test Scenarios

1. **No config file** - Verifies silent exit
2. **Host only** - Verifies host-only handling
3. **Full URL with config** - Tests authentication flow
4. **Store operation** - Tests credential storage
5. **Erase operation** - Tests credential erasure
6. **Invalid operation** - Tests error handling
7. **Multi-stage protocol** - Simulates git's behavior
8. **Different URL formats** - Tests various URL patterns

### Output

The script provides color-coded output:

- 🟢 **Green** - Test passed
- 🟡 **Yellow** - Expected failure (normal behavior)
- 🔴 **Red** - Test failed

### Testing with Real GitHub App

To test with a real GitHub App:

```bash
# 1. Setup a GitHub App
gh app-auth setup \
  --app-id 123456 \
  --key-file ~/.ssh/app-key.pem \
  --patterns "github.com/myorg/*"

# 2. Run manual tests
./test/manual/test-git-credential.sh ./gh-app-auth

# 3. Test directly
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | \
  ./gh-app-auth git-credential get
```

## Test Statistics

| Test Type | Test Files | Test Cases | Status |
|-----------|-----------|------------|--------|
| Unit Tests | 1 | 20 | ✅ Passing |
| Integration Tests | 1 | 8 | ✅ Passing |
| Manual Tests | 1 script | 8 scenarios | ✅ Available |
| **Total** | **3** | **36** | **✅ All Passing** |

## Git Credential Protocol

### How Git Calls Credential Helpers

Git uses a multi-stage protocol:

1. **Stage 1:** Request with host only

   ```
   protocol=https
   host=github.com
   
   ```

2. **Stage 2:** Request with full path

   ```
   protocol=https
   host=github.com
   path=myorg/myrepo
   
   ```

### Expected Behavior

- **No config:** Exit silently (code 0, no output)
- **No match:** Exit silently (code 0, no output)
- **Match found:** Output credentials in format:

  ```
  username=app-name[bot]
  password=ghs_token123
  ```

### Credential Helper Operations

| Operation | Purpose | Our Implementation |
|-----------|---------|-------------------|
| `get` | Provide credentials | Generate GitHub App token |
| `store` | Store credentials | No-op (we generate dynamically) |
| `erase` | Clear credentials | Clear cache (future) |

## Continuous Integration

Tests run automatically on:

- Every push to main
- Every pull request
- Via GitHub Actions workflow

### CI Configuration

```yaml
- name: Run tests
  run: make test

- name: Run integration tests
  run: go test -v ./test/integration/... -timeout 30s
```

## Debugging Tests

### Enable Verbose Output

```bash
go test -v ./... 2>&1 | tee test-output.log
```

### Debug Specific Test

```bash
go test -v ./cmd/... -run TestReadCredentialInput/complete_input
```

### Test with Debug Flag

```bash
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | \
  ./gh-app-auth --debug git-credential get
```

### Trace Git Credential Calls

```bash
GIT_TRACE=1 GIT_CURL_VERBOSE=1 git clone https://github.com/myorg/myrepo
```

## Adding New Tests

### Unit Test Template

```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "test case 1",
            input:    "input data",
            expected: "expected output",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := functionToTest(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Integration Test Template

```go
func TestGitCredentialHelper_NewScenario(t *testing.T) {
    // Setup
    tempDir := t.TempDir()
    configPath := filepath.Join(tempDir, "config.yml")
    // ... create config ...
    
    t.Setenv("GH_APP_AUTH_CONFIG", configPath)
    binaryPath := buildBinary(t)

    // Test
    input := `protocol=https
host=github.com

`
    cmd := exec.Command(binaryPath, "git-credential", "get")
    cmd.Stdin = strings.NewReader(input)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()

    // Assert
    if err != nil {
        t.Errorf("Expected success, got error: %v", err)
    }
}
```

## Troubleshooting

### Tests Fail with "binary not found"

```bash
# Build the binary first
go build -o gh-app-auth
```

### Integration Tests Timeout

```bash
# Increase timeout
go test -v ./test/integration/... -timeout 60s
```

### Manual Script Hangs

The script may hang if waiting for input. Use Ctrl+C to cancel and check:

- Binary path is correct
- Config file exists (if testing with real app)
- Network connectivity (if testing authentication)

## Best Practices

1. **Run tests before committing:**

   ```bash
   make test
   ```

2. **Test with real GitHub App periodically:**

   ```bash
   ./test/manual/test-git-credential.sh ./gh-app-auth
   ```

3. **Add tests for new features:**
   - Unit tests for new functions
   - Integration tests for new workflows
   - Update manual script for new operations

4. **Keep tests fast:**
   - Unit tests should run in milliseconds
   - Integration tests should complete in seconds
   - Use timeouts to prevent hangs

## Related Documentation

- [Git Credential Protocol](https://git-scm.com/docs/git-credential)
- [GitHub App Authentication](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app)
- [Go Testing Package](https://pkg.go.dev/testing)
