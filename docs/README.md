# Documentation

This directory contains the documentation for `gh-app-auth`, a GitHub CLI extension for Git credential authentication using GitHub Apps and Personal Access Tokens.

## User Documentation

| Document | Description | Audience |
|----------|-------------|----------|
| [Installation Guide](installation.md) | Setup and quick start | All users |
| [Configuration Reference](configuration.md) | Config file format, patterns, multi-org setup | All users |
| [Security Considerations](security.md) | Security best practices and threat model | All users |
| [Troubleshooting](troubleshooting.md) | Common issues, fixes, and diagnostic logging | All users |
| [CI/CD Integration](ci-cd-guide.md) | GitHub Actions, Jenkins, GitLab CI examples | DevOps engineers |

## Contributor Documentation

| Document | Description | Audience |
|----------|-------------|----------|
| [Architecture Overview](architecture.md) | System design, components, data flow | Contributors |
| [Testing Guide](TESTING.md) | Test suite, coverage, adding tests | Contributors |
| [E2E Testing Tutorial](E2E_TESTING_TUTORIAL.md) | End-to-end testing setup | Contributors |
| [Origin of the Project](origin_of_the_project.md) | Why this project was created | Contributors |

## Reference Documentation

These documents provide deep-dive technical details:

| Document | Description |
|----------|-------------|
| [Token Caching](TOKEN_CACHING.md) | In-memory token cache implementation |
| [Encrypted Storage](ENCRYPTED_STORAGE_ARCHITECTURE.md) | OS keyring integration design |
| [Pattern Routing](PATTERN_ROUTING.md) | Advanced pattern matching details |
| [Diagnostic Logging](DIAGNOSTIC_LOGGING.md) | Debug logging implementation |

## Quick Start

```bash
# Install
gh extension install AmadeusITGroup/gh-app-auth

# Configure a GitHub App
export GH_APP_PRIVATE_KEY="$(cat ~/my-key.pem)"
gh app-auth setup \
  --app-id 123456 \
  --patterns "github.com/myorg/*" \
  --name "My App"

# Sync git credential helpers
gh app-auth gitconfig --sync --global

# Test it works
git clone https://github.com/myorg/private-repo
```

## Getting Help

- [GitHub Issues](https://github.com/AmadeusITGroup/gh-app-auth/issues)
- [CONTRIBUTING.md](../CONTRIBUTING.md)
- Run `gh app-auth --help`
