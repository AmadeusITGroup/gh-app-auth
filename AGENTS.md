# AGENTS.md

> A guide for AI coding agents working on gh-app-auth.

## Project Overview

**gh-app-auth** is a GitHub CLI extension written in Go that provides Git credential authentication using GitHub Apps and Personal Access Tokens (PATs). It implements the Git credential helper protocol for seamless integration with Git operations.

**Key Problems Solved**:

- Cross-organization repository access with GitHub Apps
- Automatic token refresh (GitHub App tokens expire after 1 hour)
- Pattern-based credential routing
- Encrypted storage of private keys and tokens

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Language | Go | 1.21+ |
| CLI Framework | Cobra | github.com/spf13/cobra |
| GitHub API | go-gh | github.com/cli/go-gh/v2 |
| Secrets | go-keyring | github.com/zalando/go-keyring |
| Testing | Go stdlib + testify | - |
| Linting | golangci-lint | v2.1.6+ |

## Commands

```bash
# Build
go build -o gh-app-auth .

# Test (all)
make test

# Test with race detection
go test -race ./...

# Test specific package
go test -v ./pkg/auth/...

# Test with coverage
go test -coverprofile=coverage.out ./...

# Lint (comprehensive)
make quality

# Format code
make fmt

# Full CI simulation
make ci

# Security scan
make security-scan

# Install locally
gh extension install .
```

## Project Structure

```
gh-app-auth/
├── cmd/                    # CLI commands (Cobra) - YOU WRITE HERE
│   ├── root.go            # Main command entry
│   ├── setup.go           # Configure credentials
│   ├── list.go            # List configured credentials
│   ├── remove.go          # Remove credentials
│   ├── git-credential.go  # Git credential helper protocol
│   ├── gitconfig.go       # Auto-configure git
│   ├── scope.go           # Show which credential handles a URL
│   ├── migrate.go         # Migrate to encrypted storage
│   ├── test.go            # Test authentication
│   └── debug.go           # Debug utilities
├── pkg/                    # Core packages - YOU WRITE HERE
│   ├── auth/              # GitHub App authentication
│   ├── cache/             # In-memory token caching (96% coverage)
│   ├── config/            # Configuration management
│   ├── jwt/               # JWT token generation
│   ├── matcher/           # URL pattern matching (95% coverage)
│   ├── secrets/           # Encrypted key storage
│   ├── scope/             # Credential scope detection
│   └── logger/            # Diagnostic logging
├── test/                   # Integration & E2E tests
│   ├── integration/       # Integration tests
│   ├── e2e/               # End-to-end tests
│   └── testutil/          # Test utilities
├── docs/                   # Documentation - YOU WRITE HERE
│   └── adr/               # Architecture Decision Records - SEE ADR SECTION
└── .github/
    ├── workflows/         # CI/CD pipelines
    ├── actions/           # Reusable GitHub Actions
    └── prompts/           # Reusable AI prompts
```

## Code Style

### Import Organization

Always organize imports in three groups:

```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // External packages
    "github.com/spf13/cobra"

    // Internal packages
    "github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)
```

### Error Handling

```go
// ✅ Good - context and wrapping
if err != nil {
    return fmt.Errorf("failed to load config from %s: %w", path, err)
}

// ❌ Bad - no context
if err != nil {
    return err
}
```

### Cobra Command Pattern

```go
func NewExampleCmd() *cobra.Command {
    var flagValue string
    
    cmd := &cobra.Command{
        Use:   "example",
        Short: "Brief description",
        Long:  `Detailed description with examples.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return exampleRun(cmd, flagValue)
        },
    }
    
    cmd.Flags().StringVar(&flagValue, "flag", "", "Flag description")
    
    return cmd
}

func exampleRun(cmd *cobra.Command, flagValue string) error {
    // Implementation
    return nil
}
```

### Naming Conventions

- Functions: `camelCase` (`getUserData`, `calculateTotal`)
- Types/Structs: `PascalCase` (`UserService`, `GitHubApp`)
- Constants: `PascalCase` or `UPPER_SNAKE_CASE` for env vars
- Error strings: lowercase, no punctuation (`"failed to load config"`)

### Console Output

```go
// ✅ Good - use fmt.Print/Println for static strings
fmt.Println("Operation completed successfully")
fmt.Print("Processing...")

// ✅ Good - use fmt.Printf only when formatting variables
fmt.Printf("Processed %d items in %s\n", count, duration)

// ❌ Bad - unnecessary Printf for static string
fmt.Printf("Operation completed successfully\n")
```

## Testing Requirements

### Coverage Targets

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

### Test Patterns

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"empty input", Input{}, Output{}, true},
        {"edge case", edgeInput, edgeOutput, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("Function() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Isolation

```go
func TestWithTempDir(t *testing.T) {
    configDir := t.TempDir()  // Auto-cleaned
    t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(configDir, "config.yml"))
    // Test code...
}
```

## Security Guidelines

### 🔐 CRITICAL - This is an authentication project

```go
// ✅ Good - hash tokens for logging
logger.Debug("token retrieved", "hash", secrets.HashToken(token))

// ❌ NEVER - exposes token
logger.Debug("token retrieved", "token", token)
```

### Security Checklist

- [ ] No tokens, keys, or passwords logged in plain text
- [ ] Use `secrets.HashToken()` for debug logging
- [ ] Validate file permissions before reading private keys (600/400)
- [ ] Prefer OS keyring over filesystem storage
- [ ] Zero sensitive byte slices after use when possible
- [ ] No hardcoded credentials or test secrets
- [ ] Path traversal prevention (`../` rejected)

### Token Security

- Installation tokens: **memory-only**, 55-minute TTL
- Private keys: OS keyring (encrypted) or filesystem with 600/400 permissions
- Never persist tokens to disk

## Architecture Decision Records (ADR)

Significant decisions are recorded in `docs/adr/` as numbered, immutable documents. See
[docs/adr/README.md](docs/adr/README.md) for the full template, naming rules, and status lifecycle.

### 🛑 Agent rule: ASK before implementing, PROPOSE after

**When a task introduces a breaking change or a new feature, you MUST ask the user whether an ADR is
needed before you finish the change.** Do not silently create one, and do not silently skip one.

Ask when the change involves any of these:

| Trigger | Examples in this project |
|---------|--------------------------|
| **Breaking change** | Removing/renaming a CLI flag or command, changing `config.yml` format, changing the git credential helper protocol (`cmd/git-credential.go`), changing default behaviour users depend on |
| **New feature** | A new credential type (like PAT support was), a new command that adds a concept, a new authentication flow |
| **Security decision** | Token lifetimes, key/token storage location, permission checks, what gets logged and how it is redacted (`pkg/secrets/`, `pkg/auth/`, `pkg/jwt/`) |
| **Technology choice** | Adding/replacing a dependency in `go.mod`, changing the keyring or packaging tooling |
| **Significant refactoring** | Moving responsibility between packages, changing a package's public interface |
| **Rejected alternative** | You evaluated an obvious approach and chose against it — the reasoning is worth recording |

Do **not** ask for: bug fixes that restore documented behaviour, test additions, doc-only changes,
formatting, lint fixes, or dependency version bumps with no behavioural change.

### How to ask

State the decision, propose the ADR, and let the user decide. Keep it to a few lines:

```text
This adds a new credential type, which changes the config file format — a breaking change
for existing configs.

Suggested ADR: docs/adr/0002-support-oauth-device-flow-credentials.md
  Context:      Users on shared machines cannot use App private keys.
  Decision:     Add a `device_flow` credential source stored in the keyring.
  Consequences: New config key; configs written by this version are not readable by <= v0.0.7.

Want me to write it? (yes / no / adjust the framing)
```

If the user says yes:

1. Take the next unused `NNNN` (check `docs/adr/` — do not reuse numbers)
2. Create `docs/adr/NNNN-short-title.md` using the template in `docs/adr/README.md`
3. Add a row to the index table in `docs/adr/README.md`
4. Commit it with the change it documents, using `docs(adr): ...` or the change's own scope
5. Run `make markdownlint`

If the user says no, proceed without one and do not ask again for the same change.

### Writing quality ADRs

- Title states the **decision**, not the problem: "Leave Margin on JWT Timestamp Claims", not
  "JWT Clock Skew Problem"
- Anchor to the code: name the packages, files, and constants involved
- Record rejected alternatives and **why** — this is usually the most valuable section
- List real costs in Consequences, including what still does not work by design
- Name the test that prevents regression
- Never rewrite an `Accepted` ADR to reflect a new decision. Set it to
  `Superseded by NNNN` and write a new one.

[`docs/adr/0001-jwt-clock-skew-margin.md`](docs/adr/0001-jwt-clock-skew-margin.md) is the reference
example of the expected depth.

## Git Workflow

### Commit Messages (Conventional Commits)

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Scopes**: `auth`, `config`, `cli`, `cache`, `security`, `docs`, `ci`, `deps`

**Examples**:

```bash
feat(auth): add JWT token caching
fix(config): handle missing config file gracefully
docs: update installation instructions
test(auth): add integration tests for token refresh
```

### PR Requirements

- [ ] Tests pass (`make test`)
- [ ] Linting passes (`make quality`)
- [ ] New code has tests
- [ ] Documentation updated if needed
- [ ] ADR added, or explicitly declined by the user, for breaking changes and new features
- [ ] Commit messages follow Conventional Commits
- [ ] No sensitive data in code or logs

## Boundaries

### ✅ Always Do

- Run `make test` after modifying Go files
- Run `make fmt` before committing
- Add tests for new functionality
- Use table-driven tests for multiple scenarios
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Follow existing code patterns in the file you're editing
- Use `t.TempDir()` for test file isolation

### ⚠️ Ask First

- Whether an ADR is needed for any breaking change or new feature (see [ADR section](#architecture-decision-records-adr))
- Adding new dependencies to `go.mod`
- Modifying CI/CD workflows (`.github/workflows/`)
- Changing security-critical code (`pkg/secrets/`, `pkg/auth/`, `pkg/jwt/`)
- Modifying the Git credential helper protocol (`cmd/git-credential.go`)
- Breaking changes to public APIs
- Changing configuration file format

### 🚫 Never Do

- Log tokens, private keys, or secrets in plain text
- Hardcode credentials or API keys
- Commit test keys or tokens (even expired ones)
- Remove or weaken existing tests
- Disable security linters without explicit approval
- Store tokens persistently on disk
- Ignore file permission validation for private keys
- Use `panic()` instead of returning errors
- Modify `vendor/` or `node_modules/` directories

## Common Tasks

### Adding a New Command

1. Create `cmd/newcommand.go` with `NewNewCommandCmd()`
2. Create `cmd/newcommand_test.go` with tests
3. Register in `cmd/root.go`: `rootCmd.AddCommand(NewNewCommandCmd())`
4. Update README.md command reference
5. Ask the user whether the command warrants an ADR (it does if it introduces a new concept)
6. Run `make test` and `make quality`

### Adding a Feature or Making a Breaking Change

1. Implement the change following the patterns in the files you touch
2. Add or update tests covering the new behaviour and its edge cases
3. Update `README.md`, `docs/`, and `CHANGELOG.md` (`[Unreleased]` section)
4. **Ask the user whether an ADR is needed** — see [ADR section](#architecture-decision-records-adr)
5. If yes, write `docs/adr/NNNN-short-title.md` and add it to the index in `docs/adr/README.md`
6. Run `make test` and `make quality`
7. Commit with `feat(<scope>): ...`; add a `BREAKING CHANGE:` footer for breaking changes

### Releasing a Version

Releases are triggered by creating a GitHub **pre-release** on a `vX.Y.Z` tag; the workflow builds
and uploads all assets, then promotes the release to latest. Never create a final release directly —
it would ship with no assets. See [docs/RELEASE_PROCESS.md](docs/RELEASE_PROCESS.md).

### Fixing a Bug

1. Write a test that reproduces the bug
2. Verify the test fails
3. Implement the fix
4. Verify the test passes
5. Run full test suite: `make test`
6. Commit with: `fix(<scope>): <description>`

### Adding Tests

1. Use table-driven tests
2. Cover: valid cases, edge cases, error cases
3. Use `t.TempDir()` for file operations
4. Mock external dependencies via interfaces
5. Target: maintain or improve coverage

### Security Review

When touching security-critical code:

1. Check `pkg/auth/` - Authentication logic
2. Check `pkg/secrets/` - Key and token storage
3. Check `pkg/jwt/` - JWT generation
4. Check `cmd/git-credential.go` - Credential helper
5. Verify no secrets in logs
6. Run `make security-scan`

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| Lint failures | Run `make fmt` then `make quality` |
| Test failures | Check `t.Setenv()` for env vars, use `t.TempDir()` |
| Import errors | Run `go mod tidy` |
| Coverage drop | Add tests for new code paths |

### Debug Commands

```bash
# Verbose test output
go test -v ./pkg/auth/...

# Test with race detection
go test -race ./...

# Show coverage by function
go tool cover -func=coverage.out

# Run specific test
go test -v -run TestFunctionName ./pkg/...
```

---

*This file guides AI coding agents. For human contributors, see [CONTRIBUTING.md](CONTRIBUTING.md).*
