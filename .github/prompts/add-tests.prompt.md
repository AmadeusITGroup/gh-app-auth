---
mode: agent
description: "Add tests for gh-app-auth code"
---

# Add Tests

Add comprehensive tests for the specified code in gh-app-auth.

## Test Strategy

1. **Identify Test Gaps**
   - Run coverage: `go test -coverprofile=coverage.out ./...`
   - View HTML report: `go tool cover -html=coverage.out`
   - Focus on untested functions and branches

2. **Prioritize Tests**
   - Security-critical code first (auth, secrets, jwt)
   - Error handling paths
   - Edge cases and boundary conditions
   - Happy path scenarios

3. **Write Tests**
   - Use table-driven tests
   - Include positive and negative cases
   - Test error messages and types
   - Mock external dependencies

## Test Categories

### Unit Tests

Test individual functions in isolation:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"empty input", "", nil, true},
    }
    // ...
}
```

### Integration Tests

Test component interactions:

```go
func TestWorkflow(t *testing.T) {
    // Setup temporary config
    configDir := t.TempDir()
    // Create config file
    // Run command sequence
    // Verify results
}
```

### Command Tests

Test CLI commands:

```go
func TestNewExampleCmd(t *testing.T) {
    cmd := NewExampleCmd()
    
    // Test structure
    require.Equal(t, "example", cmd.Use)
    
    // Test flags
    flag := cmd.Flags().Lookup("flag-name")
    require.NotNil(t, flag)
}
```

## Test Utilities

Use helpers from `test/testutil/`:

- `testutil.CreateTestConfig()` - Create test configuration
- `testutil.MockGitHubAPI()` - Mock GitHub API responses
- `testutil.TempConfigDir()` - Temporary config directory

## Coverage Targets

Ensure these minimums:

- New code: 80%+ coverage
- Security code: 90%+ coverage
- No decrease in overall coverage

## Verification

After adding tests:

```bash
# Run tests
go test -v ./path/to/package/...

# Check coverage
go test -coverprofile=coverage.out ./path/to/package/...
go tool cover -func=coverage.out | grep -E "^total:|path/to/package"

# Run all tests
make test

# Run quality checks
make quality
```
