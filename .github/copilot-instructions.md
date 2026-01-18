# GitHub Copilot Instructions for gh-app-auth

This project is a GitHub CLI extension written in Go that provides Git credential authentication using GitHub Apps and Personal Access Tokens (PATs). It implements the Git credential helper protocol.

## Project Overview

**Purpose**: Simplify Git authentication for CI/CD pipelines and multi-organization setups by providing a unified credential helper that supports GitHub Apps, PATs, and multiple Git providers (GitHub, Bitbucket).

**Key Problems Solved**:

- Cross-organization repository access with GitHub Apps
- Automatic token refresh (GitHub App tokens expire after 1 hour)
- Pattern-based credential routing
- Encrypted storage of private keys and tokens

## Technology Stack

- **Language**: Go 1.21+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **GitHub API**: go-gh library (github.com/cli/go-gh/v2)
- **Secrets**: OS-native keyring (github.com/zalando/go-keyring)
- **Testing**: Go standard library + testify
- **Linting**: golangci-lint
- **CI**: GitHub Actions

## Project Structure

```
gh-app-auth/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Main command entry
│   ├── setup.go           # Configure credentials
│   ├── list.go            # List configured credentials
│   ├── remove.go          # Remove credentials
│   ├── git-credential.go  # Git credential helper protocol
│   ├── gitconfig.go       # Auto-configure git
│   ├── scope.go           # Show which credential handles a URL
│   ├── migrate.go         # Migrate to encrypted storage
│   └── test.go            # Test authentication
├── pkg/
│   ├── auth/              # GitHub App authentication
│   ├── cache/             # In-memory token caching
│   ├── config/            # Configuration management
│   ├── jwt/               # JWT token generation
│   ├── matcher/           # URL pattern matching
│   ├── secrets/           # Encrypted key storage
│   ├── scope/             # Credential scope detection
│   └── logger/            # Diagnostic logging
├── test/                   # Integration & E2E tests
├── docs/                   # Documentation
└── .github/
    ├── workflows/         # CI/CD pipelines
    └── actions/           # Reusable GitHub Actions
```

## Code Conventions

### Go Style

- Follow standard Go idioms and `gofmt`
- Use table-driven tests
- Error messages should be lowercase (staticcheck ST1005)
- Prefer explicit error handling over panic
- Use meaningful variable names (not single letters except loops)

### CLI Commands

- Commands use Cobra framework
- Each command has `NewXxxCmd()` constructor and `xxxRun()` execution
- Support both flags and environment variables
- Output uses color for interactive terminals, plain text for pipes

### Security Principles

- Never log tokens, private keys, or secrets in plain text
- Use token hashes (SHA-256) for debugging
- Validate file permissions for private keys (600/400)
- Prefer OS keyring over filesystem storage
- Zero sensitive data from memory when possible

### Configuration

- Config file: `~/.config/gh/extensions/gh-app-auth/config.yml`
- Support both YAML and JSON formats
- Pattern matching uses longest-prefix-first, then priority
- Graceful degradation when keyring unavailable

## Testing Requirements

- Unit tests for all exported functions
- Integration tests for command flows
- Maintain minimum 50% coverage (target: 70%)
- Use `t.TempDir()` for test isolation
- Mock external dependencies (GitHub API, keyring)

## Common Tasks

### Adding a New Command

1. Create `cmd/newcommand.go` with `NewNewCommandCmd()`
2. Register in `cmd/root.go`
3. Add corresponding tests in `cmd/newcommand_test.go`
4. Update help text and documentation

### Modifying Pattern Matching

- Changes go in `pkg/matcher/matcher.go`
- Ensure backward compatibility with existing configs
- Add tests for edge cases

### Working with Secrets

- Use `pkg/secrets/secrets.go` for all secret operations
- Never hardcode test tokens or keys
- Mock keyring in tests using interfaces

## Documentation Style

- Use Markdown with clear headings
- Include code examples that work
- Keep examples using generic placeholders (not internal references)
- Reference other docs with relative links

## Pull Request Checklist

- [ ] Tests pass (`make test`)
- [ ] Linting passes (`make quality`)
- [ ] New code has tests
- [ ] Documentation updated if needed
- [ ] Commit messages follow Conventional Commits
- [ ] No sensitive data in code or logs
