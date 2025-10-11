# GitHub App Authentication Extension

A GitHub CLI extension that enables GitHub App authentication for Git operations and API access.

## Features

- 🔐 **GitHub App Authentication**: Use GitHub Apps instead of Personal Access Tokens
- 🎯 **Repository-Specific**: Configure different apps for different repository patterns
- 🔄 **Git Integration**: Seamless git credential helper integration
- 💾 **Token Caching**: Automatic JWT token generation and caching
- 🔒 **Security**: Proper private key permission validation and secure token handling
- 🏢 **Enterprise Ready**: Multi-organization and enterprise GitHub support

## Installation

```bash
gh extension install wherka-ama/gh-app-auth
```

## Quick Start

1. **Create a GitHub App** in your organization settings
2. **Download the private key** file
3. **Configure the extension**:

```bash
gh app-auth setup \
  --app-id 123456 \
  --key-file ~/.ssh/my-app.private-key.pem \
  --patterns "github.com/myorg/*"
```

4. **Set up git credential helper**:

```bash
git config --global credential."https://github.com/myorg".helper "app-auth git-credential"
```

5. **Test and use**:

```bash
# Test authentication
gh app-auth test --repo github.com/myorg/private-repo

# Use git normally - now uses GitHub App authentication
git clone https://github.com/myorg/private-repo.git
```

## Commands

- `gh app-auth setup` - Configure GitHub App authentication
- `gh app-auth list` - List configured GitHub Apps
- `gh app-auth remove` - Remove GitHub App configuration
- `gh app-auth test` - Test authentication for a repository
- `gh app-auth git-credential` - Git credential helper (internal)

## Use Cases

### CI/CD Pipelines
```yaml
steps:
  - name: Setup GitHub App Auth
    run: |
      gh extension install wherka-ama/gh-app-auth
      gh app-auth setup --app-id ${{ secrets.GITHUB_APP_ID }} --key-file <(echo "${{ secrets.GITHUB_APP_PRIVATE_KEY }}")
```

### Multi-Organization Access
```bash
# Configure different apps for different organizations
gh app-auth setup --app-id 111111 --patterns "github.com/org1/*" --key-file org1-key.pem
gh app-auth setup --app-id 222222 --patterns "github.com/org2/*" --key-file org2-key.pem
```

### Enterprise GitHub
```bash
# Configure for GitHub Enterprise
gh app-auth setup \
  --app-id 123456 \
  --key-file enterprise-app.pem \
  --patterns "github.example.com/corp/*"
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration Reference](docs/configuration.md)
- [Security Considerations](docs/security.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Architecture Overview](docs/architecture.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and contribution guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
