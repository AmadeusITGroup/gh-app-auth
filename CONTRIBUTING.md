# Contributing to gh-app-auth

Thank you for your interest in contributing to the GitHub App Authentication extension for GitHub CLI!

## Development Setup

### Prerequisites
- Go 1.19 or later
- GitHub CLI (`gh`) installed and configured
- Git

### Getting Started

1. **Clone and setup**:
   ```bash
   git clone https://github.com/wherka-ama/gh-app-auth.git
   cd gh-app-auth
   go mod download
   ```

2. **Build the extension**:
   ```bash
   go build -o gh-app-auth .
   ```

3. **Install locally for testing**:
   ```bash
   gh extension install .
   ```

4. **Run tests**:
   ```bash
   go test ./...
   ```

## Project Structure

```
gh-app-auth/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and CLI setup
│   ├── setup.go           # Setup command implementation
│   ├── list.go            # List command implementation
│   ├── remove.go          # Remove command implementation  
│   ├── test.go            # Test command implementation
│   └── git-credential.go  # Git credential helper
├── pkg/                   # Core packages
│   ├── auth/             # Authentication logic
│   ├── cache/            # Token caching
│   ├── config/           # Configuration management
│   ├── jwt/              # JWT token generation
│   └── matcher/          # Repository pattern matching
├── docs/                 # Documentation
└── scripts/              # Build and utility scripts
```

## Development Guidelines

### Code Style
- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comprehensive tests for new functionality
- Include documentation for public APIs

### Testing
- Write unit tests for all packages
- Include integration tests for CLI commands
- Test security-critical code paths thoroughly
- Use table-driven tests where appropriate

### Security Considerations
- Never log or expose private keys or tokens
- Validate file permissions for private key files
- Use secure temporary directories for testing
- Follow principle of least privilege

## Making Changes

### Adding New Commands
1. Create command file in `cmd/` directory
2. Implement cobra.Command with appropriate flags
3. Add command to root.go
4. Write comprehensive tests
5. Update documentation

### Adding New Features
1. Design the feature with security in mind
2. Implement in appropriate package
3. Add configuration options if needed
4. Write tests covering all code paths
5. Update documentation and examples

### Bug Fixes
1. Write a test that reproduces the bug
2. Implement the fix
3. Verify the test passes
4. Consider if documentation needs updates

## Pull Request Process

1. **Fork and branch**: Create a feature branch from main
2. **Implement**: Make your changes following the guidelines
3. **Test**: Ensure all tests pass and add new tests
4. **Document**: Update relevant documentation
5. **Submit**: Create a pull request with:
   - Clear description of the change
   - Reference to any related issues
   - Test evidence (screenshots, test output)

### PR Checklist
- [ ] Code follows project style guidelines
- [ ] Tests added for new functionality
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Security considerations addressed
- [ ] Breaking changes clearly documented

## Code Review

All submissions require code review. Please:
- Be responsive to feedback
- Keep changes focused and atomic
- Write clear commit messages
- Rebase rather than merge when updating PRs

## Release Process

Releases are automated through GitHub Actions:
1. Tag a release with semantic versioning (e.g., v1.2.3)
2. GitHub Actions builds cross-platform binaries
3. Release is published to GitHub marketplace

## Getting Help

- **Issues**: Open GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Security**: Report security issues via GitHub Security Advisories

## Resources

- [GitHub CLI Extension Development](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
- [go-gh Library Documentation](https://pkg.go.dev/github.com/cli/go-gh/v2)
- [GitHub App Authentication](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps)
