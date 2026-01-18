---
applyTo: "**/*_test.go"
---

# Testing Guidelines for gh-app-auth

## Test File Organization

- Test files are named `*_test.go` next to the source file
- Integration tests go in `test/integration/`
- E2E tests go in `test/e2e/`
- Test utilities go in `test/testutil/`

## Test Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| pkg/cache | 95%+ | 96.4% ✅ |
| pkg/matcher | 95%+ | 95.4% ✅ |
| pkg/auth | 90%+ | 90.2% ✅ |
| pkg/jwt | 85%+ | 89.3% ✅ |
| pkg/config | 85%+ | 87.8% ✅ |
| pkg/secrets | 85%+ | 88.4% ✅ |
| cmd | 70%+ | 70.5% ✅ |
| **Overall** | **70%+** | **70.2%** ✅ |

## Testing Patterns

### Table-Driven Tests

Always prefer table-driven tests for functions with multiple scenarios:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        // Always include: valid case, edge cases, error cases
        {"valid", validInput, expectedOutput, false},
        {"empty", Input{}, Output{}, true},
        {"nil pointer", nilInput, Output{}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            // assertions...
        })
    }
}
```

### Testing Commands (Cobra)

Test command construction and flag parsing:

```go
func TestNewExampleCmd(t *testing.T) {
    cmd := NewExampleCmd()
    
    // Test command structure
    assert.Equal(t, "example", cmd.Use)
    assert.NotNil(t, cmd.RunE)
    
    // Test flags exist
    flag := cmd.Flags().Lookup("flag-name")
    assert.NotNil(t, flag)
}
```

### Testing with Temporary Files

```go
func TestConfigLoad(t *testing.T) {
    configDir := t.TempDir()
    configPath := filepath.Join(configDir, "config.yml")
    
    err := os.WriteFile(configPath, []byte(configContent), 0600)
    require.NoError(t, err)
    
    t.Setenv("GH_APP_AUTH_CONFIG", configPath)
    
    cfg, err := config.Load()
    // assertions...
}
```

### Testing Git Credential Protocol

Simulate git's stdin input:

```go
func TestGitCredential(t *testing.T) {
    input := `protocol=https
host=github.com
path=org/repo

`
    cmd := exec.Command(binaryPath, "git-credential", "get")
    cmd.Stdin = strings.NewReader(input)
    
    output, err := cmd.Output()
    // assertions...
}
```

### Mocking External Dependencies

Use interfaces for mockable dependencies:

```go
// Define interface
type SecretStore interface {
    Get(name string) (string, error)
    Store(name, value string) error
}

// Mock implementation
type mockSecretStore struct {
    secrets map[string]string
}

func (m *mockSecretStore) Get(name string) (string, error) {
    if v, ok := m.secrets[name]; ok {
        return v, nil
    }
    return "", errors.New("not found")
}
```

## What NOT to Test

- Third-party library internals (trust go-keyring, cobra, etc.)
- Simple getters/setters without logic
- Exact error message strings (test error types instead)

## Running Tests

```bash
# All tests
make test

# Specific package
go test -v ./pkg/auth/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific test
go test -v -run TestFunctionName ./pkg/...
```

## CI Requirements

- All tests must pass before merge
- Coverage must not decrease
- No race conditions (`go test -race`)
- Linting must pass (`make quality`)
