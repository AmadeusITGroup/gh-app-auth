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
   git clone https://github.com/AmadeusITGroup/gh-app-auth.git
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
- [ ] Commit messages follow conventional commits format

## Commit Message Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) specification for all commit messages. This enables automated changelog generation and helps maintain a clear project history.

### Conventional Commits Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Commit Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature for the user | `feat(auth): add JWT token caching` |
| `fix` | Bug fix for the user | `fix(config): handle missing config file` |
| `docs` | Documentation changes | `docs: update installation instructions` |
| `style` | Code style changes (formatting, etc.) | `style: fix gofmt issues` |
| `refactor` | Code refactoring without feature changes | `refactor(jwt): simplify token generation` |
| `perf` | Performance improvements | `perf(cache): optimize token storage` |
| `test` | Adding or fixing tests | `test(auth): add integration tests` |
| `build` | Build system or dependency changes | `build: update go version to 1.21` |
| `ci` | CI/CD configuration changes | `ci: add security scanning workflow` |
| `chore` | Other changes (maintenance, etc.) | `chore: update dependencies` |
| `revert` | Reverting previous commits | `revert: "feat: add experimental feature"` |

### Scopes (Optional)

Scopes provide additional context about the area of change:

- `auth` - Authentication and JWT handling
- `config` - Configuration management
- `cli` - Command-line interface
- `cache` - Token caching functionality
- `security` - Security-related changes
- `docs` - Documentation
- `ci` - Continuous integration
- `deps` - Dependencies

### Commit Message Examples

#### ✅ Good Examples

```bash
# New feature with scope
feat(auth): implement GitHub App authentication with JWT

# Bug fix with detailed description
fix(config): resolve panic when config file is missing

Add proper error handling for missing configuration files
instead of panicking. Now returns a user-friendly error
message and suggests running the setup command.

Fixes #42

# Documentation update
docs: add security best practices section

# Breaking change
feat(cli)!: change setup command flag from --app to --app-id

BREAKING CHANGE: The --app flag has been renamed to --app-id
for consistency with GitHub API terminology. Users should
update their scripts accordingly.

# Multiple types in one commit (avoid this)
# ❌ BAD: feat(auth): add caching and fix JWT bug
# ✅ GOOD: Split into separate commits
```

#### ❌ Examples to Avoid

```bash
# Too vague
fix: stuff

# Missing type
update readme

# Not descriptive enough  
feat: improvements

# Mixed concerns (should be separate commits)
feat: add caching and fix config bug
```

### Writing Good Commit Messages

1. **Use the imperative mood**: "Add feature" not "Added feature"
2. **Keep the subject line under 72 characters**
3. **Capitalize the subject line**
4. **Don't end the subject line with a period**
5. **Use the body to explain what and why, not how**
6. **Reference issues and pull requests when relevant**

### Breaking Changes

For breaking changes, use one of these formats:

```bash
# Method 1: ! after type/scope
feat(cli)!: change setup command interface

# Method 2: BREAKING CHANGE footer
feat(auth): improve token validation

BREAKING CHANGE: Token validation now requires additional
permissions. Users must regenerate their GitHub App tokens.
```

### Tools and Tips

#### Git Commit Templates

Create a commit template to remind yourself of the format:

```bash
# Create template file
cat > ~/.gitmessage << 'EOF'
# <type>[optional scope]: <description>
#
# [optional body]
#
# [optional footer(s)]
#
# Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
# Scopes: auth, config, cli, cache, security, docs, ci, deps
# Remember: use imperative mood, keep subject under 72 chars
EOF

# Configure git to use the template
git config --global commit.template ~/.gitmessage
```

#### Conventional Commits Tools

- **commitizen**: Interactive commit message tool
- **conventional-changelog**: Generates changelogs from commits
- **semantic-release**: Automates releases based on commit messages

#### Installation

```bash
# Install commitizen globally
npm install -g commitizen cz-conventional-changelog

# Use in project
npx cz
```

### Integration with Project Workflows

Our automated workflows rely on conventional commits for:

1. **Automated Changelog**: Generated from commit messages
2. **Semantic Versioning**: Version bumps based on commit types
3. **Release Notes**: Formatted release descriptions
4. **PR Labeling**: Automatic labels based on commit types

### Commit Message Validation

Our CI pipeline validates commit messages. If your commit doesn't follow the conventional format, the build may fail. To fix this:

1. **For the last commit**: `git commit --amend`
2. **For multiple commits**: `git rebase -i HEAD~n` (where n is the number of commits)
3. **Force push**: `git push --force-with-lease origin your-branch`

**Note**: Only force push to your own feature branches, never to main/develop.

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
